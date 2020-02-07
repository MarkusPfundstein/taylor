package structs

import (
	"github.com/google/uuid"
	"time"
)

type JobStatus int

const (
	// job is waiting to be scheduled at a worker
	JOB_STATUS_WAITING JobStatus = iota

	// job is scheduled at a worker
	JOB_STATUS_SCHEDULED 

	// job is done
	JOB_STATUS_SUCCESS

	// job is error
	JOB_STATUS_ERROR

	// job has been cancelled
	JOB_STATUS_CANCEL

	// job has been interrupted (during execution)
	JOB_STATUS_INTERRUPT

	// job has been deleted
	JOB_STATUS_DELETE
)

type UpdateHandler struct {
	Type		string			`json:"type"`
	OnEventList	[]string		`json:"on"`
	Config		map[string]interface{}	`json:"config"`
}

type Job struct {
	Id		string			`json:"id"`
	Identifier	string			`json:"identifier"`
	Status		JobStatus		`json:"status"`
	Timestamp	int64			`json:"timestamp"`
	AgentName	string			`json:"agent_name"`
	Driver		string			`json:"driver"`
	DriverConfig	map[string]interface{}	`json:"driver_config"`
	UpdateHandlers	[]UpdateHandler		`json:"update_handlers"`
	Restrict	[]string		`json:"restrict"`
	Priority	uint			`json:"priority"`
	Progress	float32			`json:"progress"`
}

func NewJob(
	id string,
	driver string,
	driverConfig map[string]interface{},
	updateHandlers []UpdateHandler,
	restrict []string,
	priority uint,
) *Job {
	// to-do: validate updatehandlers
	return &Job{
		Id:		uuid.New().String(),
		Identifier:	id,
		Status:		JOB_STATUS_WAITING,
		Timestamp:	time.Now().UnixNano() / 1000000,
		AgentName:	"",
		Driver:		driver,
		DriverConfig:	driverConfig,
		UpdateHandlers: updateHandlers,
		Restrict:	restrict,
		Progress:	0.0,
		Priority:	priority,
	}
}
