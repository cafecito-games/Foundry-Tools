namespace cafecito.game.v1
import foundry.proto.wkt

## Cases of the detail protobuf oneof.
enum_name EventDetailCase:
	Payload(payload: Any)
	Note(note: String)
