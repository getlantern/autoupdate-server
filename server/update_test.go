package server

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/getlantern/go-update"
	"github.com/getlantern/go-update/check"
)

const (
	localAddr  = "127.0.0.1:1123"
	publicAddr = localAddr
)

func init() {
	SetPrivateKey("../_resources/example-keys/private.key")
}

func TestReachServer(t *testing.T) {
	updateServer := NewUpdateServer(publicAddr, localAddr, ".", 0)

	updateServer.HandleRepo("", "getlantern", "lantern", func(next http.Handler) http.Handler {
		return next
	})
	updateServer.HandleRepo("lantern", "getlantern", "lantern", func(next http.Handler) http.Handler {
		return next
	})

	go updateServer.ListenAndServe()
	defer updateServer.Close()

	publicKey, err := os.ReadFile("../_resources/example-keys/public.pub")
	if err != nil {
		t.Fatalf("Failed to open public key: %v", err)
	}

	param := check.Params{
		AppVersion: "3.7.1",
	}

	up := update.New().ApplyPatch(update.PATCHTYPE_BSDIFF)

	if _, err = up.VerifySignatureWithPEM(publicKey); err != nil {
		t.Fatal("VerifySignatureWithPEM", err)
	}

	res, err := param.CheckForUpdate(fmt.Sprintf("http://%s/update", localAddr), up)
	if err != nil {
		t.Fatalf("CheckForUpdate: %v", err)
	}

	if res.Url == "" {
		t.Fatal("Expecting some URL.")
	}

	res, err = param.CheckForUpdate(fmt.Sprintf("http://%s/update/lantern", localAddr), up)
	if err != nil {
		t.Fatalf("CheckForUpdate: %v", err)
	}

	if res.Url == "" {
		t.Fatal("Expecting some URL.")
	}
}
