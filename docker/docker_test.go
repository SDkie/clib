package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/container"
	"github.com/docker/engine-api/types/network"

	mycontainer "github.com/kdsukhani/container"
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

func TestMain(m *testing.M) {
	var err error
	cli, err = client.NewClient(client.DefaultDockerHost, "", nil, nil)
	if err != nil {
		os.Exit(1)
	}

	responseReader, err := cli.ImageCreate(context.TODO(), "ubuntu", types.ImageCreateOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	responseDecoder := json.NewDecoder(responseReader)
	data := struct {
		Status string `json:"status"`
	}{}

	for err == nil {
		err = responseDecoder.Decode(&data)
		fmt.Println(data)

		if strings.Contains(data.Status, "Digest") {
			break
		}
	}

	if !strings.Contains(data.Status, "Digest") {
		os.Exit(1)
	} else {
		imageId = data.Status[strings.LastIndex(data.Status, ":")+1:]
		fmt.Println(imageId)
	}

	config := container.Config{
		Image: "ubuntu:latest",
	}

	resp, err := cli.ContainerCreate(context.TODO(), &config, &container.HostConfig{},
		&network.NetworkingConfig{}, "")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(resp)
	containerId = resp.ID

	os.Exit(m.Run())
}
