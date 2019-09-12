package flex

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/v3io/flex-fuse/pkg/journal"
)

type Mounter struct {
	Target string
	Spec   *VolumeSpec
	Config *Config
}

func NewMounter(target, options string) (*Mounter, error) {
	opts := VolumeSpec{}
	if options != "" {
		if err := json.Unmarshal([]byte(options), &opts); err != nil {
			return nil, err
		}
	}

	journal.Debug("Creating configuration")
	config, err := NewConfig()
	if err != nil {
		return nil, err
	}

	return &Mounter{
		Target: target,
		Config: config,
		Spec:   &opts,
	}, nil
}

func (m *Mounter) Mount() *Response {
	journal.Debug("Mounting")

	if err := m.validate(); err != nil {
		return NewFailResponse("Mount failed validation", err)
	}

	if m.Config.Type == "link" {
		return m.mountAsLink()
	}

	if isMountPoint(m.Target) {
		return NewSuccessResponse(fmt.Sprintf("Already mounted: %s", m.Target))
	}

	return m.createV3IOFUSEContainer(m.Target)
}

func (m *Mounter) Unmount() *Response {
	journal.Debug("Unmounting")

	if m.Config.Type == "link" {
		return m.unmountAsLink()
	}

	if !isMountPoint(m.Target) {
		return NewSuccessResponse("Not a mountpoint, nothing to do")
	}

	return m.removeV3IOFUSEContainer(m.Target)
}

func (m *Mounter) createV3IOFUSEContainer(targetPath string) *Response {
	journal.Info("Creating v3io-fuse container", "target", targetPath)

	dataUrls, err := m.Config.DataURLs(m.Spec.GetClusterName())
	if err != nil {
		return NewFailResponse("Could not get cluster data urls", err)
	}

	args := []string{
		"run",
		"--detach",
		"--rm",
		"--privileged",
		"--name",
		getContainerName(targetPath),
		// TODO: discover if infiniband exists and pass this
		// "--device",
		// "/dev/infiniband/uverbs0",
		"--device",
		"/dev/fuse",
		"--net=host",
		"--mount",
		fmt.Sprintf("type=bind,src=%s,target=/fuse_mount,bind-propagation=shared", targetPath),
		"quay.io/iguazio/v3io-fuse:local",
		"-o", "allow_other",
		"--connection_strings", dataUrls,
		"--mountpoint", "/fuse_mount",
		"--session_key", m.Spec.GetAccessKey(),
	}

	if m.Spec.Container != "" {
		args = append(args, "-a", m.Spec.Container)
		if m.Spec.SubPath != "" {
			args = append(args, "-p", m.Spec.SubPath)
		}
	}

	mountCmd := exec.Command("/usr/bin/docker", args...)

	journal.Debug("Running docker run command", "path", mountCmd.Path, "args", mountCmd.Args)
	if err := mountCmd.Run(); err != nil {
		return NewFailResponse(fmt.Sprintf("Could not create v3io-fuse container: %s", targetPath), err)
	}

	for _, interval := range []time.Duration{1, 2, 4, 2, 1} {
		if isMountPoint(targetPath) {
			return NewSuccessResponse("Mount completed")
		}
		time.Sleep(interval * time.Second)
	}

	return NewFailResponse(fmt.Sprintf("Could not mount due to timeout: %s", targetPath), nil)
}

func (m *Mounter) removeV3IOFUSEContainer(targetPath string) *Response {
	journal.Info("Removing v3io-fuse container", "target", targetPath)

	args := []string{
		"rm",
		"--force",
		"--name",
		getContainerName(targetPath),
	}

	mountCmd := exec.Command("/usr/bin/docker", args...)

	journal.Debug("Running docker run command", "path", mountCmd.Path, "args", mountCmd.Args)
	if err := mountCmd.Run(); err != nil {
		return NewFailResponse(fmt.Sprintf("Could not create v3io-fuse container: %s", targetPath), err)
	}

	if err := os.Remove(targetPath); err != nil {
		return NewFailResponse(fmt.Sprintf("Could not remove directory", targetPath), err)
	}

	return NewFailResponse(fmt.Sprintf("Could not umount due to timeout: %s", targetPath), nil)
}

func (m *Mounter) mountAsLink() *Response {
	journal.Info("Mounting as link", "target", m.Target)
	targetPath := path.Join("/mnt/v3io", m.Spec.Namespace, m.Spec.Container)
	response := &Response{}

	if !isMountPoint(targetPath) {
		journal.Debug("Creating folder", "target", targetPath)
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return NewFailResponse(fmt.Sprintf("Unable to create target %s", targetPath), err)
		}
		response = m.createV3IOFUSEContainer(targetPath)
	}

	if err := os.Remove(m.Target); err != nil {
		m.Unmount()
		return NewFailResponse(fmt.Sprintf("Unable to remove target %s", m.Target), err)
	}

	if err := os.Symlink(targetPath, m.Target); err != nil {
		return NewFailResponse(fmt.Sprintf("Unable to create link %s to target %s", targetPath, m.Target), err)
	}

	return response
}

func (m *Mounter) unmountAsLink() *Response {
	journal.Info("Calling unmountAsLink command", "target", m.Target)
	if err := os.Remove(m.Target); err != nil {
		return NewFailResponse("unable to remove link", err)
	}

	return NewSuccessResponse("link removed")
}

func (m *Mounter) validate() error {
	if m.Spec.AccessKey == "" && m.Spec.OverrideAccessKey == "" {
		return errors.New("required access key is missing")
	}
	if m.Spec.SubPath != "" && m.Spec.Container == "" {
		return errors.New("can't have subpath without container value")
	}
	return nil
}

func getContainerName(targetPath string) string {
	return filepath.Base(targetPath)
}

func isStaleMount(path string) bool {
	journal.Debug("Checking if mount point is stale", "target", path)
	stat := syscall.Stat_t{}
	err := syscall.Stat(path, &stat)
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			if errno == syscall.ESTALE {
				journal.Debug("Mount point is stale", "target", path)
				return true
			}
		}
	}

	journal.Debug("Mount point is not stale", "target", path)
	return false
}

func isMountPoint(path string) bool {
	journal.Debug("Checking if path is a mount point", "target", path)
	cmd := exec.Command("mountpoint", path)
	err := cmd.Run()

	if err != nil {
		journal.Debug("Path is not a mount point", "target", path)
		return false
	}

	journal.Debug("Path is a mount point", "target", path)
	return true
}
