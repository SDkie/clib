package docker

import (
	"errors"
	"time"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/kdsukhani/container"
	"github.com/kdsukhani/container/logger"
)

const (
	MinVersion = "1.18"
)

var (
	ErrConnectionFailed = errors.New("Cannot connect to the Docker daemon. Is the docker daemon running on this host?")

	ErrGettingContainerList = errors.New("Error while getting Container List")
	ErrGettingImageList     = errors.New("Error while getting Image List")

	ErrImageNotFound     = errors.New("Image Not Found")
	ErrContainerNotFound = errors.New("Container Not Found")

	ErrOnContainerInspect = errors.New("Error on Container Inspect")

	ErrFuncNotDefined = errors.New("Func Not Defined")
)

type Docker struct {
}

func (d Docker) getClient() (*client.Client, error) {
	client, err := client.NewClient(client.DefaultDockerHost, MinVersion, nil, nil)
	if err != nil {
		logger.Err(err)
		return nil, ErrConnectionFailed
	}

	return client, nil
}

func (d Docker) IsDockerInstalled() bool {
	// TODO : Check this
	_, err := d.getClient()
	if err != nil {
		logger.Err(err)
		return false
	}

	return true
}

func (d Docker) GetContainerForProcess(pid int) (containerId string, err error) {
	cli, err := d.getClient()
	if err != nil {
		return "", err
	}

	containers, err := cli.ContainerList(context.TODO(), types.ContainerListOptions{})
	if err != nil {
		logger.Err(err)
		return "", ErrGettingContainerList
	}

	for _, container := range containers {
		containerJson, err := cli.ContainerInspect(context.TODO(), container.ID)
		if err != nil {
			logger.Err(err)
			return "", ErrOnContainerInspect
		}

		if containerJson.State.Pid == pid {
			return container.ID, nil
		}
	}

	logger.Err("Container Not found for Pid: %d", pid)
	return "", ErrContainerNotFound
}

func (d Docker) GetContainerForListenPort(port int) (containerId string, err error) {
	cli, err := d.getClient()
	if err != nil {
		return "", err
	}

	containers, err := cli.ContainerList(context.TODO(), types.ContainerListOptions{})
	if err != nil {
		logger.Err(err)
		return "", ErrGettingContainerList
	}

	for _, container := range containers {
		for _, cport := range container.Ports {
			if cport.PublicPort == port {
				return container.ID, nil
			}
		}
	}

	logger.Err("Container Not found for Port: %d", port)
	return "", ErrContainerNotFound
}

func (d Docker) GetContainerForInterface(virtualEthDevice string) (string, error) {
	// TODO : Not Defined
	return "", ErrFuncNotDefined
}

func (d Docker) GetContainerData(containerId string) (*container.ContainerData, error) {
	cli, err := d.getClient()
	if err != nil {
		return nil, err
	}

	containerJson, err := cli.ContainerInspect(context.TODO(), containerId)
	if err != nil {
		logger.Err(err)
		return nil, ErrOnContainerInspect
	}

	containerData := new(container.ContainerData)
	containerData.ContainerType = "DOCKER"
	containerData.Name = containerJson.Name
	containerData.ContainerId = containerJson.ID
	containerData.ImageId = containerJson.Image
	// TODO containerData.ListenPortMap
	// TODO containerData.Proxy
	containerData.Privileged = containerJson.HostConfig.Privileged

	// NetworkType
	if containerJson.HostConfig.NetworkMode.IsBridge() {
		containerData.Network = container.NETWORK_TYPE_BRIDGE
	} else if containerJson.HostConfig.NetworkMode.IsHost() {
		containerData.Network = container.NETWORK_TYPE_HOST
	} else if containerJson.HostConfig.NetworkMode.IsContainer() {
		containerData.Network = container.NETWORK_TYPE_CONTAINER
	} else if containerJson.HostConfig.NetworkMode.IsNone() {
		containerData.Network = container.NETWORK_TYPE_NONE
	} else if containerJson.HostConfig.NetworkMode.IsDefault() {
		containerData.Network = container.NETWORK_TYPE_DEFAULT
	} else if containerJson.HostConfig.NetworkMode.IsUserDefined() {
		containerData.Network = container.NETWORK_TYPE_USER_DEFINED
	}

	// ProcessSpaceType
	if containerJson.HostConfig.PidMode.IsPrivate() {
		containerData.Process = container.PID_PRIVATE
	} else if containerJson.HostConfig.PidMode.IsHost() {
		containerData.Process = container.PID_HOST
	} else if containerJson.HostConfig.PidMode.IsContainer() {
		containerData.Process = container.PID_CONTAINER
	}

	containerData.VolumeMap = containerJson.Config.Volumes
	// TODO	containerData.VirtualEthDevice
	containerData.CreatedTime, err = time.Parse(time.RFC3339, containerJson.Created)
	if err != nil {
		logger.Err("Error while Parsing time - %s", containerJson.Created)
	}

	if len(containerJson.Config.Cmd[0]) > 0 {
		containerData.Cmdline = containerJson.Config.Cmd[0]
	}
	return containerData, nil
}

func (d Docker) GetHashForPath(path string, containerId string) ([]byte, error) {
	// TODO : Func Not Defined
	return []byte{}, ErrFuncNotDefined
}

func (d Docker) GetUsernameForUid(containerId string, uid int) (string, error) {
	// TODO : Func Not Defined
	return "", ErrFuncNotDefined
}

func (d Docker) GetImageData(id string) (*container.ImageData, error) {
	cli, err := d.getClient()
	if err != nil {
		return nil, err
	}

	images, err := cli.ImageList(context.TODO(), types.ImageListOptions{})
	if err != nil {
		logger.Err(err)
		return nil, ErrGettingImageList
	}

	imageData := new(container.ImageData)

	for _, image := range images {
		if image.ID == id {
			imageData.Id = image.ID

			// Check this - VIMP
			if len(image.RepoTags) > 2 {
				imageData.Name = image.RepoTags[0]
				imageData.Tag = image.RepoTags[1]
			} else if len(image.RepoTags) > 1 {
				imageData.Name = image.RepoTags[0]
			}

			//	TODO imageData.Mtime
			imageData.Size = image.Size
			imageData.BuildTime = time.Unix(image.Created, 0)
			return imageData, nil
		}
	}

	return nil, ErrImageNotFound
}
