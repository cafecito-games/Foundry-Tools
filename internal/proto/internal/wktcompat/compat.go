// Package wktcompat verifies that a schema identified by a canonical
// google/protobuf import path can be represented by the runtime binding that
// Foundry Tools substitutes for it.
package wktcompat

import (
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	protoparse "github.com/cafecito-games/foundry-tools/internal/proto/internal/parser"
	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
)

// SchemaFile pairs a parsed protobuf schema with the canonical import path by
// which its front end knows it.
type SchemaFile struct {
	ImportPath string
	File       *protoast.ProtoFile
}

type mismatch struct {
	path    string
	subject string
	detail  string
}

func (m mismatch) String() string {
	if m.subject == "" {
		return m.path + ": " + m.detail
	}
	return m.path + ": " + m.subject + ": " + m.detail
}

type mismatchList []mismatch

func (m mismatchList) Error() string {
	lines := make([]string, len(m))
	for i := range m {
		lines[i] = m[i].String()
	}
	return strings.Join(lines, "\n")
}

var canonicalOnce = sync.OnceValues(loadCanonical)

// Check reports every incompatibility in deterministic canonical order.
// Schemas whose paths are not one of the runtime's supported well-known files
// are ignored.
func Check(files []SchemaFile) error {
	canonical, err := canonicalOnce()
	if err != nil {
		return err
	}

	candidates := append([]SchemaFile(nil), files...)
	sort.SliceStable(candidates, func(i, j int) bool {
		return normalizePath(candidates[i].ImportPath) < normalizePath(candidates[j].ImportPath)
	})

	var mismatches mismatchList
	for _, candidate := range candidates {
		name := normalizePath(candidate.ImportPath)
		expected, ok := canonical[name]
		if !ok {
			continue
		}
		actual, normalizationErrors := normalize(candidate.File, name)
		mismatches = append(mismatches, normalizationErrors...)
		if len(normalizationErrors) == 0 {
			mismatches = append(mismatches, compare(expected, actual, name)...)
		}
	}
	if len(mismatches) == 0 {
		return nil
	}
	return mismatches
}

func loadCanonical() (map[string]schemaShape, error) {
	out := make(map[string]schemaShape, len(wellknown.Files()))
	for _, name := range wellknown.Files() {
		source, err := wellknown.Source(name)
		if err != nil {
			return nil, err
		}
		tokens, err := protoparse.Tokenize(string(source), name)
		if err != nil {
			return nil, fmt.Errorf("parse canonical %s: %w", name, err)
		}
		file, err := protoparse.Parse(tokens, name)
		if err != nil {
			return nil, fmt.Errorf("parse canonical %s: %w", name, err)
		}
		shape, normalizationErrors := normalize(file, name)
		if len(normalizationErrors) != 0 {
			return nil, fmt.Errorf("normalize canonical %s: %s", name, normalizationErrors.Error())
		}
		out[name] = shape
	}
	return out, nil
}

