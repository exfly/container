package image

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/exfly/container/config"

	log "github.com/sirupsen/logrus"
)

type rawImageEntries struct {
	Tag  string
	Hash string
}
type imagesDB map[string]rawImageEntries

func NewImageStore(configHome *config.Home) (*ImageStore, error) {
	ret := &ImageStore{
		configHome: configHome,
	}
	metaPath := ret.MetadataPath()
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		/* If it doesn't exist create an empty DB */
		if err = ioutil.WriteFile(metaPath, []byte("{}"), 0644); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

type ImageStore struct {
	configHome *config.Home
}

func (i *ImageStore) IsExistByTag(img *Image) bool {
	db, err := i.ParseImagesMetadata()
	if err != nil {
		return false
	}
	for imgM, tagEntry := range *db {
		if imgM == img.Img && tagEntry.Tag == img.Tag {
			return true
		}
	}
	return false
}

func (i *ImageStore) GetImage(img *Image) (*Image, error) {
	db, err := i.ParseImagesMetadata()
	if err != nil {
		return nil, err
	}
	for imgM, tagEntry := range *db {
		if imgM == img.Img && img.Tag == tagEntry.Tag {
			retImg := Image{
				Img:    img.Img,
				Tag:    img.Tag,
				ShaHex: tagEntry.Hash,
			}
			return &retImg, nil
		}
	}
	return nil, ErrNotExists
}

func (i *ImageStore) GetImageByHash(shaHex string) (*Image, bool, error) {
	isExists := false
	db, err := i.ParseImagesMetadata()
	if err != nil {
		return nil, isExists, err
	}
	for imgN, entries := range *db {
		if entries.Hash == shaHex {
			isExists = true
			return &Image{
				Img:    imgN,
				Tag:    entries.Tag,
				ShaHex: entries.Hash,
			}, isExists, nil
		}
	}
	return nil, isExists, nil
}

func (i *ImageStore) IsExistByHash(img *Image, shaHex string) bool {
	db, err := i.ParseImagesMetadata()
	if err != nil {
		return false
	}
	for imgM, tagEntry := range *db {
		if imgM == img.Img && shaHex == tagEntry.Hash {
			return true
		}
	}
	return false
}

func (i *ImageStore) MetadataPath() string {
	imgMetaDataPath := i.configHome.ImagesPath() + "/images.json"
	return imgMetaDataPath
}

func (i *ImageStore) MetadataReader() (*os.File, error) {
	meta, err := os.OpenFile(i.MetadataPath(), os.O_RDONLY, os.ModeAppend)
	return meta, err
}

func (i *ImageStore) MetadataWriter() (*os.File, error) {
	meta, err := os.OpenFile(i.MetadataPath(), os.O_WRONLY, os.ModeAppend)
	return meta, err
}

func (i *ImageStore) ParseImagesMetadata() (*imagesDB, error) {
	meta, err := i.MetadataReader()
	if err != nil {
		return nil, err
	}
	return i.parseImagesMetadata(meta)
}

func (i *ImageStore) parseImagesMetadata(r io.Reader) (*imagesDB, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		log.Errorf("Could not read images DB: %v\n", err)
		return nil, err
	}
	var idb imagesDB
	if err := json.Unmarshal(data, &idb); err != nil {
		log.Errorf("Unable to parse images DB: %v\n", err)
		return nil, err
	}
	return &idb, nil
}

func (i *ImageStore) StoreImgMetadata(img *Image) error {
	db, err := i.ParseImagesMetadata()
	if err != nil {
		return err
	}
	rawDB := *db
	entry := rawImageEntries{}
	if oldEntry, exists := rawDB[img.Img]; exists {
		entry = oldEntry
	}
	entry.Tag = img.Tag
	if !img.IsInit() {
		panic("img not init")
	}
	entry.Hash = img.ShaHex
	rawDB[img.Img] = entry
	metaWriter, err := i.MetadataWriter()
	if err != nil {
		return err
	}
	return i.marshalImageMetadata(&rawDB, metaWriter)
}

func (i *ImageStore) marshalImageMetadata(idb *imagesDB, w io.Writer) error {
	fileBytes, err := json.Marshal(idb)
	if err != nil {
		log.Errorf("Unable to marshall images data: %v\n", err)
		return err
	}
	if _, err := w.Write(fileBytes); err != nil {
		log.Errorf("Unable to save images writer: %v\n", err)
		return err
	}
	return nil
}
