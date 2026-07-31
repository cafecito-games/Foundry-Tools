package fsgenerator

import (
	gopath "path"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
)

// Every generated file imports foundry.proto, so a schema type sharing a name
// with something that namespace exports makes the spelling ambiguous and the
// binding will not resolve. runtimeTypeNames drives the escape that prevents
// it, and it is a hand-written list next to a set of files that grows.
//
// Deriving the expected names from the runtime source means adding a type
// there fails here rather than at some user's schema.
func TestRuntimeTypeNamesCoverEveryExportedRuntimeType(t *testing.T) {
	declared := runtimeExports("foundry.proto")
	require.NotEmpty(t, declared, "no runtime types found; the declaration pattern is probably stale")

	var missing []string
	for name, path := range declared {
		if !runtimeTypeNames[name] {
			missing = append(missing, name+" (declared in "+path+")")
		}
	}
	sort.Strings(missing)
	require.Empty(t, missing, "runtimeTypeNames is missing types foundry.proto exports; a schema declaring one of these would generate an ambiguous reference")

	// And the reverse, so a type removed from the runtime does not leave a
	// name being escaped for no reason.
	var stale []string
	for name := range runtimeTypeNames {
		if _, ok := declared[name]; !ok {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	require.Empty(t, stale, "runtimeTypeNames escapes names foundry.proto no longer exports")
}

// foundry.proto.wkt is imported only by a file that references a well-known
// type, and a reference that would be ambiguous is emitted namespace-qualified
// instead. So, unlike foundry.proto, none of its exports is escaped: a schema
// may declare its own Timestamp and keep the name.
//
// Escaping one would be wrong twice over. Every schema declaring that name
// would be renamed, including the ones that never mention a well-known type.
// And the bindings themselves are generator output, so the escape would rename
// the runtime's own Timestamp the next time `task gen-wkt` runs.
//
// Deriving the names from the runtime source means a well-known type added
// there is covered here rather than at some user's schema.
func TestWellKnownExportsAreNotEscaped(t *testing.T) {
	declared := runtimeExports(wellknown.Namespace)
	require.NotEmpty(t, declared, "no well-known bindings found; the declaration pattern is probably stale")

	var escaped []string
	for name, path := range declared {
		if escapeIdentifier(name) != name {
			escaped = append(escaped, name+" (declared in "+path+")")
		}
	}
	sort.Strings(escaped)
	require.Empty(t, escaped,
		"a name foundry.proto.wkt exports is escaped, which renames the runtime's own binding when it is regenerated")
}

// A schema generating into a namespace the runtime ships writes its bindings to
// the same output paths as the runtime files, which both entry points write
// last -- so the schema would be discarded with nothing said about it.
//
// The reserved set is derived from the shipped runtime source rather than
// listed, so this recomputes it independently from the output paths the runtime
// files occupy: a runtime namespace added later is reserved without an edit
// here or in the generator.
func TestReservedNamespacesAreEveryNamespaceTheRuntimeDeclares(t *testing.T) {
	expected := map[string]bool{}
	for path := range runtime.Files() {
		directory := gopath.Dir(path)
		require.NotEqual(t, ".", directory, "runtime file %s is not namespaced", path)
		expected[strings.ReplaceAll(directory, "/", ".")] = true
	}
	require.NotEmpty(t, expected, "no runtime files found; the embedded runtime is probably empty")

	require.Equal(t, expected, runtimeNamespaces(),
		"the reserved set must be every namespace the runtime declares, derived from its source")

	// The namespaces shipped today, so a change to either side of the
	// comparison above is still visible as a change in what is rejected.
	require.Equal(t, []string{"foundry.proto", wellknown.Namespace}, sortedRuntimeNamespaces())

	for namespace := range expected {
		require.ErrorContains(t, ValidateNamespace(namespace), "is reserved")
	}
}

// The collision is on the output path, and macOS and Windows default to
// case-insensitive filesystems, so a differently-cased spelling writes over the
// runtime's file rather than beside it.
func TestReservedNamespacesIgnoreCase(t *testing.T) {
	require.ErrorContains(t, ValidateNamespace("Foundry.proto.wkt"), "is reserved")
	require.ErrorContains(t, ValidateNamespace("FOUNDRY.PROTO"), "is reserved")

	// A namespace that cannot occupy a runtime path is still accepted.
	require.NoError(t, ValidateNamespace("Foundry.proto.wkt.mine"))
	require.NoError(t, ValidateNamespace("foundrytools.game.v1"))
}
