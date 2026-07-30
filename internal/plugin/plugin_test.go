package plugin

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/foundry-tools/internal/foundrytoolspb"
)

func TestRunGeneratesRequestedFile(t *testing.T) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"player.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("player.proto"),
			Syntax:  proto.String("proto3"),
			Package: proto.String("cafecito.game.v1"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: proto.String("Player"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   proto.String("name"),
					Number: proto.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}},
			}},
		}},
	}
	data, err := proto.Marshal(req)
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, Run(bytes.NewReader(data), &out))

	resp := &pluginpb.CodeGeneratorResponse{}
	require.NoError(t, proto.Unmarshal(out.Bytes(), resp))
	require.Empty(t, resp.GetError())
	require.NotEmpty(t, resp.GetFile())
	require.Equal(t, "cafecito/game/v1/Player.pb.fs", resp.GetFile()[0].GetName())
}

func TestRunReportsEngineTypeCollision(t *testing.T) {
	req := nodeRequest(nil)

	resp := runPlugin(t, req)

	require.Contains(t, resp.GetError(), "node.proto:4:1:")
	require.Contains(t, resp.GetError(), `native class "Node"`)
	require.Contains(t, resp.GetError(), "(foundrytools.type_prefix)")
	require.Empty(t, resp.GetFile())
}

func TestRunAppliesTypePrefix(t *testing.T) {
	options := &descriptorpb.FileOptions{}
	proto.SetExtension(options, foundrytoolspb.E_TypePrefix, "Game")
	req := nodeRequest(options)

	resp := runPlugin(t, req)

	require.Empty(t, resp.GetError())
	var generated *pluginpb.CodeGeneratorResponse_File
	for _, file := range resp.GetFile() {
		if file.GetName() == "probe/collisions/v1/GameNode.pb.fs" {
			generated = file
			break
		}
	}
	require.NotNil(t, generated)
	require.Contains(t, generated.GetContent(), "class_name GameNode")
}

func TestRunReportsMemberCollisionWithDescriptorPositions(t *testing.T) {
	resp := runPlugin(t, memberCollisionRequest(true))

	require.Equal(t, `generated Foundry member names collide:
  members.proto:5:3: field probe.members.v1.MemberProbe.Node generates Foundry member "Node_" after escaping native class "Node"
  members.proto:6:3: field probe.members.v1.MemberProbe.Node_ generates Foundry member "Node_"
  rename one protobuf declaration in probe.members.v1.MemberProbe`, resp.GetError())
	require.Empty(t, resp.GetFile())
}

func TestRunReportsMemberCollisionWithoutDescriptorPositions(t *testing.T) {
	resp := runPlugin(t, memberCollisionRequest(false))

	require.Equal(t, `generated Foundry member names collide:
  members.proto: field probe.members.v1.MemberProbe.Node generates Foundry member "Node_" after escaping native class "Node"
  members.proto: field probe.members.v1.MemberProbe.Node_ generates Foundry member "Node_"
  rename one protobuf declaration in probe.members.v1.MemberProbe`, resp.GetError())
	require.Empty(t, resp.GetFile())
}

func nodeRequest(options *descriptorpb.FileOptions) *pluginpb.CodeGeneratorRequest {
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"node.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:    proto.String("node.proto"),
			Syntax:  proto.String("proto3"),
			Package: proto.String("probe.collisions.v1"),
			Options: options,
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: proto.String("Node"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:   proto.String("label"),
					Number: proto.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				}},
			}},
			SourceCodeInfo: &descriptorpb.SourceCodeInfo{
				Location: []*descriptorpb.SourceCodeInfo_Location{{
					Path: []int32{4, 0},
					Span: []int32{3, 0, 5, 1},
				}},
			},
		}},
	}
}

func memberCollisionRequest(withPositions bool) *pluginpb.CodeGeneratorRequest {
	validFile := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("valid.proto"),
		Syntax:  proto.String("proto3"),
		Package: proto.String("probe.valid.v1"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("ValidProbe"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:   proto.String("name"),
				Number: proto.Int32(1),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			}},
		}},
	}
	collidingFile := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("members.proto"),
		Syntax:  proto.String("proto3"),
		Package: proto.String("probe.members.v1"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("MemberProbe"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:   proto.String("Node"),
					Number: proto.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				},
				{
					Name:   proto.String("Node_"),
					Number: proto.Int32(2),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				},
			},
		}},
	}
	if withPositions {
		collidingFile.SourceCodeInfo = &descriptorpb.SourceCodeInfo{
			Location: []*descriptorpb.SourceCodeInfo_Location{
				{Path: []int32{4, 0, 2, 0}, Span: []int32{4, 2, 4, 18}},
				{Path: []int32{4, 0, 2, 1}, Span: []int32{5, 2, 5, 19}},
			},
		}
	}
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"valid.proto", "members.proto"},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{validFile, collidingFile},
	}
}

