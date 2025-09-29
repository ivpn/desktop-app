package dns

import (
	"fmt"
	"net"
	"path"
	"strings"
	"sync"

	"github.com/ivpn/desktop-app/daemon/service/dns/dnscryptproxy"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

var (
	dnsCryptProxyMutex     sync.Mutex
	dnsCryptProxyInstances []*dnscryptproxy.DnsCryptProxy
)

// SetupDnsResolvers initializes DNS configuration:
// - if plain DNS servers are available, return them as is
// - if DoH resolvers available, start dnscrypt-proxy instance(s) and return their local listener addresses
//
// Returns: list of plain DNS servers ready to apply as system resolvers
// (including started local dnscrypt-proxy servers, if any)
func ResolversSetup(dnsCfg DnsSettings) ([]DnsServerConfig, error) {
	// group DNS servers into sets for separate dnscrypt-proxy instances (if needed)
	cfgGroups, err := groupConfigs(dnsCfg.Servers)
	if err != nil {
		return nil, fmt.Errorf("failed to group DNS server configs: %w", err)
	}

	// return slice of DNS server configs
	ret := make([]DnsServerConfig, 0, len(cfgGroups))
	dnsCryptConfigs := make([]*dnscryptproxy.Config, 0, len(cfgGroups))

	for _, resolversGroup := range cfgGroups {
		if len(resolversGroup) == 0 {
			continue
		}
		if resolversGroup[0].Encryption == EncryptionNone {
			// plain DNS server - return as is (only one server must be in this group)
			ret = append(ret, resolversGroup[0])
			continue
		}
		if resolversGroup[0].Encryption != EncryptionDnsOverHttps {
			return nil, fmt.Errorf("unsupported DNS encryption type %d", resolversGroup[0].Encryption)
		}
		// DoH servers - start dnscrypt-proxy instance for this group

		// assign a unique local listening IP for this instance (127.0.0.x)
		localIp := net.IPv4(127, 0, 0, byte(1+len(dnsCryptConfigs)))
		// add local dnscrypt-proxy listener as a system DNS server
		ret = append(ret, DnsServerConfig{Address: localIp.String()})

		cfg, err := createDnscryptProxyConfig(localIp, resolversGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to create dnscrypt-proxy config: %w", err)
		}
		dnsCryptConfigs = append(dnsCryptConfigs, cfg)

	}

	if len(ret) == 0 {
		return nil, fmt.Errorf("no usable DNS servers configured")
	}

	if err := startDnscryptProxyInstances(dnsCryptConfigs); err != nil {
		return nil, fmt.Errorf("failed to start dnscrypt-proxy instances: %w", err)
	}

	return ret, nil
}

// ResolversTeardown stops any running dnscrypt-proxy instances
func ResolversTeardown() error {
	return stopDnscryptProxyInstances()
}

func createDnscryptProxyConfig(listenAddress net.IP, servers []DnsServerConfig) (*dnscryptproxy.Config, error) {
	resolvers := make([]dnscryptproxy.Resolver, 0, len(servers))
	for _, svr := range servers {
		resolver, err := dnscryptproxy.NewResolver(svr.Ip(), dnscryptproxy.StampProtoTypeDoH, svr.Template)
		if err != nil {
			return nil, fmt.Errorf("failed to create resolver for %s: %w", svr.InfoString(), err)
		}
		resolvers = append(resolvers, resolver)
	}
	if len(resolvers) == 0 {
		return nil, fmt.Errorf("no usable DNS servers configured")
	}

	return dnscryptproxy.NewConfig(listenAddress, resolvers)
}

func stopDnscryptProxyInstances() error {
	dnsCryptProxyMutex.Lock()
	defer dnsCryptProxyMutex.Unlock()

	for _, proxy := range dnsCryptProxyInstances {
		if err := proxy.Stop(); err != nil {
			log.Warning("failed to stop dnscrypt-proxy: ", err)
		}
	}
	dnsCryptProxyInstances = nil
	return nil
}

func startDnscryptProxyInstances(configs []*dnscryptproxy.Config) (retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("error starting dnscrypt-proxy instances: %w", retErr)
			stopDnscryptProxyInstances()
		}
	}()

	stopDnscryptProxyInstances()

	dnsCryptProxyMutex.Lock()
	defer dnsCryptProxyMutex.Unlock()

	dnsCryptProxyInstances = make([]*dnscryptproxy.DnsCryptProxy, 0, len(configs))

	binPath, templateFile, configDir, logDir := platform.DnsCryptProxyInfo()
	if len(binPath) == 0 || len(templateFile) == 0 || len(configDir) == 0 {
		return fmt.Errorf("dnscrypt-proxy configuration not defined")
	}

	for _, cfg := range configs {
		uniqNamePostfix := cfg.ID()
		configFileOut := path.Join(configDir, fmt.Sprintf("dnscrypt-proxy-%s.toml", uniqNamePostfix))
		logFilePath := ""
		if len(logDir) > 0 {
			logFilePath = path.Join(logDir, fmt.Sprintf("dnscrypt-proxy-%s.log", uniqNamePostfix))
		}

		// save dnscrypt-proxy configuration file
		if err := cfg.Save(templateFile, configFileOut, logFilePath); err != nil {
			return fmt.Errorf("failed to save dnscrypt-proxy config file: %w", err)
		}

		proxy, err := dnscryptproxy.Start(binPath, configFileOut, logFilePath, cfg.ListenAddress())
		if err != nil {
			return fmt.Errorf("failed to start dnscrypt-proxy: %w", err)
		}
		dnsCryptProxyInstances = append(dnsCryptProxyInstances, proxy)
	}
	return nil
}

