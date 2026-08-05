package wireguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/ping"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// WireGuard connection health check logic:
//  1. Poll peer RX/TX counters every second.
//  2. If TX grows and RX does not respond within ResponseTimeout, start active probing.
//  3. If RX stays idle for IdleTimeout, also start active probing.
//  4. While probing, send ICMP echo requests to the tunnel peer from the tunnel source IP.
//  5. If no RX activity appears for ConnDeadAfterPingTimeout, report unhealthy once.
//     As soon as RX activity resumes, report healthy once and stop probing.
//
// This checker is informational only. It never disconnects the tunnel and is intended to
// surface temporary dead-path conditions while letting WireGuard recover automatically.

const (
	// Timeout to detect lack of response after sending data.
	// If TX activity is detected but no RX for this duration → start active probing (pinging).
	ResponseTimeout = 5 * time.Second
	// Timeout to detect a completely idle connection (no RX for a long time), regardless of TX activity.
	// Note: WireGuard handshake interval is 120s, so we can use this as a threshold for idle timeout.
	IdleTimeout = 120 * time.Second
	// If we are actively probing (pinging) and no response for this timeout → declare connection dead
	ConnDeadAfterPingTimeout = 10 * time.Second
)

// healthChecker monitors WireGuard tunnel liveness by watching the peer's RX/TX byte
// counters. When the connection is suspected dead (no RX after probing), it calls
// onHealthChanged(false). When RX activity resumes after a dead period, it calls
// onHealthChanged(true). The checker never disconnects - WireGuard recovers on its own.
//
// Each healthChecker instance is single-use: create a new one per start/stop cycle so a
// goroutine's self-cleanup can never affect a later generation (see startHealthCheck).
type healthChecker struct {
	tunnelName    string
	hostLocalIP   net.IP
	clientLocalIP net.IP

	ctx    context.Context
	cancel context.CancelFunc

	log *logger.Logger
}

func newHealthChecker(tunnelName string, hostLocalIP, clientLocalIP net.IP) *healthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &healthChecker{
		log:           logger.NewLogger("wgHlth"),
		tunnelName:    tunnelName,
		hostLocalIP:   hostLocalIP,
		clientLocalIP: clientLocalIP,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Stop requests run() to return. Safe to call multiple times.
func (h *healthChecker) Stop() {
	h.cancel()
}

// run blocks, monitoring tunnel liveness, until Stop() is called or an unrecoverable error occurs.
func (h *healthChecker) run(onHealthChanged func(ctx context.Context, isHealthy bool)) error {

	if onHealthChanged == nil {
		return fmt.Errorf("onHealthChanged callback is nil")
	}

	// init WG-control object
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("failed to initialize Connection Status Checker: %w", err)
	}
	defer client.Close()

	now := time.Now()
	var (
		lastRxTime         time.Time
		lastRxBytes        int64
		lastTxTimeNoResp   time.Time
		lastTxBytes        int64
		waitingForResponse bool
		probingStartTime   time.Time
		isConnectionDead   bool
	)
	lastRxTime = now
	lastTxTimeNoResp = now

	h.log.Info("Connection Status Checker started")
	defer h.log.Info("Connection Status Checker stopped")

	for {
		select {
		case <-h.ctx.Done():
			return nil
		case <-time.After(1 * time.Second):
		}

		peer, err := getPeer(h.ctx, client, h.tunnelName)
		if err != nil {
			select {
			case <-h.ctx.Done():
				return nil
			default:
			}
			return fmt.Errorf("failed to get peer: %w", err)
		}

		now = time.Now()
		//h.log.Debug(fmt.Sprintf("Tx: %v Rx: %v (+Tx: %v +Rx: %v) Handshake: %v ago", lastTxBytes, lastRxBytes, peer.TransmitBytes-lastTxBytes, peer.ReceiveBytes-lastRxBytes, now.Sub(peer.LastHandshakeTime)))

		// Transmitted bytes calculation
		if peer.TransmitBytes != lastTxBytes {
			lastTxBytes = peer.TransmitBytes
			if !waitingForResponse {
				lastTxTimeNoResp = now
				waitingForResponse = true
			}
		}

		// Received bytes calculation
		if peer.ReceiveBytes != lastRxBytes {
			lastRxBytes = peer.ReceiveBytes
			lastRxTime = now
			waitingForResponse = false

			if !probingStartTime.IsZero() {
				h.log.Info("Rx activity detected, stopping active probing")
				probingStartTime = time.Time{}
			}

			if isConnectionDead {
				isConnectionDead = false
				onHealthChanged(h.ctx, true)
			}

			continue // received data - means connection is alive.
		}

		// Check response timeout
		if waitingForResponse && now.After(lastTxTimeNoResp.Add(ResponseTimeout)) && probingStartTime.IsZero() {
			h.log.Debug(fmt.Sprintf("No response received for %v seconds, starting active probing", ResponseTimeout.Seconds()))
			probingStartTime = now
		}

		// Check Idle timeout
		if now.After(lastRxTime.Add(IdleTimeout)) && probingStartTime.IsZero() {
			h.log.Debug(fmt.Sprintf("No Rx activity for %v seconds, starting active probing", IdleTimeout.Seconds()))
			probingStartTime = now
		}

		if !probingStartTime.IsZero() {
			// Declare connection dead once when the probing timeout is exceeded.
			if !isConnectionDead && now.After(probingStartTime.Add(ConnDeadAfterPingTimeout)) {
				isConnectionDead = true
				h.log.Debug(fmt.Sprintf("No response for %v seconds, declaring connection dead", ConnDeadAfterPingTimeout.Seconds()))
				onHealthChanged(h.ctx, false)
			}

			// Keep pinging regardless of dead/alive state so recovery can be detected via RX above.
			// Actively probe tunnel liveness by pinging the server through the WG interface.
			// Source IP is set to the client's tunnel IP to force traffic through the WG interface
			// rather than the default route (important when Inverse Split Tunnel is enabled).
			// The ping reply is not checked here - any resulting RX activity is detected above.
			if err := sendPing(h.hostLocalIP, h.clientLocalIP); err != nil {
				h.log.Error("Failed to send probe ping: ", err)
			}
		}
	}
}

// sendPing sends a single ICMP echo request to the destination IP address, optionally using the specified source IP address.
func sendPing(destIP, sourceIP net.IP) error {
	pinger, err := ping.NewPinger(destIP.String())
	if err != nil {
		return err
	}

	pinger.SetPrivileged(true)
	pinger.Count = 1
	pinger.Timeout = time.Microsecond // packet is sent before the timeout loop; reply is tracked via WG RX counter
	if sourceIP != nil {
		pinger.Source = sourceIP.String()
	}

	return pinger.Run()
}

// getPeer returns the first (and only expected) peer of the named WireGuard device.
// It blocks until the device and peer are available, retrying on:
//   - ErrNotExist: device not yet created (service still starting up) or removed during Pause.
//   - No peers: window between interface creation and peer configuration.
//
// Returns when a peer is found or the context is cancelled.
func getPeer(ctx context.Context, client *wgctrl.Client, devName string) (peer *wgtypes.Peer, retErr error) {
	var dev *wgtypes.Device
	var err error

	for {
		dev, err = client.Device(devName)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		} else {
			if len(dev.Peers) > 0 {
				return &dev.Peers[0], nil // We are using only single peer
			}
		}

		// Wait before next retry
		select {
		case <-time.After(time.Second * 2):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
