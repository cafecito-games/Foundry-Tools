package fsgenerator

// Options selects what a generation run emits beyond the protobuf wire codec.
// The zero value is the wire codec alone, which is what a caller that has not
// opted into anything gets.
type Options struct {
	// JSON emits the proto3 canonical JSON mapping on every generated message:
	// to_json_variant, to_json_string, merge_from_json_variant, and a
	// from_json_string constructor. It roughly doubles the size of a generated
	// file, which is why it is opt-in rather than always on.
	JSON bool
}
