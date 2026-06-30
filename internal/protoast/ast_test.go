package protoast_test

import (
	"testing"

	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

func TestProtoFileZero(t *testing.T) {
	f := &protoast.ProtoFile{}
	if f.Imports != nil || f.Messages != nil || f.Enums != nil {
		t.Fatal("zero-value collections should be nil (caller appends)")
	}
}

func TestPositionEmbedded(t *testing.T) {
	m := &protoast.Message{Position: protoast.Position{Line: 5, Column: 12}, Name: "Foo"}
	if m.Line != 5 || m.Column != 12 {
		t.Fatalf("position not accessible via embedding: got %d:%d", m.Line, m.Column)
	}
}

func TestReservedRangeSingle(t *testing.T) {
	r := protoast.ReservedRange{Start: 7, End: 7}
	if r.Start != r.End {
		t.Fatal("single-number reserved range should have Start == End")
	}
}

func TestFieldDefaults(t *testing.T) {
	f := &protoast.Field{}
	if f.Repeated || f.Optional || f.IsEnum {
		t.Fatal("zero-value Field should have all bool fields false")
	}
	if f.OneofParent != "" {
		t.Fatal("zero-value Field should have empty OneofParent")
	}
}
