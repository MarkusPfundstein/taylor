package agent

import (
	"errors"
	"os"
	"fmt"
	"net"
	"os/signal"
	"syscall"

	"taylor/lib/tcp"
)


type Client struct {
	name string
	conn *tcp.Conn
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
	response, err := c.conn.ReadMessage()
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

func (c *Client) connect(clusterAddr string) error {
	tcpConn, err := net.Dial("tcp", clusterAddr)
	if err != nil {
		return err
	}

	c.conn = tcp.NewConn(tcpConn)

	err = c.handshake()
	if err != nil {
		return err
	}

	for {
		message, err := c.conn.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Disconnected: %v\n", err)
			break
		}
		fmt.Printf("Rec: %s", message)
	}
	return nil
}

func (c *Client) Close() {
	if c.conn != nil {
		fmt.Println("Close", c.conn)
		c.conn.Close()
	}
}

func Run() int {
	config, err := ReadConfig("./client-config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config Error: %v\n", err)
		return 1
	}
	
	client := &Client{
		name: config.Name,
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		client.Close()
		os.Exit(1)
	}()

	err = client.connect(config.ClusterAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Agent err: %+v\n", err)
		return 1
	}
	return 0
}

