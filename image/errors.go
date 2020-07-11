package image

import (
	"github.com/pkg/errors"
)

var (
	ErrNotExists error = errors.New("hash img not exists")
	ErrImgNotInit error = errors.New("img not init")
)

func IsHashImgNotExists(err error) bool {
	return err == ErrNotExists
}