func normalizePath(name string) string {
	return path.Clean(strings.ReplaceAll(name, `\`, "/"))
}

type declarationKind string

const (
	kindMessage declarationKind = "message"
	kindEnum    declarationKind = "enum"
)

type schemaShape struct {
	packageName  string
	messages     map[string]messageShape
	messageOrder []string
	enums        map[string]enumShape
	enumOrder    []string
	kinds        map[string]declarationKind
}

type messageShape struct {
	fields       map[int]fieldShape
	fieldNumbers []int
}

type enumShape struct {
	numbers     map[int]bool
	numberOrder []int
}

type fieldShape struct {
	canonicalName string
	cardinality   string
	value         typeShape
	mapKey        *typeShape
	oneof         string
}

func (f fieldShape) String() string {
	if f.mapKey != nil {
		return fmt.Sprintf("map<%s, %s>", f.mapKey.String(), f.value.String())
	}
	return f.cardinality + " " + f.value.String()
}

type typeShape struct {
	kind declarationKind
	name string
}

func (t typeShape) String() string {
	return t.name
}

var scalarTypes = map[string]bool{
	"double": true, "float": true,
	"int32": true, "int64": true, "uint32": true, "uint64": true,
	"sint32": true, "sint64": true, "fixed32": true, "fixed64": true,
	"sfixed32": true, "sfixed64": true,
	"bool": true, "string": true, "bytes": true,
}

func normalize(file *protoast.ProtoFile, importPath string) (schemaShape, mismatchList) {
	shape := schemaShape{
		messages: map[string]messageShape{},
		enums:    map[string]enumShape{},
		kinds:    map[string]declarationKind{},
	}
	if file == nil {
		return shape, mismatchList{{path: importPath, detail: "schema is missing"}}
	}
	shape.packageName = file.Package

	var errors mismatchList
	for _, message := range file.Messages {
		registerMessage(&shape, message, qualify(file.Package, "", message.Name), importPath, &errors)
	}
	for _, enum := range file.Enums {
		registerEnum(&shape, qualify(file.Package, "", enum.Name), importPath, &errors)
	}
	for _, message := range file.Messages {
		populateMessage(&shape, message, qualify(file.Package, "", message.Name), importPath, &errors)
	}
	for _, enum := range file.Enums {
		populateEnum(&shape, enum, qualify(file.Package, "", enum.Name))
	}
	return shape, errors
}

func registerMessage(shape *schemaShape, message *protoast.Message, fullName, importPath string, errors *mismatchList) {
	if existing := shape.kinds[fullName]; existing != "" {
		*errors = append(*errors, mismatch{path: importPath, subject: fullName, detail: "duplicate type declaration"})
		return
	}
	shape.kinds[fullName] = kindMessage
	shape.messageOrder = append(shape.messageOrder, fullName)
	shape.messages[fullName] = messageShape{}
	for _, nested := range message.NestedMessages {
		registerMessage(shape, nested, fullName+"."+nested.Name, importPath, errors)
	}
	for _, enum := range message.NestedEnums {
		registerEnum(shape, fullName+"."+enum.Name, importPath, errors)
	}
}

func registerEnum(shape *schemaShape, fullName, importPath string, errors *mismatchList) {
	if existing := shape.kinds[fullName]; existing != "" {
		*errors = append(*errors, mismatch{path: importPath, subject: fullName, detail: "duplicate type declaration"})
		return
	}
	shape.kinds[fullName] = kindEnum
	shape.enumOrder = append(shape.enumOrder, fullName)
	shape.enums[fullName] = enumShape{}
}

func populateMessage(shape *schemaShape, message *protoast.Message, fullName, importPath string, errors *mismatchList) {
	result := messageShape{fields: map[int]fieldShape{}}
	add := func(number int, field fieldShape) {
		if _, duplicate := result.fields[number]; duplicate {
			*errors = append(*errors, mismatch{
				path: importPath, subject: fullName,
				detail: fmt.Sprintf("duplicate field number #%d", number),
			})
			return
		}
		result.fields[number] = field
		result.fieldNumbers = append(result.fieldNumbers, number)
	}
	for _, field := range message.Fields {
		add(field.Number, fieldShape{
			canonicalName: field.Name,
			cardinality:   cardinality(field),
			value:         resolveType(shape, fullName, field.FieldType, field.FullTypePath, field.IsEnum),
		})
	}
	for _, mapField := range message.Maps {
		key := resolveType(shape, fullName, mapField.KeyType, "", false)
		add(mapField.Number, fieldShape{
			canonicalName: mapField.Name,
			cardinality:   "map",
			mapKey:        &key,
			value: resolveType(shape, fullName, mapField.ValueType,
				mapField.FullValueTypePath, mapField.ValueIsEnum),
		})
	}
	for index, oneof := range message.Oneofs {
		group := fmt.Sprintf("%s#%d", fullName, index)
		for _, field := range oneof.Fields {
			add(field.Number, fieldShape{
				canonicalName: field.Name,
				cardinality:   cardinality(field),
				value:         resolveType(shape, fullName, field.FieldType, field.FullTypePath, field.IsEnum),
				oneof:         group,
			})
		}
	}
	sort.Ints(result.fieldNumbers)
	shape.messages[fullName] = result

	for _, nested := range message.NestedMessages {
		populateMessage(shape, nested, fullName+"."+nested.Name, importPath, errors)
	}
	for _, enum := range message.NestedEnums {
		populateEnum(shape, enum, fullName+"."+enum.Name)
	}
}

func populateEnum(shape *schemaShape, enum *protoast.Enum, fullName string) {
	result := enumShape{numbers: map[int]bool{}}
	for _, value := range enum.Values {
		if result.numbers[value.Number] {
			continue
		}
		result.numbers[value.Number] = true
		result.numberOrder = append(result.numberOrder, value.Number)
	}
	sort.Ints(result.numberOrder)
	shape.enums[fullName] = result
}

func qualify(packageName, parent, name string) string {
	if parent != "" {
		return parent + "." + name
	}
	if packageName != "" {
		return packageName + "." + name
	}
	return name
}

func cardinality(field *protoast.Field) string {
	switch {
	case field.Repeated:
		return "repeated"
	case field.Optional:
		return "optional"
	default:
		return "singular"
	}
}

func resolveType(shape *schemaShape, scope, raw, fullPath string, enumHint bool) typeShape {
	if scalarTypes[raw] {
		return typeShape{name: raw}
	}
	if fullPath != "" {
		name := strings.TrimPrefix(fullPath, ".")
		return typeShape{kind: resolvedKind(shape, name, enumHint), name: name}
	}
	if strings.HasPrefix(raw, ".") {
		name := strings.TrimPrefix(raw, ".")
		return typeShape{kind: resolvedKind(shape, name, enumHint), name: name}
	}
	for current := scope; current != ""; {
		candidate := current + "." + raw
		if kind := shape.kinds[candidate]; kind != "" {
			return typeShape{kind: kind, name: candidate}
		}
		cut := strings.LastIndex(current, ".")
		if cut < 0 {
			break
		}
		current = current[:cut]
	}
	return typeShape{kind: resolvedKind(shape, raw, enumHint), name: raw}
}

func resolvedKind(shape *schemaShape, name string, enumHint bool) declarationKind {
	if kind := shape.kinds[name]; kind != "" {
		return kind
	}
	if enumHint {
		return kindEnum
	}
	return kindMessage
}

func compare(expected, actual schemaShape, importPath string) mismatchList {
	if actual.packageName != expected.packageName {
		return mismatchList{{
			path:   importPath,
			detail: fmt.Sprintf("expected package %s; found %s", expected.packageName, actual.packageName),
		}}
	}

	var out mismatchList
	for _, fullName := range expected.messageOrder {
		want := expected.messages[fullName]
		got, ok := actual.messages[fullName]
		if !ok {
			if actual.kinds[fullName] == kindEnum {
				out = append(out, mismatch{path: importPath, subject: fullName, detail: "expected message; found enum"})
			} else {
				out = append(out, mismatch{path: importPath, detail: "missing canonical message " + fullName})
			}
			continue
		}
		out = append(out, compareMessage(importPath, fullName, want, got)...)
	}
	for _, fullName := range expected.enumOrder {
		want := expected.enums[fullName]
		got, ok := actual.enums[fullName]
		if !ok && actual.kinds[fullName] == kindMessage {
			out = append(out, mismatch{path: importPath, subject: fullName, detail: "expected enum; found message"})
		} else if !ok {
			out = append(out, mismatch{path: importPath, detail: "missing canonical enum " + fullName})
		} else {
			for _, number := range want.numberOrder {
				if !got.numbers[number] {
					out = append(out, mismatch{
						path: importPath, subject: fullName,
						detail: fmt.Sprintf("missing canonical enum number %d", number),
					})
				}
			}
		}
	}
	return out
}

func compareMessage(importPath, fullName string, expected, actual messageShape) mismatchList {
	var out mismatchList
	for _, number := range expected.fieldNumbers {
		want := expected.fields[number]
		subject := fmt.Sprintf("%s.%s (#%d)", fullName, want.canonicalName, number)
		got, ok := actual.fields[number]
		if !ok {
			out = append(out, mismatch{path: importPath, subject: subject, detail: "missing canonical field"})
			continue
		}
		if want.String() != got.String() || want.value.kind != got.value.kind {
			out = append(out, mismatch{
				path: importPath, subject: subject,
				detail: fmt.Sprintf("expected %s; found %s", want.String(), got.String()),
			})
		}
		if want.oneof == "" && got.oneof != "" {
			out = append(out, mismatch{
				path: importPath, subject: subject,
				detail: "expected ordinary field; found oneof member",
			})
		}
	}

	canonicalGroups := map[string][]int{}
	var groupOrder []string
	for _, number := range expected.fieldNumbers {
		group := expected.fields[number].oneof
		if group == "" {
			continue
		}
		if _, seen := canonicalGroups[group]; !seen {
			groupOrder = append(groupOrder, group)
		}
		canonicalGroups[group] = append(canonicalGroups[group], number)
	}
	for _, group := range groupOrder {
		wantNumbers := canonicalGroups[group]
		candidateGroup := ""
		complete := true
		for _, number := range wantNumbers {
			got, ok := actual.fields[number]
			if !ok {
				complete = false
				break
			}
			if candidateGroup == "" {
				candidateGroup = got.oneof
			}
		}
		if !complete {
			continue
		}

		var gotNumbers []int
		if candidateGroup != "" {
			for _, number := range expected.fieldNumbers {
				if got, ok := actual.fields[number]; ok && got.oneof == candidateGroup {
					gotNumbers = append(gotNumbers, number)
				}
			}
		}
		if !slices.Equal(wantNumbers, gotNumbers) {
			out = append(out, mismatch{
				path: importPath, subject: fullName,
				detail: fmt.Sprintf("expected canonical oneof fields %v; found incompatible oneof partition", wantNumbers),
			})
		}
	}
	return out
}
