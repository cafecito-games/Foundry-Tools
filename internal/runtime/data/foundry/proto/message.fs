namespace foundry.proto

trait_name Message[T]

abstract func to_bytes() -> PackedByteArray

abstract func merge_from_bytes(data: PackedByteArray) -> ProtobufError
