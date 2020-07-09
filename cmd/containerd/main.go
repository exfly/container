package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"runtime"

	"github.com/exfly/container/config"
	"github.com/exfly/container/container"
	"github.com/exfly/container/image"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func main() {
	log.SetReportCaller(true)
	log.SetLevel(log.TraceLevel)
	var nameRegex = regexp.MustCompile(fmt.Sprintf(`.*%v/`, "container"))
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return nameRegex.ReplaceAllString(f.Function, ""), fmt.Sprintf("%s:%d", nameRegex.ReplaceAllString(f.File, ""), f.Line)
		},
	})

	/* We chroot and write to privileged directories. We need to be root */
	if os.Geteuid() != 0 {
		log.Fatal("You need root privileges to run this program.")
	}
	configHome := config.NewHome("/home/vagrant/containerd")
	configHome.InitDirs()
	imageConfig := image.NewImageConfig(configHome)
	// init bridge

	imgSrv, err := image.NewImageService(configHome)
	if err != nil {
		panic(err)
	}
	containerSrv := container.NewContainerService(
		configHome,
		imageConfig,
		imgSrv,
	)
	ctx := context.TODO()
	ops := opts{
		configHome:   configHome,
		imgConf:      imageConfig,
		imgSrv:       imgSrv,
		containerSrv: containerSrv,
	}
	switch os.Args[1] {
	case "run":
		fs := flag.FlagSet{}
		fs.ParseErrorsWhitelist.UnknownFlags = true

		if err := fs.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing: ", err)
		}

		img := fs.Args()[0]
		if err := runCmd(ctx, img, fs.Args()[1:], ops); err != nil {
			panic(err)
		}
	case "images":
	case "child-mode":
		log.Info("child-mode")
		fs := flag.FlagSet{}
		if err := fs.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing: ", err)
		}
		containerID := fs.Args()[0]
		if err := runChildMode(ctx, containerID, fs.Args()[1:], ops); err != nil {
			panic(err)
		}
	}
}

type opts struct {
	configHome   *config.Home
	imgConf      *image.ImageConfig
	imgSrv       *image.ImageService
	containerSrv *container.ContainerService
}

func runChildMode(ctx context.Context, containerID string, args []string, ops opts) error {
	return ops.containerSrv.RunByID(ctx, containerID, args)
}

func runCmd(ctx context.Context, rawImg string, args []string, ops opts) error {
	img, err := image.NewImage(rawImg)
	if err != nil {
		return err
	}
	pulledImg, err := ops.imgSrv.GetOrPull(ctx, img)
	if err != nil {
		return err
	}
	log.Infof("imges: %v", pulledImg)
	containerInstance := container.NewContainer(pulledImg, nil)
	if err = ops.containerSrv.Run(ctx, containerInstance, args); err != nil {
		return errors.Wrap(err, "run container error")
	}
	return nil
}
