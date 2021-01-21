package agent

import (
	"errors"
	"os"
	"fmt"
	"net"
	"time"
	"os/signal"
	"syscall"
	"sync"

	"taylor/lib/tcp"
	"taylor/lib/structs"
	"taylor/lib/util"
	"taylor/agent/drivers"
)


type Client struct {
	name		string
	config		Config
	conn		*tcp.Conn
	jobsRunningMtx  *sync.Mutex
	jobsRunning	map[string]*structs.Job
	gpuInfo		[]structs.GpuInfo
	drivers		map[string]*structs.Driver
	newJobCh	chan *structs.Job
	msgOutCh	chan interface{}
}

func (c *Client) HasCapacity() bool {
	return (c.config.Scheduler.MaxParallelJobs - uint(len(c.jobsRunning))) > 0
}

func (c *Client) updateGpuInfo(gpuInfo []structs.GpuInfo) {
	c.gpuInfo = gpuInfo
}

func (c *Client) handshake() error {

	// create handshake message
	 err := c.conn.WriteMessage(tcp.MsgHandshakeInitial{
		 MsgBase: c.GetMsgBase(tcp.MSG_HANDSHAKE_INITIAL),
		 MsgAgentInfo: c.GetMsgAgentInfo(),
		 NodeType: "agent",
	})

	fmt.Println("Wait for handshake response")
	response, _, err := c.conn.ReadMessage()
	if err != nil {
		return err
	}

	fmt.Println(response)

	msg, ok := response.(tcp.MsgHandshakeResponse)
	if ok == false {
		return errors.New("Error casting response")
	}

	if msg.Accepted == false {
		return errors.New(fmt.Sprintf("Server declined join request: %s\n", msg.RefuseReason))
	}

	fmt.Println("Handshake done. Connected to cluster", c.conn.Raddr())
	return nil
}

func (c *Client) sendJobOfferResponse(job *structs.Job, refuseReason string) {
	var accepted bool
	if refuseReason == "" {
		accepted = true
	} else {
		accepted = false
	}
	c.msgOutCh <- tcp.MsgJobAccepted{
		MsgBase: c.GetMsgBase(tcp.MSG_JOB_ACCEPTED),
		MsgAgentInfo: c.GetMsgAgentInfo(),
		Accepted: accepted,
		RefuseReason: refuseReason,
		Job: *job,
	}
}

func (c *Client) acceptJobOffer(job *structs.Job) {
	c.jobsRunningMtx.Lock()
	defer c.jobsRunningMtx.Unlock()
	fmt.Println("Have capacity")

	job.Status = structs.JOB_STATUS_SCHEDULED
	job.AgentName = c.config.Name

	c.jobsRunning[job.Id] = job

	c.sendJobOfferResponse(job, "")
}

func (c *Client) rejectJobOffer(job *structs.Job, reason string) {
	c.sendJobOfferResponse(job, reason)
}

func (c *Client) canAcceptJob(job *structs.Job) (bool, string) {
	if c.HasCapacity() == false {
		return false, "Node has no capacity left"
	}

	// dont have capabilities
	ok := util.IsSubsetString(job.Restrict, c.config.Capabilities)
	if ok == false {
		return false, "Node doesn't have required capabilities"
	}

	return true, ""
}

func (c *Client) connect(clusterAddr string) error {
	tcpConn, err := net.Dial("tcp", clusterAddr)
	if err != nil {
		return err
	}

	c.conn = tcp.NewConn(tcpConn)

	if err = c.handshake(); err != nil {
		return err
	}

	// msgOutCh
	go func() {
		for {
			pl := <-c.msgOutCh
			err := c.conn.WriteMessage(pl)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing %v\n", err)
				// break
			}
		}
	}()

	for {
		message, cmd, err := c.conn.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Disconnected: %v\n", err)
			break
		}

		switch (cmd) {
		case tcp.MSG_AGENT_INFO_REQUEST:
			c.msgOutCh <- tcp.MsgAgentInfoResponse{
				MsgBase: c.GetMsgBase(tcp.MSG_AGENT_INFO_RESPONSE),
				MsgAgentInfo: c.GetMsgAgentInfo(),
			}
		case tcp.MSG_NEW_JOB_OFFER:
			fmt.Println("Received request for work");
			jobOffer, _ := message.(tcp.MsgNewJobOffer)
			fmt.Println(jobOffer.Job)
			if can, rejectReason := c.canAcceptJob(&jobOffer.Job); can == false {
				c.rejectJobOffer(&jobOffer.Job, rejectReason)
			} else {
				c.acceptJobOffer(&jobOffer.Job)
				c.newJobCh <- &jobOffer.Job
			}
		case tcp.MSG_JOB_CANCEL_REQUEST:
			fmt.Println("Received request to cancel job")
			req, _ := message.(tcp.MsgJobCancelRequest)
			fmt.Println(req.Job)

			err := c.cancelJob(&req.Job)
			if err != nil {
				fmt.Printf("Error %v\n", err)
			} else {
				fmt.Println("Job cancelled")
			}
		default:
			fmt.Println("Unknown command received")
		}
	}
	return nil
}

