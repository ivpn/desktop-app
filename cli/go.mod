module github.com/ivpn/desktop-app/cli

go 1.24.0

require (
	github.com/ivpn/desktop-app/daemon v0.0.0
	golang.org/x/crypto v0.39.0
	golang.org/x/sys v0.33.0
	golang.org/x/term v0.32.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/native v1.1.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mdlayher/genetlink v1.3.2 // indirect
	github.com/mdlayher/netlink v1.7.2 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/mdlayher/wifi v0.5.1-0.20250704183335-1b2199ae492f // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/parsiya/golnk v0.0.0-20221103095132-740a4c27c4ff // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.zx2c4.com/wireguard/windows v0.5.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/ivpn/desktop-app/daemon => ../daemon
