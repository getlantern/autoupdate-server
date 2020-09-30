package main

import (
	"flag"
	"os"
	"strings"

	"github.com/getlantern/autoupdate-server/server"
	"github.com/getlantern/golog"
)

const (
	localPatchesDirectory = "./patches/"
)

var (
	flagRolloutRate        = flag.Float64("r", 1.0, "Rollout rate [0.0, 1.0]")
	flagPrivateKey         = flag.String("k", "", "Path to private key.")
	flagLocalAddr          = flag.String("l", ":9999", "Local bind address.")
	flagPublicAddr         = flag.String("p", "http://127.0.0.1:9999/", "Public address.")
	flagGithubOrganization = flag.String("o", "getlantern", "Github organization. For back compatibility to old clients hitting /update endpoint.")
	flagGithubProject      = flag.String("n", "lantern", "Github project name. For back compatibility to old clients hitting /update endpoint.")
	flagRepos              = flag.String("repos", "getlantern/lantern", "Comma separated Github repos in <owner/repo> format.")
	flagHelp               = flag.Bool("h", false, "Shows help.")
)

var (
	log = golog.LoggerFor("autoupdate-server")
)

func main() {

	// Parsing flags
	flag.Parse()

	if *flagHelp || *flagPrivateKey == "" {
		flag.Usage()
		os.Exit(0)
	}

	server.SetPrivateKey(*flagPrivateKey)

	updateServer := server.NewUpdateServer(*flagPublicAddr, *flagLocalAddr, localPatchesDirectory, *flagRolloutRate)
	for _, repo := range strings.Split(*flagRepos, ",") {
		parts := strings.Split(repo, "/")
		if len(parts) != 2 {
			log.Fatalf("expect repo string in <owner/repo> format, got '%s'", repo)
		}
		updateServer.HandleRepo("/update/"+repo, parts[0], parts[1])
	}
	updateServer.HandleRepo("/update", *flagGithubOrganization, *flagGithubProject)

	if err := updateServer.ListenAndServe(); err != nil {
		log.Fatalf("ListenAndServe: ", err)
	}
}
