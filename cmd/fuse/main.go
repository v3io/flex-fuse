package main

import (
	"fmt"
	"os"

	"github.com/v3io/flex-fuse/pkg/flex"
	"github.com/v3io/flex-fuse/pkg/journal"
)

func handleAction() *flex.Response {
	journal.Debug("Handling action", os.Args)

	switch action := os.Args[1]; action {
	case "init":
		result := flex.NewSuccessResponse("No initialization required")
		result.Capabilities = map[string]interface{}{
			"attach": false,
		}

		return result

	case "mount":
		mounter, err := flex.NewMounter(os.Args[2], os.Args[3])
		if err != nil {
			return flex.NewFailResponse("Failed to create mounter", err)
		}

		return mounter.Mount()

	case "unmount":
		mounter, err := flex.NewMounter(os.Args[2], "")
		if err != nil {
			return flex.NewFailResponse("Failed to create mounter", err)
		}

		return mounter.Unmount()

	default:
		return flex.NewFailResponse("Not supported",
			fmt.Errorf("Operation %s is not supported", action))
	}
}

func main() {

	// handle the action and print the result
	fmt.Printf(handleAction().ToJSON())
}
