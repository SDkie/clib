package docker

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/SDkie/clib"
	"github.com/SDkie/clib/logger"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/satori/go.uuid"
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

	ErrGivenDir = errors.New("Given Path is Dir")
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
	_, err := d.getClient()
	if err != nil {
		logger.Err(err)
		return false
	}

	return true
}

func (d Docker) GetContainerForProcess(pid int) (containerId string, err error) {
	ctx := context.Background()
	cli, err := d.getClient()
	if err != nil {
		return
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		logger.Err(err)
		return "", ErrGettingContainerList
	}

	for _, container := range containers {
		containerJson, err := cli.ContainerInspect(ctx, container.ID)
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
	ctx := context.Background()
	cli, err := d.getClient()
	if err != nil {
		return "", err
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
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
	ctx := context.Background()
	cli, err := d.getClient()
	if err != nil {
		return "", err
	}

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		logger.Err(err)
		return "", ErrGettingContainerList
	}

	for _, container := range containers {

		containerJson, err := cli.ContainerInspect(ctx, container.ID)
		if err != nil {
			logger.Err(err)
			return "", err
		}
		veth, err := getVethNameFromDockerPid(containerJson.State.Pid, containerJson.ID)
		if err != nil {
			logger.Err(err)
			return "", err
		}

		if veth == virtualEthDevice {
			return containerJson.ID, nil
		}

	}
	return "", errors.New("Not Found")
}

func (d Docker) GetContainerData(containerId string) (*clib.ContainerData, error) {
	ctx := context.Background()
	cli, err := d.getClient()
	if err != nil {
		return nil, err
	}

	containerJson, err := cli.ContainerInspect(ctx, containerId)
	if err != nil {
		logger.Err(err)
		return nil, ErrOnContainerInspect
	}

	containerData := new(clib.ContainerData)
	containerData.ContainerType = "DOCKER"
	containerData.Name = containerJson.Name
	containerData.ContainerId = containerJson.ID
	containerData.ImageId = containerJson.Image

	// ListenPortMap
	for key, value := range containerJson.NetworkSettings.Ports {
		fromPort := key.Int()
		var toPort int64

		if len(value) >= 1 {
			toPort, err = strconv.ParseInt(value[0].HostPort, 10, 64)
			if err != nil {
				logger.Err(err)
			}
		}

		containerData.ListenPortMap[fromPort] = int(toPort)
	}

	// TODO containerData.Proxy
	containerData.Privileged = containerJson.HostConfig.Privileged

	// NetworkType
	if containerJson.HostConfig.NetworkMode.IsBridge() {
		containerData.Network = clib.NETWORK_TYPE_BRIDGE
	} else if containerJson.HostConfig.NetworkMode.IsHost() {
		containerData.Network = clib.NETWORK_TYPE_HOST
	} else if containerJson.HostConfig.NetworkMode.IsContainer() {
		containerData.Network = clib.NETWORK_TYPE_CONTAINER
	} else if containerJson.HostConfig.NetworkMode.IsNone() {
		containerData.Network = clib.NETWORK_TYPE_NONE
	} else if containerJson.HostConfig.NetworkMode.IsDefault() {
		containerData.Network = clib.NETWORK_TYPE_DEFAULT
	} else if containerJson.HostConfig.NetworkMode.IsUserDefined() {
		containerData.Network = clib.NETWORK_TYPE_USER_DEFINED
	}

	// ProcessSpaceType
	if containerJson.HostConfig.PidMode.IsPrivate() {
		containerData.Process = clib.PID_PRIVATE
	} else if containerJson.HostConfig.PidMode.IsHost() {
		containerData.Process = clib.PID_HOST
	} else if containerJson.HostConfig.PidMode.IsContainer() {
		containerData.Process = clib.PID_CONTAINER
	}

	containerData.VolumeMap = containerJson.Config.Volumes
	containerData.VirtualEthDevice, err = getVethNameFromDockerPid(containerJson.State.Pid, containerJson.ID)
	if err != nil {
		logger.Err(err)
	}
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
	cli, err := d.getClient()
	if err != nil {
		return nil, err
	}

	stat, err := cli.ContainerStatPath(context.Background(), containerId, path)
	if err != nil {
		logger.Err(err)
		return nil, err
	}

	if !stat.Mode.IsDir() {
		logger.Errf("%s - %s", ErrGivenDir.Error(), path)
		return nil, ErrGivenDir
	}

	fileDir, fileName, err := getContainerFile(containerId, path, cli)
	if err != nil {
		logger.Err(err)
		return nil, err
	}
	filePath := fileDir + string(os.PathSeparator) + fileName
	defer os.Remove(fileDir)

	file, err := os.Open(filePath)
	if err != nil {
		logger.Err(err)
		return nil, err
	}
	defer file.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		logger.Err(err)
		return nil, err
	}

	hash := hasher.Sum(nil)

	return hash, nil
}

