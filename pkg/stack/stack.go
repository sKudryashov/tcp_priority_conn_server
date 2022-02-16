package stack

import (
	"os"
	"strconv"
	"sync"

	"github.com/labstack/gommon/log"
)

const (
	// StackLengthDefault rerpesents stack length
	StackLengthDefault = 100
)

// StackLength represents actual stack name
var StackLength int

func init() {
	qsize := os.Getenv("QUEUE_SIZE")
	s, err := strconv.Atoi(qsize)
	if err != nil {
		StackLength = StackLengthDefault
	}
	StackLength = s
}

// WaitConnAPI represents waiting connection API
type WaitConnAPI interface {
	WritePushResponse()
	GetData() []byte
	CheckIsActive() bool
}

// NewStack represents a stack constructor
func NewStack(lgr *log.Logger, writeWait chan WaitConnAPI) *Stack {
	return &Stack{
		// readWait:  readWait,
		writeWait: writeWait,
		lgr:       lgr,
		data:      make([]interface{}, 0, StackLength),
	}
}

// Stack represents data type stack
type Stack struct {
	lgr       *log.Logger
	wrLock    bool
	mu        sync.RWMutex
	data      []interface{}
	len       int
	writeWait chan WaitConnAPI
	readWait  chan WaitConnAPI
}

// Push push data to stack
func (s *Stack) Push(i interface{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ln := len(s.data)
	s.lgr.Infof("the stack length %d ", ln)
	if ln < StackLength {
		s.data = append(s.data, i)
		s.lgr.Infof("the stack isn't full %d", ln)
		return true
	}
	s.lgr.Infof("the stack full %d", ln)
	dataByte := s.data[len(s.data)-1].([]byte)
	s.lgr.Infof("the stack full %d last record is %s", ln, string(dataByte))
	return false
}

//PushLock pushes data with lock
func (s *Stack) PushLock(i interface{}) bool {
	s.wrLock = false
	return s.Push(i)
}

// CanWrite returns can we write or not, if we can, we must do it to
// unlock any further records
func (s *Stack) CanWrite() (func(i interface{}) bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.data) < StackLength {
		s.wrLock = true
		return s.PushLock, true
	}
	s.wrLock = false
	return nil, false
}

// CanRead returns if we can read from stack
func (s *Stack) CanRead() bool {
	if len(s.data) > 0 {
		return true
	}
	return false
}

// IsStackFull returns status stack full
func (s *Stack) IsStackFull() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if StackLength < len(s.data) {
		return false
	}
	return true
}

// Pop pops data out of the stack
func (s *Stack) Pop() (interface{}, bool) {
	s.mu.Lock()
	l := len(s.data)
	if l == 0 {
		s.mu.Unlock()
		return nil, false
	}
	ln := l - 1
	data := s.data[ln]
	s.data = s.data[:ln]
	if len(s.writeWait) > 0 {
		dataQ := <-s.writeWait
		// if it is active, then it will be processed, if not, swept by the pool collector
		if dataQ.CheckIsActive() {
			data := dataQ.GetData()
			s.lgr.Infof(" waiting push writes data to the stack %s ", string(data))
			s.data = append(s.data, data)
			dataQ.WritePushResponse()
		}
	}
	s.mu.Unlock()
	return data, true
}

//Len shows the lenghth of the stack
func (s *Stack) Len() int {
	s.mu.RLock()
	ln := len(s.data)
	s.mu.RUnlock()
	return ln
}

//IsEmpty returns if the stack is empty
func (s *Stack) IsEmpty() bool {
	s.mu.RLock()
	ln := len(s.data)
	s.mu.RUnlock()
	return ln == 0
}
