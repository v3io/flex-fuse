package flex

type ClusterConfig struct {
	Name     string   `json:"name"`
	DataUrls []string `json:"data_urls"`
}
