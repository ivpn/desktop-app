package wireguard

import (
	"io/ioutil"
	"ivpn/daemon/logger"
	"ivpn/daemon/netinfo"
	"ivpn/daemon/vpn"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("wg")
}

// ConnectionParams contains all information to make new connection
type ConnectionParams struct {
	clientLocalIP    net.IP
	clientPrivateKey string
	hostPort         int
	hostIP           net.IP
	hostPublicKey    string
	hostLocalIP      net.IP
}

// CreateConnectionParams initializing connection parameters object
func CreateConnectionParams(
	clientLocalIP net.IP,
	clientPrivateKey string,
	hostPort int,
	hostIP net.IP,
	hostPublicKey string,
	hostLocalIP net.IP) ConnectionParams {

	return ConnectionParams{
		clientLocalIP:    clientLocalIP,
		clientPrivateKey: clientPrivateKey,
		hostPort:         hostPort,
		hostIP:           hostIP,
		hostPublicKey:    hostPublicKey,
		hostLocalIP:      hostLocalIP}
}

// WireGuard structure represents all data of wireguard connection
type WireGuard struct {
	binaryPath     string
	toolBinaryPath string
	configFilePath string
	connectParams  ConnectionParams
	defGateway     net.IP

	// Must be implemeted (AND USED) in correspond file for concrete platform. Must contain platform-specified properties (or can be empty struct)
	internals internalVariables
}

// NewWireGuardObject creates new wireguard structure
func NewWireGuardObject(wgBinaryPath string, wgToolBinaryPath string, wgConfigFilePath string, connectionParams ConnectionParams) (*WireGuard, error) {

	defaultGwIP, err := netinfo.DefaultGatewayIP()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to determine default gateway IP")
	}

	return &WireGuard{
		binaryPath:     wgBinaryPath,
		toolBinaryPath: wgToolBinaryPath,
		configFilePath: wgConfigFilePath,
		defGateway:     defaultGwIP,
		connectParams:  connectionParams}, nil
}

// DestinationIPs -  Get destination IPs (VPN host server or proxy server IP address)
// This information if required, for example, to allow this address in firewall
func (wg *WireGuard) DestinationIPs() []net.IP {
	return []net.IP{wg.connectParams.hostIP}
}

// Connect - SYNCHRONOUSLY execute openvpn process (wait untill it finished)
func (wg *WireGuard) Connect(stateChan chan<- vpn.StateInfo) error {

	disconnectDescription := ""

	stateChan <- vpn.NewStateInfo(vpn.CONNECTING, "")
	defer func() {
		stateChan <- vpn.NewStateInfo(vpn.DISCONNECTED, disconnectDescription)
	}()

	err := wg.connect(stateChan)

	if err != nil {
		disconnectDescription = err.Error()
	}

	return err
}

// Disconnect stops the connection
func (wg *WireGuard) Disconnect() error {
	return wg.disconnect()
}

// IsPaused checking if we are in paused state
func (wg *WireGuard) IsPaused() bool {
	return wg.isPaused()
}

// Pause doing required operation for Pause (remporary restoring default DNS)
func (wg *WireGuard) Pause() error {
	return wg.pause()
}

// Resume doing required operation for Resume (restores DNS configuration before Pause)
func (wg *WireGuard) Resume() error {
	return wg.resume()
}

// SetManualDNS changes DNS to manual IP
func (wg *WireGuard) SetManualDNS(addr net.IP) error {
	return wg.setManualDNS(addr)
}

// ResetManualDNS restores DNS
func (wg *WireGuard) ResetManualDNS() error {
	return wg.resetManualDNS()
}

func (wg *WireGuard) generateAndSaveConfigFile(cfgFilePath string) error {
	cfg, err := wg.generateConfig()
	if err != nil {
		return errors.Wrap(err, "failed to generate WireGuard configuration")
	}

	// write configuration into temporary file
	configText := strings.Join(cfg, "\n")

	err = ioutil.WriteFile(cfgFilePath, []byte(configText), 0600)
	if err != nil {
		return errors.Wrap(err, "failed to save WireGuard configuration into a file")
	}

	log.Info("WireGuard  configuration:",
		"\n=====================\n",
		strings.ReplaceAll(configText, wg.connectParams.clientPrivateKey, "***"),
		"\n=====================\n")

	return nil
}

func (wg *WireGuard) generateConfig() ([]string, error) {
	listenPort, err := netinfo.GetFreePort()
	if err != nil {
		return nil, errors.Wrap(err, "unable to obtain free local port")
	}

	interfaceCfg := []string{
		"[Interface]",
		"PrivateKey = " + wg.connectParams.clientPrivateKey,
		"ListenPort = " + strconv.Itoa(listenPort)}

	peerCfg := []string{
		"[Peer]",
		"PublicKey = " + wg.connectParams.hostPublicKey,
		"Endpoint = " + wg.connectParams.hostIP.String() + ":" + strconv.Itoa(wg.connectParams.hostPort),
		"PersistentKeepalive = 25",
		// Same as "0.0.0.0/0" but such type of configuration is disabling internal WireGuard-s Firewall
		// It blocks everything except WireGuard traffic.
		// We need to disable WireGurd-s firewall because we have our own implementation of firewall.
		//  For details, refer to WireGuard-windows sources: tunnel\ifaceconfig.go (enableFirewall(...) method)
		"AllowedIPs = 128.0.0.0/1, 0.0.0.0/1"}

	// add some OS-specific configurations (if necessary)
	iCfg, pCgf := wg.getOSSpecificConfigParams()
	interfaceCfg = append(interfaceCfg, iCfg...)
	peerCfg = append(peerCfg, pCgf...)

	return append(interfaceCfg, peerCfg...), nil
}
