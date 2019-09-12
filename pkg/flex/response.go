package flex

import (
	"encoding/json"
	"fmt"

	"github.com/v3io/flex-fuse/pkg/journal"
)

type Response struct {
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

func newResponse(status, message string) *Response {
	return &Response{
		Status:  status,
		Message: message,
	}
}

func NewSuccessResponse(message string) *Response {
	journal.Info("Success", "message", message)

	return newResponse("Success", message)
}

func NewFailResponse(message string, err error) *Response {
	if err != nil {
		journal.Warn("Failed", "message", message, "err", err.Error())
		return newResponse("Failure", fmt.Sprintf("%s. %s", message, err))
	}
	journal.Warn("Failed", "message", message)
	return newResponse("Failure", message)
}

func (r *Response) String() string {
	if len(r.Capabilities) > 0 {
		return fmt.Sprintf("Response[Status=%s, Message=%s, Capabilities=%s]", r.Status, r.Message, r.Capabilities)
	}

	return fmt.Sprintf("Response[Status=%s, Message=%s]", r.Status, r.Message)
}

func (r *Response) ToJSON() string {
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{"status": "Failure", "Message": "%s"}`, err)
	}

	return fmt.Sprintf("%s", string(jsonBytes))
}
