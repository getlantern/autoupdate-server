package server

import (
	"fmt"
	"github.com/google/go-github/github"
	"sort"
	"sync"
)

// Arch holds architecture names.
var Arch = struct {
	X64 string
	X86 string
	ARM string
}{
	"x64",
	"x86",
	"arm",
}

// OS holds operating system names.
var OS = struct {
	Windows string
	Linux   string
	Darwin  string
}{
	"windows",
	"linux",
	"darwin",
}

// Release struct represents a single github release.
type Release struct {
	id      int
	URL     string
	Version string
	Assets  []Asset
}

type releasesByID []Release

// Asset struct represents a file included as part of a Release.
type Asset struct {
	id        int
	v         string
	Name      string
	URL       string
	LocalFile string
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

	return ghc
}

// GetReleases queries github for all product releases.
func (g *ReleaseManager) GetReleases() ([]Release, error) {
	rels, _, err := g.client.Repositories.ListReleases(g.owner, g.repo, nil)

	if err != nil {
		return nil, err
	}

	releases := make([]Release, 0, len(rels))

	for i := range rels {
		rel := Release{
			id:      *rels[i].ID,
			URL:     *rels[i].ZipballURL,
			Version: *rels[i].TagName,
		}
		rel.Assets = make([]Asset, 0, len(rels[i].Assets))
		for _, asset := range rels[i].Assets {
			rel.Assets = append(rel.Assets, Asset{
				id:   *asset.ID,
				Name: *asset.Name,
				URL:  *asset.BrowserDownloadURL,
			})
			// fmt.Printf("asset: %v -- %v -- %v\n", asset.Label, asset.State, asset.ContentType)
		}
		releases = append(releases, rel)
	}

	sort.Sort(sort.Reverse(releasesByID(releases)))

	return releases, nil
}

// UpdateAssetsMap will pull published releases, scan for compatible
// update-only binaries and will add them to the updateAssetsMap.
func (g *ReleaseManager) UpdateAssetsMap() (err error) {

	var rs []Release

	if rs, err = g.GetReleases(); err != nil {
		return err
	}

	for i := range rs {
		// Does this tag represent a release?
		if isVersionTag(rs[i].Version) {
			for j := range rs[i].Assets {
				// Does this asset represent a binary update?
				if isUpdateAsset(rs[i].Assets[j].Name) {
					asset := rs[i].Assets[j]
					asset.v = rs[i].Version
					info, err := getAssetInfo(asset.Name)
					if err != nil {
						return err
					}
					g.pushAsset(info.OS, info.Arch, &asset)
				}
			}
		}
	}

	return nil
}

func (g *ReleaseManager) pushAsset(os string, arch string, asset *Asset) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	version := asset.v
	if version == "" {
		return fmt.Errorf("Missing asset version.")
	}
	// Pushing version.
	if g.updateAssetsMap[os] == nil {
		g.updateAssetsMap[os] = make(map[string]map[string]*Asset)
	}
	if g.updateAssetsMap[os][arch] == nil {
		g.updateAssetsMap[os][arch] = make(map[string]*Asset)
	}
	g.updateAssetsMap[os][arch][version] = asset

	// Setting latest version.
	if g.latestAssetsMap[os] == nil {
		g.latestAssetsMap[os] = make(map[string]*Asset)
	}
	if g.latestAssetsMap[os][arch] == nil {
		g.latestAssetsMap[os][arch] = asset
	} else {
		// Compare against already set version
		if VersionCompare(g.latestAssetsMap[os][arch].v, asset.v) == Higher {
			g.latestAssetsMap[os][arch] = asset
		}
	}

	return nil
}

func getAssetInfo(s string) (*AssetInfo, error) {
	matches := updateAssetRe.FindStringSubmatch(s)
	if len(matches) >= 3 {
		if matches[1] != OS.Windows && matches[1] != OS.Linux && matches[1] != OS.Darwin {
			return nil, fmt.Errorf("Unknown OS: \"%s\".", matches[1])
		}
		if matches[2] != Arch.X64 && matches[2] != Arch.X86 && matches[2] != Arch.ARM {
			return nil, fmt.Errorf("Unknown architecture \"%s\".", matches[2])
		}
		info := &AssetInfo{
			OS:   matches[1],
			Arch: matches[2],
		}
		return info, nil
	}
	return nil, fmt.Errorf("Could not find asset info.")
}
