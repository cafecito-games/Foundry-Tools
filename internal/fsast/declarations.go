package fsast

import (
	"strings"

	"github.com/cafecito-games/foundry-tools/internal/fstypes"
)

// Class represents a Foundry Script class declaration.
type Class struct {
	Doc     []string
	Final   bool
	Name    string
	Extends string
	Uses    []string
	Members []Node
}

// Var represents a typed variable declaration.
type Var struct {
	Name  string
	Type  fstypes.Type
	Value string
}

// Func represents a typed function declaration.
type Func struct {
	Doc        []string
	Static     bool
	Name       string
	Parameters []Parameter
	ReturnType fstypes.Type
	ReturnVoid bool
	Body       []Node
}

// Doc adds documentation comments above another node.
type Doc struct {
	Lines []string
	Node  Node
}

// Parameter represents a typed function parameter.
type Parameter struct {
	Name string
	Type fstypes.Type
}

func renderDoc(builder *strings.Builder, indent int, lines []string) {
	for _, line := range lines {
		builder.WriteString(indentation(indent))
		builder.WriteString("##")
		if line != "" {
			builder.WriteByte(' ')
			builder.WriteString(line)
		}
		builder.WriteByte('\n')
	}
}

// RenderAt renders c at indent.
func (c Class) RenderAt(indent int) string {
	var builder strings.Builder
	renderDoc(&builder, indent, c.Doc)
	builder.WriteString(indentation(indent))
	if c.Final {
		builder.WriteString("final ")
	}
	builder.WriteString("class_name ")
	builder.WriteString(c.Name)
	if c.Extends != "" {
		builder.WriteString(" extends ")
		builder.WriteString(c.Extends)
	}
	if len(c.Uses) > 0 {
		builder.WriteString(" uses ")
		builder.WriteString(strings.Join(c.Uses, ", "))
	}
	builder.WriteByte('\n')
	for _, member := range c.Members {
		builder.WriteByte('\n')
		builder.WriteString(member.RenderAt(indent))
	}
	return builder.String()
}

// RenderAt renders d at indent.
func (d Doc) RenderAt(indent int) string {
	var builder strings.Builder
	renderDoc(&builder, indent, d.Lines)
	if d.Node != nil {
		builder.WriteString(d.Node.RenderAt(indent))
	}
	return builder.String()
}

// RenderAt renders v at indent.
func (v Var) RenderAt(indent int) string {
	var builder strings.Builder
	builder.WriteString(indentation(indent))
	builder.WriteString("var ")
	builder.WriteString(v.Name)
	if renderedType := v.Type.Render(); renderedType != "" {
		builder.WriteString(": ")
		builder.WriteString(renderedType)
	}
	if v.Value != "" {
		builder.WriteString(" = ")
		builder.WriteString(v.Value)
	}
	builder.WriteByte('\n')
	return builder.String()
}

// RenderAt renders fn at indent.
func (fn Func) RenderAt(indent int) string {
	var builder strings.Builder
	renderDoc(&builder, indent, fn.Doc)
	builder.WriteString(indentation(indent))
	if fn.Static {
		builder.WriteString("static ")
	}
	builder.WriteString("func ")
	builder.WriteString(fn.Name)
	builder.WriteByte('(')
	for i, parameter := range fn.Parameters {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(parameter.Render())
	}
	builder.WriteByte(')')
	if fn.ReturnVoid {
		builder.WriteString(" -> void")
	} else if renderedType := fn.ReturnType.Render(); renderedType != "" {
		builder.WriteString(" -> ")
		builder.WriteString(renderedType)
	}
	builder.WriteString(":\n")
	if len(fn.Body) == 0 {
		builder.WriteString(indentation(indent + 1))
		builder.WriteString("return\n")
		return builder.String()
	}
	for _, statement := range fn.Body {
		builder.WriteString(statement.RenderAt(indent + 1))
	}
	return builder.String()
}

// Render returns the parameter source.
func (p Parameter) Render() string {
	if renderedType := p.Type.Render(); renderedType != "" {
		return p.Name + ": " + renderedType
	}
	return p.Name
}
