package conn

import (
	"bufio"
	"context"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/sKudryashov/stacksrv/internal/service/formatter"
)

const (
	// ConnExpiration is a time in seconds after which conn has to be evicted from the
	// pool and closed
	ConnExpiration = 10
	// MaxConnDefault represents max connection per pool
	MaxConnDefault        = 100
	connCheckDeadline     = time.Millisecond * 10
	connCollectorInterval = time.Millisecond * 500
)

// MaxConn represents actual stack name
var MaxConn int

func init() {
	s, err := strconv.Atoi(os.Getenv("CONN_POOL_SIZE"))
	if err != nil {
		MaxConn = MaxConnDefault
	}
	MaxConn = s
	if MaxConn == 0 {
		panic("MaxConn can't be 0")
	}
}

//Conn represents app wrapper for TCP connection
type Conn struct {
	err error
	*net.TCPConn
	action    string
	mu        sync.RWMutex
	time      int64
	id        int
	data      []byte
	active    bool
	Ctx       context.Context
	CancelCtx func()
}

// SetErr sets current error
func (c *Conn) SetErr(err error) {
	c.mu.Lock()
	c.err = err
	c.mu.Unlock()
}

// GetErr returns latest error
func (c *Conn) GetErr() error {
	c.mu.RLock()
	err := c.err
	c.mu.RUnlock()
	return err
}

//CloseL is a concurrency unsafe wrapper for closing conn
func (c *Conn) CloseL() error {
	c.active = false
	err := c.TCPConn.Close()
	if c.CancelCtx != nil {
		c.CancelCtx()
	}
	return err
}

//Close is a wrapper for closing conn, also it cancels connection context
func (c *Conn) Close() error {
	c.mu.Lock()
	c.active = false
	c.mu.Unlock()
	err := c.TCPConn.Close()
	if c.CancelCtx != nil {
		c.CancelCtx()
	}
	return err
}

// SetID sets action for a connection
func (c *Conn) SetID(id int) {
	c.mu.Lock()
	c.id = id
	c.mu.Unlock()
}

// SetTime sets conn time
func (c *Conn) SetTime(time int64) {
	c.mu.Lock()
	c.time = time
	c.mu.Unlock()
}

// SetActive sets action for a connection
func (c *Conn) SetActive(active bool) {
	c.mu.Lock()
	c.active = active
	c.mu.Unlock()
}

// CheckIsActiveL checks lock-free whether conn active or not
func (c *Conn) CheckIsActiveL() bool {
	bufReader := bufio.NewReader(c)
	c.SetReadDeadline(time.Now().Add(time.Millisecond * 20))
	// if io.EOF, it means the conn is closed, but in general we are going to have read timeout err here, it means no one is writing
	if _, err := bufReader.ReadByte(); err != nil {
		if err == io.EOF {
			return false
		}
	} else {
		bufReader.UnreadByte()
	}
	return true
}

// CheckIsActive checks whether the connection is active
func (c *Conn) CheckIsActive() bool {
	c.mu.Lock()
	a := c.active
	if a {
		bufReader := bufio.NewReader(c)
		c.SetReadDeadline(time.Now().Add(time.Millisecond * 20))
		// if io.EOF, it means the conn is closed, but in general we are going to have read timeout err here, it means no one is writing
		if _, err := bufReader.ReadByte(); err != nil {
			if err == io.EOF {
				c.active = false
				a = false
			}
		} else {
			bufReader.UnreadByte()
		}
	}
	c.mu.Unlock()
	return a
}

// IsActive returns whether connection is active
func (c *Conn) IsActive() bool {
	c.mu.Lock()
	a := c.active
	c.mu.Unlock()
	return a
}

// GetID sets action for a connection
func (c *Conn) GetID() int {
	c.mu.Lock()
	id := c.id
	c.mu.Unlock()
	return id
}

// SetAction sets action for a connection
func (c *Conn) SetAction(action string) {
	c.mu.Lock()
	c.action = action
	c.mu.Unlock()
}

// GetAction sets action for a connection
func (c *Conn) GetAction() string {
	c.mu.Lock()
	a := c.action
	c.mu.Unlock()
	return a
}

// SetData sets data
func (c *Conn) SetData(data []byte) {
	c.mu.Lock()
	c.data = data
	c.mu.Unlock()
}

// GetData returns data
func (c *Conn) GetData() []byte {
	c.mu.Lock()
	data := c.data
	c.mu.Unlock()
	return data
}

// WritePushResponse writes push rsp
func (c *Conn) WritePushResponse() {
	c.Write([]byte{0})
	c.SetActive(false)
	c.Close()
}

// WriteErr writes error state
func (c *Conn) WriteErr() {
	// c.Write([]byte{0x00})
	c.SetActive(false)
	c.Close()
}

// WriteBusyState writes busy queue response
func (c *Conn) WriteBusyState() {
	c.Write([]byte{0xFF})
}

// WritePopResponse writes pop rsp
func (c *Conn) WritePopResponse(data []byte) {
	popRsp := formatter.FormatPopResponse(data)
	c.Write(popRsp)
	c.SetActive(false)
	c.Close()
}
