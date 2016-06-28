package docker

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/types"
	"github.com/kdsukhani/container"
)

func TestIsDockerInstalled(t *testing.T) {
	var c container.Container
	c = Docker{}
	if !c.IsDockerInstalled() {
		t.Error("Docker Not Installed")
	}
}

func TestGetImageData(t *testing.T) {
	d := Docker{}
	client, err := d.getClient()
	if err != nil {
		t.Error(err)
	}

	images, err := client.ImageList(context.TODO(), types.ImageListOptions{})
	if len(images) == 0 {
		// TODO: Pull Image
		return
	}

	var c container.Container
	c = d
	_, err = c.GetImageData(images[0].ID)
	if err != nil {
		t.Error(err)
	}
}
