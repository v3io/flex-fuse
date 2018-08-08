package flex

import (
	"encoding/json"
	"fmt"
	"github.com/v3io/k8svol/pkg/journal"
	"os"
	"os/exec"
	"path"
	"time"
)

/// Return status
func Init() *Response {
	resp := MakeResponse("Success", "No Initialization required")
	resp.Capabilities = map[string]interface{}{
		"attach": false,
	}
	return resp
}

type Mounter struct {
	Target string
	Spec   *VolumeSpec

	Config *Config
}

func (m *Mounter) doMount(targetPath string) *Response {
	session, err := m.Config.DataSession(m.Spec.Username, m.Spec.Password)
	if err != nil {
		return Fail("Could not create session", err)
	}
	mountCmd := exec.Command(m.Config.FusePath,
		"-o", "allow_root",
		"--connection_strings", m.Config.DataURLs(),
		"--mountpoint", targetPath,
		"-a", m.Spec.Container,
		"--session_key", session)

	journal.Debug("calling mount command", "path", mountCmd.Path, "args", mountCmd.Args)
	if err := mountCmd.Start(); err != nil {
		return Fail(fmt.Sprintf("Could not mount: %s", m.Target), err)
	}
	for _, interval := range []time.Duration{1, 2, 4} {
		if isMountPoint(targetPath) {
			return Success("Mount completed!")
		}
		time.Sleep(interval * time.Second)
	}
	return Fail(fmt.Sprintf("Could not mount due to timeout: %s", m.Target), nil)
}

func (m *Mounter) Mount() *Response {
	targetPath := path.Clean(m.Target)
	if isStaleMount(targetPath) {
		journal.Debug("calling isStaleMount command", "target", targetPath)
		unmountCmd := exec.Command("umount", targetPath)
		out, err := unmountCmd.CombinedOutput()
		if err != nil {
			return Fail(fmt.Sprintf("Could not unmount stale mount %s: %s", targetPath, out), err)
		}
	}

	journal.Debug("calling isMountPoint command", "target", targetPath)
	if !isMountPoint(targetPath) {
		return m.doMount(targetPath)
	}
	return Success(fmt.Sprintf("Already mounted: %s", m.Target))
}

func (m *Mounter) MountAsLink() *Response {
	journal.Info("calling MountAsLink command", "target", m.Target)
	targetPath := path.Join("/mnt/v3io", m.Spec.Container)
	response := &Response{}
	journal.Debug("calling isMountPoint command", "target", targetPath)
	if !isMountPoint(targetPath) {
		journal.Debug("creating folder", "target", targetPath)
		os.MkdirAll(targetPath, 0755)
		response = m.doMount(targetPath)
	}

	if err := os.Remove(m.Target); err != nil {
		m.Unmount()
		return Fail(fmt.Sprintf("unable to remove target %s", m.Target), err)
	}

	os.Symlink(targetPath, m.Target)
	return response
}

func (m *Mounter) UnmountAsLink() *Response {
	journal.Info("calling UnmountAsLink command", "target", m.Target)
	if err := os.Remove(m.Target); err != nil {
		return Fail("unable to remove link", err)
	}
	return Success("link removed")
}

func (m *Mounter) Unmount() *Response {
	journal.Info("calling Unmount command", "target", m.Target)
	targetPath := path.Clean(m.Target)
	if isMountPoint(targetPath) {
		journal.Debug("calling isMountPoint command", "target", targetPath)
		output, err := exec.Command("umount", targetPath).CombinedOutput()
		if err != nil {
			return Fail(fmt.Sprintf("cloud not unmount: %s", string(output)), err)
		}
	}
	return Success("Unmount completed!")
}

func NewMounter(target, options string) (*Mounter, error) {
	opts := VolumeSpec{}
	if options != "" {
		if err := json.Unmarshal([]byte(options), &opts); err != nil {
			return nil, err
		}
	}
	journal.Debug("reading config")
	config, err := ReadConfig()
	if err != nil {
		return nil, err
	}
	return &Mounter{
		Target: target,
		Config: config,
		Spec:   &opts,
	}, nil
}

func Mount(target, options string) *Response {
	mounter, err := NewMounter(target, options)
	if err != nil {
		return Fail("unable to create mounter", err)
	} else {
		return mounter.MountAsLink()
	}
}

func Unmount(target string) *Response {
	mounter, err := NewMounter(target, "")
	if err != nil {
		return Fail("unable to create mounter", err)
	} else {
		return mounter.UnmountAsLink()
	}
}
