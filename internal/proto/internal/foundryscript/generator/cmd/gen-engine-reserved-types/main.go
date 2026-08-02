// Package main generates the pinned Foundry engine type table.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

type namedAPIEntry struct {
	Name string `json:"name"`
}

type enumAPIEntry struct {
	Name   string          `json:"name"`
	Values []namedAPIEntry `json:"values"`
}

type classAPIEntry struct {
	Name       string          `json:"name"`
	Inherits   string          `json:"inherits"`
	Methods    []namedAPIEntry `json:"methods"`
	Properties []namedAPIEntry `json:"properties"`
	Signals    []namedAPIEntry `json:"signals"`
	Constants  []namedAPIEntry `json:"constants"`
	Enums      []enumAPIEntry  `json:"enums"`
}

type extensionAPI struct {
	BuiltinClasses []namedAPIEntry `json:"builtin_classes"`
	Classes        []classAPIEntry `json:"classes"`
}

type inheritedMemberKind uint8

const (
	inheritedMemberMethod inheritedMemberKind = iota + 1
	inheritedMemberProperty
	inheritedMemberSignal
	inheritedMemberConstant
	inheritedMemberEnum
	inheritedMemberEnumValue
)

type inheritedMember struct {
	Name  string
	Kind  inheritedMemberKind
	Owner string
}

type reservedTypes struct {
	Version          string
	Builtins         []string
	NativeClasses    []string
	InheritedMembers []inheritedMember
}

const generatedMessageBaseClass = "RefCounted"

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("gen-engine-reserved-types", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var apiPath string
	var version string
	var outputPath string
	flags.StringVar(&apiPath, "api", "", "path to Foundry's extension_api.json")
	flags.StringVar(&version, "version", "", "Foundry source version")
	flags.StringVar(&outputPath, "output", "", "path for the generated Go source")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	if apiPath == "" {
		return errors.New("--api is required")
	}
	if version == "" {
		return errors.New("--version is required")
	}
	if outputPath == "" {
		return errors.New("--output is required")
	}

	types, err := loadAPI(apiPath)
	if err != nil {
		return err
	}
	types.Version = version

	source, err := renderGo(types)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, source, 0o644); err != nil { //nolint:gosec // Generated Go source should be project-readable.
		return fmt.Errorf("write generated source %q: %w", outputPath, err)
	}
	if err := os.Chmod(outputPath, 0o644); err != nil { //nolint:gosec // Generated Go source should be project-readable.
		return fmt.Errorf("set generated source permissions %q: %w", outputPath, err)
	}

	return nil
}

