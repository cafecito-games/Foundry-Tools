package protoparse

import (
	"strconv"

	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

func (p *parser) parseOption() (*protoast.Option, error) {
	optTok := p.current()
	if _, err := p.expect(TokenOption); err != nil {
		return nil, err
	}
	name, err := p.parseOptionName()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenEquals); err != nil {
		return nil, err
	}
	value, err := p.parseOptionValue()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenSemicolon); err != nil {
		return nil, err
	}
	return &protoast.Option{
		Position: protoast.Position{Line: optTok.Line, Column: optTok.Column},
		Name:     name,
		Value:    value,
	}, nil
}

// parseOptionName handles "foo", "foo.bar", and "(foo.bar)" (parenthesized,
// which becomes "(foo.bar)" with surrounding parens included).
func (p *parser) parseOptionName() (string, error) {
	if p.match(TokenLParen) {
		p.advance()
		name, err := p.parseDottedIdent()
		if err != nil {
			return "", err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return "", err
		}
		return "(" + name + ")", nil
	}
	return p.parseDottedIdent()
}

func (p *parser) parseOneof() (*protoast.Oneof, error) {
	oneofTok := p.current()
	if _, err := p.expect(TokenOneof); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	o := &protoast.Oneof{
		Position: protoast.Position{Line: oneofTok.Line, Column: oneofTok.Column},
		Name:     nameTok.Value,
	}

	for !p.match(TokenRBrace) {
		if p.match(TokenOption) {
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			if o.Options == nil {
				o.Options = map[string]any{}
			}
			o.Options[opt.Name] = opt.Value
			continue
		}
		f, err := p.parseField(nameTok.Value)
		if err != nil {
			return nil, err
		}
		if f.Repeated {
			return nil, p.errorf(oneofTok, "Oneof fields cannot be repeated")
		}
		o.Fields = append(o.Fields, f)
	}

	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return o, nil
}

// parseOptionValue accepts string, int, float, bool, or identifier.
func (p *parser) parseOptionValue() (any, error) {
	tok := p.current()
	switch tok.Type {
	case TokenStringLiteral:
		p.advance()
		return tok.Value, nil
	case TokenIntLiteral:
		p.advance()
		v, err := strconv.ParseInt(tok.Value, 0, 64)
		if err != nil {
			return nil, p.errorf(tok, "invalid integer literal %q: %v", tok.Value, err)
		}
		return v, nil
	case TokenFloatLiteral:
		p.advance()
		v, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			return nil, p.errorf(tok, "invalid float literal %q: %v", tok.Value, err)
		}
		return v, nil
	case TokenTrue:
		p.advance()
		return true, nil
	case TokenFalse:
		p.advance()
		return false, nil
	case TokenIdentifier:
		p.advance()
		return tok.Value, nil
	default:
		return nil, p.errorf(tok, "Expected option value, got %s", tok.Type)
	}
}
