package plugin

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/stretchr/testify/require"
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
