// Package proto exposes protobuf-to-Foundry-Script generation operations.
package proto

import (
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	protodesc "github.com/cafecito-games/foundry-tools/internal/proto/internal/desc"
	fsgenerator "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/generator"
	protoparse "github.com/cafecito-games/foundry-tools/internal/proto/internal/parser"
	protovalidate "github.com/cafecito-games/foundry-tools/internal/proto/internal/validate"
)

// File is the internal protobuf AST used by the generator pipeline.
type File = protoast.ProtoFile

// ParsedFile is a parsed protobuf file and the filename used for diagnostics.
type ParsedFile = protoparse.ParsedFile

// FileEntry represents an imported proto file available to the generator.
type FileEntry = fsgenerator.FileEntry

// GeneratedFiles maps generated output paths to source text.
type GeneratedFiles = fsgenerator.GeneratedFiles

// ValidationError describes a semantic validation failure.
type ValidationError = protovalidate.ValidationError

// ParseFiles parses root proto files using importRoots.
func ParseFiles(filenames, importRoots []string) ([]ParsedFile, error) {
	return protoparse.ParseFiles(filenames, importRoots)
}

// Validate performs semantic validation on file.
func Validate(file *File, filename string) []ValidationError {
	return protovalidate.Validate(file, filename)
}

// Generate renders Foundry Script protobuf bindings for file.
func Generate(file *File, sourceName string, imports []FileEntry) (GeneratedFiles, error) {
	return fsgenerator.Generate(file, sourceName, imports)
}

// FromCodeGeneratorRequest converts protoc plugin descriptors to the internal
// protobuf AST.
func FromCodeGeneratorRequest(req *pluginpb.CodeGeneratorRequest) ([]*File, error) {
	return protodesc.FromCodeGeneratorRequest(req)
}

// FromFileDescriptorProto converts one protobuf descriptor to the internal AST.
func FromFileDescriptorProto(fdp *descriptorpb.FileDescriptorProto) (*File, error) {
	return protodesc.FromFileDescriptorProto(fdp)
}

// FormatValidationErrors renders validation errors one per line.
func FormatValidationErrors(validationErrors []ValidationError) string {
	messages := make([]string, 0, len(validationErrors))
	for i := range validationErrors {
		messages = append(messages, (&validationErrors[i]).Error())
	}
	return strings.Join(messages, "\n")
}
