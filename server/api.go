package server

import (
	"net/http"
	"fmt"
	"strconv"
	"os"

	"taylor/server/database"
	"taylor/lib/structs"
	"github.com/gin-gonic/gin"
)

type JobDefinition struct {
	Identifier	string					`json:"identifier"`
	Driver		string					`json:"driver"`
	DriverConfig	map[string]interface{}			`json:"driver_config"`
	UpdateHandlers	[]structs.UpdateHandler		`json:"update_handlers"`
}

type ApiDependencies struct {
	Store	  *database.Store
	TcpServer *TcpServer
	DiskLog	  *DiskLog
}

func postJob(deps ApiDependencies, c *gin.Context) {
	var jobDef JobDefinition
	if err := c.ShouldBindJSON(&jobDef); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if jobDef.Identifier == "" || jobDef.Driver == "" || len(jobDef.DriverConfig) == 0 {
		c.Status(http.StatusBadRequest)
		return
	}

	job := structs.NewJob(
		jobDef.Identifier,
		jobDef.Driver,
		jobDef.DriverConfig,
		jobDef.UpdateHandlers,
	)

	fmt.Printf("%+v", job)

	_, err := deps.Store.InsertJob(job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Internal Error: %v\n", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusCreated)
}

func getAllJobs(deps ApiDependencies, c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "0"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if limit < 0 {
		c.Status(http.StatusBadRequest)
		return
	}

	jobs, err := deps.Store.AllJobs(uint(limit))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Internal Error: %v\n", err)
		c.Status(http.StatusInternalServerError)
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
		fmt.Fprintf(os.Stderr, "Internal Error: %v\n", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	if job == nil {
		c.Status(http.StatusNotFound)
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
		fmt.Fprintf(os.Stderr, "Internal Error: %v\n", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	if job == nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, job)
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
		v1.GET("/jobs/:JobId/log", func (c *gin.Context) {
			getJobLog(deps, c)
		})
		v1.GET("/nodes", func (c *gin.Context) {
			getAllNodes(deps, c)
		})
	}

	return router.Run(config.Addresses.Http)
}
