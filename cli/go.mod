module github.com/ivpn/desktop-app/cli

go 1.13

require (
	github.com/ivpn/desktop-app/daemon v0.0.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
)

replace github.com/ivpn/desktop-app/daemon => ../daemon
