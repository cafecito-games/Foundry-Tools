package protodesc

import (
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/cafecito-games/foundry-tools/internal/foundrytoolspb"
	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
)

// scalarTypeNames maps the wire-type enum to the proto scalar type spelling.
// TYPE_MESSAGE, TYPE_ENUM, and TYPE_GROUP are not in this map; they use TypeName.
var scalarTypeNames = map[descriptorpb.FieldDescriptorProto_Type]string{
	descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:   "double",
	descriptorpb.FieldDescriptorProto_TYPE_FLOAT:    "float",
	descriptorpb.FieldDescriptorProto_TYPE_INT64:    "int64",
	descriptorpb.FieldDescriptorProto_TYPE_UINT64:   "uint64",
	descriptorpb.FieldDescriptorProto_TYPE_INT32:    "int32",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED64:  "fixed64",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED32:  "fixed32",
	descriptorpb.FieldDescriptorProto_TYPE_BOOL:     "bool",
	descriptorpb.FieldDescriptorProto_TYPE_STRING:   "string",
	descriptorpb.FieldDescriptorProto_TYPE_BYTES:    "bytes",
	descriptorpb.FieldDescriptorProto_TYPE_UINT32:   "uint32",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED32: "sfixed32",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED64: "sfixed64",
	descriptorpb.FieldDescriptorProto_TYPE_SINT32:   "sint32",
	descriptorpb.FieldDescriptorProto_TYPE_SINT64:   "sint64",
}

// converter holds cross-file state used while converting descriptors.
type converter struct {
	// typeRegistry maps a fully-qualified type path (no leading dot) to the
	// proto file in which it was declared. Used to populate Field.SourceFile.
	typeRegistry map[string]string
	// typeNames maps a fully-qualified type path (no leading dot) to the type
	// name relative to its defining file.
	typeNames map[string]string
}

// FromCodeGeneratorRequest converts every FileDescriptorProto in the request
// (proto_file) to a ProtoFile AST. The full proto_file list is used to build
// a cross-file type registry so that message/enum field references resolve to
// their source file path.
func FromCodeGeneratorRequest(req *pluginpb.CodeGeneratorRequest) ([]*protoast.ProtoFile, error) {
	c := newConverter(req.GetProtoFile())
	out := make([]*protoast.ProtoFile, 0, len(req.GetProtoFile()))
	sourceFiles := map[*protoast.ProtoFile]string{}
	for _, fdp := range req.GetProtoFile() {
		file, err := c.convertFile(fdp)
		if err != nil {
			return nil, err
		}
		sourceFiles[file] = fdp.GetName()
		out = append(out, file)
	}
	resolveCrossFileEnumValues(out, sourceFiles)
	return out, nil
}

// FromFileDescriptorProto converts a single FileDescriptorProto to a
// ProtoFile AST. Cross-file source-file resolution is limited to types defined
// within the given descriptor.
func FromFileDescriptorProto(fdp *descriptorpb.FileDescriptorProto) (*protoast.ProtoFile, error) {
	c := newConverter([]*descriptorpb.FileDescriptorProto{fdp})
	file, err := c.convertFile(fdp)
	if err != nil {
		return nil, err
	}
	resolveCrossFileEnumValues([]*protoast.ProtoFile{file}, map[*protoast.ProtoFile]string{file: fdp.GetName()})
	return file, nil
}

func newConverter(allFiles []*descriptorpb.FileDescriptorProto) *converter {
	c := &converter{
		typeRegistry: map[string]string{},
		typeNames:    map[string]string{},
	}
	for _, fd := range allFiles {
		pkg := fd.GetPackage()
		source := fd.GetName()
		for _, m := range fd.GetMessageType() {
			c.registerMessage(m, pkg, source, "", "")
		}
		for _, e := range fd.GetEnumType() {
			full := e.GetName()
			if pkg != "" {
				full = pkg + "." + full
			}
			c.typeRegistry[full] = source
			c.typeNames[full] = e.GetName()
		}
	}
	return c
}

func (c *converter) registerMessage(m *descriptorpb.DescriptorProto, pkg, source, parent, relativeParent string) {
	var full string
	switch {
	case parent != "":
		full = parent + "." + m.GetName()
	case pkg != "":
		full = pkg + "." + m.GetName()
	default:
		full = m.GetName()
	}
	relative := m.GetName()
	if relativeParent != "" {
		relative = relativeParent + "." + m.GetName()
	}
	c.typeRegistry[full] = source
	c.typeNames[full] = relative
	for _, nested := range m.GetNestedType() {
		if nested.GetOptions().GetMapEntry() {
			continue
		}
		c.registerMessage(nested, pkg, source, full, relative)
	}
	for _, e := range m.GetEnumType() {
		enumFull := full + "." + e.GetName()
		c.typeRegistry[enumFull] = source
		c.typeNames[enumFull] = relative + "." + e.GetName()
	}
}

