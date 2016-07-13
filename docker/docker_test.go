package docker

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"

	mycontainer "github.com/kdsukhani/container"
	"github.com/kdsukhani/container/logger"
)

var cli *client.Client
var imageId string
var containerId string

func TestIsDockerInstalled(t *testing.T) {
	var c mycontainer.Container
	c = Docker{}
	if !c.IsDockerInstalled() {
		t.Error("Docker Not Installed")
	}
}

func TestGetContainerForProcess(t *testing.T) {
	var c mycontainer.Container
	c = Docker{}

	err := cli.ContainerStart(context.TODO(), containerId, types.ContainerStartOptions{})
	if err != nil {
		logger.Err(err)
		os.Exit(1)
	}

	containerJson, err := cli.ContainerInspect(context.TODO(), containerId)
	if err != nil {
		t.Error(err)
		return
	}

	cId, err := c.GetContainerForProcess(containerJson.State.Pid)
	if err != nil {
		t.Error(err)
		return
	}

	if cId != containerId {
		t.Error("Invalid Container Id")
	}
}

func TestGetUsernameForUid(t *testing.T) {
	var c mycontainer.Container
	c = Docker{}

	err := cli.ContainerStart(context.TODO(), containerId, types.ContainerStartOptions{})
	if err != nil {
		logger.Err(err)
		os.Exit(1)
	}

	username, err := c.GetUsernameForUid(containerId, 0)
	if err != nil {
		t.Error(err)
		return
	}

	if username != "root" {
		t.Error("Invalid User Name")
	}
}

func TestMain(m *testing.M) {
	var err error

	logger.Init(logrus.ErrorLevel)
	cli, err = client.NewClient(client.DefaultDockerHost, "", nil, nil)
	if err != nil {
		os.Exit(1)
	}

	responseReader, err := cli.ImageCreate(context.TODO(), "ubuntu", types.ImageCreateOptions{})
	if err != nil {
		logger.Err(err)
		os.Exit(1)
	}

	responseDecoder := json.NewDecoder(responseReader)
	data := struct {
		Status string `json:"status"`
	}{}

	for err == nil {
		err = responseDecoder.Decode(&data)
		logger.Debug(data)

		if strings.Contains(data.Status, "Digest") {
			break
		}
	}

	if !strings.Contains(data.Status, "Digest") {
		os.Exit(1)
	} else {
		imageId = data.Status[strings.LastIndex(data.Status, ":")+1:]
		logger.Debug(imageId)
	}

	config := container.Config{
		Image: "ubuntu:latest",
		Cmd:   []string{"/bin/bash"},
	}

	resp, err := cli.ContainerCreate(context.TODO(), &config, &container.HostConfig{},
		&network.NetworkingConfig{}, "")
	if err != nil {
		logger.Err(err)
		os.Exit(1)
	}

	containerId = resp.ID

	os.Exit(m.Run())
}
