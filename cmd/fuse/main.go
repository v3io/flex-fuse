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
		return getArgumentFailResponse("Received action is not supported")
	}
}

func getArgumentFailResponse(message string) *flex.Response {
	return flex.NewFailResponse(message, fmt.Errorf("Got %s", os.Args))
}

func main() {

	// handle the action and print the result
	fmt.Printf(handleAction().ToJSON())
}
