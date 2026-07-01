package protoparse_test

import (
	"errors"
	"strings"
	"testing"

	protoparse "github.com/cafecito-games/foundry-tools/internal/proto/internal/parser"
)

func TestTokenTypeString(t *testing.T) {
	cases := map[protoparse.TokenType]string{
		protoparse.TokenEOF:        "TokenEOF",
		protoparse.TokenSyntax:     "TokenSyntax",
		protoparse.TokenIdentifier: "TokenIdentifier",
		protoparse.TokenIntLiteral: "TokenIntLiteral",
		protoparse.TokenLBrace:     "TokenLBrace",
	}
	for tt, want := range cases {
		if got := tt.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", tt, got, want)
		}
	}
}

func TestLexerErrorFormat(t *testing.T) {
	err := &protoparse.LexerError{
		File:    "test.proto",
		Line:    5,
		Column:  12,
		Message: "Unexpected character",
	}
	got := err.Error()
	want := "test.proto:5:12: error: Unexpected character"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestLexerErrorDefaultFile(t *testing.T) {
	err := &protoparse.LexerError{Line: 1, Column: 1, Message: "oops"}
	if !strings.Contains(err.Error(), "<input>") {
		t.Errorf("expected <input> in default error: %q", err.Error())
	}
}

func TestLexerErrorIsError(t *testing.T) {
	var e error = &protoparse.LexerError{}
	var le *protoparse.LexerError
	if !errors.As(e, &le) {
		t.Fatal("LexerError must implement error")
	}
}

func TestTokenizeEmpty(t *testing.T) {
	tokens, err := protoparse.Tokenize("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Type != protoparse.TokenEOF {
		t.Fatalf("got %+v, want single EOF", tokens)
	}
	if tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Errorf("EOF position = %d:%d, want 1:1", tokens[0].Line, tokens[0].Column)
	}
}