func (c *converter) convertFile(fd *descriptorpb.FileDescriptorProto) (*protoast.ProtoFile, error) {
	syntax := fd.GetSyntax()
	if syntax == "" {
		syntax = "proto3"
	}
	docs := sourceDocs(fd)
	file := &protoast.ProtoFile{
		Syntax:  syntax,
		Package: fd.GetPackage(),
		Options: map[string]any{},
	}

	if fdOpts := fd.GetOptions(); fdOpts != nil {
		copyFileOption(file.Options, fdOpts, foundrytoolspb.E_Namespace, "(foundrytools.namespace)")
		copyFileOption(file.Options, fdOpts, foundrytoolspb.E_TypePrefix, "(foundrytools.type_prefix)")
		copyFileOption(file.Options, fdOpts, foundrytoolspb.E_EmitRuntime, "(foundrytools.emit_runtime)")
	}
	// file.OptionPositions is intentionally left nil here: FileDescriptorProto
	// carries no per-option source position, so downstream error messages
	// degrade to the position-less form.

	publicSet := map[int32]struct{}{}
	for _, idx := range fd.GetPublicDependency() {
		publicSet[idx] = struct{}{}
	}
	for i, dep := range fd.GetDependency() {
		_, public := publicSet[int32(i)]
		file.Imports = append(file.Imports, &protoast.Import{Path: dep, Public: public})
	}

	for i, e := range fd.GetEnumType() {
		file.Enums = append(file.Enums, c.convertEnum(e, docs, []int32{5, int32(i)}))
	}
	for i, m := range fd.GetMessageType() {
		msg, err := c.convertMessage(m, fd.GetName(), m.GetName(), docs, []int32{4, int32(i)})
		if err != nil {
			return nil, err
		}
		file.Messages = append(file.Messages, msg)
	}
	return file, nil
}

type docIndex map[string][]string

func sourceDocs(fd *descriptorpb.FileDescriptorProto) docIndex {
	out := docIndex{}
	if fd.GetSourceCodeInfo() == nil {
		return out
	}
	for _, location := range fd.GetSourceCodeInfo().GetLocation() {
		doc := normalizeDocLines(location.GetLeadingComments(), location.GetTrailingComments())
		if len(doc) == 0 {
			continue
		}
		out[pathKey(location.GetPath())] = doc
	}
	return out
}

func (d docIndex) get(path []int32) []string {
	if len(d) == 0 {
		return nil
	}
	doc := d[pathKey(path)]
	if len(doc) == 0 {
		return nil
	}
	return append([]string(nil), doc...)
}

