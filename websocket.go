package poclient

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/gorilla/websocket"
)

// ListenForNotifications connects to the WebSocket endpoint and waits for incoming notifications
// This function is designed to run in a goroutine
// Note: This function clears all notifications after receiving them, so you should pull messages
// from the Messages channel and save them if you want to keep them
func (p Client) ListenForNotifications() error {
	if !p.loggedIn {
		return errors.New("Not logged in")
	}
	if !p.registered {
		return errors.New("Device not registered")
	}

	u := url.URL{Scheme: "wss", Host: "client.pushover.net", Path: "/push"}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("login:%s:%s\n", p.device.ID, p.user.Secret))); err != nil {
		return err
	}

	reconnect := false

	for !reconnect {
		msgType, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		if msgType != websocket.BinaryMessage { //ignore everything else
			continue
		}

		if reconnect, err = p.handleNotification(string(msgBytes)); err != nil {
			return err
		}
	}

	return p.ListenForNotifications() //reconnect
}

func (p Client) handleNotification(message string) (reconnect bool, err error) {
	switch message {
	// Keep-alive packet, no response needed.
	case "#":
		return false, nil

	// A new message has arrived; you should perform a sync.
	case "!":
		messages, err := p.GetMessages()
		if err != nil {
			return false, err
		}

		for _, msg := range messages {
			p.Messages <- msg //send messages into message channel
		}

		return false, p.DeleteOldMessages(messages)

	// Reload request; you should drop your connection and re-connect.
	case "R":
		return true, nil

	// Error; a permanent problem occurred and you should not automatically re-connect. Prompt the user to login again or re-enable the device.
	case "E":
		return false, errors.New("Received error frame")

	default:
		return false, errors.New("Unexpected message received from API")
	}
}