func TestTokenizeWhitespaceOnly(t *testing.T) {
	tokens, err := protoparse.Tokenize("   \t\n\r\n   ", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Type != protoparse.TokenEOF {
		t.Fatalf("got %+v, want single EOF", tokens)
	}
}

func TestTokenizeAllSymbols(t *testing.T) {
	tokens, err := protoparse.Tokenize("{}[]()<>;=,.", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []protoparse.TokenType{
		protoparse.TokenLBrace, protoparse.TokenRBrace,
		protoparse.TokenLBracket, protoparse.TokenRBracket,
		protoparse.TokenLParen, protoparse.TokenRParen,
		protoparse.TokenLT, protoparse.TokenGT,
		protoparse.TokenSemicolon, protoparse.TokenEquals,
		protoparse.TokenComma, protoparse.TokenDot,
		protoparse.TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(tokens), len(want), tokens)
	}
	for i, w := range want {
		if tokens[i].Type != w {
			t.Errorf("token[%d].Type = %s, want %s", i, tokens[i].Type, w)
		}
	}
}

func TestSymbolPositionTracking(t *testing.T) {
	tokens, err := protoparse.Tokenize("=\n=", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Errorf("first = at %d:%d, want 1:1", tokens[0].Line, tokens[0].Column)
	}
	if tokens[1].Line != 2 || tokens[1].Column != 1 {
		t.Errorf("second = at %d:%d, want 2:1", tokens[1].Line, tokens[1].Column)
	}
}

func TestUnexpectedCharacter(t *testing.T) {
	_, err := protoparse.Tokenize("@", "test.proto")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var le *protoparse.LexerError
	if !errors.As(err, &le) {
		t.Fatalf("expected *LexerError, got %T", err)
	}
	if !strings.Contains(le.Message, "Unexpected character") {
		t.Errorf("message = %q, want contains 'Unexpected character'", le.Message)
	}
	if le.File != "test.proto" {
		t.Errorf("file = %q, want %q", le.File, "test.proto")
	}
	if le.Line != 1 || le.Column != 1 {
		t.Errorf("position = %d:%d, want 1:1", le.Line, le.Column)
	}
}

func TestSimpleIdentifier(t *testing.T) {
	tokens, err := protoparse.Tokenize("MyMessage", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != protoparse.TokenIdentifier || tokens[0].Value != "MyMessage" {
		t.Errorf("got %+v, want Identifier 'MyMessage'", tokens[0])
	}
}

func TestIdentifierVariants(t *testing.T) {
	cases := []string{"field123", "my_field_name", "_private"}
	for _, src := range cases {
		tokens, err := protoparse.Tokenize(src, "")
		if err != nil {
			t.Errorf("Tokenize(%q) error: %v", src, err)
			continue
		}
		if tokens[0].Type != protoparse.TokenIdentifier || tokens[0].Value != src {
			t.Errorf("Tokenize(%q): got %+v, want Identifier %q", src, tokens[0], src)
		}
	}
}

func TestKeywordsAll(t *testing.T) {
	cases := map[string]protoparse.TokenType{
		"syntax":   protoparse.TokenSyntax,
		"message":  protoparse.TokenMessage,
		"enum":     protoparse.TokenEnum,
		"repeated": protoparse.TokenRepeated,
		"map":      protoparse.TokenMap,
		"oneof":    protoparse.TokenOneof,
		"import":   protoparse.TokenImport,
		"public":   protoparse.TokenPublic,
		"option":   protoparse.TokenOption,
		"packed":   protoparse.TokenPacked,
		"reserved": protoparse.TokenReserved,
		"package":  protoparse.TokenPackage,
		"service":  protoparse.TokenService,
		"rpc":      protoparse.TokenRPC,
		"returns":  protoparse.TokenReturns,
		"stream":   protoparse.TokenStream,
		"int32":    protoparse.TokenInt32,
		"int64":    protoparse.TokenInt64,
		"string":   protoparse.TokenString,
		"bool":     protoparse.TokenBool,
		"bytes":    protoparse.TokenBytes,
		"double":   protoparse.TokenDouble,
		"float":    protoparse.TokenFloat,
		"true":     protoparse.TokenTrue,
		"false":    protoparse.TokenFalse,
	}
	for word, want := range cases {
		tokens, err := protoparse.Tokenize(word, "")
		if err != nil {
			t.Errorf("Tokenize(%q) error: %v", word, err)
			continue
		}
		if tokens[0].Type != want {
			t.Errorf("Tokenize(%q): got %s, want %s", word, tokens[0].Type, want)
		}
		if tokens[0].Value != word {
			t.Errorf("Tokenize(%q): value = %q, want %q", word, tokens[0].Value, word)
		}
	}
}

func TestSimpleInteger(t *testing.T) {
	tokens, _ := protoparse.Tokenize("123", "")
	if tokens[0].Type != protoparse.TokenIntLiteral || tokens[0].Value != "123" {
		t.Errorf("got %+v, want IntLiteral 123", tokens[0])
	}
}

func TestNegativeInteger(t *testing.T) {
	tokens, _ := protoparse.Tokenize("-456", "")
	if tokens[0].Type != protoparse.TokenIntLiteral || tokens[0].Value != "-456" {
		t.Errorf("got %+v, want IntLiteral -456", tokens[0])
	}
}

func TestZero(t *testing.T) {
	tokens, _ := protoparse.Tokenize("0", "")
	if tokens[0].Type != protoparse.TokenIntLiteral || tokens[0].Value != "0" {
		t.Errorf("got %+v, want IntLiteral 0", tokens[0])
	}
}

func TestHexNumber(t *testing.T) {
	tokens, _ := protoparse.Tokenize("0x1A2B", "")
	if tokens[0].Type != protoparse.TokenIntLiteral || tokens[0].Value != "0x1A2B" {
		t.Errorf("got %+v, want IntLiteral 0x1A2B", tokens[0])
	}
}

func TestOctalNumber(t *testing.T) {
	tokens, _ := protoparse.Tokenize("0755", "")
	if tokens[0].Type != protoparse.TokenIntLiteral || tokens[0].Value != "0755" {
		t.Errorf("got %+v, want IntLiteral 0755", tokens[0])
	}
}

func TestFloatDecimal(t *testing.T) {
	tokens, _ := protoparse.Tokenize("3.14", "")
	if tokens[0].Type != protoparse.TokenFloatLiteral || tokens[0].Value != "3.14" {
		t.Errorf("got %+v, want FloatLiteral 3.14", tokens[0])
	}
}

func TestFloatExponent(t *testing.T) {
	tokens, _ := protoparse.Tokenize("1.5e10", "")
	if tokens[0].Type != protoparse.TokenFloatLiteral || tokens[0].Value != "1.5e10" {
		t.Errorf("got %+v, want FloatLiteral 1.5e10", tokens[0])
	}
}

func TestFloatNegativeExponent(t *testing.T) {
	tokens, _ := protoparse.Tokenize("2.5e-3", "")
	if tokens[0].Type != protoparse.TokenFloatLiteral || tokens[0].Value != "2.5e-3" {
		t.Errorf("got %+v, want FloatLiteral 2.5e-3", tokens[0])
	}
}

func TestMultipleKeywords(t *testing.T) {
	tokens, err := protoparse.Tokenize("message enum repeated", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []protoparse.TokenType{
		protoparse.TokenMessage, protoparse.TokenEnum, protoparse.TokenRepeated, protoparse.TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(want))
	}
	for i, w := range want {
		if tokens[i].Type != w {
			t.Errorf("token[%d] = %s, want %s", i, tokens[i].Type, w)
		}
	}
}

func TestStringDoubleQuote(t *testing.T) {
	tokens, _ := protoparse.Tokenize(`"hello world"`, "")
	if tokens[0].Type != protoparse.TokenStringLiteral || tokens[0].Value != "hello world" {
		t.Errorf("got %+v", tokens[0])
	}
}

func TestStringSingleQuote(t *testing.T) {
	tokens, _ := protoparse.Tokenize(`'hello world'`, "")
	if tokens[0].Type != protoparse.TokenStringLiteral || tokens[0].Value != "hello world" {
		t.Errorf("got %+v", tokens[0])
	}
}

func TestStringEmpty(t *testing.T) {
	tokens, _ := protoparse.Tokenize(`""`, "")
	if tokens[0].Type != protoparse.TokenStringLiteral || tokens[0].Value != "" {
		t.Errorf("got %+v", tokens[0])
	}
}

func TestStringEscapes(t *testing.T) {
	tokens, _ := protoparse.Tokenize(`"hello\nworld\t!"`, "")
	if tokens[0].Value != "hello\nworld\t!" {
		t.Errorf("got %q, want %q", tokens[0].Value, "hello\nworld\t!")
	}
}

func TestStringEscapedQuotes(t *testing.T) {
	tokens, _ := protoparse.Tokenize(`"say \"hello\""`, "")
	if tokens[0].Value != `say "hello"` {
		t.Errorf("got %q", tokens[0].Value)
	}
}

func TestStringEscapedBackslash(t *testing.T) {
	tokens, _ := protoparse.Tokenize(`"path\\to\\file"`, "")
	if tokens[0].Value != `path\to\file` {
		t.Errorf("got %q", tokens[0].Value)
	}
}

func TestStringHexEscape(t *testing.T) {
	tokens, _ := protoparse.Tokenize(`"\x41\x42\x43"`, "")
	if tokens[0].Value != "ABC" {
		t.Errorf("got %q, want ABC", tokens[0].Value)
	}
}

func TestStringUnterminated(t *testing.T) {
	_, err := protoparse.Tokenize(`"hello`, "")
	var le *protoparse.LexerError
	if !errors.As(err, &le) || !strings.Contains(le.Message, "Unterminated string literal") {
		t.Errorf("got %v, want unterminated string error", err)
	}
}

func TestStringNewline(t *testing.T) {
	_, err := protoparse.Tokenize("\"hello\nworld\"", "")
	var le *protoparse.LexerError
	if !errors.As(err, &le) || !strings.Contains(le.Message, "Newline in string literal") {
		t.Errorf("got %v, want newline-in-string error", err)
	}
}

func TestStringInvalidEscape(t *testing.T) {
	_, err := protoparse.Tokenize(`"\q"`, "")
	var le *protoparse.LexerError
	if !errors.As(err, &le) || !strings.Contains(le.Message, "Invalid escape sequence") {
		t.Errorf("got %v, want invalid escape error", err)
	}
}

func TestLineComment(t *testing.T) {
	tokens, err := protoparse.Tokenize("// this is a comment\nmessage", "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(tokens) != 3 || tokens[0].Type != protoparse.TokenComment || tokens[1].Type != protoparse.TokenMessage {
		t.Errorf("got %+v", tokens)
	}
	if tokens[0].Value != "this is a comment" {
		t.Errorf("comment value = %q", tokens[0].Value)
	}
}

func TestLineCommentAtEOF(t *testing.T) {
	tokens, err := protoparse.Tokenize("message // comment", "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(tokens) != 3 || tokens[0].Type != protoparse.TokenMessage || tokens[1].Type != protoparse.TokenComment {
		t.Errorf("got %+v", tokens)
	}
	if tokens[1].Value != "comment" {
		t.Errorf("comment value = %q", tokens[1].Value)
	}
}

func TestBlockComment(t *testing.T) {
	tokens, err := protoparse.Tokenize("/* comment */ message", "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(tokens) != 3 || tokens[0].Type != protoparse.TokenComment || tokens[1].Type != protoparse.TokenMessage {
		t.Errorf("got %+v", tokens)
	}
	if tokens[0].Value != "comment" {
		t.Errorf("comment value = %q", tokens[0].Value)
	}
}

func TestMultilineBlockComment(t *testing.T) {
	tokens, err := protoparse.Tokenize("/* line 1\nline 2\nline 3 */ message", "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(tokens) != 3 || tokens[0].Type != protoparse.TokenComment || tokens[1].Type != protoparse.TokenMessage {
		t.Errorf("got %+v", tokens)
	}
	if tokens[0].Value != "line 1\nline 2\nline 3" {
		t.Errorf("comment value = %q", tokens[0].Value)
	}
	if tokens[1].Line != 3 {
		t.Errorf("message at line %d, want 3", tokens[1].Line)
	}
}

func TestUnterminatedBlockComment(t *testing.T) {
	_, err := protoparse.Tokenize("/* unterminated", "")
	var le *protoparse.LexerError
	if !errors.As(err, &le) || !strings.Contains(le.Message, "Unterminated block comment") {
		t.Errorf("got %v, want unterminated block comment error", err)
	}
}

func TestSyntaxDeclaration(t *testing.T) {
	tokens, err := protoparse.Tokenize(`syntax = "proto3";`, "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := []protoparse.TokenType{
		protoparse.TokenSyntax,
		protoparse.TokenEquals,
		protoparse.TokenStringLiteral,
		protoparse.TokenSemicolon,
		protoparse.TokenEOF,
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(tokens), len(want), tokens)
	}
	for i, w := range want {
		if tokens[i].Type != w {
			t.Errorf("token[%d].Type = %s, want %s", i, tokens[i].Type, w)
		}
	}
	if tokens[2].Value != "proto3" {
		t.Errorf("syntax string value = %q, want proto3", tokens[2].Value)
	}
}

func TestMessageDefinition(t *testing.T) {
	src := `
message Person {
    string name = 1;
    int32 age = 2;
}
`
	tokens, err := protoparse.Tokenize(src, "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := []protoparse.TokenType{
		protoparse.TokenMessage, protoparse.TokenIdentifier, protoparse.TokenLBrace,
		protoparse.TokenString, protoparse.TokenIdentifier, protoparse.TokenEquals, protoparse.TokenIntLiteral, protoparse.TokenSemicolon,
		protoparse.TokenInt32, protoparse.TokenIdentifier, protoparse.TokenEquals, protoparse.TokenIntLiteral, protoparse.TokenSemicolon,
		protoparse.TokenRBrace,
	}
	got := make([]protoparse.TokenType, 0, len(tokens)-1)
	for _, tok := range tokens[:len(tokens)-1] {
		got = append(got, tok.Type)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("token[%d] = %s, want %s", i, got[i], w)
		}
	}
}

