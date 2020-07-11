package config

import (
	pkgdirs "github.com/exfly/container/pkg/dirs"
)

func NewHome(b string) *Home {
	ret := &Home{
		homePath: b,
	}
	return ret
}

type Home struct {
	homePath string
}

func (h *Home) HomePath() string {
	return h.homePath
}

func (h *Home) TempPath() string {
	return h.HomePath() + "/tmp"
}

func (h *Home) ImagesPath() string {
	return h.HomePath() + "/images"
}

func (h *Home) ContainersPath() string {
	return h.HomePath() + "/containers"
}

func (h *Home) NetNsPath() string {
	return h.HomePath() + "/net-ns"
}

func (h *Home) InitDirs() (err error) {
	dirs := []string{h.HomePath(), h.TempPath(), h.ImagesPath(), h.ImagesPath(), h.NetNsPath()}
	return pkgdirs.CreateDirsIfDontExist(dirs)
}
