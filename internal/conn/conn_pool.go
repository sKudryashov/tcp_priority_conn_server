package conn

import (
	"container/list"
	"sync"
	"time"

	"github.com/labstack/gommon/log"
)

//NewConnPool a ConnPool constructor
func NewConnPool(lgr *log.Logger, doneCh <-chan interface{}) *ConnPool {
	cp := &ConnPool{
		doneCh:   doneCh,
		lgr:      lgr,
		connList: list.New(),
		list:     make([]*Conn, 0, MaxConn),
	}
	go cp.connSupervisor()
	return cp
}

//ConnPool represents a connection pool
type ConnPool struct {
	lgr      *log.Logger
	mu       sync.RWMutex
	connList *list.List
	doneCh   <-chan interface{}
	list     []*Conn
}

func (c *ConnPool) isConnOutdated(cc *Conn) bool {
	now := time.Now().Unix()
	diff := (now - cc.time)
	c.lgr.Debugf("diff >= ConnExpiration diff %d now %d and conn time %d conn id %d", diff, now, cc.time, cc.GetID())
	if diff >= ConnExpiration {
		c.lgr.Debugf("conn expired ", diff, now, cc.time, cc.GetID())
		return true
	}
	return false
}

//TryPush tries to push the conn
func (c *ConnPool) TryPush(cc *Conn, readingQueue chan<- *Conn) {
	connEv, ok := c.PushS(cc, readingQueue)
	if ok && connEv != nil {
		// evicted connection, mark as inactive (to treat appropriately in waiting queues) and close
		connEv.SetActive(false)
		c.lgr.Infof("conn evicted %d", cc.GetID())
		connEv.Close()
	} else if !ok {
		//busy, nothing to evict
		c.lgr.Infof("pool busy %d", cc.GetID())
		cc.Write([]byte{0xFF}) // note#1 uncommented for test - test_server_resource_limit commented for test_pops_to_empty_stack
		cc.Close()
	}
}

// PushS pushes connection to the pool, if the pool size is exceeded, but no
// connections that could be evicted - it returns false, if it has outdated connections,
// it returns true (meaning that new connection has already been added to the pool) and
// the connection which should be then closed by caller
func (c *ConnPool) PushS(cc *Conn, readingQueue chan<- *Conn) (*Conn, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ln := len(c.list)
	c.lgr.Debugf("connection pull length: %d", ln)
	c.lgr.Debugf("max conn : %d", MaxConn)
	if ln < MaxConn {
		cc.SetID(ln)
		c.list = append(c.list, cc)
		// callback call reading socket here
		c.lgr.Debugf("pushed conn ReadingQueue %d ", cc.GetID())
		select {
		case readingQueue <- cc:
			return nil, true
		}
	}
	first := c.list[0]
	// evict outdated connection
	if c.isConnOutdated(first) {
		c.lgr.Debug("note#1 conn outdated and will be evicted ")
		c.list = c.list[1:]
		c.list = append(c.list, cc)
		readingQueue <- cc
		return first, true
	}
	// evict inactive connection
	// note#1 doesn't work with test_server_resource_limit but makes work test_pops_to_empty_stack,
	// if commented, test_server_resource_limit works.
	// if !first.CheckIsActive() {
	// 	c.list = c.list[1:]
	// 	c.list = append(c.list, cc)
	// 	readingQueue <- cc
	// 	return first, true
	// }
	// EOF note#1
	c.lgr.Debug("note#1 no more connections can be added to the pool, no evicted either 0xFF code")
	// no more connections can be added to the pool, no evicted either
	return nil, false
}

func (c *ConnPool) checkIsActive(conn *Conn) bool {
	if !conn.IsActive() {
		c.lgr.Debugf("conn %d is inactive", conn.GetID())
		return false
	}

	return true
}

func (c *ConnPool) connSupervisor() {
	for {
		select {
		case <-c.doneCh:
			c.doneCh = nil
			c.mu.Lock()
			for _, cc := range c.list {
				cc.Close()
			}
			c.list = c.list[:0]
			c.mu.Unlock()
			c.lgr.Info("conn pool supervisor stopped")
			return
		default:
			time.Sleep(connCollectorInterval)
			c.freeInactive()
		}
	}
}

// FreeInactive frees all inactive connections from the pool
func (c *ConnPool) freeInactive() {
	c.mu.Lock()
	for i := 0; i < len(c.list); i++ {
		connInPool := c.list[i]
		if !c.checkIsActive(connInPool) {
			c.releaseConnByID(i)
			// avoiding double lock
			connInPool.CloseL()
			c.lgr.Debugf("conn %d swept by the pool collector", connInPool.GetID())
		}
	}
	c.mu.Unlock()
}

func (c *ConnPool) releaseConnByID(i int) {
	// i is a single element
	ln := len(c.list)
	if ln == 1 {
		c.list = c.list[:0]
		return
	}
	// i is a first element
	if i == 0 && ln > 1 {
		c.list = c.list[i+1:]
		return
	}
	// i is a last element
	if i == (ln - 1) {
		c.list = c.list[:i-1]
		return
	}
	left := c.list[:i-1]
	right := c.list[i+1:]
	left = append(left, right...)
	c.list = left
}

// Free evicts given connection from the pool
func (c *ConnPool) Free(conn *Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lgr.Debugf("free conn id called %d", conn.GetID())
	for i, connInPool := range c.list {
		if conn.GetID() == connInPool.GetID() {
			// i is a single element
			c.releaseConnByID(i)
			return
		}
	}
}
