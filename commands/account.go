package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/service"
	"github.com/ivpn/desktop-app-daemon/vpn"
	"golang.org/x/crypto/ssh/terminal"
)

type CmdLogout struct {
	flags.CmdInfo
}

func (c *CmdLogout) Init() {
	c.Initialize("logout", "Logout from this device (if logged-in)")
}

func (c *CmdLogout) Run() error {
	return doLogout()
}

//----------------------------------------------------------------------------------------
type CmdLogin struct {
	flags.CmdInfo
	accountID string
	force     bool
}

func (c *CmdLogin) Init() {
	c.Initialize("login", "Login operation (register ACCOUNT_ID on this device)")
	c.DefaultStringVar(&c.accountID, "ACCOUNT_ID")
	c.BoolVar(&c.force, "force", false, "Log out from all other devices (applicable only with 'login' option)")
}

func (c *CmdLogin) Run() error {
	return doLogin(c.accountID, c.force)
}

func doLogin(accountID string, force bool) error {
	// checking if we are logged-in
	_proto.SessionStatus() // do not check error response (could be received 'not logged in' errors)
	helloResp := _proto.GetHelloResponse()
	if len(helloResp.Session.Session) != 0 {
		fmt.Println("Already logged in")
		PrintTips([]TipType{TipLogout})
		return fmt.Errorf("unable login (please, log out first)")
	}

	// login
	if len(accountID) == 0 {
		fmt.Print("Enter your Account ID: ")
		data, err := terminal.ReadPassword(0)
		if err != nil {
			return fmt.Errorf("failed to read accountID: %w", err)
		}
		accountID = string(data)
	}

	apiStatus, err := _proto.SessionNew(accountID, force)
	if err != nil {
		if apiStatus == types.CodeSessionsLimitReached {
			PrintTips([]TipType{TipForceLogin})
		}
		return err
	}

	fmt.Println("Logged in")
	PrintTips([]TipType{TipServers, TipConnectHelp})

	return nil
}

//----------------------------------------------------------------------------------------

type CmdAccount struct {
	flags.CmdInfo
}

func (c *CmdAccount) Init() {
	c.Initialize("account", "Get info about current account")
}

func (c *CmdAccount) Run() error {
	return checkStatus()
}

//----------------------------------------------------------------------------------------

func doLogout() error {
	// checking if we are logged-in
	_proto.SessionStatus() // do not check error response (could be received 'not logged in' errors)
	helloResp := _proto.GetHelloResponse()
	if len(helloResp.Session.Session) == 0 {
		return fmt.Errorf("already logged out")
	}

	// do not allow to logout if VPN connected
	state, _, err := _proto.GetVPNState()
	if err != nil {
		return err
	}
	if state != vpn.DISCONNECTED {
		PrintTips([]TipType{TipDisconnect})
		return fmt.Errorf("unable to log out (please, disconnect VPN first)")
	}

	// delete session
	err = _proto.SessionDelete()
	if err != nil {
		return err
	}

	fmt.Println("Logged out")
	PrintTips([]TipType{TipLogin})

	return nil
}

func checkStatus() error {
	stat, err := _proto.SessionStatus()

	helloResp := _proto.GetHelloResponse()
	if len(helloResp.Command) > 0 && (len(helloResp.Session.Session) == 0) {
		// We received 'hello' response but no session info - print tips to login
		fmt.Printf("Error: Not logged in\n\n")

		fmt.Println()
		PrintTips([]TipType{TipLogin})
		fmt.Println()

		fmt.Println(service.ErrorNotLoggedIn{})
		return nil
	}

	if err != nil {
		return err
	}

	if stat.APIStatus != types.CodeSuccess {
		return fmt.Errorf("API error: %v %v", stat.APIStatus, stat.APIErrorMessage)
	}

	acc := stat.Account
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	if acc.IsFreeTrial {
		fmt.Fprintln(w, fmt.Sprintf("Plan:\tFree Trial"))
	} else {
		fmt.Fprintln(w, fmt.Sprintf("Plan:\t%v", acc.CurrentPlan))
	}
	fmt.Fprintln(w, fmt.Sprintf("Active until:\t%v", time.Unix(acc.ActiveUntil, 0)))
	if stat.Account.Limit > 0 {
		fmt.Fprintln(w, fmt.Sprintf("Devices limit:\t%v", acc.Limit))
	}
	if acc.Upgradable == true && len(acc.UpgradeToPlan) > 0 && len(acc.UpgradeToURL) > 0 {
		fmt.Fprintln(w, fmt.Sprintf("Upgrade to:\t%v (%v)", acc.UpgradeToPlan, acc.UpgradeToURL))
	}
	w.Flush()

	return nil
}
