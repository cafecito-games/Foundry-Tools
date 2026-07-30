package fsgenerator

import (
	"regexp"
	"sort"
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
	"github.com/stretchr/testify/require"
)

// runtimeDeclaration matches the ways foundry.proto introduces an exported
// type name.
var runtimeDeclaration = regexp.MustCompile(`(?m)^(?:final\s+)?(?:class_name|tuple_name|enum_name|trait_name)\s+([A-Za-z_][A-Za-z0-9_]*)`)

// runtimeNamespace matches a runtime file's namespace declaration.
var runtimeNamespace = regexp.MustCompile(`(?m)^namespace\s+([A-Za-z_][A-Za-z0-9_.]*)`)

// exportsOf returns every type name the given runtime namespace declares,
// mapped to the file that declares it.
func exportsOf(namespace string) map[string]string {
	declared := map[string]string{}
	for path, source := range runtime.Files() {
		match := runtimeNamespace.FindStringSubmatch(source)
		if len(match) != 2 || match[1] != namespace {
			continue
		}
		for _, declaration := range runtimeDeclaration.FindAllStringSubmatch(source, -1) {
			declared[declaration[1]] = path
		}
	}
	return declared
}

// Every generated file imports foundry.proto, so a schema type sharing a name
// with something that namespace exports makes the spelling ambiguous and the
// binding will not resolve. runtimeTypeNames drives the escape that prevents
// it, and it is a hand-written list next to a set of files that grows.
//
// Deriving the expected names from the runtime source means adding a type
// there fails here rather than at some user's schema.
func TestRuntimeTypeNamesCoverEveryExportedRuntimeType(t *testing.T) {
	declared := exportsOf("foundry.proto")
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
	declared := exportsOf(wellknown.Namespace)
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
