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

type poClient struct {
	loggedIn   bool
	registered bool
	user       user
	device     device
	Messages   chan Message
}

func NewPOClient() *poClient {
	return &poClient{
		loggedIn:   false,
		registered: false,
		user:       user{},
		device:     device{},
		Messages:   make(chan Message, 32),
	}
}

//Get user object which contains id and secret
func (p poClient) User() user {
	return p.user
}

//Get device object which contains the device_id
func (p poClient) Device() device {
	return p.device
}

//Restore a previous login
func (p *poClient) RestoreLogin(secret Secret, userid UserID) {
	p.user.Secret = secret
	p.user.Id = userid
	p.loggedIn = true
}

//Set device_id parameter for future requests
func (p *poClient) RestoreDevice(devid DeviceID) {
	p.device.Id = devid
	p.registered = true
}

//Registers a new device after logging in
//The name parameter is a human readable short name (up to 25 characters long) for the device
func (p *poClient) RegisterDevice(name string) error {
	if !p.loggedIn {
		return errors.New("Not logged in")
	}
	if p.registered {
		return errors.New("Already registered")
	}
	if len(name) > 25 {
		return errors.New("Name is too long")
	}

	resp, err := http.PostForm("https://api.pushover.net/1/devices.json", url.Values{"secret": {string(p.user.Secret)}, "name": {name}, "os": {"O"}})

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

	p.device.Id = reply.Deviceid
	p.registered = true

	return nil
}

//Retrieve user id and user secret
//The password should not be saved
func (p *poClient) Login(email string, password string) error {
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
	p.user.Id = reply.Userid
	p.loggedIn = true
	p.registered = false

	return nil
}

//Get all new messages from the API
//Usually you call DeleteOldMessages right afterwards to clear all pending notifications
func (p *poClient) GetMessages() (error, []Message) {
	if !p.loggedIn {
		return errors.New("Not logged in"), []Message{}
	}
	if !p.registered {
		return errors.New("Device not registered"), []Message{}
	}

	resp, err := http.Get("https://api.pushover.net/1/messages.json?" + (url.Values{"secret": {string(p.user.Secret)}, "device_id": {string(p.device.Id)}}).Encode())

	if err != nil {
		return err, []Message{}
	}

	defer resp.Body.Close()
	reply := messagesReply{}
	err = json.NewDecoder(resp.Body).Decode(&reply)

	if err != nil {
		return err, []Message{}
	}

	if reply.Status != 1 {
		return errors.New("Getting messages led to a status != 1"), reply.Messages
	}

	//parse all timestamps into time.Time
	for i, msg := range reply.Messages {
		reply.Messages[i].Date = time.Unix(msg.Timestamp, 0)
	}

	return nil, reply.Messages
}

//Deletes all pending notifications from the server
//This action is permanent, so you need to save the messages if you want to keep them
func (p *poClient) DeleteOldMessages(messages *[]Message) error {
	highest_id := 0

	for _, msg := range *messages {
		if msg.RelativeId > highest_id {
			highest_id = msg.RelativeId
		}
	}

	resp, err := http.PostForm(fmt.Sprintf("https://api.pushover.net/1/devices/%s/update_highest_message.json", p.device.Id), url.Values{"secret": {string(p.user.Secret)}, "message": {strconv.Itoa(highest_id)}})

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
