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
package main

import (
	"fmt"
	"os"

	"github.com/v3io/flex-fuse/pkg/flex"
	"github.com/v3io/flex-fuse/pkg/journal"
)

func handleAction() *flex.Response {
	journal.Debug("Handling action", os.Args)

	if len(os.Args) < 2 {
		return getArgumentFailResponse("Fuse requires at least an action argument")
	}

	switch action := os.Args[1]; action {
	case "init":
		result := flex.NewSuccessResponse("No initialization required")
		result.Capabilities = map[string]interface{}{
			"attach": false,
		}

		return result

	case "mount":
		if len(os.Args) != 4 {
			return getArgumentFailResponse("Mount requires 2 exactly arguments")
		}

		mounter, err := flex.NewMounter()
		if err != nil {
			return flex.NewFailResponse("Failed to create mounter", err)
		}

		return mounter.Mount(os.Args[2], os.Args[3])

	case "unmount":
		if len(os.Args) != 3 {
			return getArgumentFailResponse("Mount requires 1 exactly argument")
		}

		mounter, err := flex.NewMounter()
		if err != nil {
			return flex.NewFailResponse("Failed to create mounter", err)
		}

		return mounter.Unmount(os.Args[2])

	default:
		return getArgumentFailResponse(fmt.Sprintf("Received (%s) action is not supported", action))
	}
}

func getArgumentFailResponse(message string) *flex.Response {
	return flex.NewFailResponse(message, fmt.Errorf("Got %s", os.Args))
}

func main() {

	// handle the action and print the result
	fmt.Print(handleAction().ToJSON())
}
