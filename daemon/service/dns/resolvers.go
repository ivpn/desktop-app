package dns

import (
	"fmt"
	"net"
	"path"
	"sync"

	"github.com/ivpn/desktop-app/daemon/service/dns/dnscryptproxy"
	"github.com/ivpn/desktop-app/daemon/service/platform"
)

var (
	dnsCryptProxyMutex     sync.Mutex
	dnsCryptProxyInstances []*dnscryptproxy.DnsCryptProxy
)

// ResolversSetup initializes DNS configuration for both plain and encrypted DNS servers.
// For encrypted DNS (DoH), it starts local dnscrypt-proxy instances as local resolvers.
// For plain DNS servers, it returns them unchanged.
//
// Parameters:
//   - dnsCfg: DNS configuration containing servers with various encryption types
//
// Returns:
//   - []DnsServerConfig: List of DNS server configurations ready to apply as system resolvers.
//     This includes plain DNS servers and local dnscrypt-proxy listeners (127.0.0.x) for encrypted DNS.
//   - error: Error if configuration creation or dnscrypt-proxy startup fails
func ResolversSetup(dnsCfg DnsSettings) ([]DnsServerConfig, error) {
	// return slice of DNS server configs
	ret := make([]DnsServerConfig, 0, len(dnsCfg.Servers))
	dnsCryptConfigs := make([]*dnscryptproxy.Config, 0, len(dnsCfg.Servers))

	nextResolverIP := net.IPv4(127, 0, 0, 1)

	for _, svr := range dnsCfg.Servers {

		if svr.Encryption == EncryptionNone {
			// plain DNS server - return as is (only one server must be in this group)
			ret = append(ret, svr)
			continue
		}
		if svr.Encryption != EncryptionDnsOverHttps {
			return nil, fmt.Errorf("unsupported DNS encryption type %d", svr.Encryption)
		}
		// DoH servers - start dnscrypt-proxy instance for this group

		// Get free local address for dnscrypt-proxy listener
		localIp, err := dnscryptproxy.GetFreeLocalAddressForDNS(nextResolverIP)
		if err != nil {
			log.Warning("Failed to get free local address for dnscrypt-proxy: ", err)
			// Failed to get free local address for dnscrypt-proxy.
			// Anyway, try to use nextResolverIP
			localIp = nextResolverIP
		}
		// increment last byte for next resolver
		nextResolverIP = net.IPv4(127, 0, 0, localIp.To4()[3]+1)

		// add local dnscrypt-proxy listener as a system DNS server
		ret = append(ret, DnsServerConfig{Address: localIp.String()})

		cfg, err := createDnscryptProxyConfig(localIp, []DnsServerConfig{svr})
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

	binPath, templateFile, configDir := platform.DnsCryptProxyInfo()
	if len(binPath) == 0 || len(templateFile) == 0 || len(configDir) == 0 {
		return fmt.Errorf("dnscrypt-proxy configuration not defined")
	}

	for _, cfg := range configs {
		uniqNamePostfix := cfg.ID()
		configFileOut := path.Join(configDir, fmt.Sprintf("dnscrypt-proxy-%s.toml", uniqNamePostfix))

		// save dnscrypt-proxy configuration file
		if err := cfg.Save(templateFile, configFileOut, ""); err != nil {
			return fmt.Errorf("failed to save dnscrypt-proxy config file: %w", err)
		}

		proxy, err := dnscryptproxy.Start(binPath, configFileOut, cfg.ListenAddress())
		if err != nil {
			return fmt.Errorf("failed to start dnscrypt-proxy: %w", err)
		}
		dnsCryptProxyInstances = append(dnsCryptProxyInstances, proxy)
	}
	return nil
}
