package fsgenerator

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

var variantSignaturePattern = regexp.MustCompile(`(^|[^A-Za-z0-9_])Variant\??([^A-Za-z0-9_]|$)`)

var wellKnownVariantBridgeSignatures = map[wellKnownJSONForm]map[string]bool{
	wellKnownJSONValue: {
		"func to_variant() -> Variant:":                                            true,
		"static func from_variant(_pb_value: Variant) -> (Value?, ProtobufError):": true,
	},
	wellKnownJSONStruct: {
		"func to_dictionary() -> Dictionary[String, Variant]:": true,
	},
	wellKnownJSONListValue: {
		"func to_array() -> Array[Variant]:":                                                true,
		"static func from_array(_pb_value: Array[Variant]) -> (ListValue?, ProtobufError):": true,
	},
}

// CheckPublicAPI rejects public generated function signatures that expose Variant.
func CheckPublicAPI(source string) error {
	return checkPublicAPI(source, "", "")
}

// checkPublicAPI applies the only public Variant exceptions to the canonical
// declarations that own the native Struct/Value/ListValue bridge. Matching a
// method signature is insufficient: an ordinary schema may declare the same
// method name without becoming part of the runtime's dynamic-value boundary.
func checkPublicAPI(source, sourceName, typeName string) error {
	form := wellKnownJSONNone
	if sourceName == "google/protobuf/struct.proto" {
		form = wellKnownJSONForms[sourceName][typeName]
	}
	allowed := wellKnownVariantBridgeSignatures[form]
	scanner := bufio.NewScanner(strings.NewReader(source))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if !isFunctionSignature(line) || isPrivateFunctionSignature(line) {
			continue
		}
		if variantSignaturePattern.MatchString(line) {
			if allowed[line] {
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
