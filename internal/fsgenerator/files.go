package fsgenerator

import (
	"path"
	"strings"
)

// GeneratedFiles maps generated output paths to source text.
type GeneratedFiles map[string]string

func namespacePath(namespace string) string {
	return strings.ReplaceAll(namespace, ".", "/")
}

func outputPath(namespace, typeName string) string {
	if namespace == "" {
		return typeName + ".pb.fs"
	}
	return path.Join(namespacePath(namespace), typeName+".pb.fs")
}
