package flex

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/v3io/flex-fuse/pkg/journal"
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
	cmd := exec.Command("mount")
	mountList, err := cmd.CombinedOutput()
	if err != nil {
		journal.Debug("calling isMountPoint command", "target", path, "result", false, "error", err)
		return false
	}
	mountListString := string(mountList)
	journal.Debug(mountListString)
	result := strings.Contains(mountListString, path+" type")
	journal.Debug("calling isMountPoint command", "target", path, "result", result)
	return result
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
