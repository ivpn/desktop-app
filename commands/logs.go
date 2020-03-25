package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/ivpn/desktop-app-cli/flags"
)

type CmdLogs struct {
	flags.CmdInfo
	show    bool
	enable  bool
	disable bool
}

func (c *CmdLogs) Init() {
	c.Initialize("logs", "Possibility to enable\\view logs")
	c.BoolVar(&c.show, "show", false, "(default) Show logs")
	c.BoolVar(&c.enable, "enable", false, "Enable logging")
	c.BoolVar(&c.disable, "disable", false, "Disable logging")
}
func (c *CmdLogs) Run() error {
	if c.enable && c.disable {
		return flags.BadParameter{}
	}

	var err error
	if c.enable {
		err = c.setSetLogging(true)
	} else if c.disable {
		err = c.setSetLogging(false)
	}

	if err != nil || c.enable || c.disable {
		return err
	}
	return c.doShow()
}

func (c *CmdLogs) setSetLogging(enable bool) error {
	if enable {
		_proto.SetPreferences("enable_logging", "true")
	}
	return _proto.SetPreferences("enable_logging", "false")
}

func (c *CmdLogs) doShow() error {

	fname := "/opt/ivpn/log/IVPN Agent.log"
	file, err := os.Open(fname)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	stat, err := os.Stat(fname)
	size := stat.Size()

	isPartOfFile := false
	maxBytesToRead := int64(60 * 50)
	if size > maxBytesToRead {
		isPartOfFile = true
		if _, err := file.Seek(-maxBytesToRead, io.SeekEnd); err != nil {
			return nil
		}
	}

	buff := make([]byte, maxBytesToRead)
	if _, err := file.Read(buff); err != nil {
		return nil
	}

	fmt.Println(string(buff))

	if isPartOfFile {
		fmt.Println("##############")
		fmt.Println("To view full log, please refer to file:", fname)
	}

	return nil
}
