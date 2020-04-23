package protocol

import (
	"bufio"
	"fmt"
	"time"

	"github.com/ivpn/desktop-app-daemon/logger"
	"github.com/ivpn/desktop-app-daemon/protocol/types"
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

func (c *Client) sendRecvRaw(request interface{}) (data []byte, cmdBase types.CommandBase, err error) {
	var receiver *receiverChannel

	// thread-safe receiver registration
	func() {
		c._receiversLocker.Lock()
		defer c._receiversLocker.Unlock()

		c._requestIdx++
		receiver = createReceiver(c._requestIdx, nil)

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
				if receiver.IsExpectedResponse(cmd.Idx) {
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
