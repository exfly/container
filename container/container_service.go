package container

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/exfly/container/config"
	"github.com/exfly/container/image"
	"github.com/exfly/container/pkg/dirs"
	pkgdirs "github.com/exfly/container/pkg/dirs"
	"github.com/exfly/container/pkg/file"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ContainerService struct {
	configHome *config.Home
	imgConf    *image.ImageConfig
	imgSrv     *image.ImageService
}

func NewContainerService(configHome *config.Home, imgConf *image.ImageConfig, imgSrv *image.ImageService) *ContainerService {
	return &ContainerService{
		configHome: configHome,
		imgConf:    imgConf,
		imgSrv:     imgSrv,
	}
}

func (c *ContainerService) GetContainerHome(container *Container) string {
	// TODO: new container config
	return c.GetContainerHomeByID(*container.ContainerID)
}

func (c *ContainerService) GetContainerHomeByID(containerID string) string {
	// TODO: new container config
	return c.configHome.ContainersPath() + "/" + containerID
}

func (c *ContainerService) GetContainerMetadataPath(container *Container) string {
	return c.GetContainerHome(container) + "/runtime.json"
}

func (c *ContainerService) GetContainerMetadataPathByID(containerID string) string {
	return c.GetContainerHomeByID(containerID) + "/runtime.json"
}

func (c *ContainerService) GetContainerFSHome(container *Container) string {
	// TODO: new container config
	return c.GetContainerHome(container) + "/fs"
}

func (c *ContainerService) createContainerDir(container *Container) error {
	containerHome := c.GetContainerHome(container)
	containerDirs := []string{containerHome + "/fs", containerHome + "/fs/mnt", containerHome + "/fs/upperdir", containerHome + "/fs/workdir"}
	if err := dirs.CreateDirsIfDontExist(containerDirs); err != nil {
		return errors.Wrapf(err, "Unable to create required directories: %v\n", err)
	}
	return nil
}

func (c *ContainerService) unmountOverlayFileSystem(container *Container) error {
	mountedPath := c.configHome.ContainersPath() + "/" + *container.ContainerID + "/fs/mnt"
	if err := syscall.Unmount(mountedPath, 0); err != nil {
		log.Fatalf("Uable to mount container file system: %v at %s", err, mountedPath)
		return errors.Wrapf(err, "Uable to mount container file system: %s", mountedPath)
	}
	return nil
}

func (c *ContainerService) mountOverlayFileSystem(container *Container) error {
	var srcLayers []string
	mani, err := c.imgSrv.GetManifestForImage(container.Image)
	if err != nil {
		return err
	}
	if mani.IsValid() != nil {
		return err
	}

	imageBasePath := c.imgConf.GetBasePathForImage(container.Image.ShaHex)
	for _, layer := range mani[0].Layers {
		srcLayers = append([]string{imageBasePath + "/" + layer[:12] + "/fs"}, srcLayers...)
		//srcLayers = append(srcLayers, imageBasePath + "/" + layer[:12] + "/fs")
	}

	contFSHome := c.GetContainerFSHome(container)
	log.WithField("p", contFSHome).Debug("container_fs_home")
	mntOptions := "lowerdir=" + strings.Join(srcLayers, ":") + ",upperdir=" + contFSHome + "/upperdir,workdir=" + contFSHome + "/workdir"
	log.Infof("mountOverlayFS: %v", mntOptions)
	if err := syscall.Mount("none", contFSHome+"/mnt", "overlay", 0, mntOptions); err != nil {
		return errors.Errorf("Mount failed: %v\n", err)
	}
	return nil
}

func (c *ContainerService) marshalContainer(container *Container) error {
	marshalTo := c.GetContainerMetadataPath(container)
	content, err := json.Marshal(container)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(marshalTo, content, 0644)
}

func (c *ContainerService) unmarshalContainer(containerID string) (*Container, error) {
	unmarshalFrom := c.GetContainerMetadataPathByID(containerID)
	content, err := ioutil.ReadFile(unmarshalFrom)
	if err != nil {
		return nil, err
	}
	var ret Container
	err = json.Unmarshal(content, &ret)
	return &ret, err
}

