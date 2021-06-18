package cri

type CRI interface {

	// CreateContainer creates a container
	CreateContainer(string, string, string, []string) error

	// RemoveContainer removes a container
	RemoveContainer(string) error

	// Close closes a CRI
	Close() error
}
