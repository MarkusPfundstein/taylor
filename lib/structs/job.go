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
	// job is being performed at a worker
	JOB_STATUS_RUNNING	  
	// job is done
	JOB_STATUS_STOPPED
)

type Job struct {
	Id		string		`json:"id"`
	Identifier	string		`json:"identifier"`
	Status		JobStatus	`json:"status"`
	Timestamp	int64		`json:"timestamp"`
}

func NewJob(id string) *Job {
	job := Job{
		Id:		uuid.New().String(),
		Identifier:	id,
		Status:		JOB_STATUS_WAITING,
		Timestamp:	time.Now().UnixNano() / 1000000,
	}
	fmt.Printf("make job: %v\n", job)
	return &job
}