func runPlugin(t *testing.T, req *pluginpb.CodeGeneratorRequest) *pluginpb.CodeGeneratorResponse {
	t.Helper()
	data, err := proto.Marshal(req)
	require.NoError(t, err)

	var out bytes.Buffer
	require.NoError(t, Run(bytes.NewReader(data), &out))

	resp := &pluginpb.CodeGeneratorResponse{}
	require.NoError(t, proto.Unmarshal(out.Bytes(), resp))
	return resp
}

// protoc hands the well-known descriptors over as ordinary dependencies, so
// this is the path where a reference to one is most likely to be generated a
// second time by mistake.
func TestRunRoutesWellKnownReferenceToRuntimeNamespace(t *testing.T) {
	resp := runPlugin(t, wellKnownRequest([]string{"event.proto"}))
	require.Empty(t, resp.GetError())

	files := filesByName(resp)
	source, ok := files["cafecito/game/v1/Event.pb.fs"]
	require.True(t, ok)
	require.Contains(t, source, "import foundry.proto.wkt")
	require.Contains(t, source, "var occurred_at: Timestamp? = null")
	for name := range files {
		require.NotContains(t, name, "google/protobuf/",
			"well-known types must come from the runtime, not per-project generation")
	}
}

func TestRunSkipsWellKnownFileToGenerate(t *testing.T) {
	resp := runPlugin(t, wellKnownRequest([]string{"google/protobuf/timestamp.proto", "event.proto"}))
	require.Empty(t, resp.GetError())

	files := filesByName(resp)
	require.Contains(t, files, "cafecito/game/v1/Event.pb.fs")
	require.Contains(t, files, "foundry/proto/wkt/Timestamp.pb.fs")
	for name := range files {
		require.NotContains(t, name, "google/protobuf/")
	}
}

// A request for nothing but well-known files still has an answer: the runtime
// bindings the schemas resolve to. Skipping them here would return an empty
// response with no error, which reads as a silent failure and diverges from
// what `anvil proto generate` writes for the same input.
func TestRunShipsRuntimeForAWellKnownOnlyRequest(t *testing.T) {
	resp := runPlugin(t, wellKnownRequest([]string{"google/protobuf/timestamp.proto"}))
	require.Empty(t, resp.GetError())

	files := filesByName(resp)
	require.Contains(t, files, "foundry/proto/wkt/Timestamp.pb.fs")
	require.Contains(t, files, "foundry/proto/wire.fs")
	for name := range files {
		require.NotContains(t, name, "cafecito/",
			"only the files asked for should generate a binding")
	}
}

func TestRunRejectsUnsupportedWellKnownFile(t *testing.T) {
	req := wellKnownRequest([]string{"google/protobuf/descriptor.proto"})
	req.ProtoFile = append(req.ProtoFile, &descriptorpb.FileDescriptorProto{
		Name:    proto.String("google/protobuf/descriptor.proto"),
		Syntax:  proto.String("proto2"),
		Package: proto.String("google.protobuf"),
	})

	resp := runPlugin(t, req)

	require.Contains(t, resp.GetError(), "descriptor.proto")
	require.Contains(t, resp.GetError(), "not supported")
	require.Empty(t, resp.GetFile())
}

// wellKnownRequest builds a request whose event.proto carries a
// google.protobuf.Timestamp, with the well-known descriptor supplied as a
// dependency the way protoc supplies it.
func wellKnownRequest(fileToGenerate []string) *pluginpb.CodeGeneratorRequest {
	timestamp := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("google/protobuf/timestamp.proto"),
		Syntax:  proto.String("proto3"),
		Package: proto.String("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Timestamp"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:   proto.String("seconds"),
					Number: proto.Int32(1),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
				},
				{
					Name:   proto.String("nanos"),
					Number: proto.Int32(2),
					Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				},
			},
		}},
	}
	event := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("event.proto"),
		Syntax:     proto.String("proto3"),
		Package:    proto.String("cafecito.game.v1"),
		Dependency: []string{"google/protobuf/timestamp.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Event"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("occurred_at"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".google.protobuf.Timestamp"),
			}},
		}},
	}
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: fileToGenerate,
		ProtoFile:      []*descriptorpb.FileDescriptorProto{timestamp, event},
	}
}

func filesByName(resp *pluginpb.CodeGeneratorResponse) map[string]string {
	files := make(map[string]string, len(resp.GetFile()))
	for _, file := range resp.GetFile() {
		files[file.GetName()] = file.GetContent()
	}
	return files
}
