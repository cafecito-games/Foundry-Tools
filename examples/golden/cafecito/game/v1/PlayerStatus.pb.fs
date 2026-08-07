namespace cafecito.game.v1

## Lifecycle state of a player session.
enum_name PlayerStatus:
	PLAYER_STATUS_UNSPECIFIED = 0
	PLAYER_STATUS_ONLINE = 1
	PLAYER_STATUS_AWAY = 2

	## Returns the protobuf wire value for this case.
	func to_wire() -> int:
		return self as int

	## Returns the case for a protobuf wire value, or null if it names none.
	static func from_wire(value: long) -> Self?:
		match value:
			0:
				return PlayerStatus.PLAYER_STATUS_UNSPECIFIED
			1:
				return PlayerStatus.PLAYER_STATUS_ONLINE
			2:
				return PlayerStatus.PLAYER_STATUS_AWAY
			_:
				return null
