module github.com/ivpn/desktop-app/cli

go 1.18

require (
	github.com/ivpn/desktop-app/daemon v0.0.0
	golang.org/x/crypto v0.7.0
	golang.org/x/sys v0.6.0
	golang.org/x/term v0.6.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/parsiya/golnk v0.0.0-20221103095132-740a4c27c4ff // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/stretchr/testify v1.8.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/ivpn/desktop-app/daemon => ../daemon
