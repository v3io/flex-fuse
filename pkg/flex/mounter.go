package flex

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/v3io/flex-fuse/pkg/journal"
)

type Mounter struct {
	Config *Config
}

func NewMounter() (*Mounter, error) {
	journal.Debug("Creating configuration")
	config, err := NewConfig()
	if err != nil {
		return nil, err
	}

	return &Mounter{
		Config: config,
	}, nil
}

func (m *Mounter) Mount(targetPath string, specString string) *Response {
	journal.Debug("Mounting")

	spec := Spec{}
	if err := json.Unmarshal([]byte(specString), &spec); err != nil {
		return NewFailResponse("Failed to unmarshal spec", err)
	}

	if err := spec.validate(); err != nil {
		return NewFailResponse("Mount failed validation", err)
	}

	if m.Config.Type == "link" {
		return m.mountAsLink(&spec, targetPath)
	}

	if isMountPoint(targetPath) {
		return NewSuccessResponse(fmt.Sprintf("Already mounted: %s", targetPath))
	}

	return m.createV3IOFUSEContainer(&spec, targetPath)
}

func (m *Mounter) Unmount(targetPath string) *Response {
	journal.Debug("Unmounting")

	if m.Config.Type == "link" {
		return m.unmountAsLink(targetPath)
	}

	if !isMountPoint(targetPath) {
		return NewSuccessResponse("Not a mountpoint, nothing to do")
	}

	return m.removeV3IOFUSEContainer(targetPath)
}

func (m *Mounter) createV3IOFUSEContainer(spec *Spec, targetPath string) *Response {
	journal.Info("Creating v3io-fuse container", "target", targetPath)

	dataUrls, err := m.Config.DataURLs(spec.GetClusterName())
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
		"--session_key", spec.GetAccessKey(),
	}

	if spec.Container != "" {
		args = append(args, "-a", spec.Container)
		if spec.SubPath != "" {
			args = append(args, "-p", spec.SubPath)
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

func (m *Mounter) mountAsLink(spec *Spec, targetPath string) *Response {
	journal.Info("Mounting as link", "target", targetPath)
	linkPath := path.Join("/mnt/v3io", spec.Namespace, spec.Container)
	response := &Response{}

	if !isMountPoint(linkPath) {
		journal.Debug("Creating folder", "linkPath", linkPath)
		if err := os.MkdirAll(linkPath, 0755); err != nil {
			return NewFailResponse(fmt.Sprintf("Unable to create target %s", linkPath), err)
		}
		response = m.createV3IOFUSEContainer(spec, linkPath)
	}

	if err := os.Remove(targetPath); err != nil {
		return NewFailResponse(fmt.Sprintf("Unable to remove target %s", targetPath), err)
	}

	if err := os.Symlink(linkPath, targetPath); err != nil {
		return NewFailResponse(fmt.Sprintf("Unable to create link %s to target %s", linkPath, targetPath), err)
	}

	return response
}

func (m *Mounter) unmountAsLink(targetPath string) *Response {
	journal.Info("Calling unmountAsLink command", "target", targetPath)
	if err := os.Remove(targetPath); err != nil {
		return NewFailResponse("unable to remove link", err)
	}

	return NewSuccessResponse("link removed")
}

func getContainerName(targetPath string) string {
	return filepath.Base(targetPath)
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
