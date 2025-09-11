//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2025 IVPN Limited.
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

package dns

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

type DnsEncryption int

const (
	EncryptionNone         DnsEncryption = 0
	EncryptionDnsOverTls   DnsEncryption = 1
	EncryptionDnsOverHttps DnsEncryption = 2
)

// DnsServerConfig represents a single DNS server configuration
type DnsServerConfig struct {
	Address    string        // IP address of the DNS server
	Encryption DnsEncryption // Encryption type (None, DoH, DoT)
	Template   string        // DoH/DoT template
}

func (d DnsServerConfig) Equal(x DnsServerConfig) bool {
	if d.Encryption != x.Encryption ||
		d.Template != x.Template ||
		d.Address != x.Address {
		return false
	}
	return true
}

func (d DnsServerConfig) IsIPv6() bool {
	ip := d.Ip()
	if ip == nil {
		return false
	}
	return ip.To4() == nil
}

func (d DnsServerConfig) Ip() net.IP {
	return net.ParseIP(d.Address)
}

func (d DnsServerConfig) IsEmpty() bool {
	if strings.TrimSpace(d.Address) == "" {
		return true
	}
	ip := d.Ip()
	if ip == nil || ip.Equal(net.IPv4zero) || ip.Equal(net.IPv4bcast) || ip.Equal(net.IPv6zero) {
		return true
	}
	return false
}

func (d *DnsServerConfig) ValidateAndNormalize() error {
	d.Address = strings.TrimSpace(d.Address)
	d.Template = strings.TrimSpace(d.Template)

	// Allow empty configs (they represent "no DNS server")
	if d.IsEmpty() {
		return nil
	}

	// Validate IP address for non-empty configs
	if ip := net.ParseIP(d.Address); ip == nil {
		return fmt.Errorf("invalid IP address: %s", d.Address)
	}

	// Validate encryption-specific requirements
	switch d.Encryption {
	case EncryptionNone:
		if len(d.Template) != 0 {
			return fmt.Errorf("template should be empty for 'None' encryption")
		}
	case EncryptionDnsOverHttps:
		if len(d.Template) == 0 {
			return fmt.Errorf("template URL is required for DoH")
		}
		_, err := url.Parse(d.Template)
		if err != nil {
			return fmt.Errorf("invalid template URL for DoH: %w", err)
		}
	case EncryptionDnsOverTls:
		if len(d.Template) == 0 { // TODO: any other validation for DoT template?
			return fmt.Errorf("template is required for DoT")
		}
	default:
		return fmt.Errorf("unsupported encryption type %d", d.Encryption)
	}

	return nil
}

func (d DnsServerConfig) InfoString() string {
	if d.IsEmpty() {
		return "<none>"
	}
	host := strings.TrimSpace(d.Address)
	template := strings.TrimSpace(d.Template)

	switch d.Encryption {
	case EncryptionDnsOverTls:
		return host + " (DoT " + template + ")"
	case EncryptionDnsOverHttps:
		return host + " (DoH " + template + ")"
	case EncryptionNone:
		return host
	default:
		return host + " (UNKNOWN ENCRYPTION)"
	}
}

// DnsSettings represents the DNS configuration
// It can include multiple DNS servers with different encryption methods
type DnsSettings struct {
	// List of DNS servers specified in order of preference
	Servers []DnsServerConfig
	// Internal metadata about the DNS configuration
	metadata DnsMetadata
}

type DnsMetadata struct {
	IsInternalDnsConfig bool // FALSE if DNS settings are custom (defined by user)
}

func (d DnsSettings) Metadata() DnsMetadata {
	return d.metadata
}

// Create DnsSettings object with no encryption single DNS server
func DnsSettingsCreate(ip net.IP) DnsSettings {
	if ip == nil {
		return DnsSettings{}
	}
	return DnsSettings{Servers: []DnsServerConfig{{Address: ip.String()}}, metadata: DnsMetadata{IsInternalDnsConfig: true}}
}

// Equal - compares two DnsSettings objects for equality
// Returns TRUE if both objects are equal
// (including the order of DNS servers in the list)
func (d DnsSettings) Equal(x DnsSettings) bool {
	if len(d.Servers) != len(x.Servers) {
		return false
	}
	for i := range d.Servers {
		if !d.Servers[i].Equal(x.Servers[i]) {
			return false
		}
	}
	return true
}

func (d DnsSettings) IsEmpty() bool {
	if len(d.Servers) == 0 {
		return true
	}
	for _, srv := range d.Servers {
		if !srv.IsEmpty() {
			return false
		}
	}
	return true
}

// UseEncryption - returns TRUE if at least one DNS server uses encryption (DoH or DoT)
func (d DnsSettings) UseEncryption() bool {
	for _, srv := range d.Servers {
		if srv.Encryption != EncryptionNone && !srv.IsEmpty() {
			return true
		}
	}
	return false
}

// GetUnencryptedServersAddresses - returns list of IP addresses of DNS servers without encryption
func (d DnsSettings) GetUnencryptedServersAddresses() []net.IP {
	ips := make([]net.IP, 0, len(d.Servers))
	for _, srv := range d.Servers {
		if srv.Encryption == EncryptionNone && !srv.IsEmpty() {
			ip := srv.Ip()
			if ip != nil {
				ips = append(ips, ip)
			}
		}
	}
	return ips
}

func (d DnsSettings) InfoString() string {
	if d.IsEmpty() {
		return "<none>"
	}
	var sb strings.Builder
	for i, srv := range d.Servers {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(srv.InfoString())
	}
	return sb.String()
}

func (d DnsSettings) ValidateAndNormalize() error {
	if d.IsEmpty() {
		return nil
	}

	for i, srv := range d.Servers {
		if err := srv.ValidateAndNormalize(); err != nil {
			return fmt.Errorf("DNS server %d: %w", i+1, err)
		}
	}
	return nil
}
