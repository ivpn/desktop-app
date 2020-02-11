// +build windows

package iphlpapi

// APIMibIPForwardRow - MIB_IPFORWARDROW structure. https://docs.microsoft.com/ru-ru/windows/win32/api/ipmib/ns-ipmib-mib_ipforwardrow
type APIMibIPForwardRow struct {
	DwForwardDest      [4]byte
	DwForwardMask      [4]byte
	DwForwardPolicy    uint32
	DwForwardNextHop   [4]byte
	DwForwardIfIndex   uint32
	ForwardType        uint32
	ForwardProto       uint32
	DwForwardAge       uint32
	DwForwardNextHopAS uint32
	DwForwardMetric1   uint32
	DwForwardMetric2   uint32
	DwForwardMetric3   uint32
	DwForwardMetric4   uint32
	DwForwardMetric5   uint32
}
