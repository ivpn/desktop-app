package protocol

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
)

func createReceiver(waitingIdx int, waitingObject interface{}) *receiverChannel {
	receiver := createReceiverAny(waitingIdx, waitingObject)
	receiver._waitingAny = false // accept only response with correspond Idx
	return receiver
}

// receiver waits for any response from a waitingObjects list (waitingIdx ignored when received any of expected types)
func createReceiverAny(waitingIdx int, waitingObjects ...interface{}) *receiverChannel {
	receiver := &receiverChannel{
		_waitingIdx:     waitingIdx,
		_waitingObjects: make(map[string]interface{}),
		_channel:        make(chan []byte, 1)}

	for _, wo := range waitingObjects {
		if wo == nil {
			continue
		}
		receiver._waitingObjects[types.GetTypeName(wo)] = wo
	}

	// acceptable any response with correspond 'waitingIdx' or 'waitingType'
	receiver._waitingAny = true

	return receiver
}

type receiverChannel struct {
	_channel chan []byte

	_waitingIdx     int
	_waitingObjects map[string]interface{}
	_waitingAny     bool // when true - receiver waits for any response from a waitingObjects list (waitingIdx ignored when received any of expected types)

	_receivedData    []byte
	_receivedCmdBase types.CommandBase
}

func (r *receiverChannel) GetReceivedRawData() (data []byte, cmdBaseObj types.CommandBase) {
	return r._receivedData, r._receivedCmdBase
}

func (r *receiverChannel) IsExpectedResponse(respIdx int, commandName string) bool {
	if r._waitingAny {
		if _, exist := r._waitingObjects[commandName]; exist {
			return true
		}
	}
	if r._waitingIdx == respIdx {
		return true
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
	case r._receivedData = <-r._channel: // save response to internal variable
		// check type of response
		if err := deserialize(r._receivedData, &r._receivedCmdBase); err != nil {
			return fmt.Errorf("response deserialisation failed: %w", err)
		}

		// deserialize (if response is expected)
		if len(r._waitingObjects) > 0 {
			if wo, exists := r._waitingObjects[r._receivedCmdBase.Command]; exists && wo != nil {
				// check is it Error object
				var errObj types.ErrorResp
				if r._receivedCmdBase.Command == types.GetTypeName(errObj) {
					if err := deserialize(r._receivedData, &errObj); err != nil {
						return fmt.Errorf("response deserialisation failed: %w", err)
					}
					return fmt.Errorf(errObj.ErrorMessage)
				}
				// deserialize response into expected object type
				if err := deserialize(r._receivedData, wo); err != nil {
					return fmt.Errorf("response deserialisation failed: %w", err)
				}
			} else {
				// if it is not expected response - return error
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
