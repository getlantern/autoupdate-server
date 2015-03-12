package server

import (
	"fmt"
)

type Initiative string

const (
	INITIATIVE_NEVER  Initiative = "never"
	INITIATIVE_AUTO              = "auto"
	INITIATIVE_MANUAL            = "manual"
)

// The type of a binary patch, if any. Only bsdiff is supported
type PatchType string

const (
	PATCHTYPE_BSDIFF PatchType = "bsdiff"
	PATCHTYPE_NONE             = ""
)

type Params struct {
	// protocol version
	Version int `json:"version"`
	// identifier of the application to update
	//AppId string `json:"app_id"`

	// version of the application updating itself
	AppVersion string `json:"app_version"`
	// operating system of target platform
	OS string `json:"-"`
	// hardware architecture of target platform
	Arch string `json:"-"`
	// application-level user identifier
	//UserId string `json:"user_id"`
	// checksum of the binary to replace (used for returning diff patches)
	Checksum string `json:"checksum"`
	// release channel (empty string means 'stable')
	//Channel string `json:"-"`
	// tags for custom update channels
	//Tags map[string]string `json:"tags"`
}

type Result struct {
	// should the update be applied automatically/manually
	Initiative Initiative `json:"initiative"`
	// url where to download the updated application
	Url string `json:"url"`
	// a URL to a patch to apply
	PatchUrl string `json:"patch_url"`
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
func (g *ReleaseManager) CheckForUpdate(p *Params) (res *Result, err error) {

	// Keep for the future.
	if p.Version < 1 {
		p.Version = 1
	}

	// p must not be nil.
	if p == nil {
		return nil, fmt.Errorf("Expecting params.")
	}

	if !isVersionTag(p.AppVersion) {
		return nil, fmt.Errorf("Expecting a version tag of the form vX.Y.Z.")
	}

	if p.Checksum == "" {
		return nil, fmt.Errorf("Checksum must not be nil.")
	}

	if p.OS == "" {
		return nil, fmt.Errorf("OS is required.")
	}

	if p.Arch == "" {
		return nil, fmt.Errorf("Arch is required.")
	}

	// Looking for the asset thay matches the current app checksum.
	var current *Asset
	if current, err = g.lookupAssetWithChecksum(current.OS, current.Arch, p.Checksum); err != nil {
		return nil, ErrNoSuchAsset
	}

	// Looking if there is a newer version for the os/arch.
	var update *Asset
	if update, err = g.getProductUpdate(current.OS, current.Arch); err != nil {
		return nil, fmt.Errorf("Could not lookup for updates.")
	}

	// No update available.
	if VersionCompare(p.AppVersion, update.v) != Higher {
		return nil, ErrNoUpdateAvailable
	}

	// A newer version is available!

	// Generate a binary diff of the two assets.
	var patch *Patch
	if patch, err = GeneratePatch(current.URL, update.URL); err != nil {
		return nil, fmt.Errorf("Unable to generate patch: %q", err)
	}

	// Generate result.
	r := &Result{
		Initiative: INITIATIVE_AUTO,
		Url:        assetUrl(update.URL),
		PatchUrl:   assetUrl(patch.File),
		PatchType:  PATCHTYPE_BSDIFF,
		Version:    update.v,
		Checksum:   update.Checksum,
		Signature:  update.Signature,
	}

	return r, nil
}
