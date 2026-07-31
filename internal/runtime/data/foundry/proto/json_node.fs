namespace foundry.proto

## A JSON document, modelled over the six shapes JSON has.
##
## The engine's JSON API speaks Variant, but a JSON value is dynamic over a
## closed set, so everything above this file works on a tagged union instead
## and gets exhaustiveness checking with it. The two conversions below are the
## only place in the runtime that inspects a dynamic type, which is why this is
## the one file the runtime's Variant ban exempts.
##
## Every self-reference inside the enum body is written out in full. A callers'
## file says `JsonNode`, but within the enum's own body that spelling does not
## resolve to the enum being declared, so the qualified name is what the
## analyzer accepts here.
enum_name JsonNode:
	Null
	Bool(value: bool)
	Number(value: float)
	Text(value: String)
	List(values: Array[JsonNode])
	Object(fields: Dictionary[String, JsonNode])

	## Builds the dynamic value JSON.stringify takes, total over all six cases.
	static func to_variant(_pb_node: foundry.proto.JsonNode) -> Variant:
		match _pb_node:
			foundry.proto.JsonNode.Bool(var value):
				return value
			foundry.proto.JsonNode.Number(var value):
				return value
			foundry.proto.JsonNode.Text(var value):
				return value
			foundry.proto.JsonNode.List(var values):
				var _pb_converted: Array = []
				for _pb_value in values:
					_pb_converted.append(foundry.proto.JsonNode.to_variant(_pb_value))
				return _pb_converted
			foundry.proto.JsonNode.Object(var fields):
				var _pb_converted: Dictionary = {}
				for _pb_key in fields:
					_pb_converted[_pb_key] = foundry.proto.JsonNode.to_variant(fields[_pb_key])
				return _pb_converted
			_:
				return null

	## Reads back what JSON.parse_string produced. Anything outside the six
	## shapes -- a vector, a packed array, a callable -- is not a JSON document,
	## so it is refused rather than coerced into one.
	static func from_variant(_pb_value: Variant) -> (foundry.proto.JsonNode?, ProtobufError):
		var (_pb_node, _pb_error) = foundry.proto.JsonNode._from_variant_value(_pb_value)
		if _pb_error != ProtobufError.OK:
			var _pb_failed: foundry.proto.JsonNode? = null
			return (_pb_failed, _pb_error)
		return (_pb_node, ProtobufError.OK)

	## The recursive half, kept non-nullable so a nested value can go straight
	## into a typed container without a null check the shape has already ruled
	## out. A refused value comes back as Null, which the error beside it makes
	## unreadable as a document.
	static func _from_variant_value(_pb_value: Variant) -> (foundry.proto.JsonNode, ProtobufError):
		match typeof(_pb_value):
			TYPE_NIL:
				return (foundry.proto.JsonNode.Null, ProtobufError.OK)
			TYPE_BOOL:
				var _pb_bool: bool = _pb_value
				return (foundry.proto.JsonNode.Bool(_pb_bool), ProtobufError.OK)
			## A number written without a fraction parses as an int, which is
			## the same JSON shape as one written with it.
			TYPE_INT:
				var _pb_int: int = _pb_value
				return (foundry.proto.JsonNode.Number(float(_pb_int)), ProtobufError.OK)
			TYPE_FLOAT:
				var _pb_float: float = _pb_value
				return (foundry.proto.JsonNode.Number(_pb_float), ProtobufError.OK)
			TYPE_STRING:
				var _pb_text: String = _pb_value
				return (foundry.proto.JsonNode.Text(_pb_text), ProtobufError.OK)
			TYPE_ARRAY:
				var _pb_source: Array = _pb_value
				var _pb_values: Array[foundry.proto.JsonNode] = []
				for _pb_element in _pb_source:
					var (_pb_node, _pb_error) = foundry.proto.JsonNode._from_variant_value(_pb_element)
					if _pb_error != ProtobufError.OK:
						return (foundry.proto.JsonNode.Null, _pb_error)
					_pb_values.append(_pb_node)
				return (foundry.proto.JsonNode.List(_pb_values), ProtobufError.OK)
			TYPE_DICTIONARY:
				var _pb_source: Dictionary = _pb_value
				var _pb_fields: Dictionary[String, foundry.proto.JsonNode] = {}
				for _pb_key in _pb_source:
					## A JSON object is keyed by strings; a dictionary keyed by
					## anything else came from somewhere other than a document.
					if typeof(_pb_key) != TYPE_STRING:
						return (foundry.proto.JsonNode.Null, ProtobufError.JSON_TYPE_MISMATCH)
					var _pb_name: String = _pb_key
					var (_pb_node, _pb_error) = foundry.proto.JsonNode._from_variant_value(_pb_source[_pb_key])
					if _pb_error != ProtobufError.OK:
						return (foundry.proto.JsonNode.Null, _pb_error)
					_pb_fields[_pb_name] = _pb_node
				return (foundry.proto.JsonNode.Object(_pb_fields), ProtobufError.OK)
			_:
				return (foundry.proto.JsonNode.Null, ProtobufError.JSON_TYPE_MISMATCH)
