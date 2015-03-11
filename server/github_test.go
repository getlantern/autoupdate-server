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
	r, e := testClient.GetReleases()
	fmt.Printf("r: %v, e: %v\n", r, e)
}
