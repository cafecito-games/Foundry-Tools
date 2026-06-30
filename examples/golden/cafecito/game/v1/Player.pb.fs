namespace cafecito.game.v1
import foundry.proto

final class_name Player extends RefCounted uses foundry.proto.Message[Player]

var _name: String = ""

func to_bytes() -> PackedByteArray:
	var result: PackedByteArray = PackedByteArray()
	return result

func merge_from_bytes(data: PackedByteArray) -> foundry.proto.ProtobufError:
	var _unused_size: int = data.size()
	return foundry.proto.ProtobufError.OK
