package openvpn

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/netinfo"
	"github.com/ivpn/desktop-app-daemon/service/platform"
)

// ConnectionParams represents OpenVPN connection parameters
type ConnectionParams struct {
	username          string
	password          string
	multihopExitSrvID string
	tcp               bool
	hostPort          int
	hostIPs           []net.IP
	proxyType         string
	proxyAddress      net.IP
	proxyPort         int
	proxyUsername     string
	proxyPassword     string
}

// SetCredentials update WG credentials
func (c *ConnectionParams) SetCredentials(username, password string) {
	c.password = password
	c.username = username

	// MultiHop configuration is based just by adding "@exit_server_id" to the end of username
	// And forwarding this info on server
	if len(c.multihopExitSrvID) > 0 {
		c.username = fmt.Sprintf("%s@%s", username, c.multihopExitSrvID)
	}
}

// CreateConnectionParams creates OpenVPN connection parameters object
func CreateConnectionParams(
	multihopExitSrvID string,
	tcp bool,
	hostPort int,
	hostIPs []net.IP,
	proxyType string,
	proxyAddress net.IP,
	proxyPort int,
	proxyUsername string,
	proxyPassword string) ConnectionParams {

	return ConnectionParams{
		multihopExitSrvID: multihopExitSrvID,
		tcp:               tcp,
		hostPort:          hostPort,
		hostIPs:           hostIPs,
		proxyType:         proxyType,
		proxyAddress:      proxyAddress,
		proxyPort:         proxyPort,
		proxyUsername:     proxyUsername,
		proxyPassword:     proxyPassword}
}

// WriteConfigFile saves OpenVPN connection parameters into a config file
func (c *ConnectionParams) WriteConfigFile(
	filePathToSave string,
	miAddr string,
	miPort int,
	logFile string,
	obfsproxyPort int,
	extraParameters string) error {

	cfg, err := c.generateConfiguration(miAddr, miPort, logFile, obfsproxyPort, extraParameters)
	if err != nil {
		return fmt.Errorf("failed to generate openvpn configuration : %w", err)
	}

	configText := strings.Join(cfg, "\n")

	err = ioutil.WriteFile(filePathToSave, []byte(configText), 0600)
	if err != nil {
		return fmt.Errorf("failed to save OpenVPN configuration into a file: %w", err)
	}

	log.Info("Configuring OpenVPN...\n",
		"=====================\n",
		configText,
		"\n=====================\n")

	return nil
}

