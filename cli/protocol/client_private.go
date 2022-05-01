//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the IVPN command line interface.
//
//  The IVPN command line interface is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The IVPN command line interface is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the IVPN command line interface. If not, see <https://www.gnu.org/licenses/>.
//

package protocol

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/ivpn/desktop-app/cli/helpers"
	"github.com/ivpn/desktop-app/daemon/logger"
	"github.com/ivpn/desktop-app/daemon/protocol/types"
)

func (c *Client) ensureConnected() error {
	if c._conn != nil {
		return nil
	}

	// ensure we are connected
	if err := c.Connect(); err != nil {
		return err
	}
	return nil
}

func (c *Client) sendRecv(request interface{}, response interface{}) error {
	return c.sendRecvTimeOut(request, response, c._defaultTimeout)
}

func (c *Client) sendRecvTimeOut(request interface{}, response interface{}, timeout time.Duration) error {

	doJob := func() error {
		var receiver *receiverChannel

		// thread-safe receiver registration
		func() {
			c._receiversLocker.Lock()
			defer c._receiversLocker.Unlock()

			c._requestIdx++
			if c._requestIdx == 0 {
				c._requestIdx++
			}

			receiver = createReceiver(c._requestIdx, false, response)

			c._receivers[receiver] = struct{}{}
		}()

		// do not forget to remove receiver
		defer func() {
			c._receiversLocker.Lock()
			defer c._receiversLocker.Unlock()

			delete(c._receivers, receiver)
		}()

		// send request
		if err := c.send(request, receiver._waitingIdx); err != nil {
			return err
		}

		// waiting for response
		if err := receiver.Wait(timeout); err != nil {
			return err
		}

		return nil
	}

	err := doJob()
	if errResp, ok := err.(types.ErrorResp); ok && errResp.ErrorType == types.ErrorParanoidModePasswordError {
		// Paranoid mode password error
		if len(c._paranoidModeSecret) <= 0 && c._paranoidModeSecretRequestFunc != nil {
			// request user for Password
			c._paranoidModeSecret, err = c._paranoidModeSecretRequestFunc(c)
			if err != nil {
				return err
			}
			err = doJob()
		}
	}

	return err
}

func (c *Client) sendRecvAny(request interface{}, waitingObjects ...interface{}) (data []byte, cmdBase types.CommandBase, err error) {
	isIgnoreResponseIndex := true
	return c.sendRecvAnyEx(request, isIgnoreResponseIndex, waitingObjects...)
}

func (c *Client) sendRecvAnyEx(request interface{}, isIgnoreResponseIndex bool, waitingObjects ...interface{}) (data []byte, cmdBase types.CommandBase, err error) {

	doJob := func() (data []byte, cmdBase types.CommandBase, err error) {
		var receiver *receiverChannel

		var reqIdx int
		// thread-safe receiver registration
		func() {
			c._receiversLocker.Lock()
			defer c._receiversLocker.Unlock()

			c._requestIdx++
			reqIdx = c._requestIdx

			receiver = createReceiver(c._requestIdx, isIgnoreResponseIndex, waitingObjects...)

			c._receivers[receiver] = struct{}{}
		}()

		// do not forget to remove receiver
		defer func() {
			c._receiversLocker.Lock()
			defer c._receiversLocker.Unlock()

			delete(c._receivers, receiver)
		}()

		// send request
		if err := c.send(request, reqIdx); err != nil {
			return nil, types.CommandBase{}, err
		}

		// waiting for response
		if err := receiver.Wait(c._defaultTimeout); err != nil {
			return nil, types.CommandBase{}, err
		}

		data, cmdBase = receiver.GetReceivedRawData()
		return data, cmdBase, nil
	}

	data, cmdBase, err = doJob()
	if errResp, ok := err.(types.ErrorResp); ok && errResp.ErrorType == types.ErrorParanoidModePasswordError {
		// Paranoid mode password error
		if len(c._paranoidModeSecret) <= 0 && c._paranoidModeSecretRequestFunc != nil {
			// request user for Password
			c._paranoidModeSecret, err = c._paranoidModeSecretRequestFunc(c)
			if err != nil {
				return []byte{}, types.CommandBase{}, err
			}
			data, cmdBase, err = doJob()
		}
	}

	return data, cmdBase, err
}