func normalizeDocLines(parts ...string) []string {
	var out []string
	for _, part := range parts {
		part = strings.ReplaceAll(part, "\r\n", "\n")
		part = strings.ReplaceAll(part, "\r", "\n")
		lines := strings.Split(part, "\n")
		for i, line := range lines {
			lines[i] = strings.TrimSpace(line)
		}
		for len(lines) > 0 && lines[0] == "" {
			lines = lines[1:]
		}
		for len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		out = append(out, lines...)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func pathKey(path []int32) string {
	var builder strings.Builder
	for i, part := range path {
		if i > 0 {
			builder.WriteByte('.')
		}
		builder.WriteString(strconv.FormatInt(int64(part), 10))
	}
	return builder.String()
}

func pathAppend(path []int32, parts ...int32) []int32 {
	out := make([]int32, 0, len(path)+len(parts))
	out = append(out, path...)
	out = append(out, parts...)
	return out
}

func copyFileOption(options map[string]any, fdOpts *descriptorpb.FileOptions, ext protoreflect.ExtensionType, key string) {
	if !proto.HasExtension(fdOpts, ext) {
		return
	}
	options[key] = proto.GetExtension(fdOpts, ext)
}

func (c *converter) convertMessage(
	d *descriptorpb.DescriptorProto,
	sourceFile string,
	relativeScope string,
	docs docIndex,
	path []int32,
) (*protoast.Message, error) {
	msg := &protoast.Message{
		Name:    d.GetName(),
		Doc:     docs.get(path),
		Options: map[string]any{},
	}

	// Index nested map-entry types so we can dispatch map fields.
	mapEntries := map[string]*descriptorpb.DescriptorProto{}
	for _, nested := range d.GetNestedType() {
		if nested.GetOptions().GetMapEntry() {
			mapEntries[nested.GetName()] = nested
		}
	}

	oneofNames := make([]string, len(d.GetOneofDecl()))
	for i, o := range d.GetOneofDecl() {
		oneofNames[i] = o.GetName()
	}

	var regularFields []*protoast.Field
	var mapFields []*protoast.MapField

	for i, f := range d.GetField() {
		fieldDoc := docs.get(pathAppend(path, 2, int32(i)))
		if f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED &&
			f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			short := lastSegment(f.GetTypeName())
			if entry, ok := mapEntries[short]; ok {
				mf, err := c.convertMapField(f, entry, sourceFile, relativeScope, fieldDoc)
				if err != nil {
					return nil, err
				}
				mapFields = append(mapFields, mf)
				continue
			}
		}

		field := c.convertField(f, sourceFile, relativeScope, fieldDoc)
		if f.OneofIndex != nil {
			idx := int(f.GetOneofIndex())
			if idx >= 0 && idx < len(oneofNames) {
				field.OneofParent = oneofNames[idx]
			}
		}
		regularFields = append(regularFields, field)
	}

	// Convert oneofs. Synthetic oneofs (proto3 optional) are not real oneofs:
	// Python suppresses them and keeps the field as a regular optional field.
	var oneofs []*protoast.Oneof
	oneofFieldSet := map[*protoast.Field]struct{}{}
	for i, o := range d.GetOneofDecl() {
		var fields []*protoast.Field
		for _, f := range regularFields {
			if f.OneofParent == o.GetName() {
				fields = append(fields, f)
			}
		}
		isSynthetic := len(fields) == 1 && fields[0].Optional && strings.HasPrefix(o.GetName(), "_")
		if isSynthetic {
			fields[0].OneofParent = ""
			continue
		}
		for _, f := range fields {
			oneofFieldSet[f] = struct{}{}
		}
		oneofs = append(oneofs, &protoast.Oneof{
			Name:    o.GetName(),
			Doc:     docs.get(pathAppend(path, 8, int32(i))),
			Fields:  fields,
			Options: map[string]any{},
		})
	}

	// Strip oneof-owned fields from the regular field list.
	if len(oneofFieldSet) > 0 {
		filtered := regularFields[:0]
		for _, f := range regularFields {
			if _, isOneof := oneofFieldSet[f]; !isOneof {
				filtered = append(filtered, f)
			}
		}
		regularFields = filtered
	}

	msg.Fields = regularFields
	msg.Maps = mapFields
	msg.Oneofs = oneofs

	for i, nested := range d.GetNestedType() {
		if nested.GetOptions().GetMapEntry() {
			continue
		}
		nestedScope := relativeScope + "." + nested.GetName()
		nm, err := c.convertMessage(nested, sourceFile, nestedScope, docs, pathAppend(path, 3, int32(i)))
		if err != nil {
			return nil, err
		}
		msg.NestedMessages = append(msg.NestedMessages, nm)
	}
	for i, e := range d.GetEnumType() {
		msg.NestedEnums = append(msg.NestedEnums, c.convertEnum(e, docs, pathAppend(path, 4, int32(i))))
	}

	// Reserved ranges: descriptor end is exclusive; AST uses inclusive.
	var ranges []protoast.ReservedRange
	for _, r := range d.GetReservedRange() {
		start := int(r.GetStart())
		end := int(r.GetEnd()) - 1
		ranges = append(ranges, protoast.ReservedRange{Start: start, End: end})
	}
	if len(ranges) > 0 || len(d.GetReservedName()) > 0 {
		msg.Reserved = []*protoast.Reserved{{
			Numbers: ranges,
			Names:   append([]string(nil), d.GetReservedName()...),
		}}
	}

	return msg, nil
}

func (c *converter) convertField(
	f *descriptorpb.FieldDescriptorProto,
	sourceFile string,
	relativeScope string,
	doc []string,
) *protoast.Field {
	field := &protoast.Field{
		Name:     f.GetName(),
		Doc:      doc,
		Number:   int(f.GetNumber()),
		Repeated: f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED,
		Optional: f.GetProto3Optional(),
		Options:  map[string]any{},
	}

	switch f.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		fullPath := strings.TrimPrefix(f.GetTypeName(), ".")
		field.FullTypePath = fullPath
		field.FieldType = c.typeNames[fullPath]
		field.SourceFile = c.typeRegistry[fullPath]
		if field.SourceFile == sourceFile {
			field.FieldType = localReferenceName(field.FieldType, relativeScope)
		}
		field.IsEnum = f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM
	default:
		if name, ok := scalarTypeNames[f.GetType()]; ok {
			field.FieldType = name
		} else {
			field.FieldType = "unknown"
		}
	}

	if opts := f.GetOptions(); opts != nil && opts.Packed != nil {
		field.Options["packed"] = opts.GetPacked()
	}

	return field
}

