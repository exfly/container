package image

import "github.com/exfly/container/config"

type imageMetadataDetails struct {
	Env []string `json:"Env"`
	Cmd []string `json:"Cmd"`
}
type imageMetadata struct {
	Config imageMetadataDetails `json:"config"`
}

func NewImageConfig(ch *config.Home) *ImageConfig {
	return &ImageConfig{
		configHome: ch,
	}
}

type ImageConfig struct {
	configHome *config.Home
}

func (i *ImageConfig) GetTempImagePathForImage(imageShaHex string) string {
	return i.configHome.TempPath() + "/" + imageShaHex
}

func (i *ImageConfig) GetBasePathForImage(imageShaHex string) string {
	return i.configHome.ImagesPath() + "/" + imageShaHex
}

func (i *ImageConfig) GetManifestPathForImage(imageShaHex string) string {
	return i.GetBasePathForImage(imageShaHex) + "/manifest.json"
}

func (i *ImageConfig) GetConfigPathForImage(imageShaHex string) string {
	return i.GetBasePathForImage(imageShaHex) + "/" + imageShaHex + ".json"
}
