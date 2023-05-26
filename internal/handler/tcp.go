package handler

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/sKudryashov/stacksrv/internal/conn"
	"github.com/sKudryashov/stacksrv/internal/service"
	"github.com/sKudryashov/stacksrv/internal/service/formatter"
	"github.com/sKudryashov/stacksrv/pkg/logger"
)

// TCP represents TCP connection handler
type TCP struct {
	pool  *conn.ConnPool
	queue *service.Queue
}

// NewTCP constructor
func NewTCP(pool *conn.ConnPool) *TCP {
	return &TCP{
		pool:  pool,
		queue: service.NewQService(),
	}
}

// ConnListener listens an ordered conn queue, discards slow or err connections
// and proceeds with normal ones
func (t *TCP) ConnListener(readingQueue <-chan *conn.Conn, stopCh <-chan interface{}) {
	readErr := make(chan *conn.Conn, 10)
	connReady := make(chan *conn.Conn, 10)
	bodyReaderStop := make(chan interface{}) // stopCh as well
	for {
		select {
		case <-stopCh:
			logger.App.Info("server stop signal received, closing conn listener")
			close(bodyReaderStop)
			connReady = nil
			readErr = nil
			return
		case cc := <-readErr:
			logger.App.Errorf("conn listener got an error %s", cc.GetErr())
			cc.WriteErr()
			t.pool.Free(cc)
		case cc := <-readingQueue:
			// slow down goroutines to guarantee order of execution, since bu default go routines order is not guaranteed.
			time.Sleep(time.Millisecond * 20)
			go t.readBody(cc, connReady, readErr, bodyReaderStop)
		}
	}
}

func (t *TCP) readBody(conn *conn.Conn, connReady chan *conn.Conn, cherr chan *conn.Conn, chDone <-chan interface{}) {
	bytesBuf := make([]byte, 0, 128)
	bufReader := bufio.NewReader(conn)
	var contentLn int64
	i := 0
	conn.SetReadDeadline(time.Now().Add(time.Second * 20))
	conn.SetKeepAlive(true)

	for {
		select {
		case <-chDone:
			conn.Close()
			cherr = nil
			connReady = nil
			logger.App.Info("body reader closed")
			return
		default:
		}
		bb, err := bufReader.ReadByte()
		if i == 0 {
			action, payloadSize, err := formatter.ParseRequest(bb)
			if err != nil {
				conn.SetErr(err)
				cherr <- conn
				break
			}
			conn.SetAction(action)
			contentLn = payloadSize
		}
		if err != nil {
			switch err := err.(type) {
			case *net.OpError:
				logger.App.Errorf("tcp.go conn error %v %s", err, string(conn.GetData()))
				bufReader.UnreadByte()
				conn.SetErr(err)
				cherr <- conn
				return
			default:
				if err == io.EOF {
					logger.App.Errorf("io.EOF Error: %v", err)
					bufReader.UnreadByte()
					conn.SetErr(err)
					cherr <- conn
					return
				}
				logger.App.Errorf("io.EOF Error %v", err)
				bufReader.UnreadByte()
				conn.SetErr(err)
				cherr <- conn
				return
			}
		}
		bytesBuf = append(bytesBuf, bb)
		i++
		if int64(i) == contentLn+1 {
			logger.App.Debugf("content ln %d  actual message size %d", contentLn, len(bytesBuf))
			logger.App.Debugf("socket data read %s", string(conn.GetAction()))
			break
		}
	}

	if conn.GetAction() == "0" {
		if len(bytesBuf) > 0 {
			conn.SetData(bytesBuf[1:])
		} else {
			conn.SetErr(fmt.Errorf("push connection cannot be empty"))
			cherr <- conn
		}
	} else {
		conn.SetData(bytesBuf)
	}
	// if it's not active - do nothing, it will be swept later in the pool
	if !conn.IsActive() {
		return
	}
	// a rule of thumb
	conn.Ctx = context.TODO()
	t.HandleConn(conn.Ctx, conn)
}

// HandleConn reads tcp connection, faster clients go first exactly here
func (t *TCP) HandleConn(ctx context.Context, conn *conn.Conn) {
	releaseConn, err := t.queue.ProcessRequest(ctx, conn)
	if err != nil {
		logger.App.Errorf("error processing request %d %v", conn.GetID(), err)
		t.pool.Free(conn)
		return
	}
	if releaseConn {
		t.pool.Free(conn)
	}
}
