package fsgenerator

import (
	"regexp"
	"sync"

	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
)

// runtimeDeclaration matches the ways a runtime file introduces an exported
// type name.
var runtimeDeclaration = regexp.MustCompile(`(?m)^(?:final\s+)?(?:class_name|tuple_name|enum_name|trait_name)\s+([A-Za-z_][A-Za-z0-9_]*)`)

// runtimeNamespace matches a runtime file's namespace declaration.
var runtimeNamespace = regexp.MustCompile(`(?m)^namespace\s+([A-Za-z_][A-Za-z0-9_.]*)`)

// runtimeExports returns every type name the given runtime namespace declares,
// mapped to the runtime file that declares it. Reading them out of the shipped
// source rather than listing them here means a binding added to the runtime is
// accounted for without a second edit somewhere else.
func runtimeExports(namespace string) map[string]string {
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

// wellKnownExports are the names an `import foundry.proto.wkt` brings into
// scope. The import names the namespace rather than a type, so all of them
// arrive together: a schema that references one well-known type also has every
// other well-known name in scope, and any of them competes with a same-named
// type imported from another namespace.
var wellKnownExports = sync.OnceValue(func() map[string]string {
	return runtimeExports(wellknown.Namespace)
})
