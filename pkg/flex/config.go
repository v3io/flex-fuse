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
	ImageRepository  string          `json:"image_repository"`
	ImageTag         string          `json:"image_tag"`
	RootPath         string          `json:"root_path"`
	FusePath         string          `json:"fuse_path"`
	Debug            bool            `json:"debug"`
	Type             string          `json:"type"`
	Clusters         []ClusterConfig `json:"clusters"`
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
