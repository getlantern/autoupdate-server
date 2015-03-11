package server

import (
	"fmt"
	"github.com/google/go-github/github"
	"sort"
	"sync"
)

const (
	archX64 = "x64"
	archX86 = "x86"
	archARM = "arm"
)

const (
	osWindows = "windows"
	osLinux   = "linux"
	osDarwin  = "darwin"
)

type assetArch string
type assetOS string

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
	v         []int
	Name      string
	URL       string
	LocalFile string
	AssetInfo
}

type AssetInfo struct {
	OS   string
	Arch string
}

// ReleaseManager struct defines a repository to pull releases from.
type ReleaseManager struct {
	client          *github.Client
	owner           string
	repo            string
	updateAssetsMap map[assetOS]map[assetArch][]*Asset
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
		client: github.NewClient(nil),
		owner:  owner,
		repo:   repo,
		mu:     new(sync.RWMutex),
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
	g.mu.Lock()
	defer g.mu.Unlock()

	var rs []Release

	if rs, err = g.GetReleases(); err != nil {
		return err
	}

	for i := range rs {
		fmt.Printf("release: %v\n", rs[i].Version)
		fmt.Printf("asset: %v\n", rs[i].Assets)
	}

	return nil
}

func getAssetInfo(s string) (*AssetInfo, error) {
	return nil, fmt.Errorf("Not implemented.")
}
