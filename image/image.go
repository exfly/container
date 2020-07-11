package image

import "strings"

func NewImage(s string) (*Image, error) {
	img, tag := ImageNameAndTag(s)
	return &Image{
		Img: img,
		Tag: tag,
	}, nil
}

type Image struct {
	Img    string `json:"img,omitempty"`
	Tag    string `json:"tag,omitempty"`
	ShaHex string `json:"sha_hex,omitempty"`
}

func (i Image) IsInit() bool {
	return i.ShaHex != ""
}

func (i Image) ID() string {
	return strings.Join([]string{i.Img, i.Tag}, ":")
}

func (i Image) String() string {
	return strings.Join([]string{i.Img, i.Tag, i.ShaHex}, ":")
}
