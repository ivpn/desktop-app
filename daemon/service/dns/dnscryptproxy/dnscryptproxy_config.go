//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
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

package dnscryptproxy

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

type Config struct {
	listen_address net.IP
	listen_port    byte

	// Resolvers to use with this instance
	resolvers []Resolver
}

// ID - return unique identifier for this configuration
// based on listen address (using only alphanumeric characters and '_')
func (c Config) ID() string {
	s := c.listen_address.String()
	return nonAlphanumericRegex.ReplaceAllString(s, "_")
}

func (c Config) ListenAddress() net.IP {
	return c.listen_address
}

func NewConfig(listenAddress net.IP, resolvers []Resolver) (*Config, error) {
	if listenAddress == nil || listenAddress.IsUnspecified() {
		return nil, fmt.Errorf("invalid listen address")
	}
	if len(resolvers) == 0 {
		return nil, fmt.Errorf("no DNS resolvers defined")
	}

	return &Config{
		listen_address: listenAddress,
		listen_port:    53,
		resolvers:      resolvers,
	}, nil
}

// Save updates the dnscrypt-proxy template file with resolver configuration
// and writes the result to the specified output file.
// The implementation works by replacing specific placeholder lines in the template.
//
// TODO: !IMPORTANT! This implementation have to be reviewed if the template file format is changed !!!
func (c *Config) Save(configFileTemplate, configFileOut, logFilePath string) error {
	if _, err := os.Stat(configFileTemplate); err != nil {
		return err
	}

	input, err := os.ReadFile(configFileTemplate)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	out := strings.Builder{}

	currentConfBlock := ""

	isUpdated_server_names := false
	isUpdated_static_myserver := false
	isUpdated_stamp := false
	isUpdated_listen_address := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// look for specific markers in template file to replace them with actual configuration
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentConfBlock = line
		}

		if strings.HasPrefix(line, "# server_names = ") && currentConfBlock == "" {
			server_names := make([]string, 0, len(c.resolvers))
			for _, svr := range c.resolvers {
				configSvrName := svr.ID()
				server_names = append(server_names, fmt.Sprintf("'%s'", configSvrName))
			}
			out.WriteString(fmt.Sprintf("server_names = [%s]\n", strings.Join(server_names, ", ")))
			isUpdated_server_names = true

		} else if currentConfBlock == "[static]" && strings.HasPrefix(line, "# [static.myserver]") {
			for _, svr := range c.resolvers {
				out.WriteString(fmt.Sprintf("\n[static.%s]\n", svr.ID()))
				out.WriteString(fmt.Sprintf("stamp = '%s'\n", svr.stamp))
			}
			isUpdated_static_myserver = true
			isUpdated_stamp = true

		} else if strings.HasPrefix(line, "listen_addresses =") && currentConfBlock == "" {
			out.WriteString(fmt.Sprintf("listen_addresses = ['%s:%d']\n", c.listen_address.String(), c.listen_port))
			isUpdated_listen_address = true

		} else if currentConfBlock == "[static]" && strings.HasPrefix(line, "#") && strings.Contains(line, "stamp =") && isUpdated_static_myserver {
			continue

		} else if strings.HasPrefix(line, "# log_file =") && len(logFilePath) > 0 && currentConfBlock == "" {
			out.WriteString(fmt.Sprintf("log_file = '%s'\n", logFilePath))

		} else {
			out.WriteString(line + "\n")
		}
	}

	if !isUpdated_server_names || !isUpdated_static_myserver || !isUpdated_stamp || !isUpdated_listen_address {
		return fmt.Errorf("failed to update configuration from template file, missing required markers")
	}

	err = os.WriteFile(configFileOut, []byte(out.String()), 0600) // read only for owner
	if err != nil {
		return err
	}

	return nil
}

type Resolver struct {
	serverAddr net.IP
	encryption StampProtoType
	template   string

	stamp string
}

func NewResolver(serverAddr net.IP, encryption StampProtoType, template string) (Resolver, error) {
	if serverAddr == nil || serverAddr.IsUnspecified() {
		return Resolver{}, fmt.Errorf("invalid server address")
	}
	if encryption != StampProtoTypePlain && encryption != StampProtoTypeDoH {
		return Resolver{}, fmt.Errorf("unsupported DNS encryption type %d", encryption)
	}

	stamp, err := createStamp(serverAddr, encryption, template)
	if err != nil {
		return Resolver{}, fmt.Errorf("failed to create server stamp: %w", err)
	}

	resolver := Resolver{
		stamp:      stamp,
		serverAddr: serverAddr,
		encryption: encryption,
		template:   template,
	}

	return resolver, nil
}

// ID - return unique identifier for this resolver based on server address and encryption type
// (using only alphanumeric characters and '_')
func (r Resolver) ID() string {
	s := "svr_" + r.serverAddr.String() + "_" + r.encryption.String()
	return nonAlphanumericRegex.ReplaceAllString(s, "_")
}

func (r Resolver) Stamp() string {
	return r.stamp
}

// createStamp - create server stamp from server address and encryption type
// For DNS-over-HTTPS the 'template' parameter must contain full URL to DoH endpoint
// (for example: "https://dns.example.com/dns-query")
func createStamp(serverAddr net.IP, encryption StampProtoType, template string) (string, error) {
	if encryption == StampProtoTypePlain {
		stamp := ServerStamp{ServerAddrStr: serverAddr.String(), Proto: StampProtoTypePlain}
		return stamp.String(), nil
	}

	if encryption != StampProtoTypeDoH {
		return "", fmt.Errorf("unsupported DNS encryption type %d", encryption)
	}

	template = strings.TrimSpace(template)
	if len(template) == 0 {
		return "", fmt.Errorf("empty template for DoH server")
	}

	u, err := url.Parse(template)
	if err != nil {
		return "", err
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("bad template URL scheme: %q", u.Scheme)
	}

	stamp := ServerStamp{
		ServerAddrStr: serverAddr.String(),
		Proto:         StampProtoTypeDoH,

		// DoH specific fields:
		Path:         u.Path,
		ProviderName: u.Hostname(),
	}

	//stamp.Props |= dnscryptproxy.ServerInformalPropertyDNSSEC
	//stamp.Props |= dnscryptproxy.ServerInformalPropertyNoLog
	//stamp.Props |= dnscryptproxy.ServerInformalPropertyNoFilter

	return stamp.String(), nil
}
