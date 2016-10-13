package server

import (
	"compress/bzip2"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"
)

const (
	assetsDirectory = "assets/"
)

func init() {
	err := os.MkdirAll(assetsDirectory, os.ModeDir|0700)
	if err != nil {
		log.Fatalf("Could not create directory for storing assets: %q", err)
	}
}

// downloadAsset downloads the body of the given URL and stores it into
// $ASSETS_DIRECTORY/$BASENAME.SHA256_SUM($URL)
func downloadAsset(uri string) (localfile string, err error) {
	basename := path.Base(uri)
	fileExt := path.Ext(basename)

	// The sha256 hash uses 64 chars, we'll append this hash to the name. The name
	// doesn't matter that much so we'll just use 60 chars from it.
	if len(basename) > 60 {
		basename = basename[:60]
	}

	localfile = assetsDirectory + fmt.Sprintf("%s.%x", basename, sha256.Sum256([]byte(uri)))

	if !fileExists(localfile) {
		var body io.Reader
		var res *http.Response

		c := http.Client{Timeout: time.Second * 30}
		if res, err = c.Get(uri); err != nil {
			return "", err
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Expecting 200 OK, got: %s", res.Status)
		}

		var fp *os.File

		if fp, err = os.Create(localfile); err != nil {
			return "", err
		}
		defer fp.Close()

		if fileExt == ".bz2" {
			body = bzip2.NewReader(res.Body)
		} else {
			body = res.Body
		}

		if _, err = io.Copy(fp, body); err != nil {
			return "", err
		}

	}

	return localfile, nil
}
