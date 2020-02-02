package server

import (
	"fmt"
	"net"
	"os"

	"taylor/lib/tcp"	
)

type TcpDependencies struct {}

type AddrMsgPair struct {
	addr	string
	msg	string
}

type Node struct {
	conn	*tcp.Conn
	Name	string
}

type TcpServer struct {
	Nodes		  map[string]*Node
	dependencies	  TcpDependencies
	cliChan		  chan AddrMsgPair
	config		  Config
}

func (s *TcpServer) registerNode(n *Node) bool {
	_, in := s.Nodes[n.Name]
	if in {
		return false
	}
	fmt.Printf("Register %s\n", n.Name)
	s.Nodes[n.Name] = n
	return true
}

func (s *TcpServer) deregisterNode(n *Node) {
	_, in := s.Nodes[n.Name]
	if in {
		fmt.Printf("Deregister %s\n", n.Name)
		delete(s.Nodes, n.Name)
		n.conn.Close()
	}
}

func (s *TcpServer) handshakeStart(c *tcp.Conn) (string, string, error) {
	// establish handshake
	fmt.Println("Wait for handshake message")
	data, err := c.ReadMessage()
	if err != nil {
		return "", "", err
	}

	msg, ok := data.(tcp.MsgHandshakeInitial)
	if ok == false {
		return "Invalid data", "", nil
	}

	if msg.NodeType != "agent" {
		return "Only agents can join (for now)", "", nil
	}

	return "", msg.NodeName, nil
}

func (s *TcpServer) handshakeEnd(c *tcp.Conn, refuseReason string) error {
	var accepted bool
	if refuseReason != "" {
		accepted = false
	} else {
		accepted = true
	}
	return c.WriteMessage(tcp.MsgHandshakeResponse{
		MsgBase: tcp.MsgBase{
			Command: tcp.MSG_HANDSHAKE_RESPONSE,
			NodeName: s.config.Name,
		},
		Accepted: accepted,
		RefuseReason: refuseReason,
	})
}

func (s *TcpServer) handleConn(c *tcp.Conn) {

	refuseReason, nodeName, err := s.handshakeStart(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		c.Close()
		return
	}

	// there was something wrong in handshake message
	if refuseReason != "" {
		s.handshakeEnd(c, refuseReason)
		c.Close()
		return
	}

	node := Node{
		Name: nodeName,
		conn: c,
	}

	registered := s.registerNode(&node)
	if registered == false {
		s.handshakeEnd(c, fmt.Sprintf("Already node registered with name %s\n", nodeName))
		c.Close()
		return
	}
	// will close connection 
	defer s.deregisterNode(&node)
		
	// everything ok
	s.handshakeEnd(c, "")

	fmt.Println("Handshake done for", c)
	for {
		message, err := c.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Client Error: %v\n", err)
			return
		}
		fmt.Printf("Rec: %v\n", message)
	}
}

func (s *TcpServer) ConnectedClients() []string {
	keys := make([]string, len(s.Nodes))

	i := 0
	for k := range s.Nodes {
		keys[i] = k
		i++
	}
	return keys
}

func (s *TcpServer) listen(ln net.Listener) {
	defer ln.Close()

	go func() {
		for {
			addrMsgPair := <- s.cliChan
			fmt.Printf("handle out msg %+v\n", addrMsgPair)

			addr := addrMsgPair.addr
			msg := addrMsgPair.msg

			_, in := s.Nodes[addr]
			if in {
				fmt.Printf("Send msg %s to %s\n", msg, addr)
				//c.WriteString(msg)
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

func (s *TcpServer) Broadcast(clients []string, message string) {
	for _, v := range clients {
		s.cliChan <- AddrMsgPair{v, message}
	}
}

func StartTcp(config Config, deps TcpDependencies) (*TcpServer, error) {
	ln, err := net.Listen("tcp", config.Addresses.Tcp)
	if err != nil {
		return nil, err
	}

	s := &TcpServer{
		Nodes:		   make(map[string]*Node),
		dependencies:	   deps,
		cliChan:	   make(chan AddrMsgPair, 50),
		config:		   config,
	}

	go s.listen(ln)
	return s, nil
}
