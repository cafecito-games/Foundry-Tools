namespace foundry.proto

trait_name Message

abstract static func create_message() -> Self

abstract static func protobuf_type_name() -> String

abstract func type_name() -> String

abstract func to_bytes() -> PackedByteArray

abstract func merge_from_bytes(_data: PackedByteArray) -> ProtobufError
