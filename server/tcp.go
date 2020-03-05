package server

import (
	"fmt"
	"net"
	"os"
	"errors"

	"taylor/server/database"	
	"taylor/server/handlers"	
	"taylor/lib/tcp"	
	"taylor/lib/structs"	
)

type TcpDependencies struct {
	Store	*database.Store
	DiskLog *DiskLog
}

type Node struct {
	conn		*tcp.Conn
	Name		string
	Capabilities	[]string
	Capacity	uint
	JobsRunning	uint
}

func NodeFromMessage(c *tcp.Conn, msg tcp.MsgHandshakeInitial) *Node {
	n := &Node{
		Name: msg.NodeName,
		Capacity: msg.Capacity,		// for now
		Capabilities: msg.Capabilities,
		JobsRunning: msg.JobsRunning,
		conn: c,
	}
	if n.Capabilities == nil {
		n.Capabilities = make([]string, 0)
	}
	return n
}

type NodeMsgPair struct {
	node	*Node
	payload interface{}	
}

type TcpServer struct {
	store		  *database.Store
	diskLog		  *DiskLog
	nodes		  map[string]*Node
	dependencies	  TcpDependencies
	cliChan		  chan NodeMsgPair
	config		  Config
}

func (s *TcpServer) registerNode(n *Node) bool {
	_, in := s.nodes[n.Name]
	if in {
		return false
	}
	fmt.Printf("Register agent %s\n", n.Name)
	s.nodes[n.Name] = n
	return true
}

func (s *TcpServer) deregisterNode(n *Node) {
	_, in := s.nodes[n.Name]
	if in {
		fmt.Printf("Deregister agent %s\n", n.Name)
		jobs, err := s.store.JobsFromNodeWithStatus(n.Name, structs.JOB_STATUS_SCHEDULED)
		if err == nil {
			// put jobs back on queue
			for _, job := range jobs {
				fmt.Printf("Set job %s (%s) as failed\n", job.Id, job.Identifier)
				s.deregisterScheduledJob(job, structs.JOB_STATUS_ERROR)
			}
		}
		
		delete(s.nodes, n.Name)
		n.conn.Close()
	}
}


func (s *TcpServer) handshakeStart(c *tcp.Conn) (*Node, string, error) {
	// establish handshake
	fmt.Println("Wait for handshake message")
	data, _, err := c.ReadMessage()
	if err != nil {
		return nil, "", err
	}

	msg, ok := data.(tcp.MsgHandshakeInitial)
	if ok == false {
		return nil, "Invalid data", nil
	}

	if msg.NodeType != "agent" {
		return nil, "Only agents can join (for now)", nil
	}

	node := NodeFromMessage(c, msg)

	fmt.Printf("New node: %+v\n", node)

	return node, "", nil
}

func (s *TcpServer) handshakeEnd(node *Node, refuseReason string) error {
	var accepted bool
	if refuseReason != "" {
		accepted = false
	} else {
		accepted = true
	}
	return node.conn.WriteMessage(tcp.MsgHandshakeResponse{
		MsgBase: tcp.MsgBase{
			Command: tcp.MSG_HANDSHAKE_RESPONSE,
			NodeName: s.config.Name,
		},
		Accepted: accepted,
		RefuseReason: refuseReason,
	})
}

func (s *TcpServer) handleUpdateHandlers(job *structs.Job, eventName string, progress float32, message string) {
	for _, updateHandler := range job.UpdateHandlers {
		for _, onEvent := range updateHandler.OnEventList {
			if onEvent == eventName {
				var err error
				switch (updateHandler.Type) {
				case "webhook":
					err = handlers.ExecWebhook(
						updateHandler.Config,
						job,
						eventName,
						progress,
						message,
					)
				default:
					break
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error during updateHandler %s\n", updateHandler.Type)
				}
				break
			}
		}
	}
}

func (s *TcpServer) registerScheduledJob(job *structs.Job, nodeName string) error {
	fmt.Printf("Register job: %s at agent: %s\n", job.Id, nodeName)

	if err := s.diskLog.Open(job); err != nil {
		return err 
	}
	if err := s.store.UpdateJobStatus(job.Id, structs.JOB_STATUS_SCHEDULED); err != nil {
		return err
	}
	if err := s.store.UpdateJobAgentName(job.Id, nodeName) ; err != nil {
		return err
	}
	s.handleUpdateHandlers(job, "create", 0, "")
	return nil
}

func (s *TcpServer) deregisterScheduledJob(job *structs.Job, status structs.JobStatus) error {
	fmt.Printf("Deregister job: %s\n", job.Id)

	s.handleUpdateHandlers(job, "done", 1.0, "")

	var err error
	if err = s.diskLog.Close(job); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	if err = s.store.UpdateJobStatus(job.Id, status); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
	return err
}

