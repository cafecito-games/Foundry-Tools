//go:build integration

package integration

import (
	"flag"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

var updateReference = flag.Bool("update-reference", false, "rewrite tests/foundry/scalars_reference.bin")

// referencePath is the encoding of scalarValues produced by the reference
// protobuf implementation. tests/foundry/main.fs asserts our generated encoder
// produces these bytes and our decoder reads them back, which is what makes
// the scalar support checked against protobuf rather than against itself.
const referencePath = "tests/foundry/scalars_reference.bin"

// scalarValues populates the message with the edge cases the eight non-varint
// scalars have to survive: both ends of the signed ranges, the unsigned top
// half that Foundry's signed int can only hold as a negative, a float that
// binary32 cannot represent exactly, and the infinities.
//
// NaN is deliberately absent. Its bit pattern is not unique, so a byte-exact
// comparison across two implementations would assert something neither
// promises; main.fs round-trips NaN on its own instead.
//
// Negative zero is absent for a different reason: the engine cannot reliably
// hold one written as a literal (cafecito-games/Foundry#1371), so asserting on
// it here would pin our fixture to an engine defect rather than to the wire
// format. Restore the element once that lands.
func scalarValues(message protoreflect.Message) {
	fields := message.Descriptor().Fields()
	set := func(name string, value protoreflect.Value) {
		field := fields.ByName(protoreflect.Name(name))
		if field == nil {
			panic("unknown field " + name)
		}
		message.Set(field, value)
	}
	appendTo := func(name string, values ...protoreflect.Value) {
		field := fields.ByName(protoreflect.Name(name))
		list := message.Mutable(field).List()
		for _, value := range values {
			list.Append(value)
		}
	}

	set("double_value", protoreflect.ValueOfFloat64(-2.5))
	set("float_value", protoreflect.ValueOfFloat32(0.1))
	set("fixed32_value", protoreflect.ValueOfUint32(math.MaxUint32))
	set("fixed64_value", protoreflect.ValueOfUint64(math.MaxUint64))
	set("sfixed32_value", protoreflect.ValueOfInt32(math.MinInt32))
	set("sfixed64_value", protoreflect.ValueOfInt64(math.MinInt64))
	set("sint32_value", protoreflect.ValueOfInt32(math.MinInt32))
	set("sint64_value", protoreflect.ValueOfInt64(math.MinInt64))

	appendTo("sint32_list",
		protoreflect.ValueOfInt32(0),
		protoreflect.ValueOfInt32(-1),
		protoreflect.ValueOfInt32(1),
		protoreflect.ValueOfInt32(math.MinInt32),
		protoreflect.ValueOfInt32(math.MaxInt32),
	)
	appendTo("double_list",
		protoreflect.ValueOfFloat64(0),
		protoreflect.ValueOfFloat64(1.5),
		protoreflect.ValueOfFloat64(math.Inf(1)),
		protoreflect.ValueOfFloat64(math.Inf(-1)),
	)
	appendTo("sfixed32_list",
		protoreflect.ValueOfInt32(-1),
		protoreflect.ValueOfInt32(0),
		protoreflect.ValueOfInt32(math.MaxInt32),
	)

	set("choice_delta", protoreflect.ValueOfInt64(-4096))
}

// The reference bytes are only meaningful if they are what protobuf itself
// produces, so this regenerates them and fails when the committed file has
// drifted. Run with -update-reference to accept a change.
func TestScalarReferenceBytesMatchProtobuf(t *testing.T) {
	root := repoRoot(t)
	descriptorPath := filepath.Join(t.TempDir(), "scalars.desc")

	run(t, root, "protoc",
		"--descriptor_set_out="+descriptorPath,
		"-I", "tests/foundry",
		"tests/foundry/scalars.proto",
	)

	raw, err := os.ReadFile(descriptorPath)
	require.NoError(t, err)
	var set descriptorpb.FileDescriptorSet
	require.NoError(t, proto.Unmarshal(raw, &set))

	files, err := protodesc.NewFiles(&set)
	require.NoError(t, err)
	descriptor, err := files.FindDescriptorByName("probe.scalars.v1.ScalarSuite")
	require.NoError(t, err)

	message := dynamicpb.NewMessage(descriptor.(protoreflect.MessageDescriptor))
	scalarValues(message)

	// Deterministic marshaling fixes field order, which is what makes a
	// byte-exact comparison against another implementation meaningful.
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(message)
	require.NoError(t, err)

	absolute := filepath.Join(root, referencePath)
	if *updateReference {
		require.NoError(t, os.WriteFile(absolute, encoded, 0o644))
		return
	}

	committed, err := os.ReadFile(absolute)
	require.NoError(t, err, "run with -update-reference to create %s", referencePath)
	require.Equal(t, committed, encoded,
		"%s no longer matches what protobuf encodes for scalars.proto; re-run with -update-reference if the fixture changed on purpose", referencePath)
}
