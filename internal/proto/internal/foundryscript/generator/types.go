package fsgenerator

import fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"

// ScalarType maps a protobuf scalar name to a Foundry Script type.
//
// Foundry Script's fixed-width integer types mirror the protobuf width and
// signedness exactly: the three 32-bit signed kinds and the two 32-bit unsigned
// kinds land on int and uint, and the 64-bit family on long and ulong. The wire
// codec layer still differentiates varint, zig-zag, and fixed framing — that is
// a property of the encoding, not of the type — so the type names below are
// narrower than the ten protobuf spellings.
func ScalarType(protoType string) fstypes.Type {
	switch protoType {
	case "int32", "sint32", "sfixed32":
		return fstypes.Named("int")
	case "uint32", "fixed32":
		return fstypes.Named("uint")
	case "int64", "sint64", "sfixed64":
		return fstypes.Named("long")
	case "uint64", "fixed64":
		return fstypes.Named("ulong")
	case "double", "float":
		return fstypes.Named("float")
	case "bool":
		return fstypes.Named("bool")
	case "string":
		return fstypes.Named("String")
	case "bytes":
		return fstypes.Named("PackedByteArray")
	default:
		return fstypes.Named(TypeName(protoType))
	}
}
