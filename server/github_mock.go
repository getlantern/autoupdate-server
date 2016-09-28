// +build mock

package server

import (
	"net/http"
	"strconv"
	"time"
)

const mockServerAddr = "127.0.0.1:8885"

type ghMockServer struct {
	mux *http.ServeMux
}

func handleReleasesPage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	page, _ := strconv.Atoi(r.Form.Get("page"))
	if page < 1 {
		page = 1
	}
	if page <= len(releasesPage) {
		w.Write(releasesPage[page-1])
		return
	}
	w.WriteHeader(500)
	return
}

func startMockServer(addr string) (*ghMockServer, error) {
	server := &ghMockServer{
		mux: http.NewServeMux(),
	}

	server.mux.HandleFunc("/", handleReleasesPage)

	go func() {
		http.ListenAndServe(addr, server.mux)
	}()

	time.Sleep(time.Millisecond * 100)

	return server, nil
}

func init() {
	startMockServer(mockServerAddr)
}
