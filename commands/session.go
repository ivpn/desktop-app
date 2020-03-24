package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-daemon/api/types"
	"golang.org/x/crypto/ssh/terminal"
)

type CmdLogin struct {
	flags.CmdInfo
	loginAccountID string
	forceLogin     bool
}

func (c *CmdLogin) Init() {
	c.Initialize("login", "Login operation (register accountID)")
	c.DefaultStringVar(&c.loginAccountID, "ACCOUNT_ID")
	c.BoolVar(&c.forceLogin, "force", false, "Log out from all other devices")
}

func (c *CmdLogin) Run() error {
	if len(c.loginAccountID) == 0 {
		fmt.Print("Enter your Account ID: ")
		data, err := terminal.ReadPassword(0)
		if err != nil {
			return fmt.Errorf("failed to read accountID: %w", err)
		}
		fmt.Println("")
		c.loginAccountID = string(data)
	}
	return _proto.SessionNew(c.loginAccountID, c.forceLogin)
}

//-----------------------------------------------

type CmdLogout struct {
	flags.CmdInfo
}

func (c *CmdLogout) Init() {
	c.Initialize("logout", "Logout from this device (if logged-in)")
}
func (c *CmdLogout) Run() error {
	return _proto.SessionDelete()
}

//-----------------------------------------------

type CmdAccount struct {
	flags.CmdInfo
}

func (c *CmdAccount) Init() {
	c.Initialize("account", "Get info about current account")
}
func (c *CmdAccount) Run() error {
	defer func() {
		helloResp := _proto.GetHelloResponse()
		if len(helloResp.Command) > 0 && (len(helloResp.Session.Session) == 0) {
			// We received 'hello' response but no session info - print tips to login
			fmt.Println("Tips: ")
			fmt.Println("  ivpn login        Log in with your Account ID\n")
		}
	}()

	stat, err := _proto.SessionStatus()
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
