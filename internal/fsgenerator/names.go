package fsgenerator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

const namespaceOptionKey = "(foundrytools.namespace)"
const typePrefixOptionKey = "(foundrytools.type_prefix)"

// Reserved for later generator stages that apply file-level type prefixes.
var _ = typePrefixOptionKey

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// NamespaceFor returns the Foundry Script namespace for a proto file.
func NamespaceFor(file *protoast.ProtoFile) string {
	if file == nil {
		return ""
	}
	if raw, ok := file.Options[namespaceOptionKey]; ok {
		if value, isString := raw.(string); isString && value != "" {
			return value
		}
	}
	return file.Package
}

// ValidateNamespace validates a dotted Foundry Script namespace.
func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return nil
	}
	for _, part := range strings.Split(namespace, ".") {
		if !identifierPattern.MatchString(part) {
			return fmt.Errorf("invalid namespace segment %q in %q", part, namespace)
		}
	}
	return nil
}

// TypeName converts a proto identifier to a Foundry Script type identifier.
func TypeName(name string) string {
	if name == "" {
		return ""
	}

	var builder strings.Builder
	nextUpper := true
	for _, char := range name {
		switch char {
		case '_', '-', '.':
			nextUpper = true
		default:
			if nextUpper {
				builder.WriteRune(unicode.ToUpper(char))
				nextUpper = false
			} else {
				builder.WriteRune(char)
			}
		}
	}

	return escapeIdentifier(builder.String())
}

func escapeIdentifier(name string) string {
	switch name {
	case "Class", "ClassName", "Enum", "EnumName", "Extends", "Func", "Import", "Namespace", "Trait", "TraitName", "Uses", "Var":
		return name + "_"
	default:
		return name
	}
}