func loadAPI(path string) (reservedTypes, error) {
	file, err := os.Open(path) //nolint:gosec // The caller explicitly supplies the extension API path.
	if err != nil {
		return reservedTypes{}, fmt.Errorf("open extension API %q: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	types, err := decodeAPI(file)
	if err != nil {
		return reservedTypes{}, fmt.Errorf("decode extension API %q: %w", path, err)
	}
	return types, nil
}

func decodeAPI(reader io.Reader) (reservedTypes, error) {
	var api extensionAPI
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&api); err != nil {
		return reservedTypes{}, fmt.Errorf("decode JSON: %w", err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return reservedTypes{}, errors.New("extension API contains multiple JSON values")
		}
		return reservedTypes{}, fmt.Errorf("decode trailing JSON: %w", err)
	}

	if api.BuiltinClasses == nil {
		return reservedTypes{}, errors.New("extension API builtin_classes is missing or null")
	}
	if api.Classes == nil {
		return reservedTypes{}, errors.New("extension API classes is missing or null")
	}

	builtins := make([]string, 0, len(api.BuiltinClasses)+1)
	builtinNames := make(map[string]struct{}, len(api.BuiltinClasses)+1)
	for _, entry := range api.BuiltinClasses {
		if entry.Name == "" {
			return reservedTypes{}, errors.New("built-in type has an empty name")
		}
		if _, exists := builtinNames[entry.Name]; exists {
			return reservedTypes{}, fmt.Errorf("duplicate built-in type %q", entry.Name)
		}
		builtinNames[entry.Name] = struct{}{}
		builtins = append(builtins, entry.Name)
	}

	nativeClasses := make([]string, 0, len(api.Classes))
	nativeClassNames := make(map[string]struct{}, len(api.Classes))
	for i := range api.Classes {
		entry := &api.Classes[i]
		if entry.Name == "" {
			return reservedTypes{}, errors.New("native class has an empty name")
		}
		if _, exists := nativeClassNames[entry.Name]; exists {
			return reservedTypes{}, fmt.Errorf("duplicate native class %q", entry.Name)
		}
		nativeClassNames[entry.Name] = struct{}{}
		nativeClasses = append(nativeClasses, entry.Name)
	}

	if _, exists := builtinNames["AsyncCallable"]; !exists {
		builtins = append(builtins, "AsyncCallable")
		builtinNames["AsyncCallable"] = struct{}{}
	}
	for _, name := range nativeClasses {
		if _, exists := builtinNames[name]; exists {
			return reservedTypes{}, fmt.Errorf("type %q appears in both categories", name)
		}
	}
	inherited, err := collectInheritedMembers(api.Classes, generatedMessageBaseClass)
	if err != nil {
		return reservedTypes{}, err
	}
	sort.Strings(builtins)
	sort.Strings(nativeClasses)

	return reservedTypes{
		Builtins:         builtins,
		NativeClasses:    nativeClasses,
		InheritedMembers: inherited,
	}, nil
}

func collectInheritedMembers(classes []classAPIEntry, base string) ([]inheritedMember, error) {
	byName := make(map[string]*classAPIEntry, len(classes))
	for i := range classes {
		class := &classes[i]
		byName[class.Name] = class
	}

	seenClasses := make(map[string]bool)
	seenMembers := make(map[string]bool)
	var members []inheritedMember
	for className := base; className != ""; {
		if seenClasses[className] {
			return nil, fmt.Errorf("generated message base class %q has an inheritance cycle at %q", base, className)
		}
		class, exists := byName[className]
		if !exists {
			if className == base {
				return nil, fmt.Errorf("generated message base class %q is missing", base)
			}
			return nil, fmt.Errorf("ancestor %q of generated message base class %q is missing", className, base)
		}
		seenClasses[className] = true

		categories := []struct {
			kind    inheritedMemberKind
			label   string
			entries []namedAPIEntry
		}{
			{kind: inheritedMemberMethod, label: "method", entries: class.Methods},
			{kind: inheritedMemberProperty, label: "property", entries: class.Properties},
			{kind: inheritedMemberSignal, label: "signal", entries: class.Signals},
			{kind: inheritedMemberConstant, label: "constant", entries: class.Constants},
		}
		for _, category := range categories {
			for _, entry := range category.entries {
				if err := addInheritedMember(&members, seenMembers, class.Name, category.label, entry.Name, category.kind); err != nil {
					return nil, err
				}
			}
		}
		for _, enum := range class.Enums {
			if err := addInheritedMember(&members, seenMembers, class.Name, "enum", enum.Name, inheritedMemberEnum); err != nil {
				return nil, err
			}
			for _, value := range enum.Values {
				if err := addInheritedMember(&members, seenMembers, class.Name, "enum value", value.Name, inheritedMemberEnumValue); err != nil {
					return nil, err
				}
			}
		}

		className = class.Inherits
	}

	sort.Slice(members, func(i, j int) bool {
		return members[i].Name < members[j].Name
	})
	return members, nil
}

func addInheritedMember(
	members *[]inheritedMember,
	seen map[string]bool,
	owner, category, name string,
	kind inheritedMemberKind,
) error {
	if name == "" {
		return fmt.Errorf("native class %q has a %s with an empty name", owner, category)
	}
	if seen[name] {
		return nil
	}
	seen[name] = true
	*members = append(*members, inheritedMember{Name: name, Kind: kind, Owner: owner})
	return nil
}

func renderGo(types reservedTypes) ([]byte, error) {
	builtins := append([]string(nil), types.Builtins...)
	nativeClasses := append([]string(nil), types.NativeClasses...)
	inheritedMembers := append([]inheritedMember(nil), types.InheritedMembers...)
	sort.Strings(builtins)
	sort.Strings(nativeClasses)
	sort.Slice(inheritedMembers, func(i, j int) bool {
		return inheritedMembers[i].Name < inheritedMembers[j].Name
	})

	var source strings.Builder
	source.WriteString("// Code generated by gen-engine-reserved-types. DO NOT EDIT.\n\n")
	source.WriteString("package fsgenerator\n\n")
	source.WriteString("type engineTypeKind uint8\n\n")
	source.WriteString("const (\n")
	source.WriteString("\tengineTypeBuiltin engineTypeKind = iota + 1\n")
	source.WriteString("\tengineTypeNativeClass\n")
	source.WriteString(")\n\n")
	source.WriteString("type engineTypeEntry struct {\n")
	source.WriteString("\tkind engineTypeKind\n")
	source.WriteString("}\n\n")
	source.WriteString("type engineMemberKind uint8\n\n")
	source.WriteString("const (\n")
	source.WriteString("\tengineMemberMethod engineMemberKind = iota + 1\n")
	source.WriteString("\tengineMemberProperty\n")
	source.WriteString("\tengineMemberSignal\n")
	source.WriteString("\tengineMemberConstant\n")
	source.WriteString("\tengineMemberEnum\n")
	source.WriteString("\tengineMemberEnumValue\n")
	source.WriteString(")\n\n")
	source.WriteString("type engineMemberEntry struct {\n")
	source.WriteString("\tkind engineMemberKind\n")
	source.WriteString("\towner string\n")
	source.WriteString("}\n\n")
	source.WriteString("const foundryEngineTypeSourceVersion = ")
	source.WriteString(strconv.Quote(types.Version))
	source.WriteString("\n\n")
	source.WriteString("const foundryEngineMessageBaseClass = ")
	source.WriteString(strconv.Quote(generatedMessageBaseClass))
	source.WriteString("\n\n")
	source.WriteString("var foundryEngineReservedTypes = map[string]engineTypeEntry{\n")
	for _, name := range builtins {
		source.WriteString("\t")
		source.WriteString(strconv.Quote(name))
		source.WriteString(": {kind: engineTypeBuiltin},\n")
	}
	for _, name := range nativeClasses {
		source.WriteString("\t")
		source.WriteString(strconv.Quote(name))
		source.WriteString(": {kind: engineTypeNativeClass},\n")
	}
	source.WriteString("}\n")
	source.WriteString("\nvar foundryEngineReservedMembers = map[string]engineMemberEntry{\n")
	for _, member := range inheritedMembers {
		kind, err := engineMemberKindIdentifier(member.Kind)
		if err != nil {
			return nil, err
		}
		source.WriteString("\t")
		source.WriteString(strconv.Quote(member.Name))
		source.WriteString(": {kind: ")
		source.WriteString(kind)
		source.WriteString(", owner: ")
		source.WriteString(strconv.Quote(member.Owner))
		source.WriteString("},\n")
	}
	source.WriteString("}\n")

	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated Go source: %w", err)
	}
	return formatted, nil
}

func engineMemberKindIdentifier(kind inheritedMemberKind) (string, error) {
	switch kind {
	case inheritedMemberMethod:
		return "engineMemberMethod", nil
	case inheritedMemberProperty:
		return "engineMemberProperty", nil
	case inheritedMemberSignal:
		return "engineMemberSignal", nil
	case inheritedMemberConstant:
		return "engineMemberConstant", nil
	case inheritedMemberEnum:
		return "engineMemberEnum", nil
	case inheritedMemberEnumValue:
		return "engineMemberEnumValue", nil
	default:
		return "", fmt.Errorf("unknown inherited member kind %d", kind)
	}
}
