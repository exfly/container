package image

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/exfly/container/config"
	"github.com/exfly/container/pkg/compress"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func NewImageService(configHome *config.Home) (*ImageService, error) {
	var err error
	ret := &ImageService{
		configHome: configHome,
		imgConfig:  NewImageConfig(configHome),
	}
	ret.imageStore, err = NewImageStore(configHome)
	return ret, err
}

type ImageService struct {
	configHome *config.Home
	imgConfig  *ImageConfig
	imageStore *ImageStore
}

func (s *ImageService) rawImgInTempPath(img *Image) string {
	return fmt.Sprintf("%v/%v", s.configHome.TempPath(), img.ShaHex)
}

func (s *ImageService) downloadImage(rawImg v1.Image, img *Image) error {
	if !img.IsInit() {
		panic(ErrImgNotInit)
	}
	path := s.rawImgInTempPath(img)
	os.Mkdir(path, 0755)
	path += "/package.tar"
	/* Save the image as a tar file */
	if err := crane.SaveLegacy(rawImg, img.ID(), path); err != nil {
		return errors.Wrap(err, "saving tarball")
	}
	return nil
}

func (s *ImageService) untarFile(img *Image) error {
	pathDir := s.rawImgInTempPath(img)
	pathTar := pathDir + "/package.tar"
	if err := compress.Untar(pathTar, pathDir); err != nil {
		return err
	}
	return nil
}

type manifest []struct {
	Config   string
	RepoTags []string
	Layers   []string
}

func (mani manifest) IsValid() error {
	if len(mani) == 0 || len(mani[0].Layers) == 0 {
		return errors.New("Could not find any layers.")
	}
	if len(mani) > 1 {
		return errors.New("I don't know how to handle more than one manifest.")
	}
	return nil
}

func (s *ImageService) GetManifestForImage(img *Image) (manifest, error) {
	manifestPath := s.imgConfig.GetManifestPathForImage(img.ShaHex)
	mani := manifest{}
	if err := parseManifest(manifestPath, &mani); err != nil {
		return manifest{}, err
	}
	return mani, nil
}

func (s *ImageService) processLayerTarballs(imageShaHex string, fullImageHex string) error {
	tmpPathDir := s.configHome.TempPath() + "/" + imageShaHex
	pathManifest := tmpPathDir + "/manifest.json"
	pathConfig := tmpPathDir + "/" + fullImageHex + ".json"

	mani := manifest{}
	parseManifest(pathManifest, &mani)
	if len(mani) == 0 || len(mani[0].Layers) == 0 {
		log.Fatal("Could not find any layers.")
	}
	if len(mani) > 1 {
		log.Fatal("I don't know how to handle more than one manifest.")
	}

	imagesDir := s.configHome.ImagesPath() + "/" + imageShaHex
	if err := os.Mkdir(imagesDir, 0755); err != nil {
		return err
	}
	/* untar the layer files. These become the basis of our container root fs */
	for _, layer := range mani[0].Layers {
		imageLayerDir := imagesDir + "/" + layer[:12] + "/fs"
		log.Infof("Uncompressing layer to: %s \n", imageLayerDir)
		if err := os.MkdirAll(imageLayerDir, 0755); err != nil {
			return err
		}
		srcLayer := tmpPathDir + "/" + layer
		if err := compress.Untar(srcLayer, imageLayerDir); err != nil {
			log.Fatalf("Unable to untar layer file: %s: %v\n", srcLayer, err)
		}
	}
	/* Copy the manifest file for reference later */

	if err := copyFile(pathManifest, s.imgConfig.GetManifestPathForImage(imageShaHex)); err != nil {
		return err
	}
	if err := copyFile(pathConfig, s.imgConfig.GetConfigPathForImage(imageShaHex)); err != nil {
		return err
	}
	return nil
}

func (s *ImageService) deleteTempImageFiles(imageShaHash string) error {
	tmpPath := s.imgConfig.GetTempImagePathForImage(imageShaHash)
	log.Debugf("delete temp image: %v", tmpPath)
	return errors.Wrapf(os.RemoveAll(tmpPath), "Unable to remove temporary image files: %v", tmpPath)
}

func (s *ImageService) GetImageMetadata(img *Image) (ret imageMetadata, err error) {
	content, err := ioutil.ReadFile(s.imgConfig.GetConfigPathForImage(img.ShaHex))
	if err != nil {
		return
	}
	err = json.Unmarshal(content, &ret)
	return
}

func (s *ImageService) GetOrPull(ctx context.Context, img *Image) (*Image, error) {
	if !s.imageStore.IsExistByTag(img) {
		// pull
		rawImg, err := crane.Pull(img.ID())
		if err != nil {
			return nil, err
		}
		manifest, err := rawImg.Manifest()
		if err != nil {
			return nil, err
		}
		fullImageHex := manifest.Config.Digest.Hex
		imageShaHex := fullImageHex[:12]
		log.WithField("image_hash", imageShaHex).Info("Checking if image exists under another name...")
		sameHashImg, isExists, err := s.imageStore.GetImageByHash(imageShaHex)
		if err != nil {
			return nil, err
		}
		retImg := *img
		retImg.ShaHex = imageShaHex
		if isExists {
			log.Infof("The image you requested %v is the same as %v\n", img, sameHashImg)
			err = s.imageStore.StoreImgMetadata(&retImg)
			if err != nil {
				return nil, err
			}
		} else {
			log.Info("Image doesn't exist. downloading...")
			err = s.downloadImage(rawImg, &retImg)
			if err != nil {
				return &retImg, err
			}
			if err = s.untarFile(&retImg); err != nil {
				return nil, err
			}
			if err = s.processLayerTarballs(imageShaHex, fullImageHex); err != nil {
				log.WithError(err).Error("processLayerTarballs error")
				return nil, err
			}
			err = s.imageStore.StoreImgMetadata(&retImg)
			if err = s.deleteTempImageFiles(imageShaHex); err != nil {
				return nil, err
			}
			log.Infof("pull image success: %v", retImg)
			return &retImg, nil
		}
	}
	log.Info("Image already exists. Not downloading.")
	retImg, err := s.imageStore.GetImage(img)
	if err != nil {
		return nil, err
	}
	return retImg, nil
}