func TestMapField(t *testing.T) {
	tokens, err := protoparse.Tokenize("map<string, int32> my_map = 1;", "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := []protoparse.TokenType{
		protoparse.TokenMap, protoparse.TokenLT, protoparse.TokenString, protoparse.TokenComma,
		protoparse.TokenInt32, protoparse.TokenGT, protoparse.TokenIdentifier, protoparse.TokenEquals,
		protoparse.TokenIntLiteral, protoparse.TokenSemicolon,
	}
	for i, w := range want {
		if tokens[i].Type != w {
			t.Errorf("token[%d] = %s, want %s", i, tokens[i].Type, w)
		}
	}
}

func TestLineTracking(t *testing.T) {
	tokens, _ := protoparse.Tokenize("line1\nline2\nline3", "")
	if tokens[0].Line != 1 || tokens[1].Line != 2 || tokens[2].Line != 3 {
		t.Errorf("line numbers = %d, %d, %d; want 1, 2, 3",
			tokens[0].Line, tokens[1].Line, tokens[2].Line)
	}
}

func TestColumnTracking(t *testing.T) {
	tokens, _ := protoparse.Tokenize("abc def", "")
	if tokens[0].Column != 1 {
		t.Errorf("token[0].Column = %d, want 1", tokens[0].Column)
	}
	if tokens[1].Column != 5 {
		t.Errorf("token[1].Column = %d, want 5", tokens[1].Column)
	}
}