func (c *Client) send(cmd interface{}, requestIdx int) error {
	cmdName := types.GetTypeName(cmd)

	logger.Info("--> ", cmdName)

	if err := c.initRequestFields(cmd); err != nil {
		return err
	}

	if err := types.Send(c._conn, cmd, requestIdx); err != nil {
		return fmt.Errorf("failed to send command '%s': %w", cmdName, err)
	}
	return nil
}

func (c *Client) initRequestFields(obj interface{}) error {
	if len(c._paranoidModeSecret) <= 0 && len(c._paranoidModeSecretSuAccess) <= 0 {
		return nil
	}

	valueIface := reflect.ValueOf(obj)

	// Check if the passed interface is a pointer
	if valueIface.Type().Kind() != reflect.Ptr {
		return fmt.Errorf("interface is not a pointer to a request")
	}

	if len(c._paranoidModeSecret) > 0 {
		// Get the field by name "ProtocolSecret"
		protocolSecretField := valueIface.Elem().FieldByName("ProtocolSecret")
		if !protocolSecretField.IsValid() {
			return fmt.Errorf("interface `%s` does not have the field `ProtocolSecret`", valueIface.Type())
		}
		if protocolSecretField.Type().Kind() != reflect.String {
			return fmt.Errorf("'ProtocolSecret' field of an interface `%s` is not 'string'", valueIface.Type())
		}
		protocolSecretField.Set(reflect.ValueOf(c._paranoidModeSecret))
	}

	if len(c._paranoidModeSecretSuAccess) > 0 {
		// Get the field by name "ProtocolSecretSu"
		protocolSecretField := valueIface.Elem().FieldByName("ProtocolSecretSu")
		if !protocolSecretField.IsValid() {
			return fmt.Errorf("interface `%s` does not have the field `ProtocolSecretSu`", valueIface.Type())
		}
		if protocolSecretField.Type().Kind() != reflect.String {
			return fmt.Errorf("'ProtocolSecretSu' field of an interface `%s` is not 'string'", valueIface.Type())
		}

		protocolSecretField.Set(reflect.ValueOf(base64.StdEncoding.EncodeToString(c._paranoidModeSecretSuAccess)))
	}

	return nil
}

func (c *Client) receiverRoutine() {

	defer func() {
		logger.Info("Receiver stopped")
		c._conn.Close()
	}()

	logger.Info("Receiver started")

	reader := bufio.NewReader(c._conn)

	// run loop forever
	for {
		// will listen for message to process ending in newline (\n)
		message, err := reader.ReadString('\n')
		if err != nil {
			logger.Error("Error receiving data from daemon: ", err)
			break
		}

		messageData := []byte(message)

		cmd, err := types.GetCommandBase(messageData)
		if err != nil {
			logger.Error("Failed to parse response:", err)
			return
		}

		logger.Info("<-- ", cmd.Command)

		isProcessed := false
		// thread-safe iteration trough receivers
		func() {
			c._receiversLocker.Lock()
			defer c._receiversLocker.Unlock()

			if cmd.Command == types.GetTypeName(types.HelloResp{}) {
				// update last HelloResponse object
				var hr types.HelloResp
				if err := json.Unmarshal(messageData, &hr); err == nil {
					c._helloResponse = hr

					// If we are running in privilaged environment AND if daemon informed us about secret file - read it
					// It gives us possibility to bypass EAA (if enabled)
					if len(c._helloResponse.ParanoidMode.SuAccessFile) > 0 && helpers.CheckIsAdmin() {
						if secret, err := os.ReadFile(c._helloResponse.ParanoidMode.SuAccessFile); err == nil {
							c._paranoidModeSecretSuAccess = secret
						}
					}
				}
			}

			for receiver := range c._receivers {
				if receiver.IsExpectedResponse(cmd) {
					isProcessed = true
					receiver.PushResponse(messageData)
					break
				}
			}
		}()

		if isProcessed == false {
			logger.Info(fmt.Sprintf("Response '%s:%d' not processed", cmd.Command, cmd.Idx))
		}
	}
}