func (c *Client) close() {
	if c.conn != nil {
		fmt.Println("Close", c.conn)
		c.conn.Close()
	}
}

func (c *Client) onJobUpdate(job *structs.Job, progress float32, message string) {
	c.msgOutCh <- tcp.MsgJobUpdate{
		MsgBase: c.GetMsgBase(tcp.MSG_JOB_UPDATE),
		MsgAgentInfo: c.GetMsgAgentInfo(),
		Progress: progress,
		Message:  message,
		Job:	  *job,
	}
}

func (c *Client) cancelJob(job *structs.Job) (err error) {
	driver, in := c.drivers[job.Driver]
	if in == false {
		return errors.New(fmt.Sprintf("driver %s not registered", job.Driver))
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.New(fmt.Sprintf("Exception while cancelling job: %+v", panicErr))
		}
	}()

	err = driver.Cancel(job, driver)
	return err
}

func (c *Client) execJob(job *structs.Job) (interrupted bool, err error) {

	driver, in := c.drivers[job.Driver]
	if in == false {
		return false, errors.New(fmt.Sprintf("driver %s not registered", job.Driver))
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.New(fmt.Sprintf("Exception while running job: %+v", panicErr))
			interrupted = false
		}
	}()

	interrupted, err = driver.Run(job, driver, c.onJobUpdate)
	return interrupted, err
}

func (c *Client) GetMsgBase(cmd tcp.MsgCmd) tcp.MsgBase {
	return tcp.MsgBase{
		Command: cmd,
		NodeName: c.config.Name,
	}
}


func (c *Client) GetMsgAgentInfo() tcp.MsgAgentInfo {
	return tcp.MsgAgentInfo{
		Capacity: c.config.Scheduler.MaxParallelJobs,
		JobsRunning: uint(len(c.jobsRunning)),
		Capabilities: c.config.Capabilities,
		GpuInfo: c.gpuInfo,
	}
}

func (c *Client) handleJobDone(job *structs.Job, interrupted bool, success bool, jobErrorMessage string) {
	c.jobsRunningMtx.Lock()
	defer c.jobsRunningMtx.Unlock()

	delete(c.jobsRunning, job.Id)

	if success == true {
		job.Status = structs.JOB_STATUS_SUCCESS
	} else {
		if interrupted == true {
			job.Status = structs.JOB_STATUS_CANCEL
		} else {
			job.Status = structs.JOB_STATUS_ERROR
		}
	}

	c.msgOutCh <- tcp.MsgJobDone{
		MsgBase: c.GetMsgBase(tcp.MSG_JOB_DONE),
		MsgAgentInfo: c.GetMsgAgentInfo(),
		Success: success,
		Job: *job,
		ErrorMessage: jobErrorMessage,
	}
}

func (c *Client) startJobRunner() {
	go func() {
		for {
			fmt.Println("Waiting for job data...")
			job := <-c.newJobCh
			go func(job *structs.Job) {
				fmt.Println("Received job data...")
				interrupted, err := c.execJob(job)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error executing job %s (%s)- %v\n", job.Id, job.Identifier, err)
					c.handleJobDone(job, interrupted, false, err.Error())
					return
				}

				c.handleJobDone(job, false, true, "")
			}(job)
		}
	}()
}

func initDrivers(config Config) map[string]*structs.Driver {
	driverMap := make(map[string]*structs.Driver)

	execDriver := drivers.NewExecDriver()
	driverMap[execDriver.Name] = execDriver

	return driverMap
}


func Run(args []string, devMode bool) int {

	var config Config
	if devMode == true {
		config = DevModeConfig()
	} else {
		var configPath string
		if len(args) > 0 {
			configPath = args[0]
		} else {
			configPath = "./client-config.json"
		}

		configTmp, err := ReadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Config Error: %v\n", err)
			return 1
		}
		config = configTmp
	}

	driverMap := initDrivers(config)

	client := &Client{
		config:		config,
		jobsRunningMtx: &sync.Mutex{},
		jobsRunning:	make(map[string]*structs.Job),
		drivers:	driverMap,
		newJobCh:	make(chan *structs.Job, config.Scheduler.MaxParallelJobs),
		msgOutCh:	make(chan interface{}, 5),
	}

	defer client.close()

	intC := make(chan os.Signal)
	signal.Notify(intC, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-intC
		client.close()
		os.Exit(1)
	}()

	if config.NvidiaCfg.NvidiaSmiPath != "" {
		startPollGPUDataLoop(config.NvidiaCfg, client.updateGpuInfo)
	}
	client.startJobRunner()

	for {
		err := client.connect(config.ClusterAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Agent Connect err: %+v\n", err)
		}
		time.Sleep(5 * time.Second)
		fmt.Println("Try again to connect to cluster...")
	}
}

