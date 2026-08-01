namespace foundry.proto.wkt

## `NullValue` is a singleton enumeration to represent the null value for the
## `Value` type union.
## The JSON representation for `NullValue` is JSON `null`.
enum_name NullValue:
	## Null value.
	NULL_VALUE = 0

	## Returns the protobuf wire value for this case.
	func to_wire() -> int:
		return self as int

	## Returns the case for a protobuf wire value, or null if it names none.
	static func from_wire(value: int) -> Self?:
		match value:
			0:
				return NullValue.NULL_VALUE
			_:
				return null

	## Returns the proto3 JSON name for this case.
	func to_json_name() -> String:
		var _pb_wire: int = self as int
		match _pb_wire:
			0:
				return "NULL_VALUE"
			_:
				return "NULL_VALUE"

	## Returns the case for a proto3 JSON name, or null if it names none.
	static func from_json_name(name: String) -> Self?:
		match name:
			"NULL_VALUE":
				return NullValue.NULL_VALUE
			_:
				return null
