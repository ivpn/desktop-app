package shell

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ivpn/desktop-app-daemon/logger"

	"github.com/pkg/errors"
)

// Exec - execute external process
// Synchronous operation. Waits untill proces finished
func Exec(logger *logger.Logger, name string, args ...string) error {
	if logger != nil {
		logger.Info("Shell exec: ", append([]string{name}, args...))
	}

	cmd := exec.Command(name, args...)

	if err := cmd.Start(); err != nil {
		if logger != nil {
			logger.Error("Shell exec: ", err)
		}
		return err
	}

	if err := cmd.Wait(); err != nil {
		if logger != nil {
			logger.Error("Shell exec: ", err)
		}

		exCode, e := GetCmdExitCode(err)
		if e != nil {
			return errors.Wrap(e, fmt.Sprintf("ExitCode=%d", exCode))
		}

		return err
	}

	return nil
}

// GetCmdExitCode - try to get command ExitCode from
// error received from 'Exec(...)'
func GetCmdExitCode(err error) (retCode int, retErr error) {
	if err == nil {
		return 0, errors.New("unable to get the command exit-code. Error object os nil")
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode(), nil
	}

	return -1, err
}

// ExecAndProcessOutput - execute external process
// Synchronous operation. Waits untill proces finished
func ExecAndProcessOutput(logger *logger.Logger, outProcessFunc func(text string, isError bool), textToHideInLog string, name string, args ...string) error {
	outChan := make(chan string, 1)
	errChan := make(chan string, 1)
	done := make(chan bool)
	// parsing out channels
	go func() {
		defer func() {
			done <- true
		}()

		if outProcessFunc == nil {
			return
		}

		isOutClosed := false
		isErrClosed := false
		for {
			select {
			case outText, ok := <-outChan:
				if !ok {
					isOutClosed = true
				} else {
					outProcessFunc(outText, false)
				}
			case errText, ok := <-errChan:
				if !ok {
					isErrClosed = true
				} else {
					outProcessFunc(errText, true)
				}
			}

			if isOutClosed && isErrClosed {
				return
			}
		}

	}()

	err := ExecEx(logger, outChan, errChan, textToHideInLog, name, args...)
	<-done
	return err
}

// ExecEx - execute external process
// Synchronous operation. Waits untill proces finished
func ExecEx(logger *logger.Logger, outChan chan<- string, errChan chan<- string, textToHideInLog string, name string, args ...string) error {
	if logger != nil {
		logtext := strings.Join(append([]string{name}, args...), " ")
		if len(textToHideInLog) > 0 {
			logtext = strings.ReplaceAll(logtext, textToHideInLog, "***")
		}
		logger.Info("Shell exec: ", logtext)
	}

	cmd := exec.Command(name, args...)

	if outChan != nil {
		outPipe, err := cmd.StdoutPipe()
		if err != nil {
			if logger != nil {
				logger.Error("Shell exec: ", err)
			}
			return err
		}
		outPipeScanner := bufio.NewScanner(outPipe)
		go func() {
			for outPipeScanner.Scan() {
				outChan <- outPipeScanner.Text()
			}
			close(outChan)
		}()
	}

	if errChan != nil {
		errPipe, err := cmd.StderrPipe()
		if err != nil {
			if logger != nil {
				logger.Error("Shell exec: ", err)
			}
			return err
		}
		errPipeScanner := bufio.NewScanner(errPipe)
		go func() {
			for errPipeScanner.Scan() {
				errChan <- errPipeScanner.Text()
			}
			close(errChan)
		}()
	}

	if err := cmd.Start(); err != nil {
		if logger != nil {
			logger.Error("Shell exec: ", err)
		}
		return err
	}

	if err := cmd.Wait(); err != nil {
		if logger != nil {
			logger.Error("Shell exec: ", err)
		}
		return err
	}

	return nil
}

// StartConsoleReaders - init function-reader of process console text
func StartConsoleReaders(cmd *exec.Cmd, outProcessFunc func(text string, isError bool)) error {
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	outPipeScanner := bufio.NewScanner(outPipe)
	go func() {
		for outPipeScanner.Scan() {
			outProcessFunc(outPipeScanner.Text(), false)
		}
	}()

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	errPipeScanner := bufio.NewScanner(errPipe)
	go func() {
		for errPipeScanner.Scan() {
			outProcessFunc(outPipeScanner.Text(), true)
		}
	}()

	return nil
}

// Kill trying to kill process
func Kill(cmd *exec.Cmd) error {
	// ProcessState contains information about an exited process,
	// available after a call to Wait or Run.
	// (NOT nil = process finished)
	if cmd == nil || cmd.Process == nil || cmd.ProcessState != nil {
		return nil // nothing to stop
	}

	return cmd.Process.Kill()
}

// IsRunning - true when process is currently running
func IsRunning(cmd *exec.Cmd) bool {
	// ProcessState contains information about an exited process,
	// available after a call to Wait or Run.
	// (NOT nil = process finished)
	if cmd == nil || cmd.Process == nil || cmd.ProcessState != nil {
		return false
	}
	return true
}
