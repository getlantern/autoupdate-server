package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/getlantern/autoupdate-server/server"
	"io/ioutil"
	"log"
	"net/http"
)

var releaseManager *server.ReleaseManager

const (
	patchesDirectory = "./server/patches"
)

type updateHandler struct {
}

func init() {
	// Creating release manager.
	log.Printf("Starting release manager.")
	releaseManager = server.NewReleaseManager(githubNamespace, githubRepo)
	// Updating assets...
	log.Printf("Updating assets...")
	if err := releaseManager.UpdateAssetsMap(); err != nil {
		log.Fatalf("Could not update assets: %q", err)
	}
}

func (u *updateHandler) closeWithStatus(w http.ResponseWriter, status int) {
	log.Printf("Status: %v", status)
	w.WriteHeader(status)
	w.Write([]byte(http.StatusText(status)))
}

func (u *updateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got request")

	var err error
	var res *server.Result

	if r.Method == "POST" {
		defer r.Body.Close()

		b, _ := ioutil.ReadAll(r.Body)

		// TODO: This was only for testing.
		log.Printf("body: %v", string(b))

		var params server.Params
		decoder := json.NewDecoder(bytes.NewBuffer(b))

		if err = decoder.Decode(&params); err != nil {
			u.closeWithStatus(w, http.StatusBadRequest)
			return
		}

		if res, err = releaseManager.CheckForUpdate(&params); err != nil {
			log.Printf("Failed with error: %q", err)
			if err == server.ErrNoUpdateAvailable {
				u.closeWithStatus(w, http.StatusNoContent)
			}
			u.closeWithStatus(w, http.StatusExpectationFailed)
			return
		}

		fmt.Printf("res: %v\n", res)

		var content []byte

		if content, err = json.Marshal(res); err != nil {
			u.closeWithStatus(w, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(content)
		return
	}
	u.closeWithStatus(w, http.StatusNotFound)
	return
}

func main() {

	mux := http.NewServeMux()

	mux.Handle("/update", new(updateHandler))
	mux.Handle("/patches", http.FileServer(http.Dir(patchesDirectory)))

	srv := http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}

	log.Printf("Starting up HTTP server at %s.", listenAddr)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
