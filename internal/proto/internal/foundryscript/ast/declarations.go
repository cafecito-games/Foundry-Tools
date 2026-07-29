package fsast

import (
	"strconv"
	"strings"

	fstypes "github.com/cafecito-games/foundry-tools/internal/proto/internal/foundryscript/types"
)

// Class represents a Foundry Script class declaration. A file-level class uses
// the `class_name` form with unindented members; Inner uses the `class` form
// with an indented body, which is how nested types are declared.
type Class struct {
	Doc     []string
	Inner   bool
	Final   bool
	Name    string
	Extends string
	Uses    []string
	Members []Node
}

// Enum represents a Foundry Script enum declaration. A file-level enum uses the
// `enum_name` form; Inner uses the `enum` form for declaration inside a class.
type Enum struct {
	Doc     []string
	Inner   bool
	Name    string
	Values  []EnumValue
	Members []Node
}

// EnumValue represents a single entry in an enum body. A value carrying Payload
// is a tagged-union case: cases are ordinal by declaration order and take no
// `= Number`, so the two forms cannot be mixed in one enum.
type EnumValue struct {
	Doc     []string
	Name    string
	Number  int
	Payload []Parameter
}

// SetterParameter is the name a set accessor binds its incoming value to. It
// carries the generated-name prefix for the same reason every other emitted
// name does: a member and the setter parameter share a scope, so a field named
// `value` and a parameter named `value` would both resolve to the parameter and
// the member would never be written -- silently, with no diagnostic.
const SetterParameter = "_pb_value"

// Var represents a typed variable declaration. Setter, when non-empty, makes it
// a property: the body runs on assignment, and assigning to the member inside
// that body writes the backing storage rather than recursing. The initializer
// does not run the setter.
type Var struct {
	Name   string
	Type   fstypes.Type
	Value  string
	Setter []Node
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
	if c.Inner {
		builder.WriteString("class ")
	} else {
		builder.WriteString("class_name ")
	}
	builder.WriteString(c.Name)
	if c.Extends != "" {
		builder.WriteString(" extends ")
		builder.WriteString(c.Extends)
	}
	if len(c.Uses) > 0 {
		builder.WriteString(" uses ")
		builder.WriteString(strings.Join(c.Uses, ", "))
	}
	if !c.Inner {
		builder.WriteByte('\n')
		for _, member := range c.Members {
			builder.WriteByte('\n')
			builder.WriteString(member.RenderAt(indent))
		}
		return builder.String()
	}

	builder.WriteString(":\n")
	if len(c.Members) == 0 {
		builder.WriteString(indentation(indent + 1))
		builder.WriteString("pass\n")
		return builder.String()
	}
	for i, member := range c.Members {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(member.RenderAt(indent + 1))
	}
	return builder.String()
}

// RenderAt renders e at indent.
func (e Enum) RenderAt(indent int) string {
	var builder strings.Builder
	renderDoc(&builder, indent, e.Doc)
	builder.WriteString(indentation(indent))
	if e.Inner {
		builder.WriteString("enum ")
	} else {
		builder.WriteString("enum_name ")
	}
	builder.WriteString(e.Name)
	builder.WriteString(":\n")
	// An enum body is an indented block, and an empty block is a parse error.
	if len(e.Values) == 0 && len(e.Members) == 0 {
		builder.WriteString(indentation(indent + 1))
		builder.WriteString("pass\n")
		return builder.String()
	}
	for _, value := range e.Values {
		renderDoc(&builder, indent+1, value.Doc)
		builder.WriteString(indentation(indent + 1))
		builder.WriteString(value.Name)
		if len(value.Payload) > 0 {
			builder.WriteByte('(')
			for i, parameter := range value.Payload {
				if i > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(parameter.Render())
			}
			builder.WriteByte(')')
		} else {
			builder.WriteString(" = ")
			builder.WriteString(strconv.Itoa(value.Number))
		}
		builder.WriteByte('\n')
	}
	for _, member := range e.Members {
		builder.WriteByte('\n')
		builder.WriteString(member.RenderAt(indent + 1))
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
	if len(v.Setter) == 0 {
		builder.WriteByte('\n')
		return builder.String()
	}
	builder.WriteString(":\n")
	builder.WriteString(indentation(indent + 1))
	builder.WriteString("set(" + SetterParameter + "):\n")
	for _, statement := range v.Setter {
		builder.WriteString(statement.RenderAt(indent + 2))
	}
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
