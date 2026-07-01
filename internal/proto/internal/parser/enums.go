package protoparse

import (
	"strconv"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
)

func (p *parser) parseEnum(doc []string) (*protoast.Enum, error) {
	enumTok := p.current()
	if _, err := p.expect(TokenEnum); err != nil {
		return nil, err
	}
	nameTok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	e := &protoast.Enum{
		Position: protoast.Position{Line: enumTok.Line, Column: enumTok.Column},
		Doc:      doc,
		Name:     nameTok.Value,
	}

	for {
		valueDoc := p.takeLeadingDoc()
		if p.match(TokenRBrace) {
			break
		}
		if p.match(TokenOption) {
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			if e.Options == nil {
				e.Options = map[string]any{}
			}
			e.Options[opt.Name] = opt.Value
			continue
		}
		v, err := p.parseEnumValue(valueDoc)
		if err != nil {
			return nil, err
		}
		e.Values = append(e.Values, v)
	}

	rbrace, err := p.expect(TokenRBrace)
	if err != nil {
		return nil, err
	}
	e.Doc = append(e.Doc, p.takeTrailingDoc(rbrace)...)
	return e, nil
}

func (p *parser) parseEnumValue(doc []string) (*protoast.EnumValue, error) {
	nameTok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenEquals); err != nil {
		return nil, err
	}
	numTok, err := p.expect(TokenIntLiteral)
	if err != nil {
		return nil, err
	}
	number, err := strconv.ParseInt(numTok.Value, 0, 32)
	if err != nil {
		return nil, p.errorf(numTok, "invalid enum value number %q: %v", numTok.Value, err)
	}
	var options map[string]any
	if p.match(TokenLBracket) {
		options, err = p.parseFieldOptions()
		if err != nil {
			return nil, err
		}
	}
	semicolon, err := p.expect(TokenSemicolon)
	if err != nil {
		return nil, err
	}
	doc = append(doc, p.takeTrailingDoc(semicolon)...)
	return &protoast.EnumValue{
		Position: protoast.Position{Line: nameTok.Line, Column: nameTok.Column},
		Doc:      doc,
		Name:     nameTok.Value,
		Number:   int(number),
		Options:  options,
	}, nil
}

func (p *parser) parseReserved() (*protoast.Reserved, error) {
	resTok := p.current()
	if _, err := p.expect(TokenReserved); err != nil {
		return nil, err
	}

	r := &protoast.Reserved{
		Position: protoast.Position{Line: resTok.Line, Column: resTok.Column},
	}

	if p.match(TokenStringLiteral) {
		for {
			nameTok, err := p.expect(TokenStringLiteral)
			if err != nil {
				return nil, err
			}
			r.Names = append(r.Names, nameTok.Value)
			if !p.match(TokenComma) {
				break
			}
			p.advance()
		}
	} else {
		for {
			startTok, err := p.expect(TokenIntLiteral)
			if err != nil {
				return nil, err
			}
			start, err := strconv.ParseInt(startTok.Value, 0, 32)
			if err != nil {
				return nil, p.errorf(startTok, "invalid reserved number %q: %v", startTok.Value, err)
			}

			rng := protoast.ReservedRange{Start: int(start), End: int(start)}
			if p.match(TokenIdentifier) && p.current().Value == "to" {
				p.advance()
				endTok, err := p.expect(TokenIntLiteral)
				if err != nil {
					return nil, err
				}
				end, err := strconv.ParseInt(endTok.Value, 0, 32)
				if err != nil {
					return nil, p.errorf(endTok, "invalid reserved range end %q: %v", endTok.Value, err)
				}
				rng.End = int(end)
			}
			r.Numbers = append(r.Numbers, rng)

			if !p.match(TokenComma) {
				break
			}
			p.advance()
		}
	}

	semicolon, err := p.expect(TokenSemicolon)
	if err != nil {
		return nil, err
	}
	p.takeTrailingDoc(semicolon)
	return r, nil
}
