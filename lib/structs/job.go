package structs

import (
	"github.com/google/uuid"
	"time"
	"fmt"
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
)

type Job struct {
	Id		string			`json:"id"`
	Identifier	string			`json:"identifier"`
	Status		JobStatus		`json:"status"`
	Timestamp	int64			`json:"timestamp"`
	AgentName	string			`json:"agent_name"`
	Driver		string			`json:"driver"`
	DriverConfig	map[string]interface{}	`json:"driver_config"`
}

func NewJob(id string, driver string, driverConfig map[string]interface{}) *Job {
	job := Job{
		Id:		uuid.New().String(),
		Identifier:	id,
		Status:		JOB_STATUS_WAITING,
		Timestamp:	time.Now().UnixNano() / 1000000,
		AgentName:	"",
		Driver:		driver,
		DriverConfig:	driverConfig,
	}
	fmt.Printf("make job: %v\n", job)
	return &job
}

