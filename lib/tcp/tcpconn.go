package tcp 

import (
	"bufio"
	"net"
)

type Conn struct {
	reader	*bufio.Reader
	writer	*bufio.Writer
	conn	net.Conn
}

func (t *Conn) Close() {
	t.conn.Close()
}

func (t *Conn) Conn() (net.Conn) {
	return t.conn
}

func (t *Conn) String () string {
	return "Conn(" + t.Raddr() + ")"
}

func (t *Conn) writeString(message string) error {
	_, err := t.writer.WriteString(message)
	if err != nil {
		return err
	}
	return t.writer.Flush()
}

func (t *Conn) readString(delim byte) (string, error) {
	return t.reader.ReadString(delim)
}

func (t *Conn) Raddr() string {
	return t.conn.RemoteAddr().String()
}

func (t *Conn) ReadMessage() (interface{}, int, error) {
	data, err := t.readString('\n')
	if err != nil {
		return nil, 0, err
	}
	return Decode(data)
}

func (t *Conn) WriteMessage(obj interface{}) error {
	hsMsg, err := Encode(obj)
	if err != nil {
		return err
	}

	return t.writeString(hsMsg)
}

func NewConn(c net.Conn) *Conn {
	return &Conn{
		reader:	bufio.NewReader(c),
		writer:	bufio.NewWriter(c),
		conn:	c,
	}
}