func (c *ConnectionParams) generateConfiguration(
	miAddr string,
	miPort int,
	logFile string,
	obfsproxyPort int,
	extraParameters string) (cfg []string, err error) {

	if obfsproxyPort > 0 {
		c.tcp = true
		c.hostPort = platform.ObfsproxyHostPort()
		c.proxyType = "socks"
		c.proxyAddress = net.IPv4(127, 0, 0, 1) // "127.0.0.1"
		c.proxyPort = obfsproxyPort
		c.proxyUsername = ""
		c.proxyPassword = ""
	}

	cfg = make([]string, 0, 32)

	cfg = append(cfg, "client")
	cfg = append(cfg, fmt.Sprintf("management %s %d", miAddr, miPort))
	cfg = append(cfg, "management-client")

	cfg = append(cfg, "management-hold")
	cfg = append(cfg, "auth-user-pass")
	cfg = append(cfg, "auth-nocache")

	cfg = append(cfg, "management-query-passwords")

	cfg = append(cfg, "management-signal")

	// Handshake Window --the TLS - based key exchange must finalize within n seconds of handshake initiation by any peer(default = 60 seconds).
	// If the handshake fails openvpn will attempt to reset our connection with our peer and try again.
	cfg = append(cfg, "hand-window 6")

	// To change default connection-check time - uncomment next two lines:
	cfg = append(cfg, "pull-filter ignore \"ping\"")
	cfg = append(cfg, "keepalive 8 30")

	// proxy
	if c.proxyType == "http" || c.proxyType == "socks" {

		// proxy authentication
		proxyAuthFile := ""
		if c.proxyUsername != "" && c.proxyPassword != "" {
			proxyAuthFile = "\"" + platform.OpenvpnProxyAuthFile() + "\""
			err := ioutil.WriteFile(platform.OpenvpnProxyAuthFile(), []byte(fmt.Sprintf("%s\n%s", c.proxyUsername, c.proxyPassword)), 0600)
			if err != nil {
				log.Error(err)
				return nil, fmt.Errorf("Failed to save file with proxy credentials: %w", err)
			}
		}

		// proxy config
		switch c.proxyType {
		case "http":
			cfg = append(cfg, "http-proxy-retry")
			cfg = append(cfg, fmt.Sprintf("http-proxy %s %d %s", c.proxyAddress.String(), c.proxyPort, proxyAuthFile))
			break
		case "socks":
			cfg = append(cfg, "socks-proxy-retry")
			cfg = append(cfg, fmt.Sprintf("socks-proxy %s %d %s", c.proxyAddress.String(), c.proxyPort, proxyAuthFile))
			break
		}
	}

	if logger.IsEnabled() {
		cfg = append(cfg, fmt.Sprintf(`log "%s"`, logFile))
	}

	cfg = append(cfg, "dev tun")

	if c.tcp {
		cfg = append(cfg, "proto tcp")
	} else {
		cfg = append(cfg, "proto udp")
	}

	if len(c.hostIPs) < 1 {
		return nil, errors.New("unable to connect. Host IP not defined")
	}
	if c.hostPort < 0 || c.hostPort > 65535 {
		return nil, errors.New("unable to connect. Invalid port")
	}

	for _, host := range c.hostIPs {
		cfg = append(cfg, fmt.Sprintf("remote %s %d", host, c.hostPort))
	}

	if len(c.hostIPs) > 1 {
		cfg = append(cfg, "remote-random")
	}

	cfg = append(cfg, "resolv-retry infinite")
	cfg = append(cfg, "nobind")
	cfg = append(cfg, "persist-key")

	if _, err := os.Stat(platform.OpenvpnCaKeyFile()); os.IsNotExist(err) {
		return nil, errors.New("CA certificate not found")
	}
	cfg = append(cfg, fmt.Sprintf("ca \"%s\"", platform.OpenvpnCaKeyFile()))

	if _, err := os.Stat(platform.OpenvpnTaKeyFile()); os.IsNotExist(err) {
		return nil, errors.New("TLS auth key not found")
	}
	cfg = append(cfg, fmt.Sprintf("tls-auth \"%s\" 1", platform.OpenvpnTaKeyFile()))

	cfg = append(cfg, "cipher AES-256-CBC")
	cfg = append(cfg, "remote-cert-tls server")
	cfg = append(cfg, "compress")
	cfg = append(cfg, "verb 4")

	if upCmd := platform.OpenvpnUpScript(); upCmd != "" {
		cfg = append(cfg, "up \""+upCmd+"\"")
	}
	if downCmd := platform.OpenvpnDownScript(); downCmd != "" {
		cfg = append(cfg, "down \""+downCmd+"\"")
	}

	cfg = append(cfg, "script-security 2")

	if c.proxyAddress != nil && (c.proxyType == "http" || c.proxyType == "socks") {

		localGatewayAddress, err := netinfo.DefaultGatewayIP()
		if err != nil {
			return nil, fmt.Errorf("failed to get local gateway: %w", err)
		}

		if localGatewayAddress == nil {
			return nil, errors.New("internal error: LocalGatewayAdress not defined. Unable to generate OpenVPN configuration")
		}

		if c.proxyAddress.Equal(net.IPv4(127, 0, 0, 1)) {
			for _, addr := range c.hostIPs {
				cfg = append(cfg, fmt.Sprintf("route %s 255.255.255.255 %s", addr.String(), localGatewayAddress.String()))
			}
		} else {
			cfg = append(cfg, fmt.Sprintf("route %s 255.255.255.255 %s", c.proxyAddress, localGatewayAddress.String()))
		}
	}

	cfg, err = addUserDefinedParameters(cfg, extraParameters)
	if err != nil {
		return nil, fmt.Errorf("failed to add user-defined parameters: %w", err)
	}

	return cfg, nil
}

// merge current parameters with user-defined parameters
func addUserDefinedParameters(currParams []string, userParams string) ([]string, error) {
	if len(userParams) <= 0 {
		return currParams, nil
	}

	// check OpenVPN extra parameters defined by user
	// Some parameters can be deprecated (e.g. parameters which can execute external command)
	if err := isUserParametersAllowed(userParams); err != nil {
		return nil, err
	}

	// loop trough all extraParameters defined by user
	// (looking if user-defined parameters overlap an existing parameters)
	tmpCfg := make([]string, 1)
	userLines := strings.Split(userParams, "\n")

	for _, cfgLine := range currParams {
		cfgParam := getParamFromConfigLine(cfgLine)
		cfgLineToSave := cfgLine

		for i, userLine := range userLines {
			userParam := getParamFromConfigLine(userLine)

			if len(userParam) > 0 && cfgParam == userParam {
				cfgLineToSave = userLine
				userLines[i] = ""
				break
			}
		}

		tmpCfg = append(tmpCfg, cfgLineToSave)
	}

	for _, userLine := range userLines {
		if len(userLine) > 0 {
			tmpCfg = append(tmpCfg, userLine)
		}
	}

	return tmpCfg, nil
}

// check if user parameter is allowed
func isUserParametersAllowed(userParameters string) error {

	lines := strings.Split(userParameters, "\n")

	for _, line := range lines {

		command := getParamFromConfigLine(line)
		if command == "" {
			continue
		}

		_, ok := _AllowedOpenvpnParams[command]
		if ok == false {
			return errors.New(fmt.Sprint("Parameter '", command, "' is deprecated"))
		}
	}

	return nil
}

func getParamFromConfigLine(line string) string {
	line = strings.TrimLeft(line, " \t")
	words := strings.Fields(line)

	if len(words) <= 0 || len(words[0]) <= 0 {
		return ""
	}
	// ignore comments
	if words[0][0] == '#' || words[0][0] == ';' {
		return ""
	}

	return strings.ToLower(words[0])
}
