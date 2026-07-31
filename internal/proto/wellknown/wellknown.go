// Package wellknown identifies the google/protobuf well-known types, which are
// shipped as runtime source rather than generated per project. Generating them
// per project would give every project its own incompatible Timestamp.
//
// A proto file is identified by its import path: the path it carries relative
// to an include root, which is how protoc names it. Everything here takes an
// import path and matches it exactly. A filesystem path is not an import path
// until an include root has been applied to it, which is what ImportPathFor is
// for.
package wellknown

import (
	"embed"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed proto/google/protobuf/*.proto
var protoFS embed.FS

// Namespace is the Foundry Script namespace the well-known bindings live in.
const Namespace = "foundry.proto.wkt"

const protoPrefix = "google/protobuf/"

// supported is the set of well-known files the runtime ships bindings for.
var supported = map[string]bool{
	protoPrefix + "any.proto":        true,
	protoPrefix + "duration.proto":   true,
	protoPrefix + "empty.proto":      true,
	protoPrefix + "field_mask.proto": true,
	protoPrefix + "struct.proto":     true,
	protoPrefix + "timestamp.proto":  true,
	protoPrefix + "wrappers.proto":   true,
}

// IsWellKnownImport reports whether an import path names a well-known file the
// runtime ships bindings for.
//
// The match is exact because an import path is the file's identity, not a hint
// about it. "myorg/google/protobuf/timestamp.proto" names a different file that
// happens to be spelled similarly: answering yes for it would both skip
// generating bindings the caller asked for and, during import resolution, turn
// a misspelled import into a silent substitution of the bundled schema.
func IsWellKnownImport(importedPath string) bool {
	return supported[normalize(importedPath)]
}

// ImportPathFor reduces a filesystem path to the import path protoc would give
// it: the path relative to the first include root that contains it. A vendored
// tree invoked as `-I vendor vendor/google/protobuf/timestamp.proto` therefore
// carries the import path google/protobuf/timestamp.proto and is the well-known
// file, while `-I . myorg/google/protobuf/timestamp.proto` carries a distinct
// import path and is an ordinary schema of the caller's own.
//
// A relative path no include root contains is its own import path, which is
// what protoc does with a relative path and no -I. `vend/google/protobuf/
// timestamp.proto` passed that way therefore carries the import path
// vend/google/protobuf/timestamp.proto, is not the well-known file, and is
// generated; a caller who wants the runtime's bindings names them by passing
// -I vend.
//
// That leaves one case with no answer: a path from which no relative import
// path can be derived at all -- an absolute path, or one that climbs out of
// every root -- that nonetheless spells out a google/protobuf file. There is
// nothing to make it relative to, so it could be the well-known file or an
// unrelated schema, and both guesses are damaging: one silently drops bindings
// the caller asked for, the other silently produces a second, incompatible copy
// of a runtime type. That case is an error.
func ImportPathFor(filename string, importRoots []string) (string, error) {
	for _, root := range importRoots {
		if relative, ok := relativeTo(root, filename); ok {
			return relative, nil
		}
	}
	name := normalize(filename)
	if index := strings.LastIndex(name, "/"+protoPrefix); index >= 0 && !isImportPath(name) {
		suggestedRoot := name[:index]
		if suggestedRoot == "" {
			suggestedRoot = "/"
		}
		return "", fmt.Errorf(
			"%s cannot be identified without an include path: it is not relative to the working "+
				"directory and no -I root contains it, so it is either the well-known %s or an "+
				"unrelated schema that spells its path the same way. Pass -I %s to name it as the "+
				"well-known file, or leave it off the command line entirely -- foundry-tools "+
				"already ships Foundry Script for %s",
			filename, name[index+1:], suggestedRoot, strings.Join(Files(), ", "),
		)
	}
	return name, nil
}

// isImportPath reports whether a normalized path can stand as an import path on
// its own. protoc names a file by the path it was given, so a plain relative
// path qualifies; an absolute one, or one that climbs above the working
// directory, has no spelling protoc could hand to an import statement.
func isImportPath(name string) bool {
	if path.IsAbs(name) || filepath.IsAbs(filepath.FromSlash(name)) {
		return false
	}
	return name != ".." && !strings.HasPrefix(name, "../")
}

// relativeTo reports the import path filename carries under root, if root
// contains it. Both sides are resolved against the working directory so that a
// relative root such as `-I .` still claims an absolute input.
func relativeTo(root, filename string) (string, bool) {
	absoluteRoot, err := filepath.Abs(filepath.FromSlash(normalize(root)))
	if err != nil {
		return "", false
	}
	absoluteFile, err := filepath.Abs(filepath.FromSlash(normalize(filename)))
	if err != nil {
		return "", false
	}
	relative, err := filepath.Rel(absoluteRoot, absoluteFile)
	if err != nil {
		return "", false
	}
	relative = filepath.ToSlash(relative)
	if relative == ".." || strings.HasPrefix(relative, "../") {
		return "", false
	}
	return relative, true
}

// Check rejects a google/protobuf import path the runtime does not ship.
// Falling back to generic generation would silently produce a second,
// incompatible copy of a type the runtime already defines, so an unshipped file
// is an error rather than a quiet divergence.
func Check(importedPath string) error {
	name := normalize(importedPath)
	if !strings.HasPrefix(name, protoPrefix) || supported[name] {
		return nil
	}
	return fmt.Errorf(
		"%s is not supported: foundry-tools ships Foundry Script for %s, and generating another google/protobuf file would produce types the runtime does not recognize",
		name, strings.Join(Files(), ", "),
	)
}

// Files lists the supported well-known files in stable order.
func Files() []string {
	names := make([]string, 0, len(supported))
	for name := range supported {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Source returns the vendored text of a supported well-known import path.
func Source(importedPath string) ([]byte, error) {
	name := normalize(importedPath)
	if !supported[name] {
		return nil, fmt.Errorf("%s is not a vendored well-known file", name)
	}
	return protoFS.ReadFile(path.Join("proto", name))
}

func normalize(filename string) string {
	return path.Clean(strings.ReplaceAll(filename, "\\", "/"))
}
