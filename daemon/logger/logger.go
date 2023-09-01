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

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ivpn/desktop-app/daemon/service/platform"
	"github.com/ivpn/desktop-app/daemon/service/platform/filerights"
)

var isCanPrintToConsole bool
var isLoggingEnabled bool
var filePath string
var writeMutex sync.Mutex
var globalLogFile *os.File

var log *Logger

func init() {
	log = NewLogger("log")
}

// Init - initialize logfile
func Init(logfile string) {
	filePath = logfile
}

// GetLogText returns data from saved logs
func GetLogText(maxBytesSize int64) (log string, log0 string, err error) {
	writeMutex.Lock()
	defer writeMutex.Unlock()

	logtext1, _ := getLogText(platform.LogFile(), maxBytesSize)
	logtext2, _ := getLogText(platform.LogFile()+".0", maxBytesSize)
	return logtext1, logtext2, nil
}

func getLogText(fname string, maxBytesSize int64) (text string, err error) {

	if _, err := os.Stat(filePath); err != nil {
		if isLoggingEnabled {
			return "<<< log-file not exists >>>", nil
		}
		return "<<< logging disabled >>>", nil
	}

	file, err := os.Open(fname)
	if err != nil {
		return "<<< unable to open log-file >>>", nil
	}
	defer file.Close()

	stat, _ := file.Stat()
	filesize := stat.Size()
	if filesize < maxBytesSize {
		maxBytesSize = filesize
	}

	buf := make([]byte, maxBytesSize)
	start := stat.Size() - maxBytesSize
	_, err = file.ReadAt(buf, start)
	if err != nil {
		return fmt.Sprintf("<<< failed to read log-file: %s >>>", err), nil
	}

	return string(buf), nil
}

// IsEnabled returns true if logging enabled
func IsEnabled() bool {
	return isLoggingEnabled
}

// CanPrintToConsole define if logger can print to console
//func CanPrintToConsole(isCanPrint bool) {
//	isCanPrintToConsole = isCanPrint
//}

// Enable switching on\off logging
func Enable(isEnabled bool) {
	if isLoggingEnabled == isEnabled {
		return
	}

	var infoText string
	switch isEnabled {
	case true:
		infoText = "Logging enabled"
	case false:
		infoText = "Logging disabled"
	}

	if isLoggingEnabled {
		log.Info(infoText)
	}

	isCanPrintToConsole = isEnabled
	isLoggingEnabled = isEnabled

	if isLoggingEnabled {
		log.Info(infoText)
	} else {
		deleteLogFile()
	}
}

// Info - Log info message
func Info(v ...interface{}) { _info("", v...) }

// Debug - Log Debug message
func Debug(v ...interface{}) { _debug("", v...) }

// Warning - Log Warning message
func Warning(v ...interface{}) { _warning("", v...) }

// Trace - Log Trace message
func Trace(v ...interface{}) { _trace("", v...) }

// Error - Log Error message
func Error(v ...interface{}) { _error("", 0, v...) }

// ErrorTrace - Log error with trace
func ErrorTrace(e error) { _errorTrace("", e) }

// Panic - Log Error message and call panic()
func Panic(v ...interface{}) { _panic("", v...) }

// Logger - standalone logger object
type Logger struct {
	pref       string
	isDisabled bool
}

// NewLogger - create named logger object
func NewLogger(prefix string) *Logger {
	prefix = strings.Trim(prefix, " [],./:\\")

	if len(prefix) > 6 {
		newprefix := prefix[:6]
		Debug(fmt.Sprintf("*** Logger name '%s' cut to 6 characters: '%s'***", prefix, newprefix))
		prefix = newprefix
	}

	prefix = strings.Trim(prefix, " [],./:\\")

	if prefix != "" {
		for len(prefix) < 6 {
			prefix = prefix + " "
		}
	}

	prefix = "[" + prefix + "]"
	return &Logger{pref: prefix}
}

// Info - Log info message
func (l *Logger) Info(v ...interface{}) {
	if l.isDisabled {
		return
	}
	_info(l.pref, v...)
}

// Debug - Log Debug message
func (l *Logger) Debug(v ...interface{}) {
	if l.isDisabled {
		return
	}
	_debug(l.pref, v...)
}

// Warning - Log Warning message
func (l *Logger) Warning(v ...interface{}) {
	if l.isDisabled {
		return
	}
	_warning(l.pref, v...)
}

// Trace - Log Trace message
func (l *Logger) Trace(v ...interface{}) {
	if l.isDisabled {
		return
	}
	_trace(l.pref, v...)
}

// Error - Log Error message
func (l *Logger) Error(v ...interface{}) {
	if l.isDisabled {
		return
	}
	_error(l.pref, 0, v...)
}

// ErrorE - Log Error and return same error object
// (useful in constrictions: " return log.ErrorE(err) " )
func (l *Logger) ErrorE(err error, callerStackOffset int) error {
	if l.isDisabled {
		return err
	}
	_error(l.pref, callerStackOffset, err)
	return err
}

