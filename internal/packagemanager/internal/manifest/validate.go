package manifest

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cafecito-games/foundry-tools/internal/packagemanager/internal/output"
)

// Validate checks every package entry for required and consistent fields.
func (m *Manifest) Validate() error {
	for name := range m.Packages {
		pkg := m.Packages[name]
		if err := validateSpec(name, pkg); err != nil {
			return &output.ManifestError{Err: err}
		}
	}
	return nil
}

func validatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("package name must not be empty")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("package name %q must not be an absolute path", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("package name %q must not contain path separators", name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("package name %q is not a valid directory name", name)
	}
	return nil
}

func validateSourcePath(sourcePath string) error {
	if filepath.IsAbs(sourcePath) {
		return fmt.Errorf("source_path %q must not be an absolute path", sourcePath)
	}
	cleaned := filepath.Clean(sourcePath)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("source_path %q must not escape the source root", sourcePath)
	}
	return nil
}

func validateExcludePath(excludePath string) error {
	if excludePath == "" {
		return fmt.Errorf("exclude path must not be empty")
	}
	normalized := strings.ReplaceAll(excludePath, `\`, `/`)
	if filepath.IsAbs(excludePath) || strings.HasPrefix(normalized, "/") || hasWindowsDrivePrefix(normalized) {
		return fmt.Errorf("exclude path %q must not be an absolute path", excludePath)
	}
	for _, component := range strings.Split(normalized, "/") {
		if component == ".." {
			return fmt.Errorf("exclude path %q must not escape the install root", excludePath)
		}
	}
	if path.Clean(normalized) == "." {
		return fmt.Errorf("exclude path %q must name a directory below the install root", excludePath)
	}
	return nil
}

func hasWindowsDrivePrefix(value string) bool {
	if len(value) < 2 || value[1] != ':' {
		return false
	}
	first := value[0]
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')
}

var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validateChecksum(checksum string) error {
	if !sha256Pattern.MatchString(checksum) {
		return fmt.Errorf("checksum %q must be 64 lowercase hex digits (SHA-256)", checksum)
	}
	return nil
}

func scpLikeGitURL(s string) bool {
	if strings.Contains(s, "://") {
		return false
	}
	colon := strings.IndexByte(s, ':')
	slash := strings.IndexByte(s, '/')
	return colon > 0 && (slash == -1 || colon < slash)
}

func validateGitURL(rawURL string) error {
	if strings.HasPrefix(rawURL, "-") {
		return fmt.Errorf("url %q must not begin with '-'", rawURL)
	}
	if strings.Contains(rawURL, "::") {
		return fmt.Errorf("url %q uses a disallowed git remote-helper syntax", rawURL)
	}
	if scpLikeGitURL(rawURL) {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("url %q is not a valid URL: %w", rawURL, err)
	}
	switch parsed.Scheme {
	case "https", "http", "ssh", "git", "file", "":
		return nil
	default:
		return fmt.Errorf("url %q uses unsupported scheme %q (allowed: https, http, ssh, git, file)", rawURL, parsed.Scheme)
	}
}

func validateArchiveURL(rawURL string) error {
	if strings.HasPrefix(rawURL, "-") {
		return fmt.Errorf("url %q must not begin with '-'", rawURL)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("url %q is not a valid URL: %w", rawURL, err)
	}
	switch parsed.Scheme {
	case "http", "https":
		return nil
	case "":
		return fmt.Errorf("url %q must include an http:// or https:// scheme", rawURL)
	default:
		return fmt.Errorf("url %q uses unsupported scheme %q (allowed: http, https)", rawURL, parsed.Scheme)
	}
}

func validateGitVersion(version string) error {
	if strings.HasPrefix(version, "-") {
		return fmt.Errorf("version %q must not begin with '-'", version)
	}
	return nil
}

func validateSpec(name string, pkg PackageSpec) error {
	if err := validatePackageName(name); err != nil {
		return err
	}
	if pkg.InstallAs != "" {
		if err := validatePackageName(pkg.InstallAs); err != nil {
			return fmt.Errorf("package %q: invalid install_as: %w", name, err)
		}
	}
	if pkg.SourcePath != "" {
		if err := validateSourcePath(pkg.SourcePath); err != nil {
			return fmt.Errorf("package %q: invalid source_path: %w", name, err)
		}
	}
	for _, excludePath := range pkg.Exclude {
		if err := validateExcludePath(excludePath); err != nil {
			return fmt.Errorf("package %q: invalid exclude: %w", name, err)
		}
	}
	if pkg.Checksum != "" {
		if pkg.Source == SourceGit {
			return fmt.Errorf("package %q: checksum is not supported for git sources", name)
		}
		if err := validateChecksum(pkg.Checksum); err != nil {
			return fmt.Errorf("package %q: invalid checksum: %w", name, err)
		}
	}

	switch pkg.Source {
	case SourceGit:
		if pkg.URL == "" || pkg.Version == "" {
			return fmt.Errorf("package %q: git source requires url and version", name)
		}
		if err := validateGitURL(pkg.URL); err != nil {
			return fmt.Errorf("package %q: %w", name, err)
		}
		if err := validateGitVersion(pkg.Version); err != nil {
			return fmt.Errorf("package %q: %w", name, err)
		}
	case SourceGitHubRelease:
		if pkg.Repo == "" || pkg.Version == "" {
			return fmt.Errorf("package %q: github-release source requires repo and version", name)
		}
	case SourceArchive:
		if pkg.URL == "" {
			return fmt.Errorf("package %q: archive source requires url", name)
		}
		if err := validateArchiveURL(pkg.URL); err != nil {
			return fmt.Errorf("package %q: %w", name, err)
		}
	default:
		return fmt.Errorf("package %q: unknown source %q (want git, github-release, or archive)", name, pkg.Source)
	}
	return nil
}
