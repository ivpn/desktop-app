module github.com/ivpn/desktop-app/cli

go 1.16

require (
	github.com/ivpn/desktop-app/daemon v0.0.0
	golang.org/x/sys v0.0.0-20211205182925-97ca703d548d
	golang.org/x/term v0.0.0-20220411215600-e5f449aeb171
)

replace github.com/ivpn/desktop-app/daemon => ../daemon
