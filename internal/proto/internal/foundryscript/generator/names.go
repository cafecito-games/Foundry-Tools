package fsgenerator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
)

// NamespaceOptionKey is the file option that overrides the namespace a proto
// file generates into, keyed as it appears in a parsed file's option map.
const NamespaceOptionKey = "(foundrytools.namespace)"
const typePrefixOptionKey = "(foundrytools.type_prefix)"

// generatedPrefix marks every name the emitter introduces -- locals, function
// parameters, and private members alike. protobuf identifiers may begin with an
// underscore, and protoc accepts them, so a bare `_` prefix is not enough to
// stay clear of a schema; reserving this one narrow spelling is, and it is the
// only spelling a schema may not use.
const generatedPrefix = "_pb_"

// unknownFieldsMember holds the raw bytes of fields the schema did not
// recognize, so re-encoding a message decoded from a newer peer is lossless.
const unknownFieldsMember = generatedPrefix + "unknown_fields"

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// NamespaceFor returns the Foundry Script namespace for a proto file.
func NamespaceFor(file *protoast.ProtoFile) string {
	if file == nil {
		return ""
	}
	if raw, ok := file.Options[NamespaceOptionKey]; ok {
		if value, isString := raw.(string); isString && value != "" {
			return value
		}
	}
	return file.Package
}

// ValidateNamespace validates a dotted Foundry Script namespace and rejects the
// namespaces the runtime ships bindings in.
func ValidateNamespace(namespace string) error {
	if err := validateNamespaceShape(namespace); err != nil {
		return err
	}
	// Only an exact match is reserved. A nested namespace such as
	// foundry.proto.wkt.mine generates into its own directory and so cannot
	// shadow a runtime file, and reserving the whole `foundry.` prefix would
	// reject schemas that merely start with the same word.
	if isRuntimeNamespace(namespace) {
		return reservedNamespaceError(namespace)
	}
	return nil
}

// reservedNamespaceError explains why a namespace the runtime ships cannot be
// claimed. It is shared with the resolver, which reaches the same condition
// through an import rather than through the file being generated.
func reservedNamespaceError(namespace string) error {
	return fmt.Errorf(
		"namespace %q is reserved: foundry-tools ships the runtime bindings for %s, "+
			"so generating into it would produce files the runtime replaces and silently discard this schema; "+
			"set (foundrytools.namespace) to a namespace of your own",
		namespace, strings.Join(sortedRuntimeNamespaces(), ", "),
	)
}

// validateNamespaceShape validates only the spelling of a dotted namespace,
// leaving the reserved check to the caller. The well-known bindings are the one
// dependency that legitimately resolves into a runtime namespace, so the
// resolver applies the reserved check to every other dependency itself rather
// than through ValidateNamespace.
func validateNamespaceShape(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	for _, part := range strings.Split(namespace, ".") {
		if !identifierPattern.MatchString(part) {
			return fmt.Errorf("invalid namespace segment %q in %q", part, namespace)
		}
	}
	return nil
}

type typeNamer struct {
	prefix string
}

func newTypeNamer(file *protoast.ProtoFile, sourceName string) (typeNamer, error) {
	if file == nil || file.Options == nil {
		return typeNamer{}, nil
	}

	raw, ok := file.Options[typePrefixOptionKey]
	if !ok {
		return typeNamer{}, nil
	}

	prefix, isString := raw.(string)
	if !isString || prefix == "" || !identifierPattern.MatchString(prefix) {
		message := fmt.Sprintf("must be a non-empty identifier fragment, got %s", optionValue(raw))
		return typeNamer{}, optionError(file, sourceName, typePrefixOptionKey, message)
	}

	return typeNamer{prefix: prefix}, nil
}

func optionValue(raw any) string {
	if value, ok := raw.(string); ok {
		return fmt.Sprintf("%q", value)
	}
	return fmt.Sprintf("%T(%v)", raw, raw)
}

func optionError(file *protoast.ProtoFile, sourceName, optionKey, message string) error {
	if file != nil {
		position := file.OptionPositions[optionKey]
		if position != (protoast.Position{}) {
			return fmt.Errorf("%s:%d:%d: error: %s %s",
				sourceName, position.Line, position.Column, optionKey, message)
		}
	}
	return fmt.Errorf("%s: error: %s %s", sourceName, optionKey, message)
}

// Name converts a proto identifier to a prefixed Foundry Script type identifier.
func (n typeNamer) Name(name string) string {
	return escapeIdentifier(n.prefix + normalizeTypeName(name))
}

// Reference converts a possibly-dotted proto type path to a prefixed Foundry Script reference.
func (n typeNamer) Reference(protoType string) string {
	parts := strings.Split(strings.TrimPrefix(protoType, "."), ".")
	converted := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		converted = append(converted, n.Name(part))
	}
	return strings.Join(converted, ".")
}

// TypeName converts a proto identifier to a Foundry Script type identifier.
func TypeName(name string) string {
	return (typeNamer{}).Name(name)
}

func normalizeTypeName(name string) string {
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

	return builder.String()
}

// runtimeTypeNames are the types foundry.proto exports. Every generated file
// imports that namespace, so a schema type of the same name would make the
// spelling ambiguous -- Foundry refuses to resolve a name two imported
// namespaces both export -- and would break the runtime calls the binding makes
// through it. Renaming is applied to declarations and references alike, so a
// dependency generated by this tool ends up with the same escaped name.
// TestRuntimeTypeNamesCoverEveryExportedRuntimeType keeps this in step with
// the runtime source, so a type added there cannot be forgotten here.
var runtimeTypeNames = map[string]bool{
	"BytesRead": true, "Codec": true, "FixedRead": true, "FloatRead": true,
	"Message": true, "ProtobufError": true, "SkipRead": true,
	"StringRead": true, "VarintRead": true, "Wire": true,
}

