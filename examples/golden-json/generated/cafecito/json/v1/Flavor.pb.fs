namespace cafecito.json.v1

## Where a scalar sits in the canonical JSON mapping.
enum_name Flavor:
	FLAVOR_UNSPECIFIED = 0
	FLAVOR_SWEET = 1
	FLAVOR_BITTER = 2

	## Returns the protobuf wire value for this case.
	func to_wire() -> int:
		return self as int

	## Returns the case for a protobuf wire value, or null if it names none.
	static func from_wire(value: long) -> Self?:
		match value:
			0:
				return Flavor.FLAVOR_UNSPECIFIED
			1:
				return Flavor.FLAVOR_SWEET
			2:
				return Flavor.FLAVOR_BITTER
			_:
				return null

	## Returns the proto3 JSON name for this case.
	func to_json_name() -> String:
		var _pb_wire: int = self as int
		match _pb_wire:
			0:
				return "FLAVOR_UNSPECIFIED"
			1:
				return "FLAVOR_SWEET"
			2:
				return "FLAVOR_BITTER"
			_:
				return "FLAVOR_UNSPECIFIED"

	## Returns the case for a proto3 JSON name, or null if it names none.
	static func from_json_name(name: String) -> Self?:
		match name:
			"FLAVOR_UNSPECIFIED":
				return Flavor.FLAVOR_UNSPECIFIED
			"FLAVOR_SWEET":
				return Flavor.FLAVOR_SWEET
			"FLAVOR_BITTER":
				return Flavor.FLAVOR_BITTER
			_:
				return null
