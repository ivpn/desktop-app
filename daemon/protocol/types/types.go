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

package types

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
)

type ICommandBase interface {
	LogExtraInfo() string
}

// CommandBase is a base object for communication with daemon.
// Contains fields required for all requests\responses.
type CommandBase struct {
	// this field represents command type
	Command string
	// Uses for separate request\response sessions.
	// Response messages must have same Index as request
	Idx int
}

func (cb CommandBase) LogExtraInfo() string {
	return ""
}

// RequestBase contains fields which are common for requests to a daemon
type RequestBase struct {
	CommandBase
	ProtocolSecret string
}

type ServicePreference string

const (
	Prefs_IsEnableLogging              ServicePreference = "enable_logging"
	Prefs_IsAutoconnectOnLaunch        ServicePreference = "autoconnect_on_launch"
	Prefs_IsAutoconnectOnLaunch_Daemon ServicePreference = "autoconnect_on_launch_daemon"
)

func (sp ServicePreference) Equals(key string) bool {
	return key == string(sp)
}

// Send sends a command to a connection : init+serialize+send
func Send(conn net.Conn, cmd interface{}, idx int) (retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("failed to send command to client: %w", retErr)
			log.Error(retErr)
		}
	}()

	bytesToSend, err := serialize(cmd, idx)
	if err != nil {
		return fmt.Errorf("unable to send command: %w", err)
	}

	if bytesToSend == nil {
		return fmt.Errorf("data is nil")
	}

	bytesToSend = append(bytesToSend, byte('\n'))
	if _, err := conn.Write(bytesToSend); err != nil {
		return err
	}

	return nil
}

// GetRequestBase deserializing to RequestBase object
func GetRequestBase(messageData []byte) (RequestBase, error) {
	var obj RequestBase
	if err := json.Unmarshal(messageData, &obj); err != nil {
		return obj, fmt.Errorf("failed to parse request data: %w", err)
	}

	if len(obj.Command) == 0 {
		return obj, fmt.Errorf("request name is not defined")
	}

	return obj, nil
}

// GetCommandBase deserializing to CommandBase object
func GetCommandBase(messageData []byte) (CommandBase, error) {
	var obj CommandBase
	if err := json.Unmarshal(messageData, &obj); err != nil {
		return obj, fmt.Errorf("failed to parse command data: %w", err)
	}

	if len(obj.Command) == 0 {
		return obj, fmt.Errorf("command name is not defined")
	}

	return obj, nil
}

// GetTypeName returns objects type name (without package)
func GetTypeName(cmd interface{}) string {
	t := reflect.TypeOf(cmd)
	typePath := strings.Split(t.String(), ".")
	if len(typePath) == 0 {
		return ""
	}
	return typePath[len(typePath)-1]
}

// Serialize initializing 'Command' field and serializing object
func serialize(cmd interface{}, idx int) (ret []byte, err error) {
	if err := initCmdFields(cmd, idx); err != nil {
		return nil, err
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// initCmdName initializes 'Command' field of given interface by it's type name
// (useful to initialize request or response objects)
func initCmdFields(obj interface{}, idx int) error {
	valueIface := reflect.ValueOf(obj)

	// Check if the passed interface is a pointer
	if valueIface.Type().Kind() != reflect.Ptr {
		return fmt.Errorf("interface is not a pointer")
	}

	// Get the field by name "Command"
	commandField := valueIface.Elem().FieldByName("Command")
	if !commandField.IsValid() {
		return fmt.Errorf("interface `%s` does not have the field `Command`", valueIface.Type())
	}
	if commandField.Type().Kind() != reflect.String {
		return fmt.Errorf("'Command' field of an interface `%s` is not 'string'", valueIface.Type())
	}

	// set command to a type name
	name := GetTypeName(obj)
	if len(name) == 0 {
		return fmt.Errorf("unable to determine type name of the interface")
	}
	commandField.Set(reflect.ValueOf(name))

	if idx != 0 {
		// Get the field by name "Idx"
		idxField := valueIface.Elem().FieldByName("Idx")
		if !idxField.IsValid() {
			return fmt.Errorf("interface `%s` does not have the field `Idx`", valueIface.Type())
		}
		if idxField.Type().Kind() != reflect.Int {
			return fmt.Errorf("'Idx' field of an interface `%s` is not 'string'", valueIface.Type())
		}
		// set index
		idxField.Set(reflect.ValueOf(idx))
	}
	return nil
}
