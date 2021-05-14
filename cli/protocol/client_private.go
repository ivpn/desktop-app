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
	"fmt"
	"time"

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
	var receiver *receiverChannel

	// thread-safe receiver registration
	func() {
		c._receiversLocker.Lock()
		defer c._receiversLocker.Unlock()

		c._requestIdx++
		if c._requestIdx == 0 {
			c._requestIdx++
		}

		receiver = createReceiver(c._requestIdx, response)

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
		return fmt.Errorf("failed to receive response: %w", err)
	}

	return nil
}

func (c *Client) sendRecvAny(request interface{}, waitingObjects ...interface{}) (data []byte, cmdBase types.CommandBase, err error) {
	var receiver *receiverChannel

	var reqIdx int
	// thread-safe receiver registration
	func() {
		c._receiversLocker.Lock()
		defer c._receiversLocker.Unlock()

		c._requestIdx++
		reqIdx = c._requestIdx
		receiver = createReceiver(0, waitingObjects...)

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
		return nil, types.CommandBase{}, fmt.Errorf("failed to receive response: %w", err)
	}

	data, cmdBase = receiver.GetReceivedRawData()
	return data, cmdBase, nil
}

func (c *Client) send(cmd interface{}, requestIdx int) error {
	cmdName := types.GetTypeName(cmd)

	logger.Info("--> ", cmdName)
	if err := types.Send(c._conn, cmd, requestIdx); err != nil {
		return fmt.Errorf("failed to send command '%s': %w", cmdName, err)
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

			for receiver := range c._receivers {
				if receiver.IsExpectedResponse(cmd.Idx, cmd.Command) {
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