func escapeIdentifier(name string) string {
	switch name {
	case "Class", "ClassName", "Enum", "EnumName", "Extends", "Func", "Import", "Namespace", "Trait", "TraitName", "Uses", "Var":
		return name + "_"
	}
	if runtimeTypeNames[name] {
		return name + "_"
	}
	return name
}

type memberEscapeKind uint8

const (
	memberEscapeNone memberEscapeKind = iota
	memberEscapeKeyword
	memberEscapeGenerated
	memberEscapeEngineBuiltin
	memberEscapeEngineNative
)

type memberEscape struct {
	Kind         memberEscapeKind
	ReservedName string
}

func (e memberEscape) description() string {
	switch e.Kind {
	case memberEscapeKeyword:
		return "Foundry keyword"
	case memberEscapeGenerated:
		return "generated member"
	case memberEscapeEngineBuiltin:
		return fmt.Sprintf("built-in type %q", e.ReservedName)
	case memberEscapeEngineNative:
		return fmt.Sprintf("native class %q", e.ReservedName)
	default:
		return ""
	}
}

type plannedMemberName struct {
	Generated string
	Escape    memberEscape
}

// reservedFieldNames are the Foundry Script keywords that cannot appear after
// `var`, so a proto field carrying one of these names has to be renamed rather
// than emitted verbatim. Verified against the analyzer one word at a time;
// contextual keywords such as `match`, `uses`, `case` and `emit` are legal
// identifiers and are deliberately absent.
var reservedFieldNames = map[string]bool{
	"abstract": true, "and": true, "as": true, "assert": true, "await": true,
	"break": true, "breakpoint": true, "class": true, "const": true,
	"continue": true, "elif": true, "else": true, "enum": true, "extends": true,
	"false": true, "final": true, "for": true, "func": true, "if": true,
	"import": true, "in": true, "is": true, "namespace": true, "not": true,
	"null": true, "or": true, "pass": true, "return": true, "self": true,
	"signal": true, "static": true, "super": true, "trait": true, "true": true,
	"tuple": true, "var": true, "void": true, "while": true, "yield": true,
}

// These constants are the source of truth for method spellings emitted and
// reserved by every message binding.
const (
	fromBytesMethod      = "from_bytes"
	toBytesMethod        = "to_bytes"
	mergeFromBytesMethod = "merge_from_bytes"
)

// generatedMethodNames returns a fresh ordered inventory for naming and
// collision collection.
func generatedMethodNames() []string {
	return []string{
		fromBytesMethod,
		toBytesMethod,
		mergeFromBytesMethod,
	}
}

// generatedMemberNames are all members every message binding declares. A proto
// field with one of these names would replace the generated member rather than
// sit beside it, so it is renamed for the same reason a keyword is.
var generatedMemberNames = func() map[string]bool {
	names := map[string]bool{unknownFieldsMember: true}
	for _, methodName := range generatedMethodNames() {
		names[methodName] = true
	}
	return names
}()

func planNonEngineMemberName(name string) plannedMemberName {
	switch {
	case reservedFieldNames[name]:
		return plannedMemberName{
			Generated: name + "_",
			Escape:    memberEscape{Kind: memberEscapeKeyword, ReservedName: name},
		}
	case generatedMemberNames[name]:
		return plannedMemberName{
			Generated: name + "_",
			Escape:    memberEscape{Kind: memberEscapeGenerated, ReservedName: name},
		}
	}

	return plannedMemberName{Generated: name}
}

func planMemberName(name string) plannedMemberName {
	if planned := planNonEngineMemberName(name); planned.Escape.Kind != memberEscapeNone {
		return planned
	}

	if engineType, reserved := foundryEngineReservedTypes[name]; reserved {
		kind := memberEscapeEngineNative
		if engineType.kind == engineTypeBuiltin {
			kind = memberEscapeEngineBuiltin
		}
		return plannedMemberName{
			Generated: name + "_",
			Escape:    memberEscape{Kind: kind, ReservedName: name},
		}
	}

	return plannedMemberName{Generated: name}
}

func planOneofAlternativeName(name string) plannedMemberName {
	return planNonEngineMemberName(name)
}

// FieldName converts a proto field name to the member name it is emitted as.
// Foundry keywords, generator-owned members, and engine type names receive
// exactly one trailing underscore; all other names pass through unchanged.
func FieldName(name string) string {
	return planMemberName(name).Generated
}

// localName is the name of a variable the emitter introduces inside a generated
// function body. The shared prefix is what keeps every one of them clear of a
// field named `offset`, `data` or `result`.
func localName(parts ...string) string {
	return generatedPrefix + strings.Join(parts, "_")
}

// ValidateMemberName rejects the one proto name the emitter cannot represent.
// Everything else, leading underscores included, is passed through: protoc
// accepts those, so refusing them would reject schemas that build everywhere
// else.
func ValidateMemberName(messageName, kind, name string) error {
	if strings.HasPrefix(name, generatedPrefix) {
		return fmt.Errorf("%s %s.%s: the %s prefix is reserved for generated members",
			kind, messageName, name, generatedPrefix)
	}
	return nil
}
