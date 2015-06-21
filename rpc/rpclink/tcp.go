package rpclink

import (
	"bytes"
	"io"
	"net"

	"gopkg.in/logex.v1"
)

type WriteItem struct {
	Data []byte
	Resp chan error
}

type Mux interface {
	Init(io.Reader)
	Handle(*bytes.Buffer) error
	WriteChan() (ch <-chan *WriteItem)
}

type TcpLink struct {
	mux       Mux
	conn      *net.TCPConn
	closeChan chan struct{}
}

func NewTcpLink(mux Mux) *TcpLink {
	th := &TcpLink{
		mux:       mux,
		closeChan: make(chan struct{}, 1),
	}
	return th
}

func (th *TcpLink) Init(conn net.Conn) {
	th.conn = conn.(*net.TCPConn)
	th.mux.Init(conn)
}

func (th *TcpLink) Protocol() string {
	return "tcp"
}

func (th *TcpLink) Handle() {
	go th.HandleRead()
	go th.HandleWrite()
}

func (th *TcpLink) HandleWrite() {
	var (
		item *WriteItem
		err  error
		n    int
	)

	writeChan := th.mux.WriteChan()
	defer th.Close()

	for {
		select {
		case item = <-writeChan:
		case <-th.closeChan:
			return
		}
		n, err = th.conn.Write(item.Data)
		if err == nil && n != len(item.Data) {
			err = logex.Trace(io.ErrShortWrite)
		}
		select {
		case item.Resp <- err:
		default:
		}
		if err != nil {
			logex.Error(err)
			break
		}
	}
}

func (th *TcpLink) HandleRead() {
	var (
		err    error
		buffer = bytes.NewBuffer(make([]byte, 0, 512))
	)
	defer th.Close()
	for {
		buffer.Reset()
		err = th.mux.Handle(buffer)
		if err != nil {
			logex.Error(err)
			break
		}
	}
}

func (th *TcpLink) Close() {
	th.conn.Close()
}