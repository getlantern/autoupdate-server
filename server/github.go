package server

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"sync"

	"github.com/blang/semver"
	"github.com/google/go-github/github"
)

var (
	updateAssetRe = regexp.MustCompile(`^update_(darwin|windows|linux|android)_(arm|386|amd64)\.?.*$`)

	emptyVersion semver.Version
)

// Arch holds architecture names.
var Arch = struct {
	X64 string
	X86 string
	ARM string
}{
	"amd64",
	"386",
	"arm",
}

// OS holds operating system names.
var OS = struct {
	Windows string
	Linux   string
	Darwin  string
	Android string
}{
	"windows",
	"linux",
	"darwin",
	"android",
}

// Release struct represents a single github release.
type Release struct {
	id      int64
	URL     string
	Version semver.Version // Release version.
	Assets  []Asset        // The list of assets on this release.
}

type releasesByID []Release

// Asset struct represents a file included as part of a Release.
type Asset struct {
	id        int64
	v         semver.Version
	Name      string // Name of the release.
	URL       string // URL of the patch.
	LocalFile string
	Checksum  string // SHA256 hash of the file.
	Signature string // RSASSA-PKCS1-V1_5-SIGN signature, this is the SHA256 hash against the private key.
	AssetInfo
}

// AssetInfo struct holds OS and Arch information of an asset.
type AssetInfo struct {
	OS   string
	Arch string
}

// ReleaseManager struct defines a repository to pull releases from.
type ReleaseManager struct {
	client          *github.Client
	owner           string
	repo            string
	updateAssetsMap map[string]map[string]map[string]*Asset
	latestAssetsMap map[string]map[string]*Asset
	mu              *sync.RWMutex
}

func (a releasesByID) Len() int {
	return len(a)
}

func (a releasesByID) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a releasesByID) Less(i, j int) bool {
	return a[i].id < a[j].id
}

// NewReleaseManager creates a wrapper of github.Client.
func NewReleaseManager(owner string, repo string) *ReleaseManager {

	ghc := &ReleaseManager{
		client:          github.NewClient(nil),
		owner:           owner,
		repo:            repo,
		mu:              new(sync.RWMutex),
		updateAssetsMap: make(map[string]map[string]map[string]*Asset),
		latestAssetsMap: make(map[string]map[string]*Asset),
	}

	if mockServerAddr != "" {
		uri, err := url.Parse("http://" + mockServerAddr)
		if err != nil {
			panic(err.Error())
		}
		ghc.client.BaseURL = uri
		ghc.client.UploadURL = uri
		log.Debugf("Mocking Github API.")
	}

	return ghc
}

// getReleases queries github for all product releases.
func (g *ReleaseManager) getReleases() ([]Release, error) {
	releases := []Release{}

	for page := 1; true; page++ {
		opt := &github.ListOptions{Page: page}

		rels, _, err := g.client.Repositories.ListReleases(context.Background(), g.owner, g.repo, opt)
		if err != nil {
			return nil, err
		}
		if len(rels) == 0 {
			break
		}

		for i := range rels {
			version := *rels[i].TagName
			v, err := semver.Parse(version)
			if err != nil {
				log.Debugf("Release %q of %s/%s is not semantically versioned (%q). Skipping.", g.owner, g.repo, version, err)
				continue
			}
			if v.Major < 4 {
				log.Debugf("Ignoring release %s because it is too old", version)
				continue
			}
			rel := Release{
				id:      *rels[i].ID,
				URL:     *rels[i].ZipballURL,
				Version: v,
			}
			rel.Assets = make([]Asset, 0, len(rels[i].Assets))
			for _, asset := range rels[i].Assets {
				rel.Assets = append(rel.Assets, Asset{
					id:   *asset.ID,
					Name: *asset.Name,
					URL:  *asset.BrowserDownloadURL,
				})
			}
			log.Debugf("Release %q of %s/%s has %d assets...", version, g.owner, g.repo, len(rel.Assets))
			releases = append(releases, rel)
		}
	}

	sort.Sort(sort.Reverse(releasesByID(releases)))
	return releases, nil
}

