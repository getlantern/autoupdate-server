package server

import (
	"fmt"
	"testing"
)

const (
	testAssetURL = `https://github.com/getlantern/autoupdate/releases/download/2.0.0-beta3/update_darwin_amd64`
)

func TestDownloadAsset(t *testing.T) {
	s, err := downloadAsset(testAssetURL)
	if err != nil {
		t.Fatal(fmt.Errorf("Failed to download asset: %q", err))
	}
	if s != "assets/update_darwin_amd64.6f3d15772b490fedce235ae74484a8eaa87fe329eda791c824af714398eb71d3" {
		t.Fatal("Unexpected signature.")
	}
}