func (d Docker) GetUsernameForUid(containerId string, uid int) (string, error) {
	cli, err := d.getClient()
	if err != nil {
		return "", err
	}

	fileDir, fileName, err := getContainerFile(containerId, "/etc/passwd", cli)
	if err != nil {
		logger.Err(err)
		return "", err
	}
	filePath := fileDir + string(os.PathSeparator) + fileName
	defer os.Remove(fileDir)

	file, err := os.Open(filePath)
	if err != nil {
		logger.Err(err)
		return "", err
	}
	defer file.Close()

	content, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Err(err)
		return "", err
	}

	regx := fmt.Sprintf("\n.*:.*:%d:", uid)

	re := regexp.MustCompile(regx)
	strs := re.FindAllString("\n"+string(content), 1)
	if len(strs) == 0 {
		err = fmt.Errorf("UID %d not found", uid)
		logger.Err(err)
		return "", err
	} else if len(strs) > 1 {
		err = fmt.Errorf("Invalid Request")
		logger.Err(err)
		return "", err
	}

	index := strings.Index(strs[0], ":")
	username := strs[0][1:index]
	return username, nil
}

func (d Docker) GetImageData(id string) (*clib.ImageData, error) {
	ctx := context.Background()
	cli, err := d.getClient()
	if err != nil {
		return nil, err
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		logger.Err(err)
		return nil, ErrGettingImageList
	}

	imageData := new(clib.ImageData)

	for _, image := range images {
		if image.ID == id {
			imageData.Id = image.ID

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

func getVethNameFromDockerPid(pid int, containerId string) (string, error) {
	err := os.Mkdir("/var/run/netns", 0777)
	if err != nil && !os.IsExist(err) {
		logger.Err(err)
		return "", err
	}

	path, err := exec.LookPath("ln")
	if err != nil {
		logger.Err(err)
		return "", err
	}

	cmd := exec.Command(path, "-sf", fmt.Sprintf("/proc/%d/ns/net", pid), fmt.Sprintf("/var/run/netns/%s", containerId))
	err = cmd.Run()
	if err != nil {
		logger.Err(err)
		return "", err
	}

	path, err = exec.LookPath("ip")
	if err != nil {
		logger.Err(err)
		return "", err
	}

	cmd = exec.Command(path, "netns", "exec", containerId, "ip", "link", "show", "eth0")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		logger.Err(err)
		return "", err
	}
	strs := strings.Split(out.String(), ":")
	indexString := strings.TrimSpace(strs[0])
	if indexString == "" {
		err = errors.New("eth0 Not Found")
		logger.Err(err)
		return "", err
	}

	index, err := strconv.ParseInt(indexString, 10, 64)
	if err != nil {
		logger.Err(err)
		return "", err
	}

	cmd = exec.Command(path, "link", "show")
	out.Reset()
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		logger.Err(err)
		return "", err
	}

	re := regexp.MustCompile(fmt.Sprintf("%d: .*:", index+1))
	strs = re.FindAllString(out.String(), 1)
	strs = strings.Split(strs[0], ":")
	veth := strings.TrimSpace(strs[1])
	return veth, nil
}

func getContainerFile(srcContainer string, srcPath string, cli *client.Client) (filePath string, fileName string, err error) {
	content, stat, err := cli.CopyFromContainer(context.Background(), srcContainer, srcPath)
	if err != nil {
		logger.Err(err)
		return "", "", err
	}

	// Prepare source copy info.
	srcInfo := archive.CopyInfo{
		Path:       srcPath,
		Exists:     true,
		IsDir:      stat.Mode.IsDir(),
		RebaseName: "",
	}

	preArchive := content
	if len(srcInfo.RebaseName) != 0 {
		_, srcBase := archive.SplitPathDirEntry(srcInfo.Path)
		preArchive = archive.RebaseArchiveEntries(content, srcBase, srcInfo.RebaseName)
	}

	// See comments in the implementation of `archive.CopyTo` for exactly what
	// goes into deciding how and whether the source archive needs to be
	// altered for the correct copy behavior.
	fileDir := ".tmp" + string(os.PathSeparator) + uuid.NewV4().String()
	err = os.MkdirAll(fileDir, 0777)
	if err != nil {
		logger.Err(err)
		return "", "", err
	}

	err = archive.CopyTo(preArchive, srcInfo, fileDir)
	if err != nil {
		logger.Err(err)
		return
	}

	return fileDir, stat.Name, nil
}
