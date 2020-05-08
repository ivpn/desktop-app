//
//  IVPN command line interface (CLI)
//  https://github.com/ivpn/desktop-app-cli
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
	"encoding/json"
	"fmt"
	"time"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
)

func createReceiver(waitingIdx int, waitingObjectsList ...interface{}) *receiverChannel {
	waitingObjects := make(map[string]interface{})

	for _, wo := range waitingObjectsList {
		if wo == nil {
			continue
		}
		waitingType := types.GetTypeName(wo)
		waitingObjects[waitingType] = wo
	}

	receiver := &receiverChannel{
		_waitingIdx:     waitingIdx,
		_waitingObjects: waitingObjects,
		_channel:        make(chan []byte, 1)}

	return receiver
}

type receiverChannel struct {
	_waitingIdx      int
	_waitingObjects  map[string]interface{}
	_channel         chan []byte
	_receivedData    []byte
	_receivedCmdBase types.CommandBase
}

func (r *receiverChannel) GetReceivedRawData() (data []byte, cmdBaseObj types.CommandBase) {
	return r._receivedData, r._receivedCmdBase
}

func (r *receiverChannel) IsExpectedResponse(respIdx int, command string) bool {
	// response is acceptable when:
	// - received expected responseIndex
	// - we are not waiting for response index but received one of responses from _waitingObjects
	// - when we do not care about responseIndex and response objects

	if r._waitingIdx == 0 && len(r._waitingObjects) == 0 {
		return true
	}

	if r._waitingIdx != 0 {
		if r._waitingIdx == respIdx {
			return true
		}
	} else {
		if len(r._waitingObjects) > 0 {
			if _, ok := r._waitingObjects[command]; ok {
				return true
			}
		}
	}

	return false
}

func (r *receiverChannel) PushResponse(responseData []byte) {
	select {
	case r._channel <- responseData:
	default:
		logger.Error("Receiver channel is full")
	}
}

func (r *receiverChannel) Wait(timeout time.Duration) (err error) {
	select {
	case r._receivedData = <-r._channel:

		// check type of response
		if err := deserialize(r._receivedData, &r._receivedCmdBase); err != nil {
			return fmt.Errorf("response deserialisation failed: %w", err)
		}

		if len(r._waitingObjects) > 0 {
			if wo, ok := r._waitingObjects[r._receivedCmdBase.Command]; ok {
				// deserialize response into expected object type
				if err := deserialize(r._receivedData, wo); err != nil {
					return fmt.Errorf("response deserialisation failed: %w", err)
				}
			} else {
				// check is it Error object
				var errObj types.ErrorResp
				if r._receivedCmdBase.Command == types.GetTypeName(errObj) {
					if err := deserialize(r._receivedData, &errObj); err != nil {
						return fmt.Errorf("response deserialisation failed: %w", err)
					}
					return fmt.Errorf(errObj.ErrorMessage)
				}
				return fmt.Errorf("received unexpected data (type:%s)", r._receivedCmdBase.Command)
			}
		}
		return nil

	case <-time.After(timeout):
		return ResponseTimeout{}
	}
}

func deserialize(messageData []byte, obj interface{}) error {
	if err := json.Unmarshal(messageData, obj); err != nil {
		return fmt.Errorf("failed to parse command data: %w", err)
	}
	return nil
}
