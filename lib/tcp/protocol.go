package tcp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"taylor/lib/structs"
)

type MsgCmd int

const (
	_ MsgCmd		= iota
	MSG_HANDSHAKE_INITIAL
	MSG_HANDSHAKE_RESPONSE

	MSG_NEW_JOB_OFFER
	MSG_JOB_ACCEPTED
	MSG_JOB_DONE
	MSG_JOB_UPDATE
	MSG_JOB_CANCEL_REQUEST
	MSG_JOB_CANCEL_RESPONSE
)

type MsgBase struct {
	Command		MsgCmd	`json:"command"`
	NodeName	string  `json:"node_name"`
}

type MsgAgentInfo struct {
	JobsRunning	uint		`json:"jobs_running"`
	Capacity	uint		`json:"capacity"`
	Capabilities	[]string	`json:"capabilities"`
}

type MsgHandshakeInitial struct {
	MsgBase
	MsgAgentInfo
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
	MsgAgentInfo
	Accepted	bool		`json:"accepted"`
	RefuseReason	string		`json:"refuse_reason"`
	Job		structs.Job	`json:"job"`
}

type MsgJobDone struct {
	MsgBase
	MsgAgentInfo
	Success		bool		`json:"success"`
	Job		structs.Job	`json:"job"`
}

type MsgJobUpdate struct {
	MsgBase
	MsgAgentInfo
	Progress	float32		`json:"progress"`
	Message		string		`json:"message"`
	Job		structs.Job	`json:"job"`
}

type MsgJobCancelRequest struct {
	MsgBase
	Job		structs.Job	`json:"job"`
}

type MsgJobCancelResponse struct {
	MsgBase
	MsgAgentInfo
	Job		structs.Job	`json:"job"`
	Cancelled	bool		`json:"cancelled"`
}

func Encode(message interface{}) (string, error) {
	hsMsg, err := json.Marshal(message)
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(hsMsg) + "\n", nil
}

func Decode(message string) (interface{}, MsgCmd, error) {
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
	case MSG_JOB_DONE:
		var r MsgJobDone
		err = json.Unmarshal(hsJson, &r)
		return r, r.Command, err
	case MSG_JOB_UPDATE:
		var r MsgJobUpdate
		err = json.Unmarshal(hsJson, &r)
		return r, r.Command, err
	case MSG_JOB_CANCEL_REQUEST:
		var r MsgJobCancelRequest
		err = json.Unmarshal(hsJson, &r)
		return r, r.Command, err
	case MSG_JOB_CANCEL_RESPONSE:
		var r MsgJobCancelResponse
		err = json.Unmarshal(hsJson, &r)
		return r, r.Command, err
	default:
		return nil, 0, errors.New(fmt.Sprintf("Invalid command received: %d", base.Command))
	}
}
