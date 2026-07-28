namespace cafecito.game.v1

## Cases of the payload protobuf oneof.
enum_name PlayerPayloadCase:
	Text(text: String)
	Amount(amount: int)
