package flex

import (
	"fmt"
	"os/exec"
	"syscall"
)

func isStaleMount(path string) bool {
	stat := syscall.Stat_t{}
	err := syscall.Stat(path, &stat)
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			if errno == syscall.ESTALE {
				return true
			}
		}
	}
	return false
}

func isMountPoint(path string) bool {
	cmd := exec.Command("mountpoint", path)
	err := cmd.Run()
	if err != nil {
		return false
	}
	return true
}

func MakeResponse(status, message string) *Response {
	return &Response{
		Status:  status,
		Message: message,
	}
}

func Success(message string) *Response {
	return MakeResponse("Success", message)
}

func Fail(message string, err error) *Response {
	if err != nil {
		return MakeResponse("Failure", fmt.Sprintf("%s. %s", message, err))
	}
	return MakeResponse("Failure", fmt.Sprintf("%s", message))
}
