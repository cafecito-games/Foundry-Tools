package fsgenerator

import (
	"strconv"

	fsast "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/ast"
	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

func messageIdentityMembers(plan *messagePlan, form wellKnownJSONForm) []fsast.Node {
	return []fsast.Node{
		fsast.Func{
			Static:     true,
			Name:       createMessageMethod,
			ReturnType: fstypes.Named(plan.Name),
			Body:       []fsast.Node{fsast.Return{Value: plan.Name + ".new()"}},
		},
		fsast.Func{
			Static:     true,
			Name:       protobufTypeNameMethod,
			ReturnType: fstypes.Named("String"),
			Body:       []fsast.Node{fsast.Return{Value: strconv.Quote(plan.ProtoName)}},
		},
		fsast.Func{
			Name:       typeNameMethod,
			ReturnType: fstypes.Named("String"),
			Body:       []fsast.Node{fsast.Return{Value: plan.Name + "." + protobufTypeNameMethod + "()"}},
		},
		fsast.Func{
			Static:     true,
			Name:       anyUsesValueMethod,
			ReturnType: fstypes.Named("bool"),
			Body:       []fsast.Node{fsast.Return{Value: strconv.FormatBool(form.anyUsesValue())}},
		},
	}
}
