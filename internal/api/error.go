package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type HTTPError struct {
	Message    string
	StatusCode int
	Body       []byte
}

func (e *HTTPError) Error() string {
	if len(e.Body) > 0 {
		return fmt.Sprintf("%s: %s", e.Message, string(e.Body))
	}
	return e.Message
}

func NewHTTPError(message string, resp *http.Response) *HTTPError {
	var body []byte
	if resp != nil && resp.Body != nil {
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	return &HTTPError{Message: message, StatusCode: statusCode, Body: body}
}
