package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHashImgNotExists(t *testing.T) {
	assert.True(t, IsHashImgNotExists(ErrNotExists))
}
