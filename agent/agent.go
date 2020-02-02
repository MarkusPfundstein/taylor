package agent

import (
	"errors"
	"os"
	"fmt"
	"net"
	"os/signal"
	"syscall"

	"taylor/lib/tcp"
	"taylor/lib/structs"
)


type Client struct {
	name		string
	capacity	uint
	conn		*tcp.Conn
	jobsRunning	[]*structs.Job
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

func (c *Client) sendJobOfferResponse(job *structs.Job, refuseReason string) error {
	var accepted bool
	if refuseReason == "" {
		accepted = true
	} else {
		accepted = false
	}
	return c.conn.WriteMessage(tcp.MsgJobAccepted{
		MsgBase: tcp.MsgBase{
			Command: tcp.MSG_JOB_ACCEPTED,
			NodeName: c.name,
		},
		Accepted: accepted,
		RefuseReason: refuseReason,
		Job: *job,
		JobsRunning: uint(len(c.jobsRunning)),
	})
}

func (c *Client) acceptJobOffer(job *structs.Job) error {
	fmt.Println("Have capacity")
	job.Status = structs.JOB_STATUS_RUNNING
	c.jobsRunning = append(c.jobsRunning, job)

	return c.sendJobOfferResponse(job, "")
}

func (c *Client) rejectJobOffer(job *structs.Job, reason string) error {
	return c.sendJobOfferResponse(job, reason)
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
	
	client := &Client{
		name:		config.Name,
		capacity:	config.Scheduler.MaxParallelJobs,
		jobsRunning:	make([]*structs.Job, 0),
	}

	defer client.close()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		client.close()
		os.Exit(1)
	}()

	err = client.connect(config.ClusterAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Agent err: %+v\n", err)
		return 1
	}
	return 0
}

