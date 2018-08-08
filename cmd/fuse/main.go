package main

import (
	"fmt"
	"os"

	"github.com/v3io/k8svol/pkg/flex"
)

func main() {
	var result *flex.Response
	switch action := os.Args[1]; action {
	case "init":
		result = flex.Init()
	case "mount":
		result = flex.Mount(os.Args[2], os.Args[3])
	case "unmount":
		flex.Unmount(os.Args[2])
	default:
		result = flex.MakeResponse("Not supported", fmt.Sprintf("Operation %s is not supported", action))
	}

	result.PrintJson()
}
