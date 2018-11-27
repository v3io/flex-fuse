package flex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/v3io/k8svol/pkg/journal"
	"io/ioutil"
	"strings"
)

const (
	v3ioConfig = "/etc/v3io/fuse/v3io.conf"
)

func ReadConfig() (*Config, error) {
	journal.Debug("Reading config", "path", v3ioConfig)
	content, err := ioutil.ReadFile(v3ioConfig)
	if err != nil {
		return nil, err
	}
	config := Config{}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}
	journal.Debug("Config read", "config", string(content))
	return &config, nil
}

type ClusterConfig struct {
	Name     string   `json:"name"`
	DataUrls []string `json:"data_urls"`
}

type Config struct {
	RootPath string          `json:"root_path"`
	FusePath string          `json:"fuse_path"`
	Debug    bool            `json:"debug"`
	Type     string          `json:"type"`
	Clusters []ClusterConfig `json:"clusters"`
}

func (c *Config) findCluster(cluster string) (*ClusterConfig, error) {
	for _, clusterConfig := range c.Clusters {
		if clusterConfig.Name == cluster {
			return &clusterConfig, nil
		}
	}
	return nil, fmt.Errorf("no such cluster %s", cluster)
}

func (c *Config) DataURLs(cluster string) (string, error) {
	clusterConfig, err := c.findCluster(cluster)
	if err != nil {
		return "", err
	}
	return strings.Join(clusterConfig.DataUrls, ","), nil
}

type Response struct {
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

func (r *Response) String() string {
	if len(r.Capabilities) > 0 {
		return fmt.Sprintf("Response[Status=%s, Message=%s, Capabilities=%s]", r.Status, r.Message, r.Capabilities)
	}
	return fmt.Sprintf("Response[Status=%s, Message=%s]", r.Status, r.Message)
}

func (r *Response) ToJSON() {
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		fmt.Printf(`{"status": "Failure", "Message": "%s"}`, err)
	} else {
		fmt.Printf("%s", string(jsonBytes))
	}
}

type VolumeSpec struct {
	SubPath   string `json:"subPath"`
	Container string `json:"container"`
	Cluster   string `json:"cluster"`
	Username  string `json:"kubernetes.io/secret/username"`
	Password  string `json:"kubernetes.io/secret/password"`
	AccessKey string `json:"kubernetes.io/secret/accessKey"`
	Tenant    string `json:"kubernetes.io/secret/tenant"`
	PodName   string `json:"kubernetes.io/pod.name"`
	Namespace string `json:"kubernetes.io/pod.namespace"`
	Name      string `json:"kubernetes.io/pvOrVolumeName"`
}

func (VolumeSpec) decodeOrDefault(value string) string {
	bytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value
	}
	return string(bytes)
}

func (vs *VolumeSpec) GetUsername() string {
	return vs.decodeOrDefault(vs.Username)
}

func (vs *VolumeSpec) GetTenant() string {
	return vs.decodeOrDefault(vs.Tenant)
}

func (vs *VolumeSpec) GetPassword() string {
	return vs.decodeOrDefault(vs.Password)
}

func (vs *VolumeSpec) GetAccessKey() string {
	return vs.decodeOrDefault(vs.AccessKey)
}

func (vs *VolumeSpec) GetFullUsername() string {
	if vs.Tenant != "" {
		return fmt.Sprintf("%s@%s", vs.GetUsername(), vs.GetTenant())
	}
	return vs.GetUsername()
}

func (vs *VolumeSpec) GetClusterName() string {
	if vs.Cluster == "" {
		return "default"
	}
	return vs.Cluster
}
