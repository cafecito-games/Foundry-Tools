namespace foundry.proto

trait_name Codec[T]

func encode(value: T) -> PackedByteArray

func decode(data: PackedByteArray, offset: int) -> FieldRead[T]
