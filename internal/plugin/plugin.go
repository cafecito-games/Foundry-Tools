package plugin

import (
	"fmt"
	"io"
	"sort"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	foundryproto "github.com/cafecito-games/foundry-tools/internal/proto"
	"github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
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

	files, err := foundryproto.FromCodeGeneratorRequest(req)
	if err != nil {
		return writeError(out, err.Error())
	}

	filesByName := make(map[string]*foundryproto.File, len(req.GetProtoFile()))
	for i, descriptor := range req.GetProtoFile() {
		if i < len(files) {
			filesByName[descriptor.GetName()] = files[i]
		}
	}

	// Every descriptor in the request is available to the generator as an
	// import, which is how a type declared in another file resolves to its
	// namespace and, for an enum, to its proto default.
	dependencies := make(map[string][]string, len(req.GetProtoFile()))
	for _, descriptor := range req.GetProtoFile() {
		dependencies[descriptor.GetName()] = descriptor.GetDependency()
	}

	resp := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(supportedFeatures),
	}
	for _, name := range req.GetFileToGenerate() {
		file, ok := filesByName[name]
		if !ok {
			return writeError(out, fmt.Sprintf("file to generate %q not found in request", name))
		}
		// protoc has already resolved every name in file_to_generate against
		// the include paths, so these are canonical import paths and match
		// exactly. The runtime already ships bindings for the well-known ones,
		// so generating them here would give this project a second,
		// incompatible copy.
		if wellknown.IsWellKnownImport(name) {
			continue
		}
		if err := wellknown.Check(name); err != nil {
			return writeError(out, err.Error())
		}
		if validationErrors := foundryproto.Validate(file, name); len(validationErrors) != 0 {
			return writeError(out, foundryproto.FormatValidationErrors(validationErrors))
		}
		generated, err := foundryproto.Generate(file, name, importsFor(name, dependencies, filesByName), foundryproto.Options{})
		if err != nil {
			return writeError(out, err.Error())
		}
		appendFiles(resp, generated)
	}
	// The runtime ships whenever generation was asked for, even if no schema
	// produced a binding of its own: it is what generated code compiles
	// against, and it is the only place the well-known bindings come from, so a
	// request naming nothing but well-known files is asking for exactly it.
	appendFiles(resp, runtime.Files())

	return writeResponse(out, resp)
}

// importsFor collects name's dependencies, following public re-exports through
// the same transitive walk the source parser performs.
func importsFor(name string, dependencies map[string][]string, filesByName map[string]*foundryproto.File) []foundryproto.FileEntry {
	visited := map[string]bool{name: true}
	var entries []foundryproto.FileEntry
	var walk func(string)
	walk = func(current string) {
		for _, dependency := range dependencies[current] {
			if visited[dependency] {
				continue
			}
			visited[dependency] = true
			if file, ok := filesByName[dependency]; ok {
				entries = append(entries, foundryproto.FileEntry{File: file, Filename: dependency})
			}
			walk(dependency)
		}
	}
	walk(name)
	return entries
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
