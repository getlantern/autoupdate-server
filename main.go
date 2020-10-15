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
	flagRateLimit          = flag.Int("r", 0, "Rate limit. How many updates are allowed to process per second. Defaults to no limit.")
	flagPrivateKey         = flag.String("k", "", "Path to private key.")
	flagLocalAddr          = flag.String("l", ":9999", "Local bind address.")
	flagPublicAddr         = flag.String("p", "http://127.0.0.1:9999/", "Public address.")
	flagGithubOrganization = flag.String("o", "getlantern", "Github organization. For back compatibility to old clients hitting /update endpoint.")
	flagGithubProject      = flag.String("n", "lantern", "Github project name. For back compatibility to old clients hitting /update endpoint.")
	flagRepos              = flag.String("repos", "lantern:getlantern/lantern", "Comma separated mapping of update path to Github repos mapping. The format looks like this 'app1:owner1/repo1,app2:owner2/repo2'")
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

	updateServer := server.NewUpdateServer(*flagPublicAddr, *flagLocalAddr, localPatchesDirectory, *flagRateLimit)
	for _, mapping := range strings.Split(*flagRepos, ",") {
		fatal := func() { log.Fatalf("expect repo string in 'app:owner/repo' format, got '%s'", mapping) }
		pair := strings.Split(mapping, ":")
		if len(pair) != 2 {
			fatal()
		}
		app := pair[0]
		parts := strings.Split(pair[1], "/")
		if len(parts) != 2 {
			fatal()
		}
		updateServer.HandleRepo("/update/"+app, parts[0], parts[1])
	}
	// back compatibility
	updateServer.HandleRepo("/update", *flagGithubOrganization, *flagGithubProject)

	if err := updateServer.ListenAndServe(); err != nil {
		log.Fatalf("ListenAndServe: ", err)
	}
}
