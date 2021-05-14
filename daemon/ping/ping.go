//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

// Package ping is an ICMP ping library seeking to emulate the unix "ping"
// command.
//
// Original sources: https://github.com/sparrc/go-ping
// Minor fixes added to prevent deadlocks
//
package ping

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	timeSliceLength  = 8
	trackerLength    = 8
	protocolICMP     = 1
	protocolIPv6ICMP = 58
)

var (
	ipv4Proto = map[string]string{"ip": "ip4:icmp", "udp": "udp4"}
	ipv6Proto = map[string]string{"ip": "ip6:ipv6-icmp", "udp": "udp6"}
)

// NewPinger returns a new Pinger struct pointer
func NewPinger(addr string) (*Pinger, error) {
	ipaddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return nil, err
	}

	var ipv4 bool
	if isIPv4(ipaddr.IP) {
		ipv4 = true
	} else if isIPv6(ipaddr.IP) {
		ipv4 = false
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Pinger{
		ipaddr:   ipaddr,
		addr:     addr,
		Interval: time.Second,
		Timeout:  time.Second * 100000,
		Count:    -1,
		id:       r.Intn(math.MaxInt16),
		network:  "udp",
		ipv4:     ipv4,
		Size:     timeSliceLength,
		Tracker:  r.Int63n(math.MaxInt64),
		done:     make(chan bool),
	}, nil
}

// Pinger represents ICMP packet sender/receiver
type Pinger struct {
	// Interval is the wait time between each packet send. Default is 1s.
	Interval time.Duration

	// Timeout specifies a timeout before ping exits, regardless of how many
	// packets have been received.
	Timeout time.Duration

	// Count tells pinger to stop after sending (and receiving) Count echo
	// packets. If this option is not specified, pinger will operate until
	// interrupted.
	Count int

	// Debug runs in debug mode
	Debug bool

	// Number of packets sent
	PacketsSent int

	// Number of packets received
	PacketsRecv int

	// rtts is all of the Rtts
	rtts []time.Duration

	// OnRecv is called when Pinger receives and processes a packet
	OnRecv func(*Packet)

	// OnFinish is called when Pinger exits
	OnFinish func(*Statistics)

	// Size of packet being sent
	Size int

	// Tracker: Used to uniquely identify packet when non-priviledged
	Tracker int64

	// Source is the source IP address
	Source string

	// stop chan bool
	done chan bool

	ipaddr *net.IPAddr
	addr   string

	ipv4     bool
	size     int
	id       int
	sequence int
	network  string
}

type packet struct {
	bytes  []byte
	nbytes int
	ttl    int
}

// Packet represents a received and processed ICMP echo packet.
type Packet struct {
	// Rtt is the round-trip time it took to ping.
	Rtt time.Duration

	// IPAddr is the address of the host being pinged.
	IPAddr *net.IPAddr

	// Addr is the string address of the host being pinged.
	Addr string

	// NBytes is the number of bytes in the message.
	Nbytes int

	// Seq is the ICMP sequence number.
	Seq int

	// TTL is the Time To Live on the packet.
	Ttl int
}

// Statistics represent the stats of a currently running or finished
// pinger operation.
type Statistics struct {
	// PacketsRecv is the number of packets received.
	PacketsRecv int

	// PacketsSent is the number of packets sent.
	PacketsSent int

	// PacketLoss is the percentage of packets lost.
	PacketLoss float64

	// IPAddr is the address of the host being pinged.
	IPAddr *net.IPAddr

	// Addr is the string address of the host being pinged.
	Addr string

	// Rtts is all of the round-trip times sent via this pinger.
	Rtts []time.Duration

	// MinRtt is the minimum round-trip time sent via this pinger.
	MinRtt time.Duration

	// MaxRtt is the maximum round-trip time sent via this pinger.
	MaxRtt time.Duration

	// AvgRtt is the average round-trip time sent via this pinger.
	AvgRtt time.Duration

	// StdDevRtt is the standard deviation of the round-trip times sent via
	// this pinger.
	StdDevRtt time.Duration
}

// SetIPAddr sets the ip address of the target host.
func (p *Pinger) SetIPAddr(ipaddr *net.IPAddr) {
	var ipv4 bool
	if isIPv4(ipaddr.IP) {
		ipv4 = true
	} else if isIPv6(ipaddr.IP) {
		ipv4 = false
	}

	p.ipaddr = ipaddr
	p.addr = ipaddr.String()
	p.ipv4 = ipv4
}

// IPAddr returns the ip address of the target host.
func (p *Pinger) IPAddr() *net.IPAddr {
	return p.ipaddr
}

