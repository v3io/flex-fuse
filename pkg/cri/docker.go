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
package cri

import (
	"fmt"
	"github.com/v3io/flex-fuse/pkg/journal"
	"os/exec"
)

type Docker struct {
	dockerBinaryPath string
}

func NewDocker(dockerBinaryPath string) (*Docker, error) {
	return &Docker{
		dockerBinaryPath: dockerBinaryPath,
	}, nil
}

// CreateContainer creates a container
func (d *Docker) CreateContainer(image string,
	containerName string,
	targetPath string,
	args []string) error {

	// Create the new container
	dockerCommandArgs := []string{
		"run",
		"--detach",
		"--privileged",
		"-v", "/etc/v3io/fuse:/etc/v3io/fuse",
		"--name",
		containerName,
		"--cgroup-parent",
		"/kubepods",
		"--device",
		"/dev/fuse",
		"--net=host",
		"--mount",
		fmt.Sprintf("type=bind,src=%s,target=/fuse_mount,bind-propagation=shared", targetPath),
		image,
	}

	// add the args, but skip the executable name, as the docker image already points to it
	dockerCommandArgs = append(dockerCommandArgs, args[1:]...)

	// execute the command
	dockerCommand := exec.Command(d.dockerBinaryPath, dockerCommandArgs...)

	journal.Debug("Executing docker run command", "path", dockerCommand.Path, "args", dockerCommand.Args)
	if dockerCommandOutput, err := dockerCommand.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to create v3io-fuse container %s: [%s] %s",
			targetPath,
			err.Error(),
			string(dockerCommandOutput))
	}

	return nil
}

// RemoveContainer removes a container
func (d *Docker) RemoveContainer(containerName string) error {
	args := []string{
		"rm",
		"--force",
		containerName,
	}

	dockerCommand := exec.Command(d.dockerBinaryPath, args...)

	journal.Debug("Executing docker rm command", "path", dockerCommand.Path, "args", dockerCommand.Args)
	if err := dockerCommand.Run(); err != nil {
		return err
	}

	return nil
}

func (d *Docker) Close() error {
	return nil
}
