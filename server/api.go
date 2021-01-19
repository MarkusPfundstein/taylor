package server

import (
	"net/http"
	"fmt"
	"strconv"
	"os"
	"errors"

	"taylor/server/database"
	"taylor/lib/structs"
	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error		string	`json:"error"`
}

type PatchDefinition struct {
	Op		string	`json:"op"`
	Path		string  `json:"path"`
	Value		string  `json:"value"`
}

type JobDefinition struct {
	Identifier	string			`json:"identifier"`
	Driver		string			`json:"driver"`
	DriverConfig	map[string]interface{}	`json:"driver_config"`
	UpdateHandlers	[]structs.UpdateHandler	`json:"update_handlers"`
	Restrict	[]string		`json:"restrict"`
	Priority	int			`json:"priority"`
}

type ApiDependencies struct {
	Store	  *database.Store
	TcpServer *TcpServer
	DiskLog	  *DiskLog
}

func sendError(c *gin.Context, code int, err error) {
	c.JSON(code, ErrorResponse{
		Error: err.Error(),
	})
}

func postJob(deps ApiDependencies, c *gin.Context) {
	var jobDef JobDefinition
	if err := c.ShouldBindJSON(&jobDef); err != nil {
		sendError(c, http.StatusBadRequest, errors.New("Couldn't parse Job Definition"))
		return
	}

	if jobDef.Identifier == "" || jobDef.Driver == "" || len(jobDef.DriverConfig) == 0 {
		sendError(c, http.StatusBadRequest, errors.New("Invalid Job Definition. Identifier, Driver and DirverConfig must not be empty"))
		return
	}

	if jobDef.Restrict == nil {
		jobDef.Restrict = make([]string, 0)
	}

	if jobDef.Priority < 0 {
		jobDef.Priority = 0
	}
	if jobDef.Priority > 100 {
		jobDef.Priority = 100
	}
	if jobDef.Priority == 0 {
		jobDef.Priority = 10
	}

	job := structs.NewJob(
		jobDef.Identifier,
		jobDef.Driver,
		jobDef.DriverConfig,
		jobDef.UpdateHandlers,
		jobDef.Restrict,
		uint(jobDef.Priority),
	)

	fmt.Printf("%+v", job)

	_, err := deps.Store.InsertJob(job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Internal Error: %v\n", err)
		sendError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, job)
}

func getAllJobs(deps ApiDependencies, c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "0"))
	if err != nil {
		sendError(c, http.StatusBadRequest, errors.New("limit must be integer"))
		c.Status(http.StatusBadRequest)
		return
	}
	if limit < 0 {
		sendError(c, http.StatusBadRequest, errors.New("limit must be > 0"))
		return
	}

	jobs, err := deps.Store.AllJobs(uint(limit))
	if err != nil {
		sendError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, jobs)
}

func getAllNodes(deps ApiDependencies, c *gin.Context) {
	nodes := deps.TcpServer.Nodes()

	c.JSON(http.StatusOK, nodes)
}

func getJobLog(deps ApiDependencies, c *gin.Context) {
	job, err := deps.Store.JobById(c.Param("JobId"))
	if err != nil {
		sendError(c, http.StatusInternalServerError, err)
		return
	}

	if job == nil {
		sendError(c, http.StatusNotFound, errors.New("Couldn't find job"))
		return
	}

	// open log
	logs, _:= deps.DiskLog.GetLogs(job)

	resp := make(map[string]interface{}, 0)
	resp["jobId"] = job.Id
	resp["logs"] = logs

	c.JSON(http.StatusOK, resp)
}

func getJob(deps ApiDependencies, c *gin.Context) {
	job, err := deps.Store.JobById(c.Param("JobId"))
	if err != nil {
		sendError(c, http.StatusInternalServerError, err)
		return
	}

	if job == nil {
		sendError(c, http.StatusNotFound, errors.New("Couldn't find job"))
		return
	}

	c.JSON(http.StatusOK, job)
}

func updateJobStatus(tcpServer *TcpServer, job *structs.Job, value string) (int, error) {
	switch (value) {
	case "4", "cancel":
		if job.CanCancel() {
			err := tcpServer.CancelJob(job);
			if err != nil {
				return http.StatusConflict, err
			}
			return http.StatusAccepted, nil
		} else {
			return http.StatusConflict, errors.New("Job can't be cancelled. It's not scheduled at a node or waiting")
		}
	default:
		return http.StatusBadRequest, errors.New("invalid value (4 or cancel)")
	}
}

func updateJob(tcpServer *TcpServer, job *structs.Job, path string, value string) (int, error) {
	switch (path) {
	case "/status", "status":
		return updateJobStatus(tcpServer, job, value)
	default:
		return http.StatusBadRequest, errors.New("invalid path")
	}
}

func patchJob(deps ApiDependencies, c *gin.Context) {
	var patchDef PatchDefinition
	if err := c.ShouldBindJSON(&patchDef); err != nil {
		sendError(c, http.StatusBadRequest, errors.New("Couldn't parse Patch Definition"))
		return
	}

	if patchDef.Op == "" || patchDef.Path == "" || patchDef.Value == "" {
		sendError(c, http.StatusBadRequest, errors.New("Invalid Patch Definition. Op, Path and Value field required"))
		return
	}

	job, err := deps.Store.JobById(c.Param("JobId"))
	if err != nil {
		sendError(c, http.StatusInternalServerError, err)
		return
	}

	if job == nil {
		sendError(c, http.StatusNotFound, errors.New("Couldn't find job"))
		return
	}

	// handle Operations
	switch patchDef.Op {
	case "update":
		code, err := updateJob(deps.TcpServer, job, patchDef.Path, patchDef.Value)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			sendError(c, code, err)
			return
		}

		c.Status(code)
		return
	default:
		sendError(c, http.StatusNotImplemented, errors.New("op not implemented yet"))
		return
	}
}

func deleteJob(deps ApiDependencies, c *gin.Context) {
	job, err := deps.Store.JobById(c.Param("JobId"))
	if err != nil {
		sendError(c, http.StatusInternalServerError, err)
		return
	}

	if job == nil {
		sendError(c, http.StatusNotFound, errors.New("Couldn't find job"))
		return
	}

	if job.CanDelete() == false {
		sendError(c, http.StatusConflict, errors.New("Can't delete job that is scheduled or deleted"))
		return
	}

	err = deps.Store.UpdateJobStatus(job.Id, structs.JOB_STATUS_DELETE)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err)
		return
	}

	c.Status(http.StatusOK)
}

func StartApi(config Config, deps ApiDependencies) error {

	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	v1 := router.Group("/v1")
	{
		v1.POST("/jobs", func (c *gin.Context) {
			postJob(deps, c)
		})
		v1.GET("/jobs", func (c *gin.Context) {
			getAllJobs(deps, c)
		})
		v1.GET("/jobs/:JobId", func (c *gin.Context) {
			getJob(deps, c)
		})
		v1.PATCH("/jobs/:JobId", func (c *gin.Context) {
			patchJob(deps, c)
		})
		v1.DELETE("/jobs/:JobId", func (c *gin.Context) {
			deleteJob(deps, c)
		})
		v1.GET("/jobs/:JobId/log", func (c *gin.Context) {
			getJobLog(deps, c)
		})
		v1.GET("/nodes", func (c *gin.Context) {
			getAllNodes(deps, c)
		})
	}

	return router.Run(config.Addresses.Http)
}
