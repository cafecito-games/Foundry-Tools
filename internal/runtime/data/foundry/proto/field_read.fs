namespace foundry.proto

class_name FieldRead[T] extends RefCounted

var value: T
var offset: int = 0
var error: ProtobufError = ProtobufError.OK

static func from(value_: T, offset_: int, error_: ProtobufError) -> FieldRead[T]:
	var result: FieldRead[T] = FieldRead[T].new()
	result.value = value_
	result.offset = offset_
	result.error = error_
	return result
