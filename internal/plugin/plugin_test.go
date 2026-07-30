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

	require.Contains(t, resp.GetError(), "members.proto:5:3:")
	require.Contains(t, resp.GetError(), "members.proto:6:3:")
	require.Contains(t, resp.GetError(), "field probe.members.v1.MemberProbe.Node ")
	require.Contains(t, resp.GetError(), "field probe.members.v1.MemberProbe.Node_ ")
	require.Contains(t, resp.GetError(), `Foundry member "Node_"`)
	require.Contains(t, resp.GetError(), `native class "Node"`)
	require.Empty(t, resp.GetFile())
}

func TestRunReportsMemberCollisionWithoutDescriptorPositions(t *testing.T) {
	resp := runPlugin(t, memberCollisionRequest(false))

	require.Contains(t, resp.GetError(), "members.proto:")
	require.Contains(t, resp.GetError(), "field probe.members.v1.MemberProbe.Node ")
	require.Contains(t, resp.GetError(), "field probe.members.v1.MemberProbe.Node_ ")
	require.Contains(t, resp.GetError(), `Foundry member "Node_"`)
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
	file := &descriptorpb.FileDescriptorProto{
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
		file.SourceCodeInfo = &descriptorpb.SourceCodeInfo{
			Location: []*descriptorpb.SourceCodeInfo_Location{
				{Path: []int32{4, 0, 2, 0}, Span: []int32{4, 2, 4, 18}},
				{Path: []int32{4, 0, 2, 1}, Span: []int32{5, 2, 5, 19}},
			},
		}
	}
	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"members.proto"},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{file},
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