// ErrorTrace - Log error with trace
func (l *Logger) ErrorTrace(e error) {
	if l.isDisabled {
		return
	}
	_errorTrace(l.pref, e)
}

// Panic - Log Error message and call panic()
func (l *Logger) Panic(v ...interface{}) {
	if l.isDisabled {
		return
	}
	_panic(l.pref, v...)
}

// Enable - enable\disable logger
func (l *Logger) Enable(enable bool) { l.isDisabled = !enable }

func _info(name string, v ...interface{}) {
	mes, timeStr, _, _ := getLogPrefixes(fmt.Sprint(v...), 0)
	write(timeStr, name, mes)
}

func _debug(name string, v ...interface{}) {
	mes, timeStr, runtimeInfo, _ := getLogPrefixes(fmt.Sprint(v...), 0)
	write(timeStr, name, "DEBUG", runtimeInfo, mes)
}

func _warning(name string, v ...interface{}) {
	mes, timeStr, runtimeInfo, _ := getLogPrefixes(fmt.Sprint(v...), 0)
	write(timeStr, name, "WARNING", runtimeInfo, mes)
}

func _trace(name string, v ...interface{}) {
	mes, timeStr, runtimeInfo, methodInfo := getLogPrefixes(fmt.Sprint(v...), 0)
	write(timeStr, name, "TRACE", runtimeInfo+methodInfo, mes)
}

func _error(name string, callerStackOffset int, v ...interface{}) {
	mes, timeStr, runtimeInfo, methodInfo := getLogPrefixes(fmt.Sprint(v...), callerStackOffset)
	write(timeStr, name, "ERROR", runtimeInfo+methodInfo, mes)
}

func _errorTrace(name string, err error) {
	mes, timeStr, runtimeInfo, methodInfo := getLogPrefixes(getErrorDetails(err), 0)
	write(timeStr, name, "ERROR", runtimeInfo+methodInfo, mes)
}

func _panic(name string, v ...interface{}) {
	mes, timeStr, runtimeInfo, methodInfo := getLogPrefixes(fmt.Sprint(v...), 0)

	//fmt.Println(timeStr, "PANIC", runtimeInfo+methodInfo, mes)
	write(timeStr, name, "PANIC", runtimeInfo+methodInfo, mes)

	panic(runtimeInfo + methodInfo + ": " + mes)
}

func getErrorDetails(err error) string {
	return fmt.Sprintf("%v", err)
}

func getCallerMethodName(callerStackOffset int) (string, error) {
	fpcs := make([]uintptr, 1)
	// Skip 5 levels to get the caller
	n := runtime.Callers(5+callerStackOffset, fpcs)
	if n == 0 {
		return "", fmt.Errorf("no caller")
	}

	caller := runtime.FuncForPC(fpcs[0] - 1)
	if caller == nil {
		return "", fmt.Errorf("msg caller is nil")
	}

	return caller.Name(), nil
}

func getLogPrefixes(message string, callerStackOffset int) (retMes string, timeStr string, runtimeInfo string, methodInfo string) {
	t := time.Now()

	if _, filename, line, isRuntimeInfoOk := runtime.Caller(3 + callerStackOffset); isRuntimeInfoOk {
		runtimeInfo = filepath.Base(filename) + ":" + strconv.Itoa(line) + ":"

		if methodName, err := getCallerMethodName(callerStackOffset); err == nil {
			methodInfo = "(in " + methodName + "):"
		}
	}

	timeStr = t.Format(time.StampMilli)
	retMes = strings.TrimRight(message, "\n")

	return retMes, timeStr, runtimeInfo, methodInfo
}

func write(fields ...interface{}) {
	writeMutex.Lock()
	defer writeMutex.Unlock()

	if isLoggingEnabled {
		if isCanPrintToConsole {
			// printing into console
			fmt.Println(fields...)
		}

		if globalLogFile == nil {
			createLogFile()
		}

		if globalLogFile != nil {
			// writting into log-file
			globalLogFile.WriteString(fmt.Sprintln(fields...))
		}
	}
}

func deleteLogFile() {
	writeMutex.Lock()
	defer writeMutex.Unlock()

	if globalLogFile != nil {
		globalLogFile.Close()
		globalLogFile = nil
	}

	if len(filePath) > 0 {
		os.Remove(filePath)
		os.Remove(filePath + ".0")
	}
}

func createLogFile() error {
	if globalLogFile != nil {
		globalLogFile.Close()
		globalLogFile = nil
	}

	if len(filePath) > 0 {
		if _, err := os.Stat(filePath); err == nil {
			os.Rename(filePath, filePath+".0")
		}

		var err error
		globalLogFile, err = os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600) // read\write only for privileged user
		if err != nil {
			return fmt.Errorf("failed to create log-file: %w", err)
		}
		// only for Windows: Golang is not able to change file permissins in Windows style
		if err := filerights.WindowsChmod(filePath, 0600); err != nil { // read\write only for privileged user
			return fmt.Errorf("failed to change log-file permissions: %w", err)
		}
	} else {
		return fmt.Errorf("logfile name not initialized")
	}

	return nil
}
