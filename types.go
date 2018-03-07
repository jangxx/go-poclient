package poclient

import (
	"errors"
	"fmt"
	"time"
)

type loginReply struct {
	Status int      `json:"status"`
	Userid UserID   `json:"id"`
	Secret Secret   `json:"secret"`
	Errors []string `json:"errors"`
}

type devicesReply struct {
	Status   int                 `json:"status"`
	Deviceid DeviceID            `json:"id"`
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
	Id     UserID
	Secret Secret
}

type device struct {
	Id DeviceID
}

type Secret string

type UserID string

type DeviceID string

type Message struct {
	RelativeId   int                `json:"id"`
	UniqueId     int                `json:"umid"`
	Title        string             `json:"title"`
	Text         string             `json:"message"`
	AppName      string             `json:"app"`
	AppId        int                `json:"aid"`
	IconId       string             `json:"icon"`
	Timestamp    int64              `json:"date"`
	Date         time.Time          `json:"-"`
	Priority     int                `json:"priority"`
	Sound        string             `json:"sound"`
	Url          string             `json:"url"`
	UrlTitle     string             `json:"url_title"`
	Acknowledged ConvertibleBoolean `json:"acked"`
	ReceiptCode  string             `json:"receipt"`
	ContainsHTML ConvertibleBoolean `json:"html"`
}

//https://stackoverflow.com/questions/30856454/how-to-unmarshall-both-0-and-false-as-bool-from-json
type ConvertibleBoolean bool

func (bit *ConvertibleBoolean) UnmarshalJSON(data []byte) error {
	asString := string(data)
	if asString == "1" || asString == "true" {
		*bit = true
	} else if asString == "0" || asString == "false" {
		*bit = false
	} else {
		return errors.New(fmt.Sprintf("Boolean unmarshal error: invalid input %s", asString))
	}
	return nil
}
