package tcp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"taylor/lib/structs"
)

const (
	_ int			= iota
	MSG_HANDSHAKE_INITIAL
	MSG_HANDSHAKE_RESPONSE

	MSG_NEW_JOB_OFFER
	MSG_JOB_ACCEPTED
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

type MsgNewJobOffer struct {
	MsgBase
	Job		structs.Job	`json:"job"`
}

type MsgJobAccepted struct {
	MsgBase
	Accepted	bool		`json:"accepted"`
	RefuseReason	string		`json:"refuse_reason"`
	Job		structs.Job	`json:"job"`
	JobsRunning	uint		`json:"jobs_running"`
}

func Encode(message interface{}) (string, error) {
	hsMsg, err := json.Marshal(message)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(hsMsg) + "\n", nil
}

func Decode(message string) (interface{}, int, error) {
	hsJson, err := base64.RawStdEncoding.DecodeString(strings.TrimSuffix(message, "\n"))
	if err != nil {
		return nil, 0, err
	}
	var base MsgBase
	err = json.Unmarshal(hsJson, &base)
	if err != nil {
		return nil, 0, err
	}
	switch base.Command {
	case MSG_HANDSHAKE_INITIAL:
		var r MsgHandshakeInitial
		err = json.Unmarshal(hsJson, &r)
		return r, r.Command, err
	case MSG_HANDSHAKE_RESPONSE:
		var r MsgHandshakeResponse
		err = json.Unmarshal(hsJson, &r)
		return r, r.Command, err
	case MSG_NEW_JOB_OFFER:
		var r MsgNewJobOffer
		err = json.Unmarshal(hsJson, &r)
		return r, r.Command, err
	case MSG_JOB_ACCEPTED:
		var r MsgJobAccepted
		err = json.Unmarshal(hsJson, &r)
		return r, r.Command, err
	default:
		return nil, 0, errors.New(fmt.Sprintf("Invalid command received: %d", base.Command))
	}
}
