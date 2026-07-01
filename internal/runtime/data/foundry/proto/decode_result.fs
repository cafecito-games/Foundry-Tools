namespace foundry.proto

class_name DecodeResult[T] extends RefCounted

var value: T?
var error: ProtobufError = ProtobufError.OK

static func from(value_: T?, error_: ProtobufError) -> DecodeResult[T]:
	var result: DecodeResult[T] = DecodeResult[T].new()
	result.value = value_
	result.error = error_
	return result

func is_ok() -> bool:
	return error == ProtobufError.OK
