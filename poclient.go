package poclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client represents the main Pushover Client.
// The Messages channel works in conjunction with ListenForNotifications,
// which pushes incoming Messages into this channel.
type Client struct {
	loggedIn   bool
	registered bool
	user       user
	device     device
	Messages   chan Message
}

// New creates a new POClient with default values
func New() *Client {
	return &Client{
		loggedIn:   false,
		registered: false,
		user:       user{},
		device:     device{},
		Messages:   make(chan Message, 32),
	}
}

// User returns user id and user secret
func (p Client) User() (userID string, secret string) {
	return p.user.ID, p.user.Secret
}

// Device returns the device ID
func (p Client) Device() string {
	return p.device.ID
}

// RestoreLogin sets user ID and secret to access the API from a previous login
func (p *Client) RestoreLogin(secret string, userid string) {
	p.user.Secret = secret
	p.user.ID = userid
	p.loggedIn = true
}

// RestoreDevice sets device ID from a previous device registration
func (p *Client) RestoreDevice(devid string) {
	p.device.ID = devid
	p.registered = true
}

// GetStatus returns the login and registration status
func (p *Client) GetStatus() (bool, bool) {
	return p.loggedIn, p.registered
}

// RegisterDevice registers a new device after logging in.
// The name parameter is a human readable short name (up to 25 characters long) for the device.
// After successfully registering the device you should retrieve the device_id by calling Device()
// and store it in a safe place.
func (p *Client) RegisterDevice(name string) error {
	if !p.loggedIn {
		return errors.New("Not logged in")
	}
	if p.registered {
		return errors.New("Already registered")
	}
	if len(name) > 25 {
		return errors.New("Name is too long")
	}

	resp, err := http.PostForm("https://api.pushover.net/1/devices.json", url.Values{"secret": {p.user.Secret}, "name": {name}, "os": {"O"}})

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	reply := devicesReply{}
	err = json.NewDecoder(resp.Body).Decode(&reply)

	if err != nil {
		return err
	}

	if reply.Status == 0 {
		errorMessage := ""
		for errorType, messages := range reply.Errors {
			for _, errMsg := range messages {
				errorMessage += fmt.Sprintf("%s %s, ", errorType, errMsg)
			}
		}
		return errors.New(errorMessage)
	}

	p.device.ID = reply.Deviceid
	p.registered = true

	return nil
}

// Login retrieves user id and user secret.
// After successfully logging, you should retrieve the user id and secret by calling User() and store
// them somewhere safe.
func (p *Client) Login(email string, password string) error {
	if p.loggedIn {
		return errors.New("Already logged in")
	}

	resp, err := http.PostForm("https://api.pushover.net/1/users/login.json", url.Values{"email": {email}, "password": {password}})

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	reply := loginReply{}
	err = json.NewDecoder(resp.Body).Decode(&reply)

	if err != nil {
		return err
	}

	if reply.Status == 0 {
		return errors.New(reply.Errors[0])
	}

	p.user.Secret = reply.Secret
	p.user.ID = reply.Userid
	p.loggedIn = true
	p.registered = false

	return nil
}

// GetMessages retrieves all new messages from the API.
// Usually you call DeleteOldMessages right afterwards to clear all pending notifications
func (p Client) GetMessages() ([]Message, error) {
	if !p.loggedIn {
		return nil, errors.New("Not logged in")
	}
	if !p.registered {
		return nil, errors.New("Device not registered")
	}

	resp, err := http.Get(fmt.Sprintf("https://api.pushover.net/1/messages.json?secret=%s&device_id=%s", p.user.Secret, p.device.ID))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	reply := messagesReply{}
	err = json.NewDecoder(resp.Body).Decode(&reply)

	if err != nil {
		return nil, err
	}

	if reply.Status != 1 {
		return reply.Messages, errors.New("Getting messages led to a status != 1")
	}

	//parse all timestamps into time.Time
	for i, msg := range reply.Messages {
		reply.Messages[i].Date = time.Unix(msg.Timestamp, 0)
	}

	return reply.Messages, nil
}

// DeleteMessagesByID marks all messages below the given relative ID as
// read which means they will not be transmitted again by the API
// https://pushover.net/api/client#delete
func (p Client) DeleteMessagesByID(highestID int) error {
	resp, err := http.PostForm(
		fmt.Sprintf("https://api.pushover.net/1/devices/%s/update_highest_message.json", p.device.ID),
		url.Values{"secret": {p.user.Secret}, "message": {strconv.Itoa(highestID)}},
	)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	reply := deleteOldMessagesReply{}
	err = json.NewDecoder(resp.Body).Decode(&reply)

	if err != nil {
		return err
	}

	if reply.Status == 0 {
		return errors.New(reply.Errors[0])
	}

	return nil
}

// DeleteOldMessages finds the hightest relative ID and calls DeleteMessagesByID
// This action is permanent, so you need to save the messages if you want to keep them
func (p Client) DeleteOldMessages(messages []Message) error {
	highestID := 0

	for _, msg := range messages {
		if msg.RelativeID > highestID {
			highestID = msg.RelativeID
		}
	}

	return p.DeleteMessagesByID(highestID)
}
