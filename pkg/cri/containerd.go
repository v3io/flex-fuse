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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/v3io/flex-fuse/pkg/common"
	"github.com/v3io/flex-fuse/pkg/journal"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type Containerd struct {
	containerdContext context.Context
	kubernetesContext context.Context
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

	// kubernetes namespace
	newContainerd.kubernetesContext = namespaces.WithNamespace(context.Background(), "k8s.io")

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

	// get the path to a log file
	logFilePath, err := c.getLogFilePath(containerName, targetPath)
	if err != nil {
		return err
	}

	journal.Debug("Creating log file",
		"containerName", containerName,
		"targetPath", targetPath,
		"logFilePath", logFilePath)

	v3ioFUSEContainer, err := c.createContainer(image, containerName, targetPath, args)
	if err != nil {
		return err
	}

	// create the actual process
	v3ioFUSETask, err := v3ioFUSEContainer.NewTask(c.containerdContext, cio.LogFile(logFilePath))
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

func extractECRRegion(imageName string) (string, error) {
	// Break down the regex parts:
	// \.(?:dkr\.)? - optional dkr prefix
	// ecr[^.]* - ecr followed by any non-dot characters (handles -public, -fips, etc.)
	// \. - dot separator
	// ([a-z0-9-]+) - capture group for region (allows letters, numbers, hyphens)
	// \. - dot separator
	// amazonaws\. - amazonaws prefix
	// [^\s/]+ - any non-whitespace/non-slash chars for domain suffix (.com, .com.cn, etc.)
	// Examples:
	//         "123456789012.dkr.ecr.us-east-1.amazonaws.com/myapp:latest",
	//         "123456789012.ecr.us-west-2.amazonaws.com/ml-model:v1.2.3",
	//         // Public ECR
	//         "public.ecr.aws/amazonlinux/amazonlinux:latest",
	//         "public.ecr.us-west-2.amazonaws.com/myapp:latest",
	//         // Gallery ECR
	//         "gallery.ecr.aws/amazonlinux/amazonlinux:latest",
	//         // Various ECR endpoints
	//         "123456789012.dkr.ecr-fips.us-east-1.amazonaws.com/myapp:latest",
	//         "123456789012.ecr-public.us-east-1.amazonaws.com/myapp:latest",
	//         "123456789012.dkr.ecr.cn-north-1.amazonaws.com.cn/myapp:latest",
	//         "123456789012.dkr.ecr.us-gov-west-1.amazonaws-us-gov.com/myapp:latest",
	//         // Future hypothetical endpoints
	//         "123456789012.dkr.ecr-newfeature.us-east-1.amazonaws.future/myapp:latest",
	//         "123456789012.dkr.ecr-special-endpoint.us-west-2.amazonaws.special/myapp:latest",

	re := regexp.MustCompile(`\.(?:dkr\.)?ecr[^.]*\.([a-z0-9-]+)\.amazonaws\.[^\s/]+`)

	match := re.FindStringSubmatch(imageName)

	if len(match) < 2 {
		return "", fmt.Errorf("could not extract region from ECR image name: %s", imageName)
	}

	region := match[1]

	// Handle special case where region might be 'aws'
	if region == "aws" {
		return "", fmt.Errorf("'aws' is not a region, this is likely a public ECR URL: %s", imageName)
	}

	return region, nil
}

// getECRPullCommand generates a pull command only if awsPath is found
// Assumes the image is already verified to be an ECR image and region is extracted
func getECRPullCommand(image, ctrPath, region string) (*exec.Cmd, error) {
	// Find aws CLI path
	awsPath, lookPathErr := exec.LookPath("aws")
	if lookPathErr != nil {
		return nil, fmt.Errorf("aws CLI not found in PATH: %w", lookPathErr)
	}

	// Get ECR password
	cmd := exec.Command(awsPath, "ecr", "get-login-password", "--region", region)
	ecrPasswordBytes, cmdErr := cmd.Output()
	if cmdErr != nil {
		return nil, fmt.Errorf("failed to retrieve ECR password: %w", cmdErr)
	}

	// Create pull command with ECR credentials
	ecrPassword := strings.TrimSpace(string(ecrPasswordBytes))
	pullCmd := exec.Command(ctrPath, "-n", "k8s.io", "images", "pull", "--user", fmt.Sprintf("AWS:%s", ecrPassword), image)

	return pullCmd, nil
}

func (c *Containerd) createContainer(image string,
	containerName string,
	targetPath string,
	args []string) (containerd.Container, error) {

	// The log filename incorporates the container-ID found in the `/proc/self/cgroup` file.
	// Specifically, we're scanning for a character sequence longer than 32 characters that appears after the last '/'.
	// If such a sequence isn't located, we use the term 'random'.
	// Additionally, a random number is appended to the end of the filename.
	// Here are examples of such cgroup files:
	// root@gke-zd-gke1-app-clust-zd-gke1-initial-7b135c73-jxn0:/#  cat /proc/self/cgroup
	// 13:misc:/
	// 12:rdma:/
	// 11:memory:/kubepods/besteffort/pod0404f9f9-7e8f-4cf0-848a-a7a23ef63393/466f13d55e758cf1e969744007435e2eb3d48f4d64f81fa7f2c2c7ac14690c23
	// 10:freezer:/kubepods/besteffort/pod0404f9f9-7e8f-4cf0-848a-a7a23ef63393/466f13d55e758cf1e969744007435e2eb3d48f4d64f81fa7f2c2c7ac14690c23
	// ...
	// 1:name=systemd:/kubepods/besteffort/pod0404f9f9-7e8f-4cf0-848a-a7a23ef63393/466f13d55e758cf1e969744007435e2eb3d48f4d64f81fa7f2c2c7ac14690c23
	// 0::/system.slice/containerd.service
	// [root@k8s-node1 /]# cat /proc/25512/cgroup
	// 11:perf_event:/kubepods/v3io-fuse-ef516052-8c8f-4ddc-b1ac-53a2b63c6d47-storage
	// ...
	// 2:devices:/kubepods/v3io-fuse-ef516052-8c8f-4ddc-b1ac-53a2b63c6d47-storage
	// 1:name=systemd:/kubepods/v3io-fuse-ef516052-8c8f-4ddc-b1ac-53a2b63c6d47-storage
	args = append(args, " 2>&1 | multilog s16777215 n20 /var/log/containers/flex-fuse-`awk 'match($0, /\\/([^/]+)$/) {if (RLENGTH>32) {printf \"%s.%08x\",substr($0, RSTART+1, RLENGTH-1), int(rand()*1e8) ;exit}} BEGIN {srand()} END {if (RLENGTH <= 32) { printf \"random.%08x\", int(rand()*1e8);}}' /proc/self/cgroup`")

	journal.Debug("Creating container",
		"image", image,
		"containerName", containerName,
		"targetPath", targetPath,
		"args", args)

	// try to get image from k8s namespace
	importedImages, err := c.tryImportFromK8sNamespace(image)
	if err != nil {
		journal.Debug("Failed to import image from k8s namespace. Error: " + err.Error())
	} else {
		journal.Debug("Successfully imported image from k8s namespace",
			"containerName", containerName,
			"lenImportedImages", strconv.Itoa(len(importedImages)),
			"currentImageName", image)

		// override image
		if len(importedImages) > 0 {
			image = importedImages[0].Name
		}
	}

	// assume image exists
	v3ioFUSEImage, err := c.containerdClient.GetImage(c.containerdContext, image)
	if err != nil {
		journal.Debug("Image does not exist, pulling",
			"containerName", containerName,
			"image", image)

		// pull the v3io-fuse image
		// [IG-23016] MountVolume.SetUp failed for volume storage in k8s 1.29
		//

		var err error

		// Get path to ctr
		var ctrPath string
		if ctrPath, err = exec.LookPath("ctr"); err == nil {
		} else if _, err = os.Stat("/usr/local/bin/ctr"); err == nil {
			ctrPath = "/usr/local/bin/ctr"
		} else if _, err = os.Stat("/usr/bin/ctr"); err == nil {
			ctrPath = "/usr/bin/ctr"
		}
		if err != nil {
			// Return an error if neither file exists
			journal.Error("Failed to pull image: ctr not found",
				"containerName", containerName,
				"image", image)
			return nil, err
		}

		var cmd *exec.Cmd

		err = fmt.Errorf("not ECR image")
		if strings.Contains(image, "amazonaws") {
			var ecrRegion string
			// First check if it's an ECR image
			ecrRegion, err = extractECRRegion(image)
			if err != nil {
				journal.Error("extractECRRegion return error, %v", err)
			} else {
				// If it is an ECR image, get the authenticated pull command
				cmd, err = getECRPullCommand(image, ctrPath, ecrRegion)
				if err != nil {
					journal.Error("ECR region is %s, getECRPullCommand return error, %v", ecrRegion, err)
				}
			}
		}
		// If it's not an ECR image, or if the ECR image pull command failed, fall back to the default ctr pull command
		if err != nil {
			cmd = exec.Command(ctrPath, "-n", "k8s.io", "images", "pull", "--hosts-dir", "/etc/containerd/certs.d/", image)
		}

		output, err := cmd.CombinedOutput()
		// Handle errors
		if err != nil {
			journal.Error("Failed pulling", "containerName", containerName, "image", image, "error", err, "command output", string(output))
			return nil, err
		}
		v3ioFUSEImage, err = c.containerdClient.GetImage(c.containerdContext, image)
		if err != nil {
			journal.Error("Failed to pull image",
				"containerName", containerName,
				"image", image)
			return nil, err
		}
	}

	mounts := []specs.Mount{
		{
			Destination: "/etc/v3io/fuse",
			Type:        "bind",
			Source:      "/etc/v3io/fuse",
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: "/fuse_mount",
			Type:        "bind",
			Source:      targetPath,
			Options:     []string{"rbind", "shared"},
		},
		{
			Destination: "/var/log/containers",
			Type:        "bind",
			Source:      "/var/log/containers",
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
		withCgroupParent(getCgroupParent()),
		withRootfsPropagation,
	}

	var spec specs.Spec

	snapshotterName := "overlayfs"

	// before creating, try to delete the snapshot if it exists - otherwise it'll fail
	c.containerdClient.SnapshotService(snapshotterName).Remove(c.containerdContext, containerName)

	return c.containerdClient.NewContainer(
		c.containerdContext,
		containerName,
		containerd.WithImage(v3ioFUSEImage),
		containerd.WithSnapshotter(snapshotterName),
		containerd.WithNewSnapshot(containerName, v3ioFUSEImage),
		containerd.WithImageStopSignal(v3ioFUSEImage, "SIGTERM"),
		containerd.WithRuntime("io.containerd.runc.v2", nil),
		containerd.WithSpec(&spec, options...),
	)
}

func (c *Containerd) getLogFilePath(containerName string, targetPath string) (string, error) {
	sanitizedTargetPath := strings.Replace(targetPath, "/", "-", -1)

	logFile, err := ioutil.TempFile("", fmt.Sprintf("%s-%s-", containerName, sanitizedTargetPath))
	if err != nil {
		return "", err
	}

	defer logFile.Close()

	return logFile.Name(), nil
}

func (c *Containerd) tryImportFromK8sNamespace(imageName string) ([]images.Image, error) {
	var buf bytes.Buffer
	var err error
	var imageInstance containerd.Image
	var importedImages []images.Image

	err = common.RetryFunc(c.containerdContext,
		10,
		3*time.Second,
		func(attempt int) (bool, error) {

			// make sure image is on k8s namespace
			imageInstance, err = c.containerdClient.GetImage(
				c.kubernetesContext,
				imageName,
			)
			if err != nil {
				journal.Debug("Failed to find image in k8s namespace, retrying",
					"attempt", attempt,
					"err", err.Error())
				return true, err
			}

			// reset buffer for next retry, if needed at all
			defer buf.Reset()

			// export from k8s context
			if err = c.containerdClient.Export(
				c.kubernetesContext,
				&buf,
				archive.WithImage(c.containerdClient.ImageService(), imageInstance.Name()),
			); err != nil {

				// exported failed - try again
				journal.Debug("Failed to export image from k8s namespace, retrying",
					"attempt", attempt,
					"err", err.Error())
				return true, err
			}

			// import to current containerd context
			importedImages, err = c.containerdClient.Import(c.containerdContext, &buf)
			if err != nil {

				// import failed, try again
				journal.Debug("Failed to import image to running namespace, retrying",
					"attempt", attempt,
					"err", err.Error())
				return true, err
			}

			// get imported image
			imageInstance, err = c.containerdClient.GetImage(
				c.containerdContext,
				imageName,
			)
			if err != nil {
				journal.Debug("Failed to find image in running namespace, retrying",
					"attempt", attempt,
					"err", err.Error())
				return true, err
			}

			// unpack imported
			if err = imageInstance.Unpack(c.containerdContext, ""); err != nil {
				journal.Debug("Failed to unpack imported image in running namespace, retrying",
					"attempt", attempt,
					"err", err.Error())
				return true, err
			}

			return false, nil
		})

	return importedImages, err
}

func withRootfsPropagation(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
	s.Linux.RootfsPropagation = "shared"
	return nil
}

func withCgroupParent(cgroupParentPath string) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, c *containers.Container, s *oci.Spec) error {
		s.Linux.CgroupsPath = path.Join(cgroupParentPath, c.ID)

		return nil
	}
}
func getCgroupParent() string {
	cmd := exec.Command("stat", "-fc", "%T", "/sys/fs/cgroup/")
	out, err := cmd.Output()
	if err != nil {
		return "/kubepods"
	}
	if strings.TrimSpace(string(out)) == "cgroup2fs" {
		return "/kubepods.slice"
	} else {
		return "/kubepods"
	}
}
