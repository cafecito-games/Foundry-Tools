package fsgenerator

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

var variantSignaturePattern = regexp.MustCompile(`(^|[^A-Za-z0-9_])Variant\??([^A-Za-z0-9_]|$)`)

var wellKnownVariantBridgeSignatures = map[string]bool{
	"func to_variant() -> Variant:":                                                     true,
	"static func from_variant(_pb_value: Variant) -> (Value?, ProtobufError):":          true,
	"func to_dictionary() -> Dictionary[String, Variant]:":                              true,
	"func to_array() -> Array[Variant]:":                                                true,
	"static func from_array(_pb_value: Array[Variant]) -> (ListValue?, ProtobufError):": true,
}

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
			if wellKnownVariantBridgeSignatures[line] {
				continue
			}
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
