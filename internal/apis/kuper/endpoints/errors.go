package endpoints

import (
	"encoding/json"
	"fmt"
	"strings"
)

type APIError struct {
	Status  int
	Code    any
	Message string
	Body    string
}

func (e *APIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = strings.TrimSpace(e.Body)
	}
	return fmt.Sprintf("api error: status=%d code=%v message=%s", e.Status, e.Code, msg)
}

func ParseAPIError(status int, body []byte) *APIError {
	out := &APIError{Status: status, Body: string(body)}

	var m map[string]any
	if json.Unmarshal(body, &m) == nil {
		if v, ok := m["code"]; ok {
			out.Code = v
		}
		if v, ok := m["message"].(string); ok {
			out.Message = v
		}
	}
	return out
}
