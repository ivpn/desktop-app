package winlib

import "syscall"

// Condition GUIDs
var (
	FwpmConditionAleAppID        = syscall.GUID{Data1: 0xd78e1e87, Data2: 0x8644, Data3: 0x4ea5, Data4: [8]byte{0x94, 0x37, 0xd8, 0x09, 0xec, 0xef, 0xc9, 0x71}}
	FwpmConditionIPLocalAddress  = syscall.GUID{Data1: 0xd9ee00de, Data2: 0xc1ef, Data3: 0x4617, Data4: [8]byte{0xbf, 0xe3, 0xff, 0xd8, 0xf5, 0xa0, 0x89, 0x57}}
	FwpmConditionIPLocalPort     = syscall.GUID{Data1: 0x0c1ba1af, Data2: 0x5765, Data3: 0x453f, Data4: [8]byte{0xaf, 0x22, 0xa8, 0xf7, 0x91, 0xac, 0x77, 0x5b}}
	FwpmConditionIPRemoteAddress = syscall.GUID{Data1: 0xb235ae9a, Data2: 0x1d64, Data3: 0x49b8, Data4: [8]byte{0xa4, 0x4c, 0x5f, 0xf3, 0xd9, 0x09, 0x50, 0x45}}
	FwpmConditionIPRemotePort    = syscall.GUID{Data1: 0xc35a604d, Data2: 0xd22b, Data3: 0x4e1a, Data4: [8]byte{0x91, 0xb4, 0x68, 0xf6, 0x74, 0xee, 0x67, 0x4b}}

	/*
		FwpmConditionInterfaceMacAddress             = syscall.GUID{Data1: 0xf6e63dce, Data2: 0x1f4b, Data3: 0x4c6b, Data4: [8]byte{0xb6, 0xef, 0x11, 0x65, 0xe7, 0x1f, 0x8e, 0xe7}}
		FwpmConditionMacLocalAddress                 = syscall.GUID{Data1: 0xd999e981, Data2: 0x7948, Data3: 0x4c8e, Data4: [8]byte{0xb7, 0x42, 0xc8, 0x4e, 0x3b, 0x67, 0x8f, 0x8f}}
		FwpmConditionMacRemoteAddress                = syscall.GUID{Data1: 0x408f2ed4, Data2: 0x3a70, Data3: 0x4b4d, Data4: [8]byte{0x92, 0xa6, 0x41, 0x5a, 0xc2, 0x0e, 0x2f, 0x12}}
		FwpmConditionEtherType                       = syscall.GUID{Data1: 0xfd08948d, Data2: 0xa219, Data3: 0x4d52, Data4: [8]byte{0xbb, 0x98, 0x1a, 0x55, 0x40, 0xee, 0x7b, 0x4e}}
		FwpmConditionVlanID                          = syscall.GUID{Data1: 0x938eab21, Data2: 0x3618, Data3: 0x4e64, Data4: [8]byte{0x9c, 0xa5, 0x21, 0x41, 0xeb, 0xda, 0x1c, 0xa2}}
		FwpmConditionVswitchTenantNetworkID          = syscall.GUID{Data1: 0xdc04843c, Data2: 0x79e6, Data3: 0x4e44, Data4: [8]byte{0xa0, 0x25, 0x65, 0xb9, 0xbb, 0x0f, 0x9f, 0x94}}
		FwpmConditionNdisPort                        = syscall.GUID{Data1: 0xdb7bb42b, Data2: 0x2dac, Data3: 0x4cd4, Data4: [8]byte{0xa5, 0x9a, 0xe0, 0xbd, 0xce, 0x1e, 0x68, 0x34}}
		FwpmConditionNdisMediaType                   = syscall.GUID{Data1: 0xcb31cef1, Data2: 0x791d, Data3: 0x473b, Data4: [8]byte{0x89, 0xd1, 0x61, 0xc5, 0x98, 0x43, 0x04, 0xa0}}
		FwpmConditionNdisPhysicalMediaType           = syscall.GUID{Data1: 0x34c79823, Data2: 0xc229, Data3: 0x44f2, Data4: [8]byte{0xb8, 0x3c, 0x74, 0x02, 0x08, 0x82, 0xae, 0x77}}
		FwpmConditionL2Flags                         = syscall.GUID{Data1: 0x7bc43cbf, Data2: 0x37ba, Data3: 0x45f1, Data4: [8]byte{0xb7, 0x4a, 0x82, 0xff, 0x51, 0x8e, 0xeb, 0x10}}
		FwpmConditionMacLocalAddressType             = syscall.GUID{Data1: 0xcc31355c, Data2: 0x3073, Data3: 0x4ffb, Data4: [8]byte{0xa1, 0x4f, 0x79, 0x41, 0x5c, 0xb1, 0xea, 0xd1}}
		FwpmConditionMacRemoteAddressType            = syscall.GUID{Data1: 0x027fedb4, Data2: 0xf1c1, Data3: 0x4030, Data4: [8]byte{0xb5, 0x64, 0xee, 0x77, 0x7f, 0xd8, 0x67, 0xea}}
		FwpmConditionAlePackageID                    = syscall.GUID{Data1: 0x71BC78FA, Data2: 0xF17C, Data3: 0x4997, Data4: [8]byte{0xA6, 0x02, 0x6A, 0xBB, 0x26, 0x1F, 0x35, 0x1C}}
		FwpmConditionMacSourceAddress                = syscall.GUID{Data1: 0x7b795451, Data2: 0xf1f6, Data3: 0x4d05, Data4: [8]byte{0xb7, 0xcb, 0x21, 0x77, 0x9d, 0x80, 0x23, 0x36}}
		FwpmConditionMacDestinationAddress           = syscall.GUID{Data1: 0x04ea2a93, Data2: 0x858c, Data3: 0x4027, Data4: [8]byte{0xb6, 0x13, 0xb4, 0x31, 0x80, 0xc7, 0x85, 0x9e}}
		FwpmConditionMacSourceAddressType            = syscall.GUID{Data1: 0x5c1b72e4, Data2: 0x299e, Data3: 0x4437, Data4: [8]byte{0xa2, 0x98, 0xbc, 0x3f, 0x01, 0x4b, 0x3d, 0xc2}}
		FwpmConditionMacDestinationAddressType       = syscall.GUID{Data1: 0xae052932, Data2: 0xef42, Data3: 0x4e99, Data4: [8]byte{0xb1, 0x29, 0xf3, 0xb3, 0x13, 0x9e, 0x34, 0xf7}}
		FwpmConditionIPSourcePort                    = syscall.GUID{Data1: 0xa6afef91, Data2: 0x3df4, Data3: 0x4730, Data4: [8]byte{0xa2, 0x14, 0xf5, 0x42, 0x6a, 0xeb, 0xf8, 0x21}}
		FwpmConditionIPDestinationPort               = syscall.GUID{Data1: 0xce6def45, Data2: 0x60fb, Data3: 0x4a7b, Data4: [8]byte{0xa3, 0x04, 0xaf, 0x30, 0xa1, 0x17, 0x00, 0x0e}}
		FwpmConditionVswitchID                       = syscall.GUID{Data1: 0xc4a414ba, Data2: 0x437b, Data3: 0x4de6, Data4: [8]byte{0x99, 0x46, 0xd9, 0x9c, 0x1b, 0x95, 0xb3, 0x12}}
		FwpmConditionVswitchNetworkType              = syscall.GUID{Data1: 0x11d48b4b, Data2: 0xe77a, Data3: 0x40b4, Data4: [8]byte{0x91, 0x55, 0x39, 0x2c, 0x90, 0x6c, 0x26, 0x08}}
		FwpmConditionVswitchSourceInterfaceID        = syscall.GUID{Data1: 0x7f4ef24b, Data2: 0xb2c1, Data3: 0x4938, Data4: [8]byte{0xba, 0x33, 0xa1, 0xec, 0xbe, 0xd5, 0x12, 0xba}}
		FwpmConditionVswitchDestinationInterfaceID   = syscall.GUID{Data1: 0x8ed48be4, Data2: 0xc926, Data3: 0x49f6, Data4: [8]byte{0xa4, 0xf6, 0xef, 0x30, 0x30, 0xe3, 0xfc, 0x16}}
		FwpmConditionVswitchSourceVMID               = syscall.GUID{Data1: 0x9c2a9ec2, Data2: 0x9fc6, Data3: 0x42bc, Data4: [8]byte{0xbd, 0xd8, 0x40, 0x6d, 0x4d, 0xa0, 0xbe, 0x64}}
		FwpmConditionVswitchDestinationVMID          = syscall.GUID{Data1: 0x6106aace, Data2: 0x4de1, Data3: 0x4c84, Data4: [8]byte{0x96, 0x71, 0x36, 0x37, 0xf8, 0xbc, 0xf7, 0x31}}
		FwpmConditionVswitchSourceInterfaceType      = syscall.GUID{Data1: 0xe6b040a2, Data2: 0xedaf, Data3: 0x4c36, Data4: [8]byte{0x90, 0x8b, 0xf2, 0xf5, 0x8a, 0xe4, 0x38, 0x07}}
		FwpmConditionVswitchDestinationInterfaceType = syscall.GUID{Data1: 0xfa9b3f06, Data2: 0x2f1a, Data3: 0x4c57, Data4: [8]byte{0x9e, 0x68, 0xa7, 0x09, 0x8b, 0x28, 0xdb, 0xfe}}
		FwpmConditionIPLocalAddress                  = syscall.GUID{Data1: 0xd9ee00de, Data2: 0xc1ef, Data3: 0x4617, Data4: [8]byte{0xbf, 0xe3, 0xff, 0xd8, 0xf5, 0xa0, 0x89, 0x57}}
		FwpmConditionIPRemoteAddress                 = syscall.GUID{Data1: 0xb235ae9a, Data2: 0x1d64, Data3: 0x49b8, Data4: [8]byte{0xa4, 0x4c, 0x5f, 0xf3, 0xd9, 0x09, 0x50, 0x45}}
		FwpmConditionIPSourceAddress                 = syscall.GUID{Data1: 0xae96897e, Data2: 0x2e94, Data3: 0x4bc9, Data4: [8]byte{0xb3, 0x13, 0xb2, 0x7e, 0xe8, 0x0e, 0x57, 0x4d}}
		FwpmConditionIPDestinationAddress            = syscall.GUID{Data1: 0x2d79133b, Data2: 0xb390, Data3: 0x45c6, Data4: [8]byte{0x86, 0x99, 0xac, 0xac, 0xea, 0xaf, 0xed, 0x33}}
		FwpmConditionIPLocalAddressType              = syscall.GUID{Data1: 0x6ec7f6c4, Data2: 0x376b, Data3: 0x45d7, Data4: [8]byte{0x9e, 0x9c, 0xd3, 0x37, 0xce, 0xdc, 0xd2, 0x37}}
		FwpmConditionIPDestinationAddressType        = syscall.GUID{Data1: 0x1ec1b7c9, Data2: 0x4eea, Data3: 0x4f5e, Data4: [8]byte{0xb9, 0xef, 0x76, 0xbe, 0xaa, 0xaf, 0x17, 0xee}}
		FwpmConditionIPNexthopAddress                = syscall.GUID{Data1: 0xeabe448a, Data2: 0xa711, Data3: 0x4d64, Data4: [8]byte{0x85, 0xb7, 0x3f, 0x76, 0xb6, 0x52, 0x99, 0xc7}}
		FwpmConditionIPLocalInterface                = syscall.GUID{Data1: 0x4cd62a49, Data2: 0x59c3, Data3: 0x4969, Data4: [8]byte{0xb7, 0xf3, 0xbd, 0xa5, 0xd3, 0x28, 0x90, 0xa4}}
		FwpmConditionIPArrivalInterface              = syscall.GUID{Data1: 0x618a9b6d, Data2: 0x386b, Data3: 0x4136, Data4: [8]byte{0xad, 0x6e, 0xb5, 0x15, 0x87, 0xcf, 0xb1, 0xcd}}
		FwpmConditionArrivalInterfaceType            = syscall.GUID{Data1: 0x89f990de, Data2: 0xe798, Data3: 0x4e6d, Data4: [8]byte{0xab, 0x76, 0x7c, 0x95, 0x58, 0x29, 0x2e, 0x6f}}
		FwpmConditionArrivalTunnelType               = syscall.GUID{Data1: 0x511166dc, Data2: 0x7a8c, Data3: 0x4aa7, Data4: [8]byte{0xb5, 0x33, 0x95, 0xab, 0x59, 0xfb, 0x03, 0x40}}
		FwpmConditionArrivalInterfaceIndex           = syscall.GUID{Data1: 0xcc088db3, Data2: 0x1792, Data3: 0x4a71, Data4: [8]byte{0xb0, 0xf9, 0x03, 0x7d, 0x21, 0xcd, 0x82, 0x8b}}
		FwpmConditionNexthopSubInterfaceIndex        = syscall.GUID{Data1: 0xef8a6122, Data2: 0x0577, Data3: 0x45a7, Data4: [8]byte{0x9a, 0xaf, 0x82, 0x5f, 0xbe, 0xb4, 0xfb, 0x95}}
		FwpmConditionIPNexthopInterface              = syscall.GUID{Data1: 0x93ae8f5b, Data2: 0x7f6f, Data3: 0x4719, Data4: [8]byte{0x98, 0xc8, 0x14, 0xe9, 0x74, 0x29, 0xef, 0x04}}
		FwpmConditionNexthopInterfaceType            = syscall.GUID{Data1: 0x97537c6c, Data2: 0xd9a3, Data3: 0x4767, Data4: [8]byte{0xa3, 0x81, 0xe9, 0x42, 0x67, 0x5c, 0xd9, 0x20}}
		FwpmConditionNexthopTunnelType               = syscall.GUID{Data1: 0x72b1a111, Data2: 0x987b, Data3: 0x4720, Data4: [8]byte{0x99, 0xdd, 0xc7, 0xc5, 0x76, 0xfa, 0x2d, 0x4c}}
		FwpmConditionNexthopInterfaceIndex           = syscall.GUID{Data1: 0x138e6888, Data2: 0x7ab8, Data3: 0x4d65, Data4: [8]byte{0x9e, 0xe8, 0x05, 0x91, 0xbc, 0xf6, 0xa4, 0x94}}
		FwpmConditionOriginalProfileID               = syscall.GUID{Data1: 0x46ea1551, Data2: 0x2255, Data3: 0x492b, Data4: [8]byte{0x80, 0x19, 0xaa, 0xbe, 0xee, 0x34, 0x9f, 0x40}}
		FwpmConditionCurrentProfileID                = syscall.GUID{Data1: 0xab3033c9, Data2: 0xc0e3, Data3: 0x4759, Data4: [8]byte{0x93, 0x7d, 0x57, 0x58, 0xc6, 0x5d, 0x4a, 0xe3}}
		FwpmConditionLocalInterfaceProfileID         = syscall.GUID{Data1: 0x4ebf7562, Data2: 0x9f18, Data3: 0x4d06, Data4: [8]byte{0x99, 0x41, 0xa7, 0xa6, 0x25, 0x74, 0x4d, 0x71}}
		FwpmConditionArrivalInterfaceProfileID       = syscall.GUID{Data1: 0xcdfe6aab, Data2: 0xc083, Data3: 0x4142, Data4: [8]byte{0x86, 0x79, 0xc0, 0x8f, 0x95, 0x32, 0x9c, 0x61}}
		FwpmConditionNexthopInterfaceProfileID       = syscall.GUID{Data1: 0xd7ff9a56, Data2: 0xcdaa, Data3: 0x472b, Data4: [8]byte{0x84, 0xdb, 0xd2, 0x39, 0x63, 0xc1, 0xd1, 0xbf}}
		FwpmConditionReauthorizeReason               = syscall.GUID{Data1: 0x11205e8c, Data2: 0x11ae, Data3: 0x457a, Data4: [8]byte{0x8a, 0x44, 0x47, 0x70, 0x26, 0xdd, 0x76, 0x4a}}
		FwpmConditionOriginalIcmpType                = syscall.GUID{Data1: 0x076dfdbe, Data2: 0xc56c, Data3: 0x4f72, Data4: [8]byte{0xae, 0x8a, 0x2c, 0xfe, 0x7e, 0x5c, 0x82, 0x86}}
		FwpmConditionIPPhysicalArrivalInterface      = syscall.GUID{Data1: 0xda50d5c8, Data2: 0xfa0d, Data3: 0x4c89, Data4: [8]byte{0xb0, 0x32, 0x6e, 0x62, 0x13, 0x6d, 0x1e, 0x96}}
		FwpmConditionIPPhysicalNexthopInterface      = syscall.GUID{Data1: 0xf09bd5ce, Data2: 0x5150, Data3: 0x48be, Data4: [8]byte{0xb0, 0x98, 0xc2, 0x51, 0x52, 0xfb, 0x1f, 0x92}}
		FwpmConditionInterfaceQuarantineEpoch        = syscall.GUID{Data1: 0xcce68d5e, Data2: 0x053b, Data3: 0x43a8, Data4: [8]byte{0x9a, 0x6f, 0x33, 0x38, 0x4c, 0x28, 0xe4, 0xf6}}
		FwpmConditionInterfaceType                   = syscall.GUID{Data1: 0xdaf8cd14, Data2: 0xe09e, Data3: 0x4c93, Data4: [8]byte{0xa5, 0xae, 0xc5, 0xc1, 0x3b, 0x73, 0xff, 0xca}}
		FwpmConditionTunnelType                      = syscall.GUID{Data1: 0x77a40437, Data2: 0x8779, Data3: 0x4868, Data4: [8]byte{0xa2, 0x61, 0xf5, 0xa9, 0x02, 0xf1, 0xc0, 0xcd}}
		FwpmConditionIPForwardInterface              = syscall.GUID{Data1: 0x1076b8a5, Data2: 0x6323, Data3: 0x4c5e, Data4: [8]byte{0x98, 0x10, 0xe8, 0xd3, 0xfc, 0x9e, 0x61, 0x36}}
		FwpmConditionIPProtocol                      = syscall.GUID{Data1: 0x3971ef2b, Data2: 0x623e, Data3: 0x4f9a, Data4: [8]byte{0x8c, 0xb1, 0x6e, 0x79, 0xb8, 0x06, 0xb9, 0xa7}}
		FwpmConditionIPLocalPort                     = syscall.GUID{Data1: 0x0c1ba1af, Data2: 0x5765, Data3: 0x453f, Data4: [8]byte{0xaf, 0x22, 0xa8, 0xf7, 0x91, 0xac, 0x77, 0x5b}}
		FwpmConditionIPRemotePort                    = syscall.GUID{Data1: 0xc35a604d, Data2: 0xd22b, Data3: 0x4e1a, Data4: [8]byte{0x91, 0xb4, 0x68, 0xf6, 0x74, 0xee, 0x67, 0x4b}}
		FwpmConditionEmbeddedLocalAddressType        = syscall.GUID{Data1: 0x4672a468, Data2: 0x8a0a, Data3: 0x4202, Data4: [8]byte{0xab, 0xb4, 0x84, 0x9e, 0x92, 0xe6, 0x68, 0x09}}
		FwpmConditionEmbeddedRemoteAddress           = syscall.GUID{Data1: 0x77ee4b39, Data2: 0x3273, Data3: 0x4671, Data4: [8]byte{0xb6, 0x3b, 0xab, 0x6f, 0xeb, 0x66, 0xee, 0xb6}}
		FwpmConditionEmbeddedProtocol                = syscall.GUID{Data1: 0x07784107, Data2: 0xa29e, Data3: 0x4c7b, Data4: [8]byte{0x9e, 0xc7, 0x29, 0xc4, 0x4a, 0xfa, 0xfd, 0xbc}}
		FwpmConditionEmbeddedLocalPort               = syscall.GUID{Data1: 0xbfca394d, Data2: 0xacdb, Data3: 0x484e, Data4: [8]byte{0xb8, 0xe6, 0x2a, 0xff, 0x79, 0x75, 0x73, 0x45}}
		FwpmConditionEmbeddedRemotePort              = syscall.GUID{Data1: 0xcae4d6a1, Data2: 0x2968, Data3: 0x40ed, Data4: [8]byte{0xa4, 0xce, 0x54, 0x71, 0x60, 0xdd, 0xa8, 0x8d}}
		FwpmConditionFlags                           = syscall.GUID{Data1: 0x632ce23b, Data2: 0x5167, Data3: 0x435c, Data4: [8]byte{0x86, 0xd7, 0xe9, 0x03, 0x68, 0x4a, 0xa8, 0x0c}}
		FwpmConditionDirection                       = syscall.GUID{Data1: 0x8784c146, Data2: 0xca97, Data3: 0x44d6, Data4: [8]byte{0x9f, 0xd1, 0x19, 0xfb, 0x18, 0x40, 0xcb, 0xf7}}
		FwpmConditionInterfaceIndex                  = syscall.GUID{Data1: 0x667fd755, Data2: 0xd695, Data3: 0x434a, Data4: [8]byte{0x8a, 0xf5, 0xd3, 0x83, 0x5a, 0x12, 0x59, 0xbc}}
		FwpmConditionSubInterfaceIndex               = syscall.GUID{Data1: 0x0cd42473, Data2: 0xd621, Data3: 0x4be3, Data4: [8]byte{0xae, 0x8c, 0x72, 0xa3, 0x48, 0xd2, 0x83, 0xe1}}
		FwpmConditionSourceInterfaceIndex            = syscall.GUID{Data1: 0x2311334d, Data2: 0xc92d, Data3: 0x45bf, Data4: [8]byte{0x94, 0x96, 0xed, 0xf4, 0x47, 0x82, 0x0e, 0x2d}}
		FwpmConditionSourceSubInterfaceIndex         = syscall.GUID{Data1: 0x055edd9d, Data2: 0xacd2, Data3: 0x4361, Data4: [8]byte{0x8d, 0xab, 0xf9, 0x52, 0x5d, 0x97, 0x66, 0x2f}}
		FwpmConditionDestinationInterfaceIndex       = syscall.GUID{Data1: 0x35cf6522, Data2: 0x4139, Data3: 0x45ee, Data4: [8]byte{0xa0, 0xd5, 0x67, 0xb8, 0x09, 0x49, 0xd8, 0x79}}
		FwpmConditionDestinationSubInterfaceIndex    = syscall.GUID{Data1: 0x2b7d4399, Data2: 0xd4c7, Data3: 0x4738, Data4: [8]byte{0xa2, 0xf5, 0xe9, 0x94, 0xb4, 0x3d, 0xa3, 0x88}}
		FwpmConditionAleAppID                        = syscall.GUID{Data1: 0xd78e1e87, Data2: 0x8644, Data3: 0x4ea5, Data4: [8]byte{0x94, 0x37, 0xd8, 0x09, 0xec, 0xef, 0xc9, 0x71}}
		FwpmConditionAleOriginalAppID                = syscall.GUID{Data1: 0x0e6cd086, Data2: 0xe1fb, Data3: 0x4212, Data4: [8]byte{0x84, 0x2f, 0x8a, 0x9f, 0x99, 0x3f, 0xb3, 0xf6}}
		FwpmConditionAleUserID                       = syscall.GUID{Data1: 0xaf043a0a, Data2: 0xb34d, Data3: 0x4f86, Data4: [8]byte{0x97, 0x9c, 0xc9, 0x03, 0x71, 0xaf, 0x6e, 0x66}}
		FwpmConditionAleRemoteUserID                 = syscall.GUID{Data1: 0xf63073b7, Data2: 0x0189, Data3: 0x4ab0, Data4: [8]byte{0x95, 0xa4, 0x61, 0x23, 0xcb, 0xfa, 0xb8, 0x62}}
		FwpmConditionAleRemoteMachineID              = syscall.GUID{Data1: 0x1aa47f51, Data2: 0x7f93, Data3: 0x4508, Data4: [8]byte{0xa2, 0x71, 0x81, 0xab, 0xb0, 0x0c, 0x9c, 0xab}}
		FwpmConditionAlePromiscuousMode              = syscall.GUID{Data1: 0x1c974776, Data2: 0x7182, Data3: 0x46e9, Data4: [8]byte{0xaf, 0xd3, 0xb0, 0x29, 0x10, 0xe3, 0x03, 0x34}}
		FwpmConditionAleSioFirewallSystemPort        = syscall.GUID{Data1: 0xb9f4e088, Data2: 0xcb98, Data3: 0x4efb, Data4: [8]byte{0xa2, 0xc7, 0xad, 0x07, 0x33, 0x26, 0x43, 0xdb}}
		FwpmConditionAleReauthReason                 = syscall.GUID{Data1: 0xb482d227, Data2: 0x1979, Data3: 0x4a98, Data4: [8]byte{0x80, 0x44, 0x18, 0xbb, 0xe6, 0x23, 0x75, 0x42}}
		FwpmConditionAleNapContext                   = syscall.GUID{Data1: 0x46275a9d, Data2: 0xc03f, Data3: 0x4d77, Data4: [8]byte{0xb7, 0x84, 0x1c, 0x57, 0xf4, 0xd0, 0x27, 0x53}}
		FwpmConditionKmAuthNapContext                = syscall.GUID{Data1: 0x35d0ea0e, Data2: 0x15ca, Data3: 0x492b, Data4: [8]byte{0x90, 0x0e, 0x97, 0xfd, 0x46, 0x35, 0x2c, 0xce}}
		FwpmConditionRemoteUserToken                 = syscall.GUID{Data1: 0x9bf0ee66, Data2: 0x06c9, Data3: 0x41b9, Data4: [8]byte{0x84, 0xda, 0x28, 0x8c, 0xb4, 0x3a, 0xf5, 0x1f}}
		FwpmConditionRPCIfUUID                       = syscall.GUID{Data1: 0x7c9c7d9f, Data2: 0x0075, Data3: 0x4d35, Data4: [8]byte{0xa0, 0xd1, 0x83, 0x11, 0xc4, 0xcf, 0x6a, 0xf1}}
		FwpmConditionRPCIfVersion                    = syscall.GUID{Data1: 0xeabfd9b7, Data2: 0x1262, Data3: 0x4a2e, Data4: [8]byte{0xad, 0xaa, 0x5f, 0x96, 0xf6, 0xfe, 0x32, 0x6d}}
		FwpmConditionRPCIfFlag                       = syscall.GUID{Data1: 0x238a8a32, Data2: 0x3199, Data3: 0x467d, Data4: [8]byte{0x87, 0x1c, 0x27, 0x26, 0x21, 0xab, 0x38, 0x96}}
		FwpmConditionDcomAppID                       = syscall.GUID{Data1: 0xff2e7b4d, Data2: 0x3112, Data3: 0x4770, Data4: [8]byte{0xb6, 0x36, 0x4d, 0x24, 0xae, 0x3a, 0x6a, 0xf2}}
		FwpmConditionImageName                       = syscall.GUID{Data1: 0xd024de4d, Data2: 0xdeaa, Data3: 0x4317, Data4: [8]byte{0x9c, 0x85, 0xe4, 0x0e, 0xf6, 0xe1, 0x40, 0xc3}}
		FwpmConditionRPCProtocol                     = syscall.GUID{Data1: 0x2717bc74, Data2: 0x3a35, Data3: 0x4ce7, Data4: [8]byte{0xb7, 0xef, 0xc8, 0x38, 0xfa, 0xbd, 0xec, 0x45}}
		FwpmConditionRPCAuthType                     = syscall.GUID{Data1: 0xdaba74ab, Data2: 0x0d67, Data3: 0x43e7, Data4: [8]byte{0x98, 0x6e, 0x75, 0xb8, 0x4f, 0x82, 0xf5, 0x94}}
		FwpmConditionRPCAuthLevel                    = syscall.GUID{Data1: 0xe5a0aed5, Data2: 0x59ac, Data3: 0x46ea, Data4: [8]byte{0xbe, 0x05, 0xa5, 0xf0, 0x5e, 0xcf, 0x44, 0x6e}}
		FwpmConditionSecEncryptAlgorithm             = syscall.GUID{Data1: 0x0d306ef0, Data2: 0xe974, Data3: 0x4f74, Data4: [8]byte{0xb5, 0xc7, 0x59, 0x1b, 0x0d, 0xa7, 0xd5, 0x62}}
		FwpmConditionSecKeySize                      = syscall.GUID{Data1: 0x4772183b, Data2: 0xccf8, Data3: 0x4aeb, Data4: [8]byte{0xbc, 0xe1, 0xc6, 0xc6, 0x16, 0x1c, 0x8f, 0xe4}}
		FwpmConditionIPLocalAddressV4                = syscall.GUID{Data1: 0x03a629cb, Data2: 0x6e52, Data3: 0x49f8, Data4: [8]byte{0x9c, 0x41, 0x57, 0x09, 0x63, 0x3c, 0x09, 0xcf}}
		FwpmConditionIPLocalAddressV6                = syscall.GUID{Data1: 0x2381be84, Data2: 0x7524, Data3: 0x45b3, Data4: [8]byte{0xa0, 0x5b, 0x1e, 0x63, 0x7d, 0x9c, 0x7a, 0x6a}}
		FwpmConditionPipe                            = syscall.GUID{Data1: 0x1bd0741d, Data2: 0xe3df, Data3: 0x4e24, Data4: [8]byte{0x86, 0x34, 0x76, 0x20, 0x46, 0xee, 0xf6, 0xeb}}
		FwpmConditionIPRemoteAddressV4               = syscall.GUID{Data1: 0x1febb610, Data2: 0x3bcc, Data3: 0x45e1, Data4: [8]byte{0xbc, 0x36, 0x2e, 0x06, 0x7e, 0x2c, 0xb1, 0x86}}
		FwpmConditionIPRemoteAddressV6               = syscall.GUID{Data1: 0x246e1d8c, Data2: 0x8bee, Data3: 0x4018, Data4: [8]byte{0x9b, 0x98, 0x31, 0xd4, 0x58, 0x2f, 0x33, 0x61}}
		FwpmConditionProcessWithRPCIfUUID            = syscall.GUID{Data1: 0xe31180a8, Data2: 0xbbbd, Data3: 0x4d14, Data4: [8]byte{0xa6, 0x5e, 0x71, 0x57, 0xb0, 0x62, 0x33, 0xbb}}
		FwpmConditionRPCEpValue                      = syscall.GUID{Data1: 0xdccea0b9, Data2: 0x0886, Data3: 0x4360, Data4: [8]byte{0x9c, 0x6a, 0xab, 0x04, 0x3a, 0x24, 0xfb, 0xa9}}
		FwpmConditionRPCEpFlags                      = syscall.GUID{Data1: 0x218b814a, Data2: 0x0a39, Data3: 0x49b8, Data4: [8]byte{0x8e, 0x71, 0xc2, 0x0c, 0x39, 0xc7, 0xdd, 0x2e}}
		FwpmConditionClientToken                     = syscall.GUID{Data1: 0xc228fc1e, Data2: 0x403a, Data3: 0x4478, Data4: [8]byte{0xbe, 0x05, 0xc9, 0xba, 0xa4, 0xc0, 0x5a, 0xce}}
		FwpmConditionRPCServerName                   = syscall.GUID{Data1: 0xb605a225, Data2: 0xc3b3, Data3: 0x48c7, Data4: [8]byte{0x98, 0x33, 0x7a, 0xef, 0xa9, 0x52, 0x75, 0x46}}
		FwpmConditionRPCServerPort                   = syscall.GUID{Data1: 0x8090f645, Data2: 0x9ad5, Data3: 0x4e3b, Data4: [8]byte{0x9f, 0x9f, 0x80, 0x23, 0xca, 0x09, 0x79, 0x09}}
		FwpmConditionRPCProxyAuthType                = syscall.GUID{Data1: 0x40953fe2, Data2: 0x8565, Data3: 0x4759, Data4: [8]byte{0x84, 0x88, 0x17, 0x71, 0xb4, 0xb4, 0xb5, 0xdb}}
		FwpmConditionClientCertKeyLength             = syscall.GUID{Data1: 0xa3ec00c7, Data2: 0x05f4, Data3: 0x4df7, Data4: [8]byte{0x91, 0xf2, 0x5f, 0x60, 0xd9, 0x1f, 0xf4, 0x43}}
		FwpmConditionClientCertOid                   = syscall.GUID{Data1: 0xc491ad5e, Data2: 0xf882, Data3: 0x4283, Data4: [8]byte{0xb9, 0x16, 0x43, 0x6b, 0x10, 0x3f, 0xf4, 0xad}}
		FwpmConditionNetEventType                    = syscall.GUID{Data1: 0x206e9996, Data2: 0x490e, Data3: 0x40cf, Data4: [8]byte{0xb8, 0x31, 0xb3, 0x86, 0x41, 0xeb, 0x6f, 0xcb}}
		FwpmConditionPeerName                        = syscall.GUID{Data1: 0x9b539082, Data2: 0xeb90, Data3: 0x4186, Data4: [8]byte{0xa6, 0xcc, 0xde, 0x5b, 0x63, 0x23, 0x50, 0x16}}
		FwpmConditionRemoteID                        = syscall.GUID{Data1: 0xf68166fd, Data2: 0x0682, Data3: 0x4c89, Data4: [8]byte{0xb8, 0xf5, 0x86, 0x43, 0x6c, 0x7e, 0xf9, 0xb7}}
		FwpmConditionAuthenticationType              = syscall.GUID{Data1: 0xeb458cd5, Data2: 0xda7b, Data3: 0x4ef9, Data4: [8]byte{0x8d, 0x43, 0x7b, 0x0a, 0x84, 0x03, 0x32, 0xf2}}
		FwpmConditionKmType                          = syscall.GUID{Data1: 0xff0f5f49, Data2: 0x0ceb, Data3: 0x481b, Data4: [8]byte{0x86, 0x38, 0x14, 0x79, 0x79, 0x1f, 0x3f, 0x2c}}
		FwpmConditionKmMode                          = syscall.GUID{Data1: 0xfeef4582, Data2: 0xef8f, Data3: 0x4f7b, Data4: [8]byte{0x85, 0x8b, 0x90, 0x77, 0xd1, 0x22, 0xde, 0x47}}
		FwpmConditionIPsecPolicyKey                  = syscall.GUID{Data1: 0xad37dee3, Data2: 0x722f, Data3: 0x45cc, Data4: [8]byte{0xa4, 0xe3, 0x06, 0x80, 0x48, 0x12, 0x44, 0x52}}
		FwpmConditionQmMode                          = syscall.GUID{Data1: 0xf64fc6d1, Data2: 0xf9cb, Data3: 0x43d2, Data4: [8]byte{0x8a, 0x5f, 0xe1, 0x3b, 0xc8, 0x94, 0xf2, 0x65}}
	*/
)
