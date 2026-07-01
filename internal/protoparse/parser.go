package protoparse

import (
	"fmt"
	"strings"

	"github.com/cafecito-games/foundry-tools/internal/protoast"
)

// Parse consumes a token stream from the lexer and produces a *protoast.ProtoFile.
// The filename is used only for error messages; pass "" for "<input>".
func Parse(tokens []Token, filename string) (*protoast.ProtoFile, error) {
	p := &parser{tokens: tokens, filename: filename}
	return p.parseFile()
}

type parser struct {
	tokens   []Token
	filename string
	pos      int
}

// current returns the token at the cursor, or the last token (EOF) if past
// the end. The lexer always emits a trailing TokenEOF, so this is safe.
func (p *parser) current() Token {
	if p.pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens)-1 {
		p.pos++
	}
	return tok
}

func (p *parser) match(types ...TokenType) bool {
	p.skipComments()
	cur := p.current().Type
	for _, t := range types {
		if cur == t {
			return true
		}
	}
	return false
}

func (p *parser) expect(t TokenType) (Token, error) {
	p.skipComments()
	tok := p.current()
	if tok.Type != t {
		return Token{}, p.errorf(tok, "Expected %s, got %s", t, tok.Type)
	}
	return p.advance(), nil
}

func (p *parser) skipComments() {
	for p.current().Type == TokenComment {
		p.advance()
	}
}

func (p *parser) takeLeadingDoc() []string {
	var docs []string
	lastCommentEndLine := 0
	for p.current().Type == TokenComment {
		tok := p.advance()
		lines := normalizeDocLines(tok.Value)
		if len(lines) != 0 {
			if len(docs) != 0 && tok.Line > lastCommentEndLine+1 {
				docs = docs[:0]
			}
			docs = append(docs, lines...)
		}
		lastCommentEndLine = tokenEndLine(tok)
	}
	if len(docs) == 0 {
		return nil
	}
	next := p.current()
	if next.Type == TokenEOF || next.Line > lastCommentEndLine+1 {
		return nil
	}
	return docs
}

func (p *parser) takeTrailingDoc(after Token) []string {
	var docs []string
	line := tokenEndLine(after)
	for p.current().Type == TokenComment && p.current().Line == line {
		tok := p.advance()
		docs = append(docs, normalizeDocLines(tok.Value)...)
	}
	if len(docs) == 0 {
		return nil
	}
	return docs
}

func normalizeDocLines(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func tokenEndLine(tok Token) int {
	if tok.EndLine != 0 {
		return tok.EndLine
	}
	return tok.Line
}

func (p *parser) errorf(tok Token, format string, args ...any) *ParserError {
	return &ParserError{
		File:    p.filename,
		Token:   tok,
		Message: fmt.Sprintf(format, args...),
	}
}

func (p *parser) parseFile() (*protoast.ProtoFile, error) {
	first := p.current()

	syntax, err := p.parseSyntax()
	if err != nil {
		return nil, err
	}

	file := &protoast.ProtoFile{
		Position: protoast.Position{Line: first.Line, Column: first.Column},
		Syntax:   syntax,
	}

	for {
		doc := p.takeLeadingDoc()
		if p.match(TokenEOF) {
			break
		}
		switch {
		case p.match(TokenImport):
			imp, err := p.parseImport()
			if err != nil {
				return nil, err
			}
			file.Imports = append(file.Imports, imp)
		case p.match(TokenPackage):
			pkg, err := p.parsePackage()
			if err != nil {
				return nil, err
			}
			file.Package = pkg
		case p.match(TokenOption):
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			if file.Options == nil {
				file.Options = map[string]any{}
			}
			if file.OptionPositions == nil {
				file.OptionPositions = map[string]protoast.Position{}
			}
			file.Options[opt.Name] = opt.Value
			file.OptionPositions[opt.Name] = opt.Position
		case p.match(TokenMessage):
			m, err := p.parseMessage(doc)
			if err != nil {
				return nil, err
			}
			file.Messages = append(file.Messages, m)
		case p.match(TokenEnum):
			e, err := p.parseEnum(doc)
			if err != nil {
				return nil, err
			}
			file.Enums = append(file.Enums, e)
		default:
			tok := p.current()
			return nil, p.errorf(tok, "Unexpected token: %s", tok.Type)
		}
	}

	return file, nil
}

func (p *parser) parseImport() (*protoast.Import, error) {
	impTok := p.current()
	if _, err := p.expect(TokenImport); err != nil {
		return nil, err
	}
	public := false
	if p.match(TokenPublic) {
		public = true
		p.advance()
	}
	pathTok, err := p.expect(TokenStringLiteral)
	if err != nil {
		return nil, err
	}
	semicolon, err := p.expect(TokenSemicolon)
	if err != nil {
		return nil, err
	}
	p.takeTrailingDoc(semicolon)
	return &protoast.Import{
		Position: protoast.Position{Line: impTok.Line, Column: impTok.Column},
		Path:     pathTok.Value,
		Public:   public,
	}, nil
}

func (p *parser) parsePackage() (string, error) {
	if _, err := p.expect(TokenPackage); err != nil {
		return "", err
	}
	name, err := p.parseDottedIdent()
	if err != nil {
		return "", err
	}
	semicolon, err := p.expect(TokenSemicolon)
	if err != nil {
		return "", err
	}
	p.takeTrailingDoc(semicolon)
	return name, nil
}

// parseDottedIdent parses Foo, Foo.Bar, Foo.Bar.Baz (identifiers separated
// by dots). At least one identifier required.
func (p *parser) parseDottedIdent() (string, error) {
	head, err := p.expect(TokenIdentifier)
	if err != nil {
		return "", err
	}
	parts := []string{head.Value}
	for p.match(TokenDot) {
		p.advance()
		next, err := p.expect(TokenIdentifier)
		if err != nil {
			return "", err
		}
		parts = append(parts, next.Value)
	}
	return strings.Join(parts, "."), nil
}

func (p *parser) parseSyntax() (string, error) {
	if _, err := p.expect(TokenSyntax); err != nil {
		return "", err
	}
	if _, err := p.expect(TokenEquals); err != nil {
		return "", err
	}
	tok, err := p.expect(TokenStringLiteral)
	if err != nil {
		return "", err
	}
	semicolon, err := p.expect(TokenSemicolon)
	if err != nil {
		return "", err
	}
	p.takeTrailingDoc(semicolon)
	return tok.Value, nil
}
