// Package output defines package-manager error classes and exit codes.
package output

import "errors"

// ExitCode is a process exit status with a defined meaning.
type ExitCode int

const (
	// ExitOK indicates success.
	ExitOK ExitCode = 0
	// ExitGeneric indicates an unclassified failure.
	ExitGeneric ExitCode = 1
	// ExitUsage indicates bad flags or arguments.
	ExitUsage ExitCode = 2
	// ExitManifest indicates a project, manifest, or lockfile failure.
	ExitManifest ExitCode = 3
	// ExitFetch indicates a network, auth, or source-resolution failure.
	ExitFetch ExitCode = 4
	// ExitInstall indicates an extraction or filesystem install failure.
	ExitInstall ExitCode = 5
)

// ManifestError wraps a manifest, project discovery, or lockfile failure.
type ManifestError struct{ Err error }

func (e *ManifestError) Error() string { return e.Err.Error() }
func (e *ManifestError) Unwrap() error { return e.Err }

// FetchError wraps a network/auth/source-resolution failure.
type FetchError struct{ Err error }

func (e *FetchError) Error() string { return e.Err.Error() }
func (e *FetchError) Unwrap() error { return e.Err }

// InstallError wraps a filesystem/extraction failure.
type InstallError struct{ Err error }

func (e *InstallError) Error() string { return e.Err.Error() }
func (e *InstallError) Unwrap() error { return e.Err }

// CodeFor maps an error to the exit code the process should return.
func CodeFor(err error) ExitCode {
	if err == nil {
		return ExitOK
	}
	var me *ManifestError
	var fe *FetchError
	var ie *InstallError
	switch {
	case errors.As(err, &me):
		return ExitManifest
	case errors.As(err, &fe):
		return ExitFetch
	case errors.As(err, &ie):
		return ExitInstall
	default:
		return ExitGeneric
	}
}
