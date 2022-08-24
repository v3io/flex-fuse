/*
Copyright 2018 Iguazio Systems Ltd.

Licensed under the Apache License, Version 2.0 (the "License") with
an addition restriction as set forth herein. You may not use this
file except in compliance with the License. You may obtain a copy of
the License at http://www.apache.org/licenses/LICENSE-2.0.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing
permissions and limitations under the License.

In addition, you may not use the software for any purposes that are
illegal under applicable law, and the grant of the foregoing license
under the Apache 2.0 license is conditioned upon your compliance with
such restriction.
*/
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

	return string(jsonBytes)
}
