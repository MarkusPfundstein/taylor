package agent

import (
	"errors"
	"os"
	"fmt"
	"net"
	"os/signal"
	"syscall"
	"sync"

	"taylor/lib/tcp"
	"taylor/lib/structs"
	"taylor/agent/drivers"
)


type Client struct {
	name		string
	capacity	uint
	conn		*tcp.Conn
	jobsRunningMtx  *sync.Mutex
	jobsRunning	map[string]*structs.Job
	drivers		map[string]*structs.Driver
	jobCh		chan *structs.Job
	msgOutCh	chan interface{}
}

func (c *Client) HasCapacity() bool {
	return (c.capacity - uint(len(c.jobsRunning))) > 0
}

func (c *Client) handshake() error {

	// create handshake message
	 err := c.conn.WriteMessage(tcp.MsgHandshakeInitial{
		MsgBase: tcp.MsgBase{
			Command: tcp.MSG_HANDSHAKE_INITIAL,
			NodeName: c.name,
		},
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
	
	fmt.Println("Handhsake done. Connected to cluster", c.conn.Raddr())
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
		MsgBase: tcp.MsgBase{
			Command: tcp.MSG_JOB_ACCEPTED,
			NodeName: c.name,
			JobsRunning: uint(len(c.jobsRunning)),
		},
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
	job.AgentName = c.name

	c.jobsRunning[job.Id] = job

	c.sendJobOfferResponse(job, "")
}

func (c *Client) rejectJobOffer(job *structs.Job, reason string) {
	c.sendJobOfferResponse(job, reason)
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
			pl := <- c.msgOutCh
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
		case tcp.MSG_NEW_JOB_OFFER:
			fmt.Println("Request for work");
			jobOffer, _ := message.(tcp.MsgNewJobOffer)
			fmt.Println(jobOffer.Job)
			if c.HasCapacity() {
				c.acceptJobOffer(&jobOffer.Job)
				c.jobCh <- &jobOffer.Job
			} else {
				c.rejectJobOffer(&jobOffer.Job, "No capacity")
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

func (c *Client) execJob(job *structs.Job) (err error) {
	
	driver, in := c.drivers[job.Driver]
	if in == false {
		return errors.New(fmt.Sprintf("driver %s not registered", job.Driver))
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.New(fmt.Sprintf("Exception while running job: %+v", panicErr))
		}
	}()

	err = driver.Run(job, driver, nil)
	return err
}

func (c *Client) handleJobDone(job *structs.Job, success bool) {
	c.jobsRunningMtx.Lock()
	defer c.jobsRunningMtx.Unlock()

	delete(c.jobsRunning, job.Id)

	if success == true {
		job.Status = structs.JOB_STATUS_SUCCESS
	} else {
		job.Status = structs.JOB_STATUS_ERROR
	}

	c.msgOutCh <- tcp.MsgJobDone{
		MsgBase: tcp.MsgBase{
			Command: tcp.MSG_JOB_DONE,
			NodeName: c.name,
			JobsRunning: uint(len(c.jobsRunning)),
		},
		Success: success,
		Job: *job,
	}
}

func (c *Client) startJobRunner() {
	go func() {
		for {
			fmt.Println("Waiting for job")
			job := <- c.jobCh
			go func(job *structs.Job) {
				fmt.Println("Got a job to do...")
				err := c.execJob(job)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error executing job %s (%s)- %v\n", job.Id, job.Identifier, err)
					c.handleJobDone(job, false)
					return
				}

				c.handleJobDone(job, true)
			}(job)
		}
	}()
}

func initDrivers(config *Config) map[string]*structs.Driver {
	driverMap := make(map[string]*structs.Driver)

	execDriver := drivers.NewExecDriver()
	driverMap[execDriver.Name] = execDriver

	return driverMap
}


func Run(args []string) int {

	fmt.Println("Args", args)
	var configPath string
	if len(args) > 0 {
		configPath = args[0]
	} else {
		configPath = "./client-config.json"
	}

	config, err := ReadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config Error: %v\n", err)
		return 1
	}

	driverMap := initDrivers(config)
	
	client := &Client{
		name:		config.Name,
		capacity:	config.Scheduler.MaxParallelJobs,
		jobsRunningMtx: &sync.Mutex{},
		jobsRunning:	make(map[string]*structs.Job),
		drivers:	driverMap,
		jobCh:		make(chan *structs.Job, config.Scheduler.MaxParallelJobs),
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

	client.startJobRunner()

	err = client.connect(config.ClusterAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Agent err: %+v\n", err)
		return 1
	}
	return 0
}