// UpdateAssetsMap will pull published releases, scan for compatible
// update-only binaries and will add them to the updateAssetsMap.
func (g *ReleaseManager) UpdateAssetsMap() (err error) {

	var rs []Release

	log.Debugf("Getting releases...")
	if rs, err = g.getReleases(); err != nil {
		return err
	}
	log.Debugf("Found %d releases under %s/%s", len(rs), g.owner, g.repo)

	// Resetting file hashes.
	fileHashMapMu.Lock()
	fileHashMap = map[string]string{}
	fileHashMapMu.Unlock()

	log.Debugf("Getting assets...")
	for i := range rs {
		log.Debugf("Getting assets for release %q...", rs[i].Version)
		for j := range rs[i].Assets {
			log.Debugf("Found %q.", rs[i].Assets[j].Name)
			/*
				if rs[i].Assets[j].v.Major < 4 {
					log.Debugf("Ignoring asset %s because it is too old", rs[i].Assets[j].Name)
					continue
				}
			*/
			// Does this asset represent a binary update?
			if isUpdateAsset(rs[i].Assets[j].Name) {
				log.Debugf("%q is an auto-update asset.", rs[i].Assets[j].Name)
				asset := rs[i].Assets[j]
				asset.v = rs[i].Version
				info, err := getAssetInfo(asset.Name)
				if err != nil {
					return fmt.Errorf("could not get asset info: %q", err)
				}
				if err = g.pushAsset(info.OS, info.Arch, &asset); err != nil {
					return fmt.Errorf("could not push asset: %q", err)
				}
			} else {
				log.Debugf("%q is not an auto-update asset. Skipping.", rs[i].Assets[j].Name)
			}
		}
	}

	return nil
}

func (g *ReleaseManager) getProductUpdate(os string, arch string) (asset *Asset, err error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.latestAssetsMap == nil {
		return nil, fmt.Errorf("no updates available")
	}

	if g.latestAssetsMap[os] == nil {
		return nil, fmt.Errorf("no such OS")
	}

	if g.latestAssetsMap[os][arch] == nil {
		return nil, fmt.Errorf("no such Arch")
	}

	return g.latestAssetsMap[os][arch], nil
}

func (g *ReleaseManager) lookupAssetWithChecksum(os string, arch string, checksum string) (asset *Asset, err error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if os == OS.Android {
		return nil, fmt.Errorf("checksums disabled for Android")
	}

	if g.updateAssetsMap == nil {
		return nil, fmt.Errorf("no updates available")
	}

	if g.updateAssetsMap[os] == nil {
		return nil, fmt.Errorf("no such OS")
	}

	if g.updateAssetsMap[os][arch] == nil {
		return nil, fmt.Errorf("no such Arch")
	}

	for _, a := range g.updateAssetsMap[os][arch] {
		if a.Checksum == checksum {
			return a, nil
		}
	}

	return nil, fmt.Errorf("could not find a matching checksum in assets list")
}

func (g *ReleaseManager) lookupAssetWithVersion(os string, arch string, version string) (asset *Asset, err error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.updateAssetsMap == nil {
		return nil, fmt.Errorf("no updates available")
	}

	if g.updateAssetsMap[os] == nil {
		return nil, fmt.Errorf("no such OS")
	}

	if g.updateAssetsMap[os][arch] == nil {
		return nil, fmt.Errorf("no such Arch")
	}

	for _, a := range g.updateAssetsMap[os][arch] {
		if a.v.String() == version {
			return a, nil
		}
	}

	return nil, fmt.Errorf("could not find a matching version in assets list")
}

func (g *ReleaseManager) pushAsset(os string, arch string, asset *Asset) (err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	version := asset.v

	asset.OS = os
	asset.Arch = arch

	if version.EQ(emptyVersion) {
		return fmt.Errorf("missing asset version")
	}

	var localfile string
	if localfile, err = downloadAsset(asset.URL); err != nil {
		return err
	}

	if asset.Checksum, err = checksumForFile(localfile); err != nil {
		return err
	}

	if asset.Signature, err = signatureForFile(localfile); err != nil {
		return err
	}

	// Pushing version.
	if g.updateAssetsMap[os] == nil {
		g.updateAssetsMap[os] = make(map[string]map[string]*Asset)
	}
	if g.updateAssetsMap[os][arch] == nil {
		g.updateAssetsMap[os][arch] = make(map[string]*Asset)
	}
	g.updateAssetsMap[os][arch][version.String()] = asset

	// Setting latest version.
	if g.latestAssetsMap[os] == nil {
		g.latestAssetsMap[os] = make(map[string]*Asset)
	}

	if g.latestAssetsMap[os][arch] == nil {
		g.latestAssetsMap[os][arch] = asset
	} else {
		// Compare against already set version.
		if asset.v.GT(g.latestAssetsMap[os][arch].v) {
			g.latestAssetsMap[os][arch] = asset
		}
	}

	return nil
}

func getAssetInfo(s string) (*AssetInfo, error) {
	matches := updateAssetRe.FindStringSubmatch(s)
	if len(matches) >= 3 {
		if matches[1] != OS.Windows && matches[1] != OS.Linux && matches[1] != OS.Darwin && matches[1] != OS.Android {
			return nil, fmt.Errorf("unknown OS: \"%s\"", matches[1])
		}
		if matches[2] != Arch.X64 && matches[2] != Arch.X86 && matches[2] != Arch.ARM {
			return nil, fmt.Errorf("unknown architecture \"%s\"", matches[2])
		}
		info := &AssetInfo{
			OS:   matches[1],
			Arch: matches[2],
		}
		return info, nil
	}
	return nil, fmt.Errorf("could not find asset info")
}

func isUpdateAsset(s string) bool {
	return updateAssetRe.MatchString(s)
}
