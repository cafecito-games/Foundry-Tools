package manifest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAcceptsSupportedSources(t *testing.T) {
	m := &Manifest{Packages: map[string]PackageSpec{
		"git":     {Source: SourceGit, URL: "https://example.com/repo.git", Version: "v1.0.0"},
		"release": {Source: SourceGitHubRelease, Repo: "owner/repo", Version: "v1.0.0", Asset: "*.zip"},
		"archive": {Source: SourceArchive, URL: "https://example.com/pkg.zip"},
	}}

	require.NoError(t, m.Validate())
}

func TestValidateRejectsUnsafeNames(t *testing.T) {
	cases := []string{"", "../escape", "nested/name", `nested\name`, ".", "..", "/abs"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			m := &Manifest{Packages: map[string]PackageSpec{
				name: {Source: SourceArchive, URL: "https://example.com/pkg.zip"},
			}}
			require.Error(t, m.Validate())
		})
	}
}

func TestValidateRejectsUnsafeSourcePath(t *testing.T) {
	m := &Manifest{Packages: map[string]PackageSpec{
		"pkg": {Source: SourceArchive, URL: "https://example.com/pkg.zip", SourcePath: "../escape"},
	}}
	require.Error(t, m.Validate())
}

func TestValidateRejectsUnsafeExcludes(t *testing.T) {
	cases := [][]string{
		{""},
		{"."},
		{".."},
		{"../escape"},
		{"/absolute"},
		{`C:\absolute`},
	}
	for _, excludes := range cases {
		m := &Manifest{Packages: map[string]PackageSpec{
			"pkg": {Source: SourceArchive, URL: "https://example.com/pkg.zip", Exclude: excludes},
		}}
		require.Error(t, m.Validate(), "exclude %q", excludes)
	}
}

func TestValidateRejectsBadChecksums(t *testing.T) {
	m := &Manifest{Packages: map[string]PackageSpec{
		"pkg": {Source: SourceArchive, URL: "https://example.com/pkg.zip", Checksum: "ABC"},
	}}
	require.Error(t, m.Validate())
}

func TestValidateRejectsChecksumForGit(t *testing.T) {
	m := &Manifest{Packages: map[string]PackageSpec{
		"pkg": {Source: SourceGit, URL: "https://example.com/repo.git", Version: "v1", Checksum: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
	}}
	require.Error(t, m.Validate())
}

func TestValidateRejectsMissingRequiredFields(t *testing.T) {
	cases := map[string]PackageSpec{
		"git":     {Source: SourceGit, URL: "https://example.com/repo.git"},
		"release": {Source: SourceGitHubRelease, Repo: "owner/repo"},
		"archive": {Source: SourceArchive},
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			m := &Manifest{Packages: map[string]PackageSpec{name: spec}}
			require.Error(t, m.Validate())
		})
	}
}

func TestValidateRejectsUnknownSource(t *testing.T) {
	m := &Manifest{Packages: map[string]PackageSpec{
		"pkg": {Source: SourceType("unknown"), URL: "https://example.com/pkg.zip"},
	}}
	require.Error(t, m.Validate())
}

func TestValidateRejectsArchiveURLWithoutHTTPScheme(t *testing.T) {
	m := &Manifest{Packages: map[string]PackageSpec{
		"pkg": {Source: SourceArchive, URL: "file:///tmp/pkg.zip"},
	}}
	require.Error(t, m.Validate())
}

func TestValidateRejectsUnsafeGitURLAndVersion(t *testing.T) {
	cases := []PackageSpec{
		{Source: SourceGit, URL: "-bad", Version: "v1"},
		{Source: SourceGit, URL: "ext::sh -c bad", Version: "v1"},
		{Source: SourceGit, URL: "https://example.com/repo.git", Version: "-bad"},
	}
	for _, spec := range cases {
		m := &Manifest{Packages: map[string]PackageSpec{"pkg": spec}}
		require.Error(t, m.Validate())
	}
}
