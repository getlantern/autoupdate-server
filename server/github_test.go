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
	if info.OS != OS.Darwin || info.Arch != Arch.X86 {
		t.Fatal("Failed to identify update asset.")
	}

	if info, err = getAssetInfo("autoupdate-binary-darwin-x64.v1"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != OS.Darwin || info.Arch != Arch.X64 {
		t.Fatal("Failed to identify update asset.")
	}

	if info, err = getAssetInfo("autoupdate-binary-linux-arm"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != OS.Linux || info.Arch != Arch.ARM {
		t.Fatal("Failed to identify update asset.")
	}

	if info, err = getAssetInfo("autoupdate-binary-windows-x86"); err != nil {
		t.Fatal(fmt.Errorf("Failed to get asset info: %q", err))
	}
	if info.OS != OS.Windows || info.Arch != Arch.X86 {
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
	if testClient.updateAssetsMap == nil {
		t.Fatal("Assets map should not be nil at this point.")
	}
	if testClient.latestAssetsMap == nil {
		t.Fatal("Assets map should not be nil at this point.")
	}
}

func TestDownloadLowestVersionAndUpgradeIt(t *testing.T) {
	// We can use the updateAssetsMap to look for the lowest version.
}
