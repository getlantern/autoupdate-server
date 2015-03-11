package server

import (
	"fmt"
	"testing"
)

var testClient *ReleaseManager

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
