package protoparse

// scalarTypeTokens is the set of token types that name a built-in scalar.
var scalarTypeTokens = map[TokenType]bool{
	TokenDouble:   true,
	TokenFloat:    true,
	TokenInt32:    true,
	TokenInt64:    true,
	TokenUInt32:   true,
	TokenUInt64:   true,
	TokenSInt32:   true,
	TokenSInt64:   true,
	TokenFixed32:  true,
	TokenFixed64:  true,
	TokenSFixed32: true,
	TokenSFixed64: true,
	TokenBool:     true,
	TokenString:   true,
	TokenBytes:    true,
}

// parseType parses a field type. Built-in scalars return their keyword
// string. Identifier paths return "Foo", "Foo.Bar", or ".pkg.Foo" for
// absolute references.
func (p *parser) parseType() (string, error) {
	tok := p.current()
	if scalarTypeTokens[tok.Type] {
		p.advance()
		return tok.Value, nil
	}

	// Absolute (.pkg.Foo) or relative (Foo.Bar) message type.
	if tok.Type == TokenDot {
		p.advance()
		rest, err := p.parseDottedIdent()
		if err != nil {
			return "", err
		}
		return "." + rest, nil
	}
	if tok.Type == TokenIdentifier {
		return p.parseDottedIdent()
	}

	return "", p.errorf(tok, "Expected type name, got %s", tok.Type)
}
