// Package wellknown identifies the google/protobuf well-known types, which are
// shipped as runtime source rather than generated per project. Generating them
// per project would give every project its own incompatible Timestamp.
package wellknown

import (
	"embed"
	"fmt"
	"path"
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

// IsWellKnown reports whether filename names a well-known file the runtime
// ships bindings for.
func IsWellKnown(filename string) bool {
	return supported[normalize(filename)]
}

// Check rejects a google/protobuf file the runtime does not ship. Falling back
// to generic generation would silently produce a second, incompatible copy of a
// type the runtime already defines, so an unshipped file is an error rather
// than a quiet divergence.
func Check(filename string) error {
	name := normalize(filename)
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

// Source returns the vendored text of a supported well-known file.
func Source(filename string) ([]byte, error) {
	name := normalize(filename)
	if !supported[name] {
		return nil, fmt.Errorf("%s is not a vendored well-known file", name)
	}
	return protoFS.ReadFile(path.Join("proto", name))
}

func normalize(filename string) string {
	return path.Clean(strings.ReplaceAll(filename, "\\", "/"))
}
