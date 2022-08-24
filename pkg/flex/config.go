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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/v3io/flex-fuse/pkg/journal"
)

const (
	v3ioConfig = "/etc/v3io/fuse/v3io.conf"
)

type Config struct {
	ImageRepository string          `json:"image_repository"`
	ImageTag        string          `json:"image_tag"`
	RootPath        string          `json:"root_path"`
	FusePath        string          `json:"fuse_path"`
	Debug           bool            `json:"debug"`
	Type            string          `json:"type"`
	Clusters        []ClusterConfig `json:"clusters"`
	V3ioConfigPath  string          `json:"v3io_config_path"`
}

func NewConfig() (*Config, error) {
	content, err := ioutil.ReadFile(v3ioConfig)
	if err != nil {
		return nil, err
	}

	config := Config{}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	journal.Debug("Created configuration", "content", string(content))

	return &config, nil
}

func (c *Config) DataURLs(cluster string) (string, error) {
	clusterConfig, err := c.findCluster(cluster)
	if err != nil {
		return "", err
	}
	return strings.Join(clusterConfig.DataUrls, ","), nil
}

func (c *Config) findCluster(cluster string) (*ClusterConfig, error) {
	for _, clusterConfig := range c.Clusters {
		if clusterConfig.Name == cluster {
			return &clusterConfig, nil
		}
	}
	return nil, fmt.Errorf("no such cluster %s", cluster)
}
