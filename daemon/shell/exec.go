//
//  Daemon for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2023 IVPN Limited.
//
//  This file is part of the Daemon for IVPN Client Desktop.
//
//  The Daemon for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The Daemon for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the Daemon for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

package shell

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type LogInterface interface {
	Info(v ...interface{})
	Error(v ...interface{})
}

// Exec - execute external process
// Synchronous operation. Waits until process finished
func Exec(logger LogInterface, name string, args ...string) error {
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
			return fmt.Errorf("ExitCode=%d: %w", exCode, e)
		}

		return err
	}

	return nil
}

// GetCmdExitCode - try to get command ExitCode from
// error received from 'Exec(...)'
func GetCmdExitCode(err error) (retCode int, retErr error) {
	if err == nil {
		return 0, fmt.Errorf("unable to get the command exit-code. Error object is nil")
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode(), nil
	}

	return -1, err
}

// ExecAndProcessOutput - execute external process
// Synchronous operation. Waits until process finished
func ExecAndProcessOutput(logger LogInterface, outProcessFunc func(text string, isError bool), textToHideInLog string, name string, args ...string) error {
	outChan := make(chan string, 1)
	errChan := make(chan string, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	// parsing out channels
	go func() {
		defer wg.Done()

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
	wg.Wait()

	return err
}

// ExecAndGetOutput - execute external process and return it's console output
func ExecAndGetOutput(logger LogInterface, maxRetBuffSize int, textToHideInLog string, name string, args ...string) (outText string, outErrText string, exitCode int, isBufferTooSmall bool, err error) {
	strOut := strings.Builder{}
	strErr := strings.Builder{}
	isBufferTooSmall = false
	outProcessFunc := func(text string, isError bool) {
		if len(text) == 0 {
			return
		}
		if isError {
			if strErr.Len() > maxRetBuffSize {
				isBufferTooSmall = true
				return
			}
			strErr.WriteString(text + "\n")
		} else {
			if strOut.Len() > maxRetBuffSize {
				isBufferTooSmall = true
				return
			}
			strOut.WriteString(text + "\n")
		}
	}

	retErr := ExecAndProcessOutput(logger, outProcessFunc, textToHideInLog, name, args...)

	retExitCode := 0
	if retErr != nil {
		if code, e := GetCmdExitCode(retErr); e == nil {
			retExitCode = code
		}
	}

	return strOut.String(), strErr.String(), retExitCode, isBufferTooSmall, retErr
}

// ExecEx - execute external process
// Synchronous operation. Waits until process finished
func ExecEx(logger LogInterface, outChan chan<- string, errChan chan<- string, textToHideInLog string, name string, args ...string) error {
	if logger != nil {
		logtext := strings.Join(append([]string{name}, args...), " ")
		if len(textToHideInLog) > 0 {
			logtext = strings.ReplaceAll(logtext, textToHideInLog, "***")
		}
		logger.Info("Shell exec: ", logtext)
	}

	cmd := exec.Command(name, args...)

	var wg sync.WaitGroup

	if outChan != nil {
		outPipe, err := cmd.StdoutPipe()
		if err != nil {
			if logger != nil {
				logger.Error("Shell exec: ", err)
			}
			return err
		}
		outPipeScanner := bufio.NewScanner(outPipe)
		wg.Add(1)
		go func() {
			defer wg.Done()
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
		wg.Add(1)
		go func() {
			defer wg.Done()
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

	wg.Wait()
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

	go func() {
		outPipeScanner := bufio.NewScanner(outPipe)
		for outPipeScanner.Scan() {
			outProcessFunc(outPipeScanner.Text(), false)
		}
	}()

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	go func() {
		errPipeScanner := bufio.NewScanner(errPipe)
		for errPipeScanner.Scan() {
			outProcessFunc(errPipeScanner.Text(), true)
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
