package server

import (
	"fmt"
	"testing"
)

var testClient *ReleaseManager

func TestSplitUpdateAsset(t *testing.T) {
	var err error
	var info *AssetInfo

	if info, err = getAssetInfo("autoupdate-binary-darwin-x86.dmg"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != osDarwin || info.Arch != archX86 {
		t.Fatal("Failed to identify update asset.")
	}

	if info, err = getAssetInfo("autoupdate-binary-darwin-x64.v1"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != osDarwin || info.Arch != archX64 {
		t.Fatal("Failed to identify update asset.")
	}

	if info, err = getAssetInfo("autoupdate-binary-linux-arm"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != osLinux || info.Arch != archARM {
		t.Fatal("Failed to identify update asset.")
	}

	if info, err = getAssetInfo("autoupdate-binary-windows-x86"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != osWindows || info.Arch != archX86 {
		t.Fatal("Failed to identify update asset.")
	}

	if _, err = getAssetInfo("autoupdate-binary-osx-x86"); err == nil {
		t.Fatalf("Should have ignored the release, \"osx\" is not a valid OS value.")
	}
}

func TestNewClient(t *testing.T) {
	testClient = NewReleaseManager("getlantern", "autoupdate-server")
	if testClient == nil {
		t.Fatal("Failed to create new client.")
	}
}

func TestListReleases(t *testing.T) {
	if _, err := testClient.GetReleases(); err != nil {
		t.Fatal(fmt.Errorf("Failed to pull releases: %q", err))
	}
}

func TestUpdateAssetsMap(t *testing.T) {
	if err := testClient.UpdateAssetsMap(); err != nil {
		t.Fatal(fmt.Errorf("Failed to update assets map: %q", err))
	}
}
