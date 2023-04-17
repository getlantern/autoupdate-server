package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/blang/semver"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/getlantern/autoupdate-server/instrument"
	"github.com/getlantern/golog"
	"golang.org/x/time/rate"
)

const (
	githubRefreshTime = 30 * time.Minute
	httpPathPrefix    = "/update"
	appLantern        = "lantern"
)

var (
	v360 = semver.MustParse("3.6.0")
	// Windows XP/2003 or below
	windowsXPMinus                 = semver.MustParse("6.0.0")
	lastLanternVersionForWindowsXP = "5.4.1"
	// 0SX 10.10 Yosemite or below
	osxYosemiteMinus                 = semver.MustParse("15.0.0")
	lastLanternVersionForOSXYosemite = "5.4.1"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var log = golog.LoggerFor("autoupdate-server")

// Initiative type.
type Initiative string

const (
	INITIATIVE_NEVER  Initiative = "never"
	INITIATIVE_AUTO              = "auto"
	INITIATIVE_MANUAL            = "manual"
)

// PatchType represents the type of a binary patch, if any. Only bsdiff is supported
type PatchType string

const (
	PATCHTYPE_BSDIFF PatchType = "bsdiff"
	PATCHTYPE_NONE             = ""
)

// Params represent parameters sent by the go-update client.
type Params struct {
	// protocol version
	Version int `json:"version"`
	// version of the application updating itself
	AppVersion string `json:"app_version"`
	// operating system of target platform
	OS string `json:"-"`
	// hardware architecture of target platform
	Arch string `json:"-"`
	// Semantic version of the OS
	OSVersion string `json:"os_version"`
	// checksum of the binary to replace (used for returning diff patches)
	Checksum string `json:"checksum"`
	// tags for custom update channels
	Tags map[string]string `json:"tags"`
}

// Result represents the answer to be sent to the client.
type Result struct {
	// should the update be applied automatically/manually
	Initiative Initiative `json:"initiative"`
	// url where to download the updated application
	URL string `json:"url"`
	// a URL to a patch to apply
	PatchURL string `json:"patch_url"`
	// the patch format (only bsdiff supported at the moment)
	PatchType PatchType `json:"patch_type"`
	// version of the new application
	Version string `json:"version"`
	// expected checksum of the new application
	Checksum string `json:"checksum"`
	// signature for verifying update authenticity
	Signature string `json:"signature"`
}

// CheckForUpdate receives a *Params message and emits a *Result. If both res
// and err are nil it means no update is available.
func (g *ReleaseManager) CheckForUpdate(p *Params, isLantern bool) (res *Result, err error) {

	// Keep for the future.
	if p.Version < 1 {
		p.Version = 1
	}

	// p must not be nil.
	if p == nil {
		return nil, fmt.Errorf("expecting params")
	}

	if p.Tags != nil {
		// Compatibility with go-check.
		if p.Tags["os"] != "" {
			p.OS = p.Tags["os"]
		}
		if p.Tags["arch"] != "" {
			p.Arch = p.Tags["arch"]
		}
	}

	if p.OS == "" {
		return nil, fmt.Errorf("OS is required")
	}

	// The checksum is optional if the OS is Android
	// since we aren't doing binary diffs
	if p.OS != OS.Android && p.Checksum == "" {
		return nil, fmt.Errorf("checksum must not be nil")
	}

	if p.Arch == "" {
		return nil, fmt.Errorf("arch is required")
	}

	// One APK to support both ARM and ARM64 on Android
	if p.OS == OS.Android && p.Arch == "arm64" {
		p.Arch = Arch.ARM
	}

	appVersion, err := semver.Parse(p.AppVersion)
	if err != nil {
		return nil, fmt.Errorf("bad app version string %v: %v", p.AppVersion, err)
	}

	var update *Asset
	if isLantern {
		if update, err = g.specificLanternVersionToUpgrade(p); err != nil {
			return nil, fmt.Errorf("no upgrade for version %s %s/%s: %v", p.AppVersion, p.OS, p.Arch, err)
		}
	}

	// Looking if there is a newer version for the os/arch.
	if update == nil {
		if update, err = g.getProductUpdate(p.OS, p.Arch); err != nil {
			return nil, fmt.Errorf("could not lookup for updates: %s", err)
		}
	}

	// No update available.
	if update.v.LTE(appVersion) {
		return nil, ErrNoUpdateAvailable
	}

	// A newer version is available!

	// Looking for the asset thay matches the current app checksum.
	var current *Asset
	if current, err = g.lookupAssetWithChecksum(p.OS, p.Arch, p.Checksum); err != nil {
		// No such asset with the given checksum, nothing to compare. Tell the
		// client to download the full binary
		r := &Result{
			Initiative: INITIATIVE_AUTO,
			URL:        update.URL,
			PatchType:  PATCHTYPE_NONE,
			Version:    update.v.String(),
			Checksum:   update.Checksum,
			Signature:  update.Signature,
		}

		return r, nil
	}

	// Generate a binary diff of the two assets.
	var patch *Patch
	if patch, err = generatePatch(current.URL, update.URL); err != nil {
		return nil, fmt.Errorf("unable to generate patch: %q", err)
	}

	// Generate result with the patch URL.
	r := &Result{
		Initiative: INITIATIVE_AUTO,
		URL:        update.URL,
		PatchURL:   patch.File,
		PatchType:  PATCHTYPE_BSDIFF,
		Version:    update.v.String(),
		Checksum:   update.Checksum,
		Signature:  update.Signature,
	}

	return r, nil
}

func (g *ReleaseManager) specificLanternVersionToUpgrade(p *Params) (*Asset, error) {
	var specificVersion string
	if osVersion, err := semver.Parse(p.OSVersion); err == nil {
		if p.OS == "windows" && osVersion.LT(windowsXPMinus) {
			specificVersion = lastLanternVersionForWindowsXP
		} else if p.OS == "darwin" && osVersion.LT(osxYosemiteMinus) {
			specificVersion = lastLanternVersionForOSXYosemite
		}
	}
	if specificVersion != "" {
		return g.lookupAssetWithVersion(p.OS, p.Arch, specificVersion)
	}
	return nil, nil
}

type UpdateServer struct {
	chClose          chan struct{}
	localAddr        string
	mux              *http.ServeMux
	patchesDirectory string
	publicAddr       string
	rateLimit        rate.Limit
	limiter          *rate.Limiter
}

func NewUpdateServer(publicAddr, localAddr, localpatchesDirectory string, rateLimit int) *UpdateServer {
	u := &UpdateServer{
		chClose:          make(chan struct{}),
		localAddr:        localAddr,
		patchesDirectory: localpatchesDirectory,
		publicAddr:       publicAddr,
		rateLimit:        rate.Limit(rateLimit),
	}
	if u.rateLimit == 0 {
		u.rateLimit = rate.Inf
	}
	u.limiter = rate.NewLimiter(u.rateLimit, int(u.rateLimit))
	u.mux = http.NewServeMux()
	u.mux.Handle("/patches/", http.StripPrefix("/patches/", http.FileServer(http.Dir(u.patchesDirectory))))
	return u
}

func (u *UpdateServer) HandleRepo(app, owner, repo string, otelHandler func(next http.Handler) http.Handler) {
	path := httpPathPrefix
	if app != "" {
		path = path + "/" + app
	} else {
		app = appLantern
	}
	log.Debugf("HTTP path %q maps to repo %s/%s", path, owner, repo)
	u.mux.Handle(path, otelHandler(u.handlerFor(app, owner, repo)))
}

func (u *UpdateServer) handlerFor(app, owner, repo string) http.Handler {
	releaseManager := NewReleaseManager(owner, repo)
	// Getting assets...
	if err := releaseManager.UpdateAssetsMap(); err != nil {
		// In this case we will not be able to continue.
		log.Fatal(err)
	}
	// Setting a goroutine for pulling updates periodically
	go u.backgroundUpdate(releaseManager)

	//u.configureHoneycomb(context.Background())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := r.Context()
		userID := userIDFromRequest(c)
		_, span := instrument.Tracer.Start(c, "autoupdate_download")
		defer span.End()
		span.SetAttributes(attribute.Int64("userId", userID))

		var err error
		var res *Result

		recordError := func(w http.ResponseWriter, statusCode int, msg string, args ...any) {
			msg = fmt.Sprintf("%s", args...)
			log.Error(msg)
			span.RecordError(errors.New(msg))
			span.SetStatus(codes.Error, msg)
			closeWithStatus(w, statusCode)
		}

		if r.Method != "POST" {
			recordError(w, http.StatusNotFound, "Invalid HTTP method")
			return
		}
		defer r.Body.Close()

		var params Params
		decoder := json.NewDecoder(r.Body)

		if err = decoder.Decode(&params); err != nil {
			recordError(w, http.StatusBadRequest, "JSON decode error")
			return
		}

		span.SetAttributes(attribute.String("appVersion", params.AppVersion))
		span.SetAttributes(attribute.String("arch", params.Arch))
		span.SetAttributes(attribute.String("platform", params.OS))

		isLantern := app == appLantern
		if res, err = releaseManager.CheckForUpdate(&params, isLantern); err != nil {
			if err == ErrNoUpdateAvailable {
				log.Debugf("No update available for: %s/%s/%s", app, params.OS, params.AppVersion)
				closeWithStatus(w, http.StatusNoContent)
				return
			}
			recordError(w, http.StatusExpectationFailed, "CheckForUpdate failed. App/OS/Version: %s/%s/%s, error: %q", app, params.OS, params.AppVersion, err)
			return
		}

		if !u.limiter.Allow() {
			recordError(w, http.StatusNoContent, "Update skipped because current rate limit (%.0f/s) is hit.", u.rateLimit)
			return
		}

		if isLantern && params.OS == "darwin" {
			currentVersion, err := semver.Parse(params.AppVersion)
			if err != nil {
				recordError(w, http.StatusNoContent, "Failed to parse version (%q): %v", params.AppVersion, err)
				return
			}
			if currentVersion.LT(v360) {
				recordError(w, http.StatusNoContent, "Got %q version %q on OSX, but we cannot update it. Skipped", app, params.AppVersion)
				return
			}
		}

		log.Debugf("Got query from client %q/%q/%q, resolved to upgrade to %q using %q strategy.", app, params.AppVersion, params.OS, res.Version, res.PatchType)

		if res.PatchURL != "" {
			res.PatchURL = u.publicAddr + res.PatchURL
		}

		var content []byte
		if content, err = json.Marshal(res); err != nil {
			log.Debugf("Failed to marshal response: %s", err)
			recordError(w, http.StatusInternalServerError, "Failed to marshal response: %w", err)
			return
		}

		nonce, _ := strconv.ParseInt(r.Header.Get("X-Message-Nonce"), 10, 64) // Can be zero for old clients.
		hash := sha256.Sum256(append(content, []byte(fmt.Sprintf("%d", nonce))...))
		messageAuth, err := Sign(hash[:])
		if err != nil {
			recordError(w, http.StatusInternalServerError, "Could not sign body: %w", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Message-Signature", hex.EncodeToString(messageAuth))

		if _, err := w.Write(content); err != nil {
			log.Debugf("Unable to write response: %s", err)
		}
	})
}

func (u *UpdateServer) ListenAndServe() error {
	srv := http.Server{
		Addr:    u.localAddr,
		Handler: u.mux,
	}
	log.Debugf("Starting up HTTP server at %s.", u.localAddr)
	go func() {
		<-u.chClose
		log.Debugf("Closing HTTP server at %s.", u.localAddr)
		_ = srv.Close()
	}()
	return srv.ListenAndServe()
}

func (u *UpdateServer) Close() {
	close(u.chClose)
}

// backgroundUpdate periodically looks for releases.
func (u *UpdateServer) backgroundUpdate(releaseManager *ReleaseManager) {
	tk := time.NewTicker(githubRefreshTime)
	for {
		select {
		case <-tk.C:
			log.Debug("Updating assets...")
			if err := releaseManager.UpdateAssetsMap(); err != nil {
				log.Debugf("updateAssets: %s", err)
			}
		case <-u.chClose:
			return
		}
	}
}

func closeWithStatus(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
	if status == http.StatusNoContent {
		return
	}
	if _, err := w.Write([]byte(http.StatusText(status))); err != nil {
		log.Debugf("Unable to write status %d: %v", status, err)
	}
}
