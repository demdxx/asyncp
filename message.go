package asyncp

import (
	"encoding/json"
	"errors"

	"github.com/geniusrabbit/notificationcenter"
)

// ErrNullMessageValue if value is nil
var ErrNullMessageValue = errors.New(`the value message is nil`)

// Message this is the internal type of message
type Message = notificationcenter.Message

type message []byte

func messageFrom(value interface{}) (message, error) {
	switch v := value.(type) {
	case nil:
	case []byte:
		return message(v), nil
	case string:
		return message(v), nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return message(data), nil
	}
	return nil, ErrNullMessageValue
}

func mustMessageFrom(value interface{}) message {
	msg, err := messageFrom(value)
	if err != nil {
		panic(err)
	}
	return msg
}

// Unical message ID (depends on transport)
func (m message) ID() string { return `` }

// Body returns message data as bytes
func (m message) Body() []byte { return []byte(m) }

// Acknowledgment of the message processing
func (m message) Ack() error { return nil }
