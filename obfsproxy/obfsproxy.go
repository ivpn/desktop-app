package obfsproxy

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/shell"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger("obfpxy")
	rand.Seed(time.Now().UnixNano())
}

const (
	constDefLocalPort         = 1050
	constMaxConnectionRetries = 7
)

// Obfsproxy structure. Contains info about obfsproxy binary
type Obfsproxy struct {
	binaryPath string
	proc       *startedCmd
}

// CreateObfsproxy creates new obfsproxy object
func CreateObfsproxy(theBinaryPath string) (obj *Obfsproxy) {
	return &Obfsproxy{binaryPath: theBinaryPath}
}

// Start - asynchronously start obfsproxy
func (p *Obfsproxy) Start() (port int, err error) {

	localPort := constDefLocalPort
	log.Info(fmt.Sprintf("Starting obfsproxy on local port %d", localPort))
	defer func() {
		if err != nil || port <= 0 {
			log.Error("Error starting obfsproxy")
			p.Stop()
		}
	}()

	// retry-connection loop (required local port may be already in use)
	for doNextTry, tryNo := true, 0; doNextTry == true && tryNo < constMaxConnectionRetries; tryNo++ {
		doNextTry = false

		prepareFoNextTry := func() {
			doNextTry = true
			newPort := getRandPort()
			log.Info(fmt.Sprintf("Local port %d already in use. Trying another port %d.", localPort, newPort))
			localPort = newPort
		}

		if checkIsPortInUse(localPort) {
			prepareFoNextTry()
			continue
		}

		command, err := p.start(localPort)
		if err != nil {
			if _, ok := err.(*obfsStartErrorPortInUse); ok {
				prepareFoNextTry()
				continue
			}
			return 0, fmt.Errorf("failed to start obfsproxy: %w", err)
		}

		p.proc = command
		return localPort, nil
	}

	return 0, errors.New("obfsproxy not started")
}

// Wait - wait for obfsproxy process stop
func (p *Obfsproxy) Wait() error {
	prc := p.proc
	if prc == nil {
		return nil
	}

	<-prc.stopped
	return prc.exitError
}

// Stop - stop obfsproxy
func (p *Obfsproxy) Stop() {
	prc := p.proc
	if prc == nil {
		return
	}

	log.Info("Stopping obfsproxy...")
	if err := shell.Kill(prc.command); err != nil {
		log.Error(err)
	}
}

type obfsStartErrorPortInUse struct {
}

func (e *obfsStartErrorPortInUse) Error() string {
	return "Port already in use"
}

type startedCmd struct {
	command   *exec.Cmd
	stopped   <-chan struct{}
	exitError error
}

func (p *Obfsproxy) start(localPort int) (command *startedCmd, err error) {
	// obfsproxy command with arguments
	cmd := exec.Command(p.binaryPath, "obfs3", "socks", fmt.Sprintf("127.0.0.1:%d", localPort))

	defer func() {
		if err != nil {
			// in case of error - ensure process is stopped
			shell.Kill(cmd)
			command = nil
		}
	}()

	// process console output
	isPortInUse := false
	isFirstOutput := false

	outputParseFunc := func(text string, isError bool) {
		// notify first console output received
		isFirstOutput = true

		if isPortInUse { // Do not log other obfsproxy output. We already logged the problem "address already in use"
			return
		}

		if isError { // logging error output
			log.Error("[ERR] ", text)
		} else {
			log.Debug("[OUT] ", text)
		}

		// TODO: NOTE! hardcoded string!
		// Output example: [ERROR] Couldn't listen on 127.0.0.1:1050: [Errno 48] Address already in use.
		if strings.Contains(strings.ToLower(text), "errno 48") || strings.Contains(strings.ToLower(text), "address already in use") {
			isPortInUse = true
		}
	}

	// register colsole output reader for a process
	if err := shell.StartConsoleReaders(cmd, outputParseFunc); err != nil {
		log.Error("Failed to init obfsproxy command: ", err.Error())
		return nil, err
	}

	// Start obfsxproxy process
	if err := cmd.Start(); err != nil {
		log.Error("Failed to start obfsproxy: ", err.Error())
		return nil, err
	}

	stoppedChan := make(chan struct{}, 1)
	var porocStoppedError error
	go func() {
		porocStoppedError = cmd.Wait()
		log.Info("Obfsproxy stopped")
		stoppedChan <- struct{}{}
		close(stoppedChan)
	}()

	started := time.Now()
	// waiting for first channel output (ensure process is started)
	for isFirstOutput == false && shell.IsRunning(cmd) {
		time.Sleep(time.Millisecond * 10)

		if time.Since(started) > time.Second*10 { // timeout limit to start obfsproxy process = 10 seconds
			return nil, errors.New("obfsproxy start timeout")
		}
		if isPortInUse {
			return nil, &obfsStartErrorPortInUse{}
		}
	}

	// wait some to ensure process succesfully started
	// TODO: necessary to think how to avoid using hardcoded 'Sleep()'
	time.Sleep(time.Millisecond * 10)

	if isPortInUse {
		return nil, &obfsStartErrorPortInUse{}
	}

	log.Info(fmt.Sprintf("Started on port %d", localPort))
	return &startedCmd{command: cmd, stopped: stoppedChan, exitError: porocStoppedError}, nil
}

func getRandPort() int {
	return constDefLocalPort + rand.Intn(3000)
}

func checkIsPortInUse(localPort int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		return false
	}

	defer conn.Close()
	return true
}