// groupConfigs sorts DNS servers into groups based on their type and properties.
// Each group is later represented as a single DNS resolver in the system's DNS settings.
//
// Grouping rules:
//   - Plain DNS servers are each placed in their own group.
//   - DoH (DNS-over-HTTPS) servers are grouped together to be handled by a single dnscrypt-proxy instance.
//   - A new group is created for a DoH server if its URI is already present in the current group.
//     This is necessary because dnscrypt-proxy de-duplicates DoH resolvers with the same URI,
//     which would prevent failover.
//
// Example:
// Input DNS servers (in order):
//   - 8.8.8.8 (DoH, https://dns.google/dns-query)
//   - 9.9.9.9 (DoH, https://dns.quad9.net/dns-query)
//   - 8.8.4.4 (Plain)
//   - 1.2.3.4 (Plain)
//   - 1.1.1.1 (DoH, https://cloudflare-dns.com/dns-query)
//   - 1.0.0.1 (DoH, https://cloudflare-dns.com/dns-query) <- Same URI as 1.1.1.1
//   - 4.4.4.4 (DoH, https://dns.google/dns-query)      <- Same URI as 8.8.8.8
//
// Resulting groups:
//   - [8.8.8.8 (DoH), 9.9.9.9 (DoH)] -> dnscrypt-proxy instance 1
//   - [8.8.4.4 (Plain)]              -> Direct use
//   - [1.2.3.4 (Plain)]              -> Direct use
//   - [1.1.1.1 (DoH)]                -> dnscrypt-proxy instance 2
//   - [1.0.0.1 (DoH), 4.4.4.4 (DoH)] -> dnscrypt-proxy instance 3
func groupConfigs(servers []DnsServerConfig) (groups [][]DnsServerConfig, retErr error) {
	if len(servers) == 0 {
		return nil, fmt.Errorf("no DNS servers provided")
	}

	groups = make([][]DnsServerConfig, 0)
	var (
		processedServers   = make(map[string]struct{}) // track processed servers to skip duplicates
		currentGroup       = make([]DnsServerConfig, 0)
		currentUsedDoHUris = make(map[string]struct{}) // track DoH URIs in current group
	)

	for _, server := range servers {
		server.Template = strings.TrimSpace(server.Template)

		// create a unique key for this server to detect duplicates
		var serverKey string
		if server.Encryption == EncryptionDnsOverHttps {
			serverKey = fmt.Sprintf("doh:%s:%s", server.Address, server.Template)
		} else {
			serverKey = fmt.Sprintf("plain:%s", server.Address)
		}

		// skip duplicate servers
		if _, exists := processedServers[serverKey]; exists {
			continue
		}
		processedServers[serverKey] = struct{}{}

		// validate encryption type
		if server.Encryption != EncryptionNone && server.Encryption != EncryptionDnsOverHttps {
			return nil, fmt.Errorf("unsupported DNS encryption type %d", server.Encryption)
		}

		// handle plain DNS servers - each gets its own group
		if server.Encryption == EncryptionNone {
			// finalize current DoH group if it exists
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = make([]DnsServerConfig, 0)
				currentUsedDoHUris = make(map[string]struct{})
			}
			// create a single-server group for plain DNS
			groups = append(groups, []DnsServerConfig{server})
			continue
		}

		// handle DoH servers
		if server.Encryption == EncryptionDnsOverHttps {
			// check if this DoH URI is already in the current group
			if _, exists := currentUsedDoHUris[server.Template]; exists {
				// finalize current group and start a new one
				if len(currentGroup) > 0 {
					groups = append(groups, currentGroup)
					currentGroup = make([]DnsServerConfig, 0)
					currentUsedDoHUris = make(map[string]struct{})
				}
			}

			// add server to current group
			currentGroup = append(currentGroup, server)
			currentUsedDoHUris[server.Template] = struct{}{}
		}
	}

	// finalize the last group if it exists
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups, nil
}
