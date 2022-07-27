module github.com/ivpn/desktop-app/cli

go 1.18

require (
	github.com/ivpn/desktop-app/daemon v0.0.0
	golang.org/x/crypto v0.0.0-20220427172511-eb4f295cb31f
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e
	golang.org/x/term v0.0.0-20220411215600-e5f449aeb171
)

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/parsiya/golnk v0.0.0-20200515071614-5db3107130ce // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.7.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/ivpn/desktop-app/daemon => ../daemon
