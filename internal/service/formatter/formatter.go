package formatter

import (
	"fmt"
	"strconv"
)

const (
	// ActionPush represents push action
	ActionPush = "0"

	// ActionPop represents POP action
	ActionPop = "1"
)

// ParseRequest parses the first request byte
func ParseRequest(header byte) (string, int64, error) {
	actionStr := fmt.Sprintf("%08b", header)
	action := actionStr[0]
	payloadLnStr := actionStr[1:]
	payloadLnInt, err := strconv.ParseInt(payloadLnStr, 2, 16)
	actionStr = string(action)
	return actionStr, payloadLnInt, err
}

// FormatPopResponse formats rsp for pop
func FormatPopResponse(data []byte) []byte {
	ln := len(data)
	response := make([]byte, 0, ln+1)
	response = append(response, byte(int64(ln)))
	response = append(response, data...)
	return response
}
