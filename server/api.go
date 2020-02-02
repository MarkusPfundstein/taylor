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
	Identifier string `json:"identifier" binding:"required"`
}

type ApiDependencies struct {
	Store *database.Store
	TcpServer *TcpServer
}

func postJob(deps *ApiDependencies, c *gin.Context) {
	var jobDef JobDefinition
	if err := c.ShouldBindJSON(&jobDef); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	job := structs.NewJob(jobDef.Identifier)

	fmt.Println(job)

	_, err := deps.Store.InsertJob(job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Internal Error: %v\n", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	deps.TcpServer.Broadcast([]string{"client1", "client2"}, "Hello Client")

	c.Status(http.StatusCreated)
}

func getAllJobs(deps *ApiDependencies, c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "0"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	jobs, err := deps.Store.AllJobs(limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Internal Error: %v\n", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, jobs)
}

func getAllNodes(deps *ApiDependencies, c *gin.Context) {
	nodes := deps.TcpServer.ConnectedClients()

	c.JSON(http.StatusOK, nodes)
}

func getJob(deps *ApiDependencies, c *gin.Context) {
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

func StartApi(config *Config, deps *ApiDependencies) error {

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
		v1.GET("/nodes", func (c *gin.Context) {
			getAllNodes(deps, c)
		})
	}

	return router.Run(config.Addresses.Http)
}
