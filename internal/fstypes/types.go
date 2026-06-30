package fstypes

import "strings"

// Type represents a Foundry Script type annotation.
type Type struct {
	name     string
	args     []Type
	nullable bool
}

// Named returns a named type annotation.
func Named(name string) Type {
	return Type{name: name}
}

// Generic returns a generic type annotation with the provided type arguments.
func Generic(name string, args ...Type) Type {
	return Type{
		name: name,
		args: append([]Type(nil), args...),
	}
}

// Nullable returns a nullable type annotation.
func Nullable(typ Type) Type {
	typ.nullable = true
	return typ
}

// Array returns an Array type annotation for element.
func Array(element Type) Type {
	return Generic("Array", element)
}

// Dictionary returns a Dictionary type annotation for key and value.
func Dictionary(key, value Type) Type {
	return Generic("Dictionary", key, value)
}

// Render returns the Foundry Script source representation of t.
func (t Type) Render() string {
	var builder strings.Builder
	builder.WriteString(t.name)
	if len(t.args) > 0 {
		builder.WriteByte('[')
		for i, arg := range t.args {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(arg.Render())
		}
		builder.WriteByte(']')
	}
	if t.nullable {
		builder.WriteByte('?')
	}
	return builder.String()
}

// IsVariant reports whether t is the explicit public Variant type.
func (t Type) IsVariant() bool {
	return t.name == "Variant" && len(t.args) == 0 && !t.nullable
}
