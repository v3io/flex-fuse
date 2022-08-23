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
package flex

import (
	"encoding/base64"
	"errors"
	"os"
)

type DirToCreate struct {
	Name        string      `json:"name"`
	Permissions os.FileMode `json:"permissions"`
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
