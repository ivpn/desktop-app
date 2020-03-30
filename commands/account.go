package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-daemon/api/types"
	"github.com/ivpn/desktop-app-daemon/service"
	"golang.org/x/crypto/ssh/terminal"
)

type CmdAccount struct {
	flags.CmdInfo
	login     bool
	accountID string
	force     bool
	logout    bool
	status    bool
}

func (c *CmdAccount) Init() {
	c.Initialize("account", "Get info about current account (ACCOUNT_ID applicable only with 'login' option)")
	c.DefaultStringVar(&c.accountID, "ACCOUNT_ID")
	c.BoolVar(&c.login, "login", false, "Login operation (register ACCOUNT_ID on this device)")
	c.BoolVar(&c.force, "force", false, "Log out from all other devices (applicable only with 'login' option)")
	c.BoolVar(&c.logout, "logout", false, "Logout from this device (if logged-in)")
	c.BoolVar(&c.status, "status", false, "(default) Information about account status")
}

func (c *CmdAccount) Run() error {
	if c.login && (c.logout || c.status) {
		return flags.BadParameter{}
	}
	if c.force && c.logout {
		return flags.BadParameter{}
	}
	if c.logout && (c.status || len(c.accountID) > 0) {
		return flags.BadParameter{}
	}
	if c.status && (c.login || c.logout) {
		return flags.BadParameter{}
	}

	if c.logout {
		return c.doLogout()
	}

	if c.login {
		return c.doLogin(c.accountID, c.force)
	}

	if len(c.accountID) > 0 {
		return flags.BadParameter{}
	}

	return c.checkStatus()
}

func (c *CmdAccount) doLogin(accountID string, force bool) error {
	if len(accountID) == 0 {
		fmt.Print("Enter your Account ID: ")
		data, err := terminal.ReadPassword(0)
		if err != nil {
			return fmt.Errorf("failed to read accountID: %w", err)
		}
		fmt.Println("")
		accountID = string(data)
	}
	apiStatus, err := _proto.SessionNew(accountID, force)
	if err != nil {
		if apiStatus == types.CodeSessionsLimitReached {
			fmt.Println("Tips: ")
			fmt.Printf("  %s account -force -login  ACCOUNT_ID         Log in with your Account ID and logout from all other devices\n\n", os.Args[0])
		}
		return err
	}
	return nil
}

func (c *CmdAccount) doLogout() error {
	return _proto.SessionDelete()
}

func (c *CmdAccount) checkStatus() error {
	stat, err := _proto.SessionStatus()

	helloResp := _proto.GetHelloResponse()
	if len(helloResp.Command) > 0 && (len(helloResp.Session.Session) == 0) {
		// We received 'hello' response but no session info - print tips to login
		fmt.Printf("Error: Not logged in\n\n")
		fmt.Println("Tips: ")
		fmt.Println(" ", service.ErrorNotLoggedIn{})
		fmt.Printf("  %s account -login  ACCOUNT_ID         Log in with your Account ID\n\n", os.Args[0])
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
