package service

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/sKudryashov/stacksrv/internal/service/formatter"
	"github.com/sKudryashov/stacksrv/pkg/stack"
)

// ErrAction represents an error when action isn't registered
type ErrAction struct {
	error
}

//WriterAPI represents API in which we can write
type WriterAPI interface {
	SetActive(bool)
	IsActive() bool
	CheckIsActive() bool
	WritePushResponse()
	WriteBusyState()
	WritePopResponse([]byte)
	GetAction() string
	GetData() []byte
	GetID() int
}

// Queue service operates on queue on a highlevel providing any business logic on top of
// the data structure itself
type Queue struct {
	lgr         *log.Logger
	st          *stack.Stack
	waitReadCh  chan WriterAPI
	waitWriteCh chan stack.WaitConnAPI
	readWait    []WriterAPI
	writeWait   []WriterAPI
}

// NewQService constructor
func NewQService(lgr *log.Logger) *Queue {
	wwr := make(chan stack.WaitConnAPI, 100)
	q := &Queue{
		st:          stack.NewStack(lgr, wwr),
		waitReadCh:  make(chan WriterAPI, 100),
		waitWriteCh: wwr,
		lgr:         lgr,
	}
	go q.processWaits()
	return q
}

func (q *Queue) processWaits() {
	for {
		time.Sleep(time.Second * 1)
		if len(q.waitReadCh) > 0 {
			if d, ok := q.st.Pop(); ok {
				data := d.([]byte)
				select {
				case conn := <-q.waitReadCh:
					if conn.CheckIsActive() {
						conn.WritePopResponse(data)
					}
				}
			}
		}
	}
}

func (q *Queue) addWaitingRead(conn WriterAPI) {
	q.waitReadCh <- conn
}

func (q *Queue) addWaitingWrite(conn WriterAPI) {
	q.waitWriteCh <- conn
}

// ProcessRequest processes single queue request
func (q *Queue) ProcessRequest(ctx context.Context, conn WriterAPI) (bool, error) {
	action := conn.GetAction()
	switch action {
	case formatter.ActionPop:
		q.lgr.Debugf("action POP")
		if !conn.CheckIsActive() {
			q.lgr.Debugf("connection is not active and can't be processed %d", conn.GetID())
			return false, nil
		}
		data, ok := q.st.Pop()
		if !ok {
			q.lgr.Debugf("there is nothing to read, waiting")
			q.addWaitingRead(conn)
			return false, nil
		}
		dataByte := data.([]byte)
		q.lgr.Infof("POP from the stack %s", string(dataByte))
		conn.WritePopResponse(dataByte)

		return true, nil
	case formatter.ActionPush:
		if !conn.CheckIsActive() {
			q.lgr.Debugf("connection is not active and can't be processed %d", conn.GetID())
			return false, nil
		}
		data := conn.GetData()
		if ok := q.st.Push(data); !ok {
			q.lgr.Infof("no place to push %s left, waiting", string(data))
			q.addWaitingWrite(conn)
			return false, nil
		}
		q.lgr.Infof("data PUSHed to the stack %s", string(data))
		conn.SetActive(false)
		conn.WritePushResponse()

		return true, nil
	default:
		return false, ErrAction{fmt.Errorf("unregistered action %s", action)}
	}
}
