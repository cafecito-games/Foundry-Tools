package plugin

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/cafecito-games/foundry-tools/internal/fsgenerator"
	"github.com/cafecito-games/foundry-tools/internal/protoast"
	"github.com/cafecito-games/foundry-tools/internal/protodesc"
	"github.com/cafecito-games/foundry-tools/internal/protovalidate"
	"github.com/cafecito-games/foundry-tools/internal/runtime"
)

const supportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)

// Run reads a protoc CodeGeneratorRequest from in and writes a
// CodeGeneratorResponse to out.
func Run(in io.Reader, out io.Writer) error {
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("read request: %w", err)
	}

	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(data, req); err != nil {
		return writeError(out, fmt.Sprintf("unmarshal request: %v", err))
	}

	files, err := protodesc.FromCodeGeneratorRequest(req)
	if err != nil {
		return writeError(out, err.Error())
	}

	filesByName := make(map[string]*protoast.ProtoFile, len(req.GetProtoFile()))
	for i, descriptor := range req.GetProtoFile() {
		if i < len(files) {
			filesByName[descriptor.GetName()] = files[i]
		}
	}

	resp := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(supportedFeatures),
	}
	generatedAny := false
	for _, name := range req.GetFileToGenerate() {
		file, ok := filesByName[name]
		if !ok {
			return writeError(out, fmt.Sprintf("file to generate %q not found in request", name))
		}
		if validationErrors := protovalidate.Validate(file, name); len(validationErrors) != 0 {
			return writeError(out, formatValidationErrors(validationErrors))
		}
		generated, err := fsgenerator.Generate(file, name, nil)
		if err != nil {
			return writeError(out, err.Error())
		}
		if len(generated) != 0 {
			generatedAny = true
		}
		appendFiles(resp, generated)
	}
	if generatedAny {
		appendFiles(resp, runtime.Files())
	}

	return writeResponse(out, resp)
}

func formatValidationErrors(validationErrors []protovalidate.ValidationError) string {
	messages := make([]string, 0, len(validationErrors))
	for i := range validationErrors {
		messages = append(messages, (&validationErrors[i]).Error())
	}
	return strings.Join(messages, "\n")
}

func appendFiles(resp *pluginpb.CodeGeneratorResponse, files map[string]string) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		resp.File = append(resp.File, &pluginpb.CodeGeneratorResponse_File{
			Name:    proto.String(name),
			Content: proto.String(files[name]),
		})
	}
}

func writeError(out io.Writer, message string) error {
	return writeResponse(out, &pluginpb.CodeGeneratorResponse{
		Error:             proto.String(message),
		SupportedFeatures: proto.Uint64(supportedFeatures),
	})
}

func writeResponse(out io.Writer, resp *pluginpb.CodeGeneratorResponse) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}
