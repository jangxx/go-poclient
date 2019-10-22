package poclient

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// ListenForNotifications connects to the WebSocket endpoint and waits for incoming notifications
// This function is designed to run in a goroutine
// If no keep-alive packet is received for one minute, the function exits with a timeout error (net.Error).
// Note: This function clears all notifications after receiving them, so you should pull messages
// from the Messages channel and save them if you want to keep them
func (p *Client) ListenForNotifications() error {
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

	p.wsConn = conn // store reference to connection

	if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("login:%s:%s\n", p.device.ID, p.user.Secret))); err != nil {
		return err
	}

	reconnect := false

	for !reconnect {
		// time out after no keep-alive has been received for one minute
		conn.SetReadDeadline(time.Now().Add(1 * time.Minute))

		msgType, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		if msgType != websocket.BinaryMessage { // ignore everything else
			continue
		}

		if reconnect, err = p.handleNotification(string(msgBytes)); err != nil {
			return err
		}
	}

	return p.ListenForNotifications() // reconnect
}

// CloseWebsocket forcefully closes a open websocket connection, if one exists
// This also causes a running ListenForNotifications to return an error,
// which you can use to reconnect
func (p *Client) CloseWebsocket() {
	if p.wsConn != nil {
		p.wsConn.Close()
		p.wsConn = nil
	}
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
		return false, &ErrorFrameError{}

	default:
		return false, errors.New("Unexpected message received from API")
	}
}
