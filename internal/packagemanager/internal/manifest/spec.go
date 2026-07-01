package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// SourceType identifies how a package is obtained.
type SourceType string

const (
	// SourceGit fetches a Git repository ref.
	SourceGit SourceType = "git"
	// SourceGitHubRelease fetches one GitHub release asset.
	SourceGitHubRelease SourceType = "github-release"
	// SourceArchive fetches a direct HTTP(S) archive.
	SourceArchive SourceType = "archive"
)

// PackageSpec is one package entry declared in packages.toml.
type PackageSpec struct {
	Name string `toml:"-"`

	Source     SourceType `toml:"source"`
	URL        string     `toml:"url,omitempty"`
	Repo       string     `toml:"repo,omitempty"`
	Version    string     `toml:"version,omitempty"`
	Asset      string     `toml:"asset,omitempty"`
	SourcePath string     `toml:"source_path,omitempty"`
	InstallAs  string     `toml:"install_as,omitempty"`
	Exclude    []string   `toml:"exclude,omitempty"`
	Checksum   string     `toml:"checksum,omitempty"`
}

// Manifest is the parsed contents of packages.toml.
type Manifest struct {
	Packages map[string]PackageSpec `toml:"packages"`
}

// InstallName returns the directory name under addons/ for this package.
func (s PackageSpec) InstallName() string {
	if s.InstallAs != "" {
		return s.InstallAs
	}
	return s.Name
}

// Hash returns a stable hash of the spec's resolvable fields.
func (s PackageSpec) Hash() string {
	exclude := append([]string(nil), s.Exclude...)
	sort.Strings(exclude)
	representation := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s",
		s.Source, s.URL, s.Repo, s.Version, s.Asset, s.SourcePath, s.InstallAs, s.Checksum, strings.Join(exclude, "\x00"))
	checksum := sha256.Sum256([]byte(representation))
	return hex.EncodeToString(checksum[:])
}