func (c *converter) convertEnum(e *descriptorpb.EnumDescriptorProto, docs docIndex, path []int32) *protoast.Enum {
	out := &protoast.Enum{
		Name:    e.GetName(),
		Doc:     docs.get(path),
		Options: map[string]any{},
	}
	for i, v := range e.GetValue() {
		out.Values = append(out.Values, &protoast.EnumValue{
			Name:    v.GetName(),
			Doc:     docs.get(pathAppend(path, 2, int32(i))),
			Number:  int(v.GetNumber()),
			Options: map[string]any{},
		})
	}
	if e.GetOptions().GetAllowAlias() {
		out.Options["allow_alias"] = true
	}
	return out
}

func (c *converter) convertMapField(
	f *descriptorpb.FieldDescriptorProto,
	entry *descriptorpb.DescriptorProto,
	sourceFile string,
	relativeScope string,
	doc []string,
) (*protoast.MapField, error) {
	if len(entry.GetField()) != 2 {
		return nil, &mapEntryError{name: f.GetName()}
	}
	var keyDescriptor, valueDescriptor *descriptorpb.FieldDescriptorProto
	for _, ef := range entry.GetField() {
		switch ef.GetName() {
		case "key":
			keyDescriptor = ef
		case "value":
			valueDescriptor = ef
		}
	}
	if keyDescriptor == nil || valueDescriptor == nil {
		// Fall back to positional: descriptors guarantee key=1, value=2.
		keyDescriptor = entry.GetField()[0]
		valueDescriptor = entry.GetField()[1]
	}

	mf := &protoast.MapField{
		Name:    f.GetName(),
		Doc:     doc,
		Number:  int(f.GetNumber()),
		Options: map[string]any{},
	}

	if name, ok := scalarTypeNames[keyDescriptor.GetType()]; ok {
		mf.KeyType = name
	} else {
		mf.KeyType = "unknown"
	}

	switch valueDescriptor.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		fullPath := strings.TrimPrefix(valueDescriptor.GetTypeName(), ".")
		mf.FullValueTypePath = fullPath
		mf.ValueType = c.typeNames[fullPath]
		mf.ValueSourceFile = c.typeRegistry[fullPath]
		if mf.ValueSourceFile == sourceFile {
			mf.ValueType = localReferenceName(mf.ValueType, relativeScope)
		}
		mf.ValueIsEnum = valueDescriptor.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM
	default:
		if name, ok := scalarTypeNames[valueDescriptor.GetType()]; ok {
			mf.ValueType = name
		} else {
			mf.ValueType = "unknown"
		}
	}

	return mf, nil
}

type mapEntryError struct {
	name string
}

func (e *mapEntryError) Error() string {
	return "invalid map entry for field " + e.name
}

func lastSegment(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// resolveCrossFileEnumValues walks every file's enums (including those nested
// inside messages) to build a fully-qualified-name index, then attaches the
// resolved enum's values to fields that reference an enum from another file.
// Same-file references are skipped because the generator can locate the enum
// AST node directly.
func resolveCrossFileEnumValues(files []*protoast.ProtoFile, sourceFiles map[*protoast.ProtoFile]string) {
	index := map[string]*protoast.Enum{}
	for _, file := range files {
		prefix := ""
		if file.Package != "" {
			prefix = file.Package + "."
		}
		for _, e := range file.Enums {
			index[prefix+e.Name] = e
		}
		for _, m := range file.Messages {
			indexNestedEnums(m, prefix+m.Name, index)
		}
	}
	for _, file := range files {
		for _, m := range file.Messages {
			attachEnumValues(m, sourceFiles[file], index)
		}
	}
}

func indexNestedEnums(m *protoast.Message, scope string, index map[string]*protoast.Enum) {
	for _, e := range m.NestedEnums {
		index[scope+"."+e.Name] = e
	}
	for _, nested := range m.NestedMessages {
		indexNestedEnums(nested, scope+"."+nested.Name, index)
	}
}

func attachEnumValues(m *protoast.Message, currentSourceFile string, index map[string]*protoast.Enum) {
	resolve := func(f *protoast.Field) {
		if !f.IsEnum || f.FullTypePath == "" {
			return
		}
		if f.SourceFile == currentSourceFile {
			return
		}
		if e, ok := index[f.FullTypePath]; ok {
			f.EnumValues = e.Values
		}
	}
	for _, f := range m.Fields {
		resolve(f)
	}
	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			resolve(f)
		}
	}
	for _, nested := range m.NestedMessages {
		attachEnumValues(nested, currentSourceFile, index)
	}
}

func localReferenceName(typeName, relativeScope string) string {
	if relativeScope == "" {
		return typeName
	}
	prefix := relativeScope + "."
	if strings.HasPrefix(typeName, prefix) {
		return strings.TrimPrefix(typeName, prefix)
	}
	return typeName
}
