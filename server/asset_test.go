package server

import (
	"fmt"
	"testing"
)

const (
	testAssetURL = `https://github.com/getlantern/autoupdate-server/releases/download/v0.4/autoupdate-binary-darwin-x86.v4`
)

func TestDownloadAsset(t *testing.T) {
	if _, err := downloadAsset(testAssetURL); err != nil {
		t.Fatal(fmt.Errorf("Failed to download asset: %q", err))
	}
}