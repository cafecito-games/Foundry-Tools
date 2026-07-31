package fsgenerator

import (
	"regexp"
	"sort"
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

// runtimeNamespaces is the set of namespaces the embedded runtime declares.
// Every runtime file is written to an output path derived from its namespace,
// so a schema generating into one of these namespaces would emit files that the
// runtime then overwrites. Reading the set out of the shipped source rather than
// listing it here means a namespace added to the runtime is reserved without a
// second edit somewhere else.
var runtimeNamespaces = sync.OnceValue(func() map[string]bool {
	namespaces := map[string]bool{}
	for _, source := range runtime.Files() {
		if match := runtimeNamespace.FindStringSubmatch(source); len(match) == 2 {
			namespaces[match[1]] = true
		}
	}
	return namespaces
})

// sortedRuntimeNamespaces lists the reserved namespaces in stable order so a
// diagnostic reads the same way on every run.
func sortedRuntimeNamespaces() []string {
	namespaces := make([]string, 0, len(runtimeNamespaces()))
	for namespace := range runtimeNamespaces() {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	return namespaces
}

// wellKnownExports are the names an `import foundry.proto.wkt` brings into
// scope. The import names the namespace rather than a type, so all of them
// arrive together: a schema that references one well-known type also has every
// other well-known name in scope, and any of them competes with a same-named
// type imported from another namespace.
var wellKnownExports = sync.OnceValue(func() map[string]string {
	return runtimeExports(wellknown.Namespace)
})