// SetAddr resolves and sets the ip address of the target host, addr can be a
// DNS name like "www.google.com" or IP like "127.0.0.1".
func (p *Pinger) SetAddr(addr string) error {
	ipaddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return err
	}

	p.SetIPAddr(ipaddr)
	p.addr = addr
	return nil
}

// Addr returns the string ip address of the target host.
func (p *Pinger) Addr() string {
	return p.addr
}

// SetPrivileged sets the type of ping pinger will send.
// false means pinger will send an "unprivileged" UDP ping.
// true means pinger will send a "privileged" raw ICMP ping.
// NOTE: setting to true requires that it be run with super-user privileges.
func (p *Pinger) SetPrivileged(privileged bool) {
	if privileged {
		p.network = "ip"
	} else {
		p.network = "udp"
	}
}

// Privileged returns whether pinger is running in privileged mode.
func (p *Pinger) Privileged() bool {
	return p.network == "ip"
}

// Run runs the pinger. This is a blocking function that will exit when it's
// done. If Count or Interval are not specified, it will run continuously until
// it is interrupted.
func (p *Pinger) Run() {
	p.run()
}

func (p *Pinger) run() {
	var conn *icmp.PacketConn
	if p.ipv4 {
		if conn = p.listen(ipv4Proto[p.network]); conn == nil {
			return
		}
		conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)
	} else {
		if conn = p.listen(ipv6Proto[p.network]); conn == nil {
			return
		}
		conn.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true)
	}
	defer conn.Close()
	defer p.finish()

	var wg sync.WaitGroup
	recv := make(chan *packet, 5)
	defer close(recv)
	wg.Add(1)
	go p.recvICMP(conn, recv, &wg)

	err := p.sendICMP(conn)
	if err != nil {
		fmt.Println(err.Error())
	}

	timeout := time.NewTicker(p.Timeout)
	defer timeout.Stop()
	interval := time.NewTicker(p.Interval)
	defer interval.Stop()

	for {
		select {
		case <-p.done:
			wg.Wait()
			return
		case <-timeout.C:
			close(p.done)
			wg.Wait()
			return
		case <-interval.C:
			if p.Count > 0 && p.PacketsSent >= p.Count {
				continue
			}
			err = p.sendICMP(conn)
			if err != nil {
				fmt.Println("FATAL: ", err.Error())
			}
		case r := <-recv:
			err := p.processPacket(r)
			if err != nil {
				fmt.Println("FATAL: ", err.Error())
			}
		}
		if p.Count > 0 && p.PacketsRecv >= p.Count {
			close(p.done)
			wg.Wait()
			return
		}
	}
}

func (p *Pinger) Stop() {
	close(p.done)
}

func (p *Pinger) finish() {
	handler := p.OnFinish
	if handler != nil {
		s := p.Statistics()
		handler(s)
	}
}

// Statistics returns the statistics of the pinger. This can be run while the
// pinger is running or after it is finished. OnFinish calls this function to
// get it's finished statistics.
func (p *Pinger) Statistics() *Statistics {
	loss := float64(p.PacketsSent-p.PacketsRecv) / float64(p.PacketsSent) * 100
	var min, max, total time.Duration
	if len(p.rtts) > 0 {
		min = p.rtts[0]
		max = p.rtts[0]
	}
	for _, rtt := range p.rtts {
		if rtt < min {
			min = rtt
		}
		if rtt > max {
			max = rtt
		}
		total += rtt
	}
	s := Statistics{
		PacketsSent: p.PacketsSent,
		PacketsRecv: p.PacketsRecv,
		PacketLoss:  loss,
		Rtts:        p.rtts,
		Addr:        p.addr,
		IPAddr:      p.ipaddr,
		MaxRtt:      max,
		MinRtt:      min,
	}
	if len(p.rtts) > 0 {
		s.AvgRtt = total / time.Duration(len(p.rtts))
		var sumsquares time.Duration
		for _, rtt := range p.rtts {
			sumsquares += (rtt - s.AvgRtt) * (rtt - s.AvgRtt)
		}
		s.StdDevRtt = time.Duration(math.Sqrt(
			float64(sumsquares / time.Duration(len(p.rtts)))))
	}
	return &s
}

func (p *Pinger) recvICMP(
	conn *icmp.PacketConn,
	recv chan<- *packet,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	for {
		select {
		case <-p.done:
			return
		default:
			bytes := make([]byte, 512)
			conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
			var n, ttl int
			var err error
			if p.ipv4 {
				var cm *ipv4.ControlMessage
				n, cm, _, err = conn.IPv4PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.TTL
				}
			} else {
				var cm *ipv6.ControlMessage
				n, cm, _, err = conn.IPv6PacketConn().ReadFrom(bytes)
				if cm != nil {
					ttl = cm.HopLimit
				}
			}
			if err != nil {
				if neterr, ok := err.(*net.OpError); ok {
					if neterr.Timeout() {
						// Read timeout
						continue
					} else {
						close(p.done)
						return
					}
				}
			}

			select {
			case <-p.done:
				return
			case recv <- &packet{bytes: bytes, nbytes: n, ttl: ttl}:
			}
		}
	}
}

