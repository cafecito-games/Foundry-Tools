package fsgenerator

import fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"

// ScalarType maps a protobuf scalar name to a Foundry Script type.
func ScalarType(protoType string) fstypes.Type {
	switch protoType {
	case "int32", "int64", "uint32", "uint64", "sint32", "sint64", "fixed32", "fixed64", "sfixed32", "sfixed64":
		return fstypes.Named("int")
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
