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
	"reflect"
	"strings"
)

// CommandBase is a base object for communication with daemon.
// Contains fields required for all requests\responses.
type CommandBase struct {
	// this field represents command type
	Command string
	// Uses for separate request\response sessions.
	// Response messages must have same Index as request
	Idx int
}

func (cb *CommandBase) Init(name string, idx int) {
	cb.Command = name
	cb.Idx = idx
}
func (cb *CommandBase) Name() string {
	return cb.Command
}
func (cb *CommandBase) Index() int {
	return cb.Idx
}
func (cb *CommandBase) LogExtraInfo() string {
	return ""
}

type ResponseBase struct {
	CommandBase
	Error string
}

func (rb *ResponseBase) GetError() string {
	return rb.Error
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

// GetTypeName returns objects type name (without package)
func GetTypeName(cmd interface{}) string {
	t := reflect.TypeOf(cmd)
	typePath := strings.Split(t.String(), ".")
	if len(typePath) == 0 {
		return ""
	}
	return typePath[len(typePath)-1]
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
