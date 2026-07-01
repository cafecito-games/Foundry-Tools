package foundrytoolspb

import _ "embed"

//go:embed options.proto
var optionsProto []byte

// Bytes returns the embedded foundrytools/options.proto schema.
func Bytes() []byte {
	out := make([]byte, len(optionsProto))
	copy(out, optionsProto)
	return out
}
