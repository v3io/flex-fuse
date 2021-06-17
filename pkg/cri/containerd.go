package cri

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/v3io/flex-fuse/pkg/journal"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/cmd/ctr/commands/tasks"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type Containerd struct {
	containerdContext context.Context
	containerdClient  *containerd.Client
}

func NewContainerd(containerdSock string, contextName string) (*Containerd, error) {
	var err error

	newContainerd := Containerd{}

	newContainerd.containerdClient, err = containerd.New(containerdSock)
	if err != nil {
		return nil, err
	}

	// specify a namespace
	newContainerd.containerdContext = namespaces.WithNamespace(context.Background(), contextName)

	return &newContainerd, nil
}

func (c *Containerd) Close() error {
	return c.containerdClient.Close()
}

// CreateContainer creates a container
func (c *Containerd) CreateContainer(image string,
	containerName string,
	targetPath string,
	args []string) error {

	v3ioFUSEContainer, err := c.createContainer(image, containerName, targetPath, args)
	if err != nil {
		return err
	}

	// create the actual process
	v3ioFUSETask, err := tasks.NewTask(c.containerdContext,
		c.containerdClient,
		v3ioFUSEContainer,
		"",
		nil,
		false,
		"",
		[]cio.Opt{})

	if err != nil {
		return err
	}

	if err := v3ioFUSETask.Start(c.containerdContext); err != nil {
		return err
	}

	return nil
}

// RemoveContainer removes a container
func (c *Containerd) RemoveContainer(containerName string) error {
	journal.Debug("Removing container", "containerName", containerName)

	container, err := c.containerdClient.LoadContainer(c.containerdContext, containerName)
	if err != nil {
		return err
	}

	task, err := container.Task(c.containerdContext, cio.Load)
	if err != nil {
		journal.Debug("No task found for container, removing container",
			"containerName", containerName)

		return container.Delete(c.containerdContext)
	}

	journal.Debug("Got task for container",
		"containerName", containerName,
		"id", task.ID())

	status, err := task.Status(c.containerdContext)
	if err != nil {
		return err
	}

	journal.Debug("Got task status for container",
		"containerName", containerName,
		"status", status.Status)

	if status.Status != containerd.Stopped && status.Status != containerd.Created {
		journal.Debug("Killing task", "containerName", containerName)

		err = task.Kill(c.containerdContext,
			syscall.SIGTERM,
			containerd.WithKillAll)

		if err != nil {
			return fmt.Errorf("Failed killing %s's task: %s", containerName, err)
		}

		journal.Debug("Waiting for task to die", "containerName", containerName)

		// wait for task to exit
		taskExitStatusChan, err := task.Wait(c.containerdContext)
		if err != nil {
			return fmt.Errorf("Failed waiting for %s's task: %s", containerName, err)
		}

		select {
		case exitStatus := <-taskExitStatusChan:
			journal.Debug("Done waiting for task to exist",
				"containerName", containerName, "exitStatus", exitStatus)
		case <-time.After(20 * time.Second):
			return fmt.Errorf("Timed out waiting for %s's task to exit", containerName)
		}
	}

	if _, err := task.Delete(c.containerdContext); err != nil {
		return fmt.Errorf("Failed to delete %s's task: %s", containerName, err)
	}

	journal.Debug("Task deleted, deleting container", "containerName", containerName)

	return container.Delete(c.containerdContext)
}

func (c *Containerd) createContainer(image string,
	containerName string,
	targetPath string,
	args []string) (containerd.Container, error) {

	journal.Debug("Creating container",
		"image", image,
		"containerName", containerName,
		"targetPath", targetPath,
		"args", args)

	// pull the v3io-fuse image
	v3ioFUSEImage, err := c.containerdClient.Pull(c.containerdContext, image, containerd.WithPullUnpack)
	if err != nil {
		return nil, err
	}

	mounts := []specs.Mount{
		{
			Destination: "/etc/v3io/fuse",
			Type:        "bind",
			Source:      "/home/iguazio/fuse/etc",
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: "/fuse_mount",
			Type:        "bind",
			Source:      targetPath,
			Options:     []string{"rbind", "shared"},
		},
	}

	options := []oci.SpecOpts{
		oci.WithDefaultSpec(),
		oci.WithDefaultUnixDevices,
		oci.WithMounts(mounts),
		oci.WithImageConfig(v3ioFUSEImage),
		oci.WithProcessArgs(args...),
		oci.WithPrivileged,
		oci.WithAllDevicesAllowed,
		oci.WithHostDevices,
		oci.WithHostNamespace(specs.NetworkNamespace),
		oci.WithHostHostsFile,
		oci.WithHostResolvconf,
		oci.WithDevices("/dev/fuse", "", "rwm"),
		withRootfsPropagation,
	}

	var spec specs.Spec

	return c.containerdClient.NewContainer(
		c.containerdContext,
		containerName,
		containerd.WithImage(v3ioFUSEImage),
		containerd.WithSnapshotter("overlayfs"),
		containerd.WithNewSnapshot(containerName, v3ioFUSEImage),
		containerd.WithImageStopSignal(v3ioFUSEImage, "SIGTERM"),
		containerd.WithRuntime("io.containerd.runc.v2", nil),
		containerd.WithSpec(&spec, options...),
	)
}

func withRootfsPropagation(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
	s.Linux.RootfsPropagation = "shared"
	return nil
}
