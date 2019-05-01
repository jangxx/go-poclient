package poclient

import (
	"fmt"
	"time"
)

type loginReply struct {
	Status int      `json:"status"`
	Userid string   `json:"id"`
	Secret string   `json:"secret"`
	Errors []string `json:"errors"`
}

type devicesReply struct {
	Status   int                 `json:"status"`
	Deviceid string              `json:"id"`
	Errors   map[string][]string `json:"errors"`
}

type messagesReply struct {
	Status   int       `json:"status"`
	Errors   []string  `json:"errors"`
	Messages []Message `json:"messages"`
}

type deleteOldMessagesReply struct {
	Status int      `json:"status"`
	Errors []string `json:"errors"`
}

type user struct {
	ID     string
	Secret string
}

type device struct {
	ID string
}

// Message represents a message loaded from the API
// The fields are documented here: https://pushover.net/api/client#download
type Message struct {
	RelativeID   int                `json:"id"`
	UniqueID     int                `json:"umid"`
	Title        string             `json:"title"`
	Text         string             `json:"message"`
	AppName      string             `json:"app"`
	AppID        int                `json:"aid"`
	IconID       string             `json:"icon"`
	Timestamp    int64              `json:"date"`
	Date         time.Time          `json:"-"`
	Priority     int                `json:"priority"`
	Sound        string             `json:"sound"`
	URL          string             `json:"url"`
	URLTitle     string             `json:"url_title"`
	Acknowledged convertibleBoolean `json:"acked"`
	ReceiptCode  string             `json:"receipt"`
	ContainsHTML convertibleBoolean `json:"html"`
}

// Taken from https://stackoverflow.com/questions/30856454/how-to-unmarshall-both-0-and-false-as-bool-from-json
type convertibleBoolean bool

func (bit *convertibleBoolean) UnmarshalJSON(data []byte) error {
	asString := string(data)
	if asString == "1" || asString == "true" {
		*bit = true
	} else if asString == "0" || asString == "false" {
		*bit = false
	} else {
		return fmt.Errorf("Boolean unmarshal error: invalid input %s", asString)
	}
	return nil
}

type ErrorFrameError struct{}

func (e *ErrorFrameError) Error() string {
	return "Received error frame"
}

type Missing2FAError struct{}

func (e *Missing2FAError) Error() string {
	return "2 Factor Authentication required"
}
