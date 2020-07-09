package image

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func ImageNameAndTag(src string) (string, string) {
	s := strings.Split(src, ":")
	var img, tag string
	if len(s) > 1 {
		img, tag = s[0], s[1]
	} else {
		img = s[0]
		tag = "latest"
	}
	return img, tag
}

func parseManifest(manifestPath string, mani *manifest) error {
	data, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, mani); err != nil {
		return err
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
