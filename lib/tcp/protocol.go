package tcp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	_ int			= iota
	MSG_HANDSHAKE_INITIAL
	MSG_HANDSHAKE_RESPONSE
)

type MsgBase struct {
	Command		int	`json:"command"`
	NodeName	string  `json:"node_name"`
}

type MsgHandshakeInitial struct {
	MsgBase
	NodeType	string	`json:"node_type"`
}

type MsgHandshakeResponse struct {
	MsgBase
	Accepted	bool	`json:"accepted"`
	RefuseReason	string	`json:"refuse_reason"`
}

func Encode(message interface{}) (string, error) {
	hsMsg, err := json.Marshal(message)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(hsMsg) + "\n", nil
}

func Decode(message string) (interface{}, error) {
	hsJson, err := base64.RawStdEncoding.DecodeString(strings.TrimSuffix(message, "\n"))
	if err != nil {
		return nil, err
	}
	var base MsgBase
	err = json.Unmarshal(hsJson, &base)
	if err != nil {
		return nil, err
	}
	fmt.Println("decoded", base)
	switch base.Command {
	case MSG_HANDSHAKE_INITIAL:
		var r MsgHandshakeInitial
		err = json.Unmarshal(hsJson, &r)
		return r, err
	case MSG_HANDSHAKE_RESPONSE:
		var r MsgHandshakeResponse
		err = json.Unmarshal(hsJson, &r)
		return r, err
	default:
		return nil, errors.New(fmt.Sprintf("Invalid command received: %s", base.Command))
	}

	return base, nil
}
