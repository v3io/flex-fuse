package flex

import (
	"fmt"
	"github.com/v3io/k8svol/pkg/journal"
	"os/exec"
	"syscall"
)

func isStaleMount(path string) bool {
	journal.Debug("calling isStaleMount command", "target", path)
	stat := syscall.Stat_t{}
	err := syscall.Stat(path, &stat)
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			if errno == syscall.ESTALE {
				journal.Debug("calling isStaleMount command", "target", path, "result", true)
				return true
			}
		}
	}
	journal.Debug("calling isStaleMount command", "target", path, "result", false)
	return false
}

func isMountPoint(path string) bool {
	journal.Debug("calling isMountPoint command", "target", path)
	cmd := exec.Command("mountpoint", path)
	err := cmd.Run()
	if err != nil {
		journal.Debug("calling isMountPoint command", "target", path, "result", false)
		return false
	}
	journal.Debug("calling isMountPoint command", "target", path, "result", true)
	return true
}

func MakeResponse(status, message string) *Response {
	return &Response{
		Status:  status,
		Message: message,
	}
}

func Success(message string) *Response {
	journal.Info("Success", "message", message)

	return MakeResponse("Success", message)
}

func Fail(message string, err error) *Response {
	if err != nil {
		journal.Warn("Failed", "message", message, "err", err.Error())
		return MakeResponse("Failure", fmt.Sprintf("%s. %s", message, err))
	}
	journal.Warn("Failed", "message", message)
	return MakeResponse("Failure", message)
}
