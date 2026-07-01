package fsgenerator

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

var variantSignaturePattern = regexp.MustCompile(`(^|[^A-Za-z0-9_])Variant\??([^A-Za-z0-9_]|$)`)

// CheckPublicAPI rejects public generated function signatures that expose Variant.
func CheckPublicAPI(source string) error {
	scanner := bufio.NewScanner(strings.NewReader(source))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if !isFunctionSignature(line) || isPrivateFunctionSignature(line) {
			continue
		}
		if variantSignaturePattern.MatchString(line) {
			return fmt.Errorf("public Variant in generated signature at line %d: %s", lineNumber, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func isFunctionSignature(line string) bool {
	return strings.HasPrefix(line, "func ") || strings.HasPrefix(line, "static func ")
}

func isPrivateFunctionSignature(line string) bool {
	return strings.HasPrefix(line, "func _") || strings.HasPrefix(line, "static func _")
}
