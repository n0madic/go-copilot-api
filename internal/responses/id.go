package responses

import (
	"crypto/rand"
	"fmt"
	"time"
)

func NewResponseID() string {
	return newID("resp")
}

func newMessageID() string {
	return newID("msg")
}

func newFunctionID() string {
	return newID("fc")
}

func newCallID() string {
	return newID("call")
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%x", prefix, b[:])
}