func (p *Pinger) processPacket(recv *packet) error {
	receivedAt := time.Now()
	var proto int
	if p.ipv4 {
		proto = protocolICMP
	} else {
		proto = protocolIPv6ICMP
	}

	var m *icmp.Message
	var err error
	if m, err = icmp.ParseMessage(proto, recv.bytes); err != nil {
		return fmt.Errorf("error parsing icmp message: %s", err.Error())
	}

	if m.Type != ipv4.ICMPTypeEchoReply && m.Type != ipv6.ICMPTypeEchoReply {
		// Not an echo reply, ignore it
		return nil
	}

	outPkt := &Packet{
		Nbytes: recv.nbytes,
		IPAddr: p.ipaddr,
		Addr:   p.addr,
		Ttl:    recv.ttl,
	}

	switch pkt := m.Body.(type) {
	case *icmp.Echo:
		// If we are privileged, we can match icmp.ID
		if p.network == "ip" {
			// Check if reply from same ID
			if pkt.ID != p.id {
				return nil
			}
		}

		if len(pkt.Data) < timeSliceLength+trackerLength {
			return fmt.Errorf("insufficient data received; got: %d %v",
				len(pkt.Data), pkt.Data)
		}

		tracker := bytesToInt(pkt.Data[timeSliceLength:])
		timestamp := bytesToTime(pkt.Data[:timeSliceLength])

		if tracker != p.Tracker {
			return nil
		}

		outPkt.Rtt = receivedAt.Sub(timestamp)
		outPkt.Seq = pkt.Seq
		p.PacketsRecv++
	default:
		// Very bad, not sure how this can happen
		return fmt.Errorf("invalid ICMP echo reply; type: '%T', '%v'", pkt, pkt)
	}

	p.rtts = append(p.rtts, outPkt.Rtt)
	handler := p.OnRecv
	if handler != nil {
		handler(outPkt)
	}

	return nil
}

func (p *Pinger) sendICMP(conn *icmp.PacketConn) error {
	var typ icmp.Type
	if p.ipv4 {
		typ = ipv4.ICMPTypeEcho
	} else {
		typ = ipv6.ICMPTypeEchoRequest
	}

	var dst net.Addr = p.ipaddr
	if p.network == "udp" {
		dst = &net.UDPAddr{IP: p.ipaddr.IP, Zone: p.ipaddr.Zone}
	}

	t := append(timeToBytes(time.Now()), intToBytes(p.Tracker)...)
	if remainSize := p.Size - timeSliceLength - trackerLength; remainSize > 0 {
		t = append(t, bytes.Repeat([]byte{1}, remainSize)...)
	}

	body := &icmp.Echo{
		ID:   p.id,
		Seq:  p.sequence,
		Data: t,
	}

	msg := &icmp.Message{
		Type: typ,
		Code: 0,
		Body: body,
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	for {
		if _, err := conn.WriteTo(msgBytes, dst); err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
		}
		p.PacketsSent++
		p.sequence++
		break
	}

	return nil
}

func (p *Pinger) listen(netProto string) *icmp.PacketConn {
	conn, err := icmp.ListenPacket(netProto, p.Source)
	if err != nil {
		fmt.Printf("Error listening for ICMP packets: %s\n", err.Error())
		close(p.done)
		return nil
	}
	return conn
}

func bytesToTime(b []byte) time.Time {
	var nsec int64
	for i := uint8(0); i < 8; i++ {
		nsec += int64(b[i]) << ((7 - i) * 8)
	}
	return time.Unix(nsec/1000000000, nsec%1000000000)
}

func isIPv4(ip net.IP) bool {
	return len(ip.To4()) == net.IPv4len
}

func isIPv6(ip net.IP) bool {
	return len(ip) == net.IPv6len
}

func timeToBytes(t time.Time) []byte {
	nsec := t.UnixNano()
	b := make([]byte, 8)
	for i := uint8(0); i < 8; i++ {
		b[i] = byte((nsec >> ((7 - i) * 8)) & 0xff)
	}
	return b
}

func bytesToInt(b []byte) int64 {
	return int64(binary.BigEndian.Uint64(b))
}

func intToBytes(tracker int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(tracker))
	return b
}
