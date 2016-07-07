package container

import (
	"time"
)

// Network type, Bridge, host etc.
type NetworkType int

type ProcessSpaceType int

const (
	PID_PRIVATE = iota
	PID_HOST
	PID_CONTAINER
)

type ContainerData struct {
	ContainerType    string //docket, rkt etc
	Name             string
	ContainerId      string
	ImageId          string
	ListenPortMap    map[int]int
	Proxy            int // Pid of docker-proxy
	Privileged       bool
	Network          NetworkType
	Process          ProcessSpaceType
	VolumeMap        map[string]struct{}
	VirtualEthDevice string
	CreatedTime      time.Time
	Cmdline          string
}

type ImageData struct {
	Id        string
	Name      string
	Tag       string
	Mtime     time.Time
	Size      int64
	BuildTime time.Time
}

type Container interface {

	// Is docker installed on host?
	IsDockerInstalled() bool

	// Get container associated with various objects
	GetContainerForProcess(pid int) (containerId string, err error)
	GetContainerForListenPort(port int) (containerId string, err error)
	GetContainerForInterface(virtualEthDevice string) (containerId string, err error)

	//Get data about a container.
	GetContainerData(containerId string) (*ContainerData, error)

	//Get Sha-256 of an internal path in container.
	GetHashForPath(path string, containerId string) (hash []byte, err error)

	//Get username for internal UID
	GetUsernameForUid(containerId string, uid int) (string, error)

	// Get information about the image
	GetImageData(id string) (*ImageData, error)
}
