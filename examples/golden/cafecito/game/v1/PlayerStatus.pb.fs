namespace cafecito.game.v1

## Lifecycle state of a player session.
enum_name PlayerStatus:
	PLAYER_STATUS_UNSPECIFIED = 0
	PLAYER_STATUS_ONLINE = 1
	PLAYER_STATUS_AWAY = 2

	## Returns the protobuf wire value for this case.
	func to_wire() -> int:
		match self:
			PlayerStatus.PLAYER_STATUS_ONLINE:
				return 1
			PlayerStatus.PLAYER_STATUS_AWAY:
				return 2
			_:
				return 0

	## Returns the case for a protobuf wire value, tolerating unknown values.
	static func from_wire(value: int) -> PlayerStatus:
		match value:
			1:
				return PlayerStatus.PLAYER_STATUS_ONLINE
			2:
				return PlayerStatus.PLAYER_STATUS_AWAY
			_:
				return PlayerStatus.PLAYER_STATUS_UNSPECIFIED
