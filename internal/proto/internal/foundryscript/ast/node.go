package fsast

import "strings"

// Node is a renderable Foundry Script AST node.
type Node interface {
	RenderAt(indent int) string
}

// File represents a complete Foundry Script source file.
type File struct {
	Namespace    string
	Imports      []string
	Declarations []Node
}

// Render returns the source for f with a trailing newline.
func (f File) Render() string {
	var builder strings.Builder
	if f.Namespace != "" {
		builder.WriteString("namespace ")
		builder.WriteString(f.Namespace)
		builder.WriteByte('\n')
	}
	for _, imp := range f.Imports {
		builder.WriteString("import ")
		builder.WriteString(imp)
		builder.WriteByte('\n')
	}
	if (f.Namespace != "" || len(f.Imports) > 0) && len(f.Declarations) > 0 {
		builder.WriteByte('\n')
	}
	for i, declaration := range f.Declarations {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(declaration.RenderAt(0))
	}
	source := builder.String()
	if !strings.HasSuffix(source, "\n") {
		source += "\n"
	}
	return source
}

func indentation(level int) string {
	return strings.Repeat("\t", level)
}
