package flex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/v3io/k8svol/pkg/journal"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	v3ioConfig                 = "/etc/v3io/fuse/v3io.conf"
	v3ioSessionPayloadTemplate = `{
        "data": {
            "type": "session",
            "attributes": {
                "plane": "%s",
                "interface_kind": "fuse",
                "username": "%s",
                "password": "%s"
            }
        }
    }`
)

func ReadConfig() (*Config, error) {
	content, err := ioutil.ReadFile(v3ioConfig)
	if err != nil {
		return nil, err
	}
	config := Config{}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

type Config struct {
	RootPath string `json:"root_path"`
	FusePath string `json:"fuse_path"`
	Debug    bool   `json:"debug"`
	Type     string `json:"type"`
	Clusters []struct {
		Name    string `json:"name"`
		DataUrl string `json:"data_url"`
		ApiUrl  string `json:"api_url"`
	} `json:"clusters"`
}

type sessionResponse struct {
	Data struct {
		Id string `json:"id"`
	} `json:"data"`
}

func (c *Config) DataURLs() string {
	result := make([]string, len(c.Clusters), len(c.Clusters))
	for index, item := range c.Clusters {
		result[index] = item.DataUrl
	}
	return strings.Join(result, ",")
}

func (c *Config) ControlSession(username, password string) (string, error) {
	return c.Session(username, password, "control")
}

func (c *Config) DataSession(username, password string) (string, error) {
	return c.Session(username, password, "data")
}

func (c *Config) Session(username, password, plane string) (string, error) {
	payload := strings.NewReader(fmt.Sprintf(v3ioSessionPayloadTemplate, plane, username, password))
	journal.Debug("creating session", "plane", plane, "url", fmt.Sprintf("%s/api/sessions", c.Clusters[0].ApiUrl))
	response, err := http.Post(
		fmt.Sprintf("%s/api/sessions", c.Clusters[0].ApiUrl),
		"application/json",
		payload)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	journal.Debug("result from creating session", "status", response.Status, "body", string(bodyBytes))
	if response.StatusCode != 201 {
		return "", fmt.Errorf("error creating session. %d : %s", response.StatusCode, response.Status)
	}
	responseM := sessionResponse{}
	if err := json.Unmarshal(bodyBytes, &responseM); err != nil {
		return "", err
	}
	journal.Info("created session id", responseM.Data.Id)
	return responseM.Data.Id, nil
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

func (r *Response) ToJson() {
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
	Username  string `json:"kubernetes.io/secret/username"`
	Password  string `json:"kubernetes.io/secret/password"`
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

func (vs *VolumeSpec) GetFullUsername() string {
	if vs.Tenant != "" {
		return fmt.Sprintf("%s@%s", vs.GetUsername(), vs.GetTenant())
	}
	return vs.GetUsername()
}
