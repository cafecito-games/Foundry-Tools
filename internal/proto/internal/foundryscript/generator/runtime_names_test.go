package fsgenerator

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/runtime"
	"github.com/stretchr/testify/require"
)

// runtimeDeclaration matches the ways foundry.proto introduces an exported
// type name.
var runtimeDeclaration = regexp.MustCompile(`(?m)^(?:final\s+)?(?:class_name|tuple_name|enum_name|trait_name)\s+([A-Za-z_][A-Za-z0-9_]*)`)

// Every generated file imports foundry.proto, so a schema type sharing a name
// with something that namespace exports makes the spelling ambiguous and the
// binding will not resolve. runtimeTypeNames drives the escape that prevents
// it, and it is a hand-written list next to a set of files that grows.
//
// Deriving the expected names from the runtime source means adding a type
// there fails here rather than at some user's schema.
func TestRuntimeTypeNamesCoverEveryExportedRuntimeType(t *testing.T) {
	declared := map[string]string{}
	for path, source := range runtime.Files() {
		if !strings.HasPrefix(path, "foundry/proto/") {
			continue
		}
		for _, match := range runtimeDeclaration.FindAllStringSubmatch(source, -1) {
			declared[match[1]] = path
		}
	}
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
