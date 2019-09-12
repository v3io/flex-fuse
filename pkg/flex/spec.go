package flex

import "encoding/base64"

type VolumeSpec struct {
	SubPath           string `json:"subPath"`
	Container         string `json:"container"`
	Cluster           string `json:"cluster"`
	OverrideAccessKey string `json:"accessKey"`
	AccessKey         string `json:"kubernetes.io/secret/accessKey"`
	PodName           string `json:"kubernetes.io/pod.name"`
	Namespace         string `json:"kubernetes.io/pod.namespace"`
	Name              string `json:"kubernetes.io/pvOrVolumeName"`
}

func (VolumeSpec) decodeOrDefault(value string) string {
	bytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value
	}
	return string(bytes)
}

func (vs *VolumeSpec) GetAccessKey() string {
	if vs.OverrideAccessKey == "" {
		return vs.decodeOrDefault(vs.AccessKey)
	}
	return vs.OverrideAccessKey
}

func (vs *VolumeSpec) GetClusterName() string {
	if vs.Cluster == "" {
		return "default"
	}
	return vs.Cluster
}
