package main

import (
	"flag"
	"os"
	"time"

	"github.com/getlantern/autoupdate-server/server"
	"github.com/getlantern/golog"
)

var (
	flagRolloutRate        = flag.String("r", "1.0", "Rollout rate [0.0, 1.0]")
	flagPrivateKey         = flag.String("k", "", "Path to private key.")
	flagLocalAddr          = flag.String("l", ":9999", "Local bind address.")
	flagPublicAddr         = flag.String("p", "http://127.0.0.1:9999/", "Public address.")
	flagGithubOrganization = flag.String("o", "getlantern", "Github organization.")
	flagGithubProject      = flag.String("n", "lantern", "Github project name.")
	flagHelp               = flag.Bool("h", false, "Shows help.")
)

var (
	log            = golog.LoggerFor("autoupdate-server")
	releaseManager *server.ReleaseManager
)

// updateAssets checks for new assets released on the github releases page.
func updateAssets() error {
	log.Debug("Updating assets...")
	if err := releaseManager.UpdateAssetsMap(); err != nil {
		return err
	}
	return nil
}

// backgroundUpdate periodically looks for releases.
func backgroundUpdate() {
	for {
		time.Sleep(githubRefreshTime)
		// Updating assets...
		if err := updateAssets(); err != nil {
			log.Debugf("updateAssets: %s", err)
		}
	}
}

func main() {

	// Parsing flags
	flag.Parse()

	if *flagHelp || *flagPrivateKey == "" {
		flag.Usage()
		os.Exit(0)
	}

	server.SetPrivateKey(*flagPrivateKey)

	// Creating release manager.
	log.Debug("Starting release manager.")
	releaseManager = server.NewReleaseManager(*flagGithubOrganization, *flagGithubProject)
	// Getting assets...
	if err := updateAssets(); err != nil {
		// In this case we will not be able to continue.
		log.Fatal(err)
	}

	// Setting a goroutine for pulling updates periodically
	go backgroundUpdate()

	updateServer := &server.UpdateServer{
		ReleaseManager:   releaseManager,
		PublicAddr:       *flagPublicAddr,
		LocalAddr:        *flagLocalAddr,
		RolloutRate:      *flagRolloutRate,
		PatchesDirectory: localPatchesDirectory,
	}

	if err := updateServer.ListenAndServe(); err != nil {
		log.Fatalf("ListenAndServe: ", err)
	}
}
