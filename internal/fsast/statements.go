package fsast

import "strings"

// Return represents a return statement.
type Return struct {
	Value string
}

// Assign represents an assignment statement.
type Assign struct {
	Target string
	Value  string
}

// Expr represents an expression statement.
type Expr struct {
	Code string
}

// RenderAt renders r at indent.
func (r Return) RenderAt(indent int) string {
	var builder strings.Builder
	builder.WriteString(indentation(indent))
	builder.WriteString("return")
	if r.Value != "" {
		builder.WriteByte(' ')
		builder.WriteString(r.Value)
	}
	builder.WriteByte('\n')
	return builder.String()
}

// RenderAt renders a at indent.
func (a Assign) RenderAt(indent int) string {
	var builder strings.Builder
	builder.WriteString(indentation(indent))
	builder.WriteString(a.Target)
	builder.WriteString(" = ")
	builder.WriteString(a.Value)
	builder.WriteByte('\n')
	return builder.String()
}

// RenderAt renders e at indent.
func (e Expr) RenderAt(indent int) string {
	var builder strings.Builder
	builder.WriteString(indentation(indent))
	builder.WriteString(e.Code)
	builder.WriteByte('\n')
	return builder.String()
}