func (c *ContainerService) prepareAndExecuteContainer(ctx context.Context, container *Container, args []string) error {
	if err := c.marshalContainer(container); err != nil {
		return err
	}
	args = append([]string{"child-mode", *container.ContainerID}, args...)
	log.Infof("CMD: %v", args)
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC,
	}
	return cmd.Run()
}

func (c *ContainerService) copyNameserverConfig(container *Container) error {
	resolvFilePaths := []string{
		"/var/run/systemd/resolve/resolv.conf",
		"/etc/gockerresolv.conf",
		"/etc/resolv.conf",
	}
	for _, resolvFilePath := range resolvFilePaths {
		if _, err := os.Stat(resolvFilePath); os.IsNotExist(err) {
			continue
		} else {
			return file.CopyFile(
				resolvFilePath,
				c.GetContainerFSHome(container)+"/mnt/etc/resolv.conf",
			)
		}
	}
	return nil
}

func (c *ContainerService) RunByID(ctx context.Context, containerID string, args []string) error {
	container, err := c.unmarshalContainer(containerID)
	if err != nil {
		return err
	}
	log.Debug(spew.Sdump(container))
	imgMetadata, err := c.imgSrv.GetImageMetadata(container.Image)
	if err != nil {
		return err
	}
	mntPath := c.GetContainerFSHome(container) + "/mnt"

	rawCmd := args[0]
	rawCmdArgs := args[1:]
	log.Infof("CMD: %v %v", rawCmd, rawCmdArgs)
	cmd := exec.Command(rawCmd, rawCmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = imgMetadata.Config.Env

	if err = syscall.Sethostname([]byte(containerID)); err != nil {
		return err
	}
	// createCGroup
	// configCGroup
	if err = c.copyNameserverConfig(container); err != nil {
		return errors.Wrap(err, "copy nameserver config")
	}
	if err = syscall.Chroot(mntPath); err != nil {
		return errors.Wrap(err, "chroot")
	}
	if err = os.Chdir("/"); err != nil {
		return errors.Wrap(err, "chdir")
	}
	if err = pkgdirs.CreateDirsIfDontExist([]string{"/proc", "/sys", "/tmp", "/dev"}); err != nil {
		return errors.Wrap(err, "create proc sys")
	}
	if err = syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return errors.Wrap(err, "mount proc")
	}
	if err = syscall.Mount("tmpfs", "/tmp", "tmpfs", 0, ""); err != nil {
		return errors.Wrap(err, "mount tmp")
	}
	if err = syscall.Mount("tmpfs", "/dev", "tmpfs", 0, ""); err != nil {
		return errors.Wrap(err, "mount dev")
	}
	if err = pkgdirs.CreateDirsIfDontExist([]string{"/dev/pts"}); err != nil {
		return errors.Wrap(err, "create /dev/pts")
	}
	if err = syscall.Mount("devpts", "/dev/pts", "devpts", 0, ""); err != nil {
		return errors.Wrap(err, "mount /dev/pts")
	}
	if err = syscall.Mount("sysfs", "/sys", "sysfs", 0, ""); err != nil {
		return errors.Wrap(err, "mount /sys")
	}
	if err = cmd.Run(); err != nil {
		return errors.Wrap(err, "run")
	}
	if err = (syscall.Unmount("/dev/pts", 0)); err != nil {
		return err
	}
	if err = (syscall.Unmount("/dev", 0)); err != nil {
		return err
	}
	if err = (syscall.Unmount("/sys", 0)); err != nil {
		return err
	}
	if err = (syscall.Unmount("/proc", 0)); err != nil {
		return err
	}
	if err = (syscall.Unmount("/tmp", 0)); err != nil {
		return err
	}
	return nil
}

func (c *ContainerService) Run(ctx context.Context, container *Container, args []string) error {
	if err := c.createContainerDir(container); err != nil {
		return err
	}
	if err := c.mountOverlayFileSystem(container); err != nil {
		return err
	}
	if err := c.prepareAndExecuteContainer(ctx, container, args); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("clean contaier?")
	_, _ = reader.ReadString('\n')
	if err := c.unmountOverlayFileSystem(container); err != nil {
		return err
	}
	if err := os.RemoveAll(c.GetContainerHome(container)); err != nil {
		return err
	}
	log.Info("finish")
	return nil
}
