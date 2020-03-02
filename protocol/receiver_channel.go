package protocol

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
)

func createReceiver(waitingIdx int, waitingObject interface{}) *receiverChannel {
	var waitingType string
	if waitingObject != nil {
		waitingType = types.GetTypeName(waitingObject)
	}

	receiver := &receiverChannel{
		_waitingIdx:    waitingIdx,
		_waitingType:   waitingType,
		_waitingObject: waitingObject,
		_channel:       make(chan []byte, 1)}

	return receiver
}

type receiverChannel struct {
	_waitingIdx      int
	_waitingType     string
	_waitingObject   interface{}
	_channel         chan []byte
	_receivedData    []byte
	_receivedCmdBase types.CommandBase
}

func (r *receiverChannel) GetReceivedRawData() (data []byte, cmdBaseObj types.CommandBase) {
	return r._receivedData, r._receivedCmdBase
}

func (r *receiverChannel) IsExpectedResponse(respIdx int) bool {
	if r._waitingIdx != 0 && r._waitingIdx != respIdx {
		return false
	}
	return true
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

		if r._waitingObject != nil && len(r._waitingType) > 0 {
			// if it is not expected response - return error
			if r._receivedCmdBase.Command != r._waitingType {
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

			// deserialize response into expected object type
			if err := deserialize(r._receivedData, r._waitingObject); err != nil {
				return fmt.Errorf("response deserialisation failed: %w", err)
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
