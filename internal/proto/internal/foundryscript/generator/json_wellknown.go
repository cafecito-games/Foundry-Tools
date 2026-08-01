package fsgenerator

import (
	"path"
	"strings"

	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
)

// wellKnownJSONForm is the canonical JSON form of a google.protobuf type.
//
// The well-known types do not serialize as ordinary messages: a Timestamp is a
// string, an Empty is an empty object, a wrapper is the bare scalar it wraps.
// The form is what both JSON emitters branch on -- the serializer picks the
// body of to_json from it, the decoder picks the body of from_json -- so the
// mapping is stated once here rather than twice in terms of type names.
type wellKnownJSONForm int

const (
	// wellKnownJSONNone is an ordinary message, serialized field by field.
	wellKnownJSONNone wellKnownJSONForm = iota
	// wellKnownJSONTimestamp is an RFC-3339 string, always UTC on output.
	wellKnownJSONTimestamp
	// wellKnownJSONDuration is seconds with up to nine fractional digits and
	// an `s` suffix.
	wellKnownJSONDuration
	// wellKnownJSONFieldMask is one string of comma-joined camelCase paths.
	wellKnownJSONFieldMask
	// wellKnownJSONStruct is a JSON object of Value members.
	wellKnownJSONStruct
	// wellKnownJSONValue is whichever JSON value its kind names.
	wellKnownJSONValue
	// wellKnownJSONListValue is a JSON array of Value members.
	wellKnownJSONListValue
	// wellKnownJSONWrapper is the bare scalar the wrapper carries, written
	// whatever its value, since the wrapper's own presence is the point of it.
	wellKnownJSONWrapper
	// wellKnownJSONEmpty is an empty JSON object.
	wellKnownJSONEmpty
	// wellKnownJSONAny has no JSON form until the type-URL registry lands.
	wellKnownJSONAny
)

// wellKnownJSONForms is the well-known table, keyed first on the import path of
// the proto file that declares the type and then on the type's proto name.
//
// Keying on the import path is what makes a well-known type identifiable at
// all: `Timestamp` is an ordinary name a schema of the caller's own may declare,
// and only the file it was declared in says whether it is google's. The
// generated bindings for these files are the generator's own output, so the
// special form is emitted onto the declaring binding rather than onto every
// message that references one -- a referencing field then recurses through the
// trait exactly as it would for any other message.
var wellKnownJSONForms = map[string]map[string]wellKnownJSONForm{
	"google/protobuf/any.proto":        {"Any": wellKnownJSONAny},
	"google/protobuf/duration.proto":   {"Duration": wellKnownJSONDuration},
	"google/protobuf/empty.proto":      {"Empty": wellKnownJSONEmpty},
	"google/protobuf/field_mask.proto": {"FieldMask": wellKnownJSONFieldMask},
	"google/protobuf/struct.proto": {
		"ListValue": wellKnownJSONListValue,
		"Struct":    wellKnownJSONStruct,
		"Value":     wellKnownJSONValue,
	},
	"google/protobuf/timestamp.proto": {"Timestamp": wellKnownJSONTimestamp},
	"google/protobuf/wrappers.proto": {
		"BoolValue":   wellKnownJSONWrapper,
		"BytesValue":  wellKnownJSONWrapper,
		"DoubleValue": wellKnownJSONWrapper,
		"FloatValue":  wellKnownJSONWrapper,
		"Int32Value":  wellKnownJSONWrapper,
		"Int64Value":  wellKnownJSONWrapper,
		"StringValue": wellKnownJSONWrapper,
		"UInt32Value": wellKnownJSONWrapper,
		"UInt64Value": wellKnownJSONWrapper,
	},
}

// wellKnownJSONFormFor reports the canonical JSON form of the type declared as
// typeName in the file carrying importPath. typeName is the generated type's
// own name, which for these files is the proto name unchanged: the checked-in
// bindings are generated with no type prefix, and a caller's copy of a
// google/protobuf file is rejected before it reaches an emitter.
func wellKnownJSONFormFor(importPath, typeName string) wellKnownJSONForm {
	if !wellknown.IsWellKnownImport(importPath) {
		return wellKnownJSONNone
	}
	return wellKnownJSONForms[path.Clean(strings.ReplaceAll(importPath, `\`, "/"))][typeName]
}

// Helper is the runtime class that converts this form to and from its JSON
// text, or empty for a form the emitters build inline. These shipped in the
// foundations epic and are the one place the formatting rules live, so a
// Timestamp written by a message and one written by hand agree.
func (f wellKnownJSONForm) Helper() string {
	switch f {
	case wellKnownJSONTimestamp:
		return "JsonTimestamp"
	case wellKnownJSONDuration:
		return "JsonDuration"
	case wellKnownJSONFieldMask:
		return "JsonFieldMask"
	default:
		return ""
	}
}

// jsonAnyUnsupportedMessage explains why google.protobuf.Any has no JSON form.
// It leads with the ProtobufError case name so the category stays greppable
// across both directions, which is the same convention the decoder's
// JsonDecodeError messages follow.
const jsonAnyUnsupportedMessage = "JSON_ANY_UNSUPPORTED: " +
	"google.protobuf.Any has no JSON form until the type-URL registry lands"

// wellKnownField is the field a well-known form is defined in terms of, found
// by its protobuf name rather than by position: a well-known binding is
// generated like any other, so its members carry whatever escaping the general
// naming rules applied.
func wellKnownField(plans []fieldPlan, protoName string) *fieldPlan {
	for i := range plans {
		if plans[i].RawName == protoName {
			return &plans[i]
		}
	}
	return nil
}
