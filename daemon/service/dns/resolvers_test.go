package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupConfigs(t *testing.T) {
	// Define some common servers for reuse in tests
	dohGoogle1 := DnsServerConfig{Address: "8.8.8.8", Encryption: EncryptionDnsOverHttps, Template: "https://dns.google/dns-query"}
	dohGoogle2 := DnsServerConfig{Address: "8.8.4.4", Encryption: EncryptionDnsOverHttps, Template: "https://dns.google/dns-query"}
	dohCloudflare1 := DnsServerConfig{Address: "1.1.1.1", Encryption: EncryptionDnsOverHttps, Template: "https://cloudflare-dns.com/dns-query"}
	dohCloudflare2 := DnsServerConfig{Address: "1.0.0.1", Encryption: EncryptionDnsOverHttps, Template: "https://cloudflare-dns.com/dns-query"}
	dohQuad9 := DnsServerConfig{Address: "9.9.9.9", Encryption: EncryptionDnsOverHttps, Template: "https://dns.quad9.net/dns-query"}
	plain1 := DnsServerConfig{Address: "208.67.222.222", Encryption: EncryptionNone}
	plain2 := DnsServerConfig{Address: "208.67.220.220", Encryption: EncryptionNone}

	testCases := []struct {
		name           string
		servers        []DnsServerConfig
		expectedGroups [][]DnsServerConfig
		expectedErr    string
	}{
		{
			name:        "empty input",
			servers:     []DnsServerConfig{},
			expectedErr: "no DNS servers provided",
		},
		{
			name: "unsupported encryption type",
			servers: []DnsServerConfig{
				{Address: "1.2.3.4", Encryption: 99},
			},
			expectedErr: "unsupported DNS encryption type 99",
		},
		{
			name: "only plain DNS servers",
			servers: []DnsServerConfig{
				plain1,
				plain2,
			},
			expectedGroups: [][]DnsServerConfig{
				{plain1},
				{plain2},
			},
		},
		{
			name: "only DoH servers with unique URIs",
			servers: []DnsServerConfig{
				dohGoogle1,
				dohCloudflare1,
				dohQuad9,
			},
			expectedGroups: [][]DnsServerConfig{
				{dohGoogle1, dohCloudflare1, dohQuad9},
			},
		},
		{
			name: "only DoH servers with duplicate URIs",
			servers: []DnsServerConfig{
				dohGoogle1,
				dohCloudflare1,
				dohGoogle2, // Same URI as dohGoogle1, should start a new group
			},
			expectedGroups: [][]DnsServerConfig{
				{dohGoogle1, dohCloudflare1},
				{dohGoogle2},
			},
		},
		{
			name: "mixed servers with plain DNS breaking DoH groups",
			servers: []DnsServerConfig{
				dohGoogle1,
				dohCloudflare1,
				plain1, // Breaks the DoH group
				dohQuad9,
				dohGoogle2,
			},
			expectedGroups: [][]DnsServerConfig{
				{dohGoogle1, dohCloudflare1},
				{plain1},
				{dohQuad9, dohGoogle2},
			},
		},
		{
			name: "complex grouping from function documentation",
			servers: []DnsServerConfig{
				dohGoogle1,     // Group 1
				dohQuad9,       // Group 1
				plain2,         // Group 2 (plain)
				plain1,         // Group 3 (plain)
				dohCloudflare1, // Group 4
				dohCloudflare2, // Same URI as previous, starts Group 5
				dohGoogle2,     // Group 5
			},
			expectedGroups: [][]DnsServerConfig{
				{dohGoogle1, dohQuad9},
				{plain2},
				{plain1},
				{dohCloudflare1},
				{dohCloudflare2, dohGoogle2},
			},
		},
		{
			name: "duplicate servers are skipped",
			servers: []DnsServerConfig{
				dohGoogle1,
				plain1,
				dohGoogle1, // Duplicate DoH
				plain1,     // Duplicate plain
				dohCloudflare1,
			},
			expectedGroups: [][]DnsServerConfig{
				{dohGoogle1},
				{plain1},
				{dohCloudflare1},
			},
		},
		{
			name: "template with whitespace is trimmed",
			servers: []DnsServerConfig{
				{Address: "8.8.8.8", Encryption: EncryptionDnsOverHttps, Template: " https://dns.google/dns-query "},
				{Address: "8.8.4.4", Encryption: EncryptionDnsOverHttps, Template: "https://dns.google/dns-query"}, // Same URI
			},
			expectedGroups: [][]DnsServerConfig{
				{{Address: "8.8.8.8", Encryption: EncryptionDnsOverHttps, Template: "https://dns.google/dns-query"}},
				{{Address: "8.8.4.4", Encryption: EncryptionDnsOverHttps, Template: "https://dns.google/dns-query"}},
			},
		},
		{
			name: "starts with plain dns",
			servers: []DnsServerConfig{
				plain1,
				dohGoogle1,
				dohCloudflare1,
			},
			expectedGroups: [][]DnsServerConfig{
				{plain1},
				{dohGoogle1, dohCloudflare1},
			},
		},
		{
			name: "ends with plain dns",
			servers: []DnsServerConfig{
				dohGoogle1,
				dohCloudflare1,
				plain1,
			},
			expectedGroups: [][]DnsServerConfig{
				{dohGoogle1, dohCloudflare1},
				{plain1},
			},
		},
		{
			name: "single plain dns",
			servers: []DnsServerConfig{
				plain1,
			},
			expectedGroups: [][]DnsServerConfig{
				{plain1},
			},
		},
		{
			name: "single doh dns",
			servers: []DnsServerConfig{
				dohGoogle1,
			},
			expectedGroups: [][]DnsServerConfig{
				{dohGoogle1},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			groups, err := groupConfigs(tc.servers)

			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
				assert.Nil(t, groups)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedGroups, groups)
			}
		})
	}
}
