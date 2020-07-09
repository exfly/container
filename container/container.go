package container

import "github.com/exfly/container/image"

func NewContainer(img *image.Image, id *string) *Container {
	if id == nil {
		tmpID := CreateContainerID()
		id = &tmpID
	}
	return &Container{
		ContainerID: id,
		Image:       img,
	}
}

type Container struct {
	ContainerID *string      `json:"container_id,omitempty"`
	Image       *image.Image `json:"image,omitempty"`

	Mem  int      `json:"mem,omitempty"`
	Swap int      `json:"swap,omitempty"`
	Pids int      `json:"pids,omitempty"`
	Cpus float64  `json:"cpus,omitempty"`
	Src  string   `json:"src,omitempty"`
	Args []string `json:"args,omitempty"`
}
