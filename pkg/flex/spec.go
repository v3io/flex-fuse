package flex

import (
	"encoding/base64"
	"errors"
	"os"
)

type DirToCreate struct {
	Name              string `json:"name"`
	Permissions       os.FileMode `json:"permissions"`
}
type Spec struct {
	SubPath           string `json:"subPath"`
	Container         string `json:"container"`
	Cluster           string `json:"cluster"`
	OverrideAccessKey string `json:"accessKey"`
	AccessKey         string `json:"kubernetes.io/secret/accessKey"`
	PodName           string `json:"kubernetes.io/pod.name"`
	Namespace         string `json:"kubernetes.io/pod.namespace"`
	Name              string `json:"kubernetes.io/pvOrVolumeName"`
	DirsToCreate      string `json:"dirsToCreate"`
}

func (s *Spec) decodeOrDefault(value string) string {
	bytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value
	}
	return string(bytes)
}

func (s *Spec) validate() error {
	if s.AccessKey == "" && s.OverrideAccessKey == "" {
		return errors.New("required access key is missing")
	}

	if s.SubPath != "" && s.Container == "" {
		return errors.New("can't have subpath without container value")
	}

	return nil
}

func (s *Spec) GetAccessKey() string {
	if s.OverrideAccessKey == "" {
		return s.decodeOrDefault(s.AccessKey)
	}
	return s.OverrideAccessKey
}

func (s *Spec) GetClusterName() string {
	if s.Cluster == "" {
		return "default"
	}

	return s.Cluster
}
