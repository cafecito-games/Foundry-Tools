namespace foundry.proto.wkt

## Cases of the kind protobuf oneof.
enum_name ValueKindCase:
	## Represents a null value.
	NullValue(null_value: NullValue)
	## Represents a double value.
	NumberValue(number_value: float)
	## Represents a string value.
	StringValue(string_value: String)
	## Represents a boolean value.
	BoolValue(bool_value: bool)
	## Represents a structured value.
	StructValue(struct_value: Struct)
	## Represents a repeated `Value`.
	ListValue(list_value: ListValue)