func (s *TcpServer) handleMsgJobUpdate(response *tcp.MsgJobUpdate) error {
	s.handleUpdateHandlers(&response.Job, "update", response.Progress, response.Message)

	if err := s.store.UpdateJobProgress(response.Job.Id, response.Progress); err != nil {
		return err
	}
	if _, err := s.diskLog.WriteString(&response.Job, response.Message + "\n"); err != nil {
		return err
	}
	return nil
}

func (s *TcpServer) handleMsgJobDone(response *tcp.MsgJobDone) error {
	fmt.Printf("Job %s (%s) success status: %v\n", response.Job.Id, response.Job.Identifier, response.Success)

	if err := s.store.UpdateJobProgress(response.Job.Id, 1.0); err != nil {
		return err
	}
	return s.deregisterScheduledJob(&response.Job, response.Job.Status)
}

func (s *TcpServer) handleMsgJobAccepted(response *tcp.MsgJobAccepted) error {
	if response.Accepted == false {
		return errors.New(fmt.Sprintf("Node %s rejected work. Reason %s", response.NodeName, response.RefuseReason))
	}

	fmt.Printf("Node %s accepted work\n", response.NodeName);

	return s.registerScheduledJob(&response.Job, response.NodeName)
}

func (s *TcpServer) updateNodeFromMessage(msgBase tcp.MsgBase, agentInfo tcp.MsgAgentInfo) error {
	node, in := s.nodes[msgBase.NodeName]
	if in == false {
		return errors.New(fmt.Sprintf("Node %s not available anymore.", msgBase.NodeName))
	}

	node.JobsRunning = agentInfo.JobsRunning
	node.Capacity = agentInfo.Capacity
	return nil
}

func (s *TcpServer) handleConn(c *tcp.Conn) {

	node, refuseReason, err := s.handshakeStart(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		c.Close()
		return
	}

	// there was something wrong in handshake message
	if refuseReason != "" {
		s.handshakeEnd(node, refuseReason)
		node.conn.Close()
		return
	}

	// we can't register node with same name twice
	registered := s.registerNode(node)
	if registered == false {
		s.handshakeEnd(node, fmt.Sprintf("Already node registered with name %s\n", node.Name))
		node.conn.Close()
		return
	}
	// will close connection 
	defer s.deregisterNode(node)
		
	// everything ok
	s.handshakeEnd(node, "")

	fmt.Println("Handshake done for", node.Name)
	for {
		message, cmd, err := node.conn.ReadMessage()
		if err != nil {
			// To-Do: go through all jobs at that agent that are SCHEDULED and put them back into queue
			fmt.Fprintf(os.Stderr, "Client Error: %v\n", err)
			return
		}

		switch (cmd) {
		case tcp.MSG_JOB_ACCEPTED:
			response, _ := message.(tcp.MsgJobAccepted)
			err := s.updateNodeFromMessage(response.MsgBase, response.MsgAgentInfo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			err = s.handleMsgJobAccepted(&response)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		case tcp.MSG_JOB_DONE:
			response, _ := message.(tcp.MsgJobDone)
			err := s.updateNodeFromMessage(response.MsgBase, response.MsgAgentInfo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			err = s.handleMsgJobDone(&response)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		case tcp.MSG_JOB_UPDATE:
			response, _ := message.(tcp.MsgJobUpdate)
			err := s.updateNodeFromMessage(response.MsgBase, response.MsgAgentInfo)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			err = s.handleMsgJobUpdate(&response)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		default:
			fmt.Println("Unknown command received")
		}
	}
}

func (s *TcpServer) Nodes() []*Node {
	nodes := make([]*Node, len(s.nodes))


	i := 0
	for _, v := range s.nodes {
		nodes[i] = v
		i++
	}
	return nodes
}

func (s *TcpServer) listen(ln net.Listener) {
	defer ln.Close()

	go func() {
		for {
			nodeMsgPair := <- s.cliChan

			node := nodeMsgPair.node
			payload := nodeMsgPair.payload

			// check again if node is still connected
			_, in := s.nodes[node.Name]
			if !in {
				// discard message
				continue
			}

			err := node.conn.WriteMessage(payload)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error sending to %s\n", node.Name)
				// notify node
			}
		}
	}()

	for {
		c, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Accept Error: %v\n", err)
			continue
		}

		tcpConn := tcp.NewConn(c)

		go s.handleConn(tcpConn)
	}
}

func (s *TcpServer) CancelJob(job *structs.Job) {
}

func (s *TcpServer) Unicast(node *Node, payload interface{}) {
	s.cliChan <- NodeMsgPair{node, payload}
}

func StartTcp(config Config, deps TcpDependencies) (*TcpServer, error) {
	ln, err := net.Listen("tcp", config.Addresses.Tcp)
	if err != nil {
		return nil, err
	}

	s := &TcpServer{
		nodes:		   make(map[string]*Node),
		store:		   deps.Store,
		cliChan:	   make(chan NodeMsgPair, 50),
		config:		   config,
		diskLog:	   deps.DiskLog,
	}

	go s.listen(ln)
	return s, nil
}
