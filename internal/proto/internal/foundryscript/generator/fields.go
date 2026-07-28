package fsgenerator

import (
	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

// fieldMember is the public member for a field. A plain field carries exactly
// the information a get/set pair would, so no accessors are emitted.
func fieldMember(plan *fieldPlan) fsast.Node {
	return fsast.Doc{
		Lines: docOrFallback(plan.Doc, fieldDoc(plan.Name)),
		Node: fsast.Var{
			Name:  plan.Name,
			Type:  plan.DeclaredType(),
			Value: plan.DeclaredDefault(),
		},
	}
}

// oneofMember is the nullable tagged-union member backing a oneof. Making the
// union the only representation means "two cases set at once" cannot be built.
func oneofMember(oneof *oneofPlan) fsast.Node {
	return fsast.Doc{
		Lines: docOrFallback(oneof.Doc, oneofDoc(oneof.Field)),
		Node: fsast.Var{
			Name:  oneof.Field,
			Type:  fstypes.Nullable(fstypes.Named(oneof.Type)),
			Value: "null",
		},
	}
}

// oneofUnion is the tagged union declared for a oneof. Cases are ordinal by
// declaration order and take no explicit value.
func oneofUnion(oneof *oneofPlan) fsast.Enum {
	values := make([]fsast.EnumValue, 0, len(oneof.Members))
	for i := range oneof.Members {
		member := &oneof.Members[i]
		values = append(values, fsast.EnumValue{
			Doc:     docOrFallback(member.Doc, nil),
			Name:    TypeName(member.Name),
			Payload: []fsast.Parameter{{Name: member.Name, Type: member.Value.Type}},
		})
	}
	return fsast.Enum{
		Doc:    oneofUnionDoc(oneof.Field),
		Name:   oneof.Type,
		Values: values,
	}
}

func fromBytesFactory(className string) fsast.Func {
	return fsast.Func{
		Doc:        fromBytesDoc(className),
		Static:     true,
		Name:       "from_bytes",
		Parameters: []fsast.Parameter{{Name: "data", Type: fstypes.Named("PackedByteArray")}},
		ReturnType: fstypes.Tuple(fstypes.Nullable(fstypes.Named(className)), fstypes.Named("ProtobufError")),
		Body: []fsast.Node{
			line(0, "var message: "+className+" = "+className+".new()"),
			line(0, "var error: ProtobufError = message.merge_from_bytes(data)"),
			line(0, "if error != ProtobufError.OK:"),
			// A bare null does not carry the nullable element type.
			line(1, "var failed: "+className+"? = null"),
			line(1, "return (failed, error)"),
			fsast.Return{Value: "(message, ProtobufError.OK)"},
		},
	}
}
