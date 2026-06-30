package fsgenerator

import (
	"github.com/cafecito-games/foundry-tools/internal/fsast"
	"github.com/cafecito-games/foundry-tools/internal/fstypes"
	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

func fieldMembers(field *protoast.Field) []fsast.Node {
	typ := fieldType(field)
	name := field.Name
	backing := "_" + name
	return []fsast.Node{
		fsast.Func{
			Name:       "set_" + name,
			Parameters: []fsast.Parameter{{Name: "value", Type: typ}},
			ReturnVoid: true,
			Body:       []fsast.Node{fsast.Assign{Target: backing, Value: "value"}},
		},
		fsast.Func{
			Name:       "get_" + name,
			ReturnType: typ,
			Body:       []fsast.Node{fsast.Return{Value: backing}},
		},
	}
}

func fieldType(field *protoast.Field) fstypes.Type {
	return ScalarType(field.FieldType)
}

func fromBytesFactory(className string) fsast.Func {
	return fsast.Func{
		Static:     true,
		Name:       "from_bytes",
		Parameters: []fsast.Parameter{{Name: "data", Type: fstypes.Named("PackedByteArray")}},
		ReturnType: fstypes.Generic("DecodeResult", fstypes.Named(className)),
		Body: []fsast.Node{
			fsast.Raw{Code: "\tvar message: " + className + " = " + className + ".new()\n"},
			fsast.Raw{Code: "\tvar err: foundry.proto.ProtobufError = message.merge_from_bytes(data)\n"},
			fsast.Return{Value: "DecodeResult[" + className + "].from(message, err)"},
		},
	}
}
