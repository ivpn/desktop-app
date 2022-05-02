module github.com/ivpn/desktop-app/cli

go 1.16

require (
	github.com/ivpn/desktop-app/daemon v0.0.0
	golang.org/x/crypto v0.0.0-20220427172511-eb4f295cb31f // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e
	golang.org/x/term v0.0.0-20220411215600-e5f449aeb171
)

replace github.com/ivpn/desktop-app/daemon => ../daemon
