namespace foundry.proto

trait_name Codec[T]

abstract func encode(value: T) -> PackedByteArray

abstract func decode(data: PackedByteArray, offset: int) -> (T, int, ProtobufError)