func TestMultilineColumnTracking(t *testing.T) {
	tokens, _ := protoparse.Tokenize("first\n  second\n    third", "")
	cases := []struct{ line, col int }{
		{1, 1}, {2, 3}, {3, 5},
	}
	for i, c := range cases {
		if tokens[i].Line != c.line || tokens[i].Column != c.col {
			t.Errorf("token[%d] at %d:%d, want %d:%d",
				i, tokens[i].Line, tokens[i].Column, c.line, c.col)
		}
	}
}

func TestErrorMessageFormat(t *testing.T) {
	_, err := protoparse.Tokenize("@", "test.proto")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"test.proto", "1:1", "error"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q missing %q", msg, want)
		}
	}
}

func TestStringRemainingEscapes(t *testing.T) {
	tokens, err := protoparse.Tokenize(`"\r\'\0"`, "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := "\r'\x00"
	if tokens[0].Value != want {
		t.Errorf("got %q, want %q", tokens[0].Value, want)
	}
}

func TestStringUnterminatedEscape(t *testing.T) {
	_, err := protoparse.Tokenize(`"abc\`, "")
	var le *protoparse.LexerError
	if !errors.As(err, &le) {
		t.Fatalf("got %v, want LexerError", err)
	}
	if !strings.Contains(le.Message, "Unterminated") {
		t.Errorf("got %q, want unterminated escape", le.Message)
	}
}

func TestStringInvalidHexEscape(t *testing.T) {
	_, err := protoparse.Tokenize(`"\xZZ"`, "")
	var le *protoparse.LexerError
	if !errors.As(err, &le) || !strings.Contains(le.Message, "Invalid hex escape") {
		t.Errorf("got %v, want invalid hex escape", err)
	}
}

func TestStringInvalidHexEscapeShort(t *testing.T) {
	_, err := protoparse.Tokenize(`"\x1"`, "")
	var le *protoparse.LexerError
	if !errors.As(err, &le) || !strings.Contains(le.Message, "Invalid hex escape") {
		t.Errorf("got %v, want invalid hex escape", err)
	}
}

func TestTokenTypeStringOutOfRange(t *testing.T) {
	tt := protoparse.TokenType(-1)
	if !strings.Contains(tt.String(), "TokenType(-1)") {
		t.Errorf("got %q, want fallback format", tt.String())
	}
	tt2 := protoparse.TokenType(99999)
	if !strings.Contains(tt2.String(), "TokenType(99999)") {
		t.Errorf("got %q, want fallback format", tt2.String())
	}
}

func TestFloatExponentNoSign(t *testing.T) {
	tokens, err := protoparse.Tokenize("1e10", "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if tokens[0].Type != protoparse.TokenFloatLiteral || tokens[0].Value != "1e10" {
		t.Errorf("got %s %q, want TokenFloatLiteral 1e10", tokens[0].Type, tokens[0].Value)
	}
}
