package poclient

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/gorilla/websocket"
)

//Connects to the WebSocket endpoint and waits for incoming notifications
//This function is designed to run in a goroutine
//Note: This function clears all notifications after receiving them, so you should pull messages
//from the Messages channel and save them if you want to keep them
func (p *POClient) ListenForNotifications() error {
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

	conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("login:%s:%s\n", p.device.Id, p.user.Secret)))

	reconnect := false

	for {
		msgType, msgBytes, err := conn.ReadMessage()

		if msgType != websocket.BinaryMessage { //ignore everything else
			continue
		}

		if err != nil {
			return err
		}

		message := string(msgBytes)

		switch message {
		case "#": //do nothing
		case "!":
			err, messages := p.GetMessages()
			if err != nil {
				return err
			}
			for _, msg := range messages {
				p.Messages <- msg //send messages into message channel
			}
			p.DeleteOldMessages(&messages)
		case "R":
			reconnect = true
		case "E":
			return errors.New("Received error frame")
		}

		if reconnect {
			break
		}
	}

	return p.ListenForNotifications() //reconnect
}
