namespace cafecito.game.v1
import foundry.proto
import foundry.proto.wkt

## An event carrying a well-known type in every position a reference can take.
final class_name Event extends RefCounted uses Message

## The occurred_at protobuf field.
var occurred_at: Timestamp? = null

## The attributes protobuf field.
var attributes: Struct? = null

## The attachments protobuf field.
var attachments: Array[Any] = []

## The checkpoints protobuf field.
var checkpoints: Dictionary[String, Timestamp] = {}

## The detail protobuf oneof; null when no case is set.
var detail: EventDetailCase? = null

## Fields this schema does not recognize, kept verbatim so a re-encode is lossless.
var _pb_unknown_fields: PackedByteArray = PackedByteArray()

## Decodes protobuf wire data into a new Event message.
static func from_bytes(_pb_data: PackedByteArray) -> (Event?, ProtobufError):
	var _pb_message: Event = Event.new()
	var _pb_error: ProtobufError = _pb_message.merge_from_bytes(_pb_data)
	if _pb_error != ProtobufError.OK:
		var _pb_failed: Event? = null
		return (_pb_failed, _pb_error)
	return (_pb_message, ProtobufError.OK)

## Serializes this message to protobuf wire data.
func to_bytes() -> PackedByteArray:
	var _pb_result: PackedByteArray = PackedByteArray()
	if occurred_at is Timestamp:
		var _pb_occurred_at_data: PackedByteArray = occurred_at.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_occurred_at_data.size()))
		_pb_result.append_array(_pb_occurred_at_data)
	if attributes is Struct:
		var _pb_attributes_data: PackedByteArray = attributes.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_attributes_data.size()))
		_pb_result.append_array(_pb_attributes_data)
	for _pb_attachments_item: Any in attachments:
		var _pb_attachments_data: PackedByteArray = _pb_attachments_item.to_bytes()
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(3, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_attachments_data.size()))
		_pb_result.append_array(_pb_attachments_data)
	for _pb_checkpoints_key: String in checkpoints:
		var _pb_checkpoints_entry: PackedByteArray = PackedByteArray()
		_pb_checkpoints_entry.append_array(Wire.encode_varint(Wire.make_tag(1, Wire.WIRE_LENGTH_DELIMITED)))
		var _pb_checkpoints_key_data: PackedByteArray = Wire.encode_string(_pb_checkpoints_key)
		_pb_checkpoints_entry.append_array(Wire.encode_varint(_pb_checkpoints_key_data.size()))
		_pb_checkpoints_entry.append_array(_pb_checkpoints_key_data)
		var _pb_checkpoints_value_data: PackedByteArray = checkpoints[_pb_checkpoints_key].to_bytes()
		_pb_checkpoints_entry.append_array(Wire.encode_varint(Wire.make_tag(2, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_checkpoints_entry.append_array(Wire.encode_varint(_pb_checkpoints_value_data.size()))
		_pb_checkpoints_entry.append_array(_pb_checkpoints_value_data)
		_pb_result.append_array(Wire.encode_varint(Wire.make_tag(4, Wire.WIRE_LENGTH_DELIMITED)))
		_pb_result.append_array(Wire.encode_varint(_pb_checkpoints_entry.size()))
		_pb_result.append_array(_pb_checkpoints_entry)
	match detail:
		EventDetailCase.Payload(var _pb_detail_payload):
			var _pb_detail_payload_data: PackedByteArray = _pb_detail_payload.to_bytes()
			_pb_result.append_array(Wire.encode_varint(Wire.make_tag(5, Wire.WIRE_LENGTH_DELIMITED)))
			_pb_result.append_array(Wire.encode_varint(_pb_detail_payload_data.size()))
			_pb_result.append_array(_pb_detail_payload_data)
		EventDetailCase.Note(var _pb_detail_note):
			_pb_result.append_array(Wire.encode_varint(Wire.make_tag(6, Wire.WIRE_LENGTH_DELIMITED)))
			var _pb_detail_note_data: PackedByteArray = Wire.encode_string(_pb_detail_note)
			_pb_result.append_array(Wire.encode_varint(_pb_detail_note_data.size()))
			_pb_result.append_array(_pb_detail_note_data)
		_:
			pass
	_pb_result.append_array(_pb_unknown_fields)
	return _pb_result

## Merges protobuf wire data into this message.
func merge_from_bytes(_pb_data: PackedByteArray) -> ProtobufError:
	var _pb_offset: int = 0
	while _pb_offset < _pb_data.size():
		var _pb_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
		if _pb_tag.error != ProtobufError.OK:
			return _pb_tag.error
		_pb_offset = _pb_tag.offset
		var _pb_wire_type: int = Wire.get_wire_type(_pb_tag.value)
		match Wire.get_field_number(_pb_tag.value):
			1:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (occurred_at is Timestamp):
					occurred_at = Timestamp.new()
				var _pb_occurred_at_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, occurred_at)
				if _pb_occurred_at_read.error != ProtobufError.OK:
					return _pb_occurred_at_read.error
				_pb_offset = _pb_occurred_at_read.offset
			2:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				if not (attributes is Struct):
					attributes = Struct.new()
				var _pb_attributes_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, attributes)
				if _pb_attributes_read.error != ProtobufError.OK:
					return _pb_attributes_read.error
				_pb_offset = _pb_attributes_read.offset
			3:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_attachments_message: Any = Any.new()
				var _pb_attachments_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_attachments_message)
				if _pb_attachments_read.error != ProtobufError.OK:
					return _pb_attachments_read.error
				attachments.append(_pb_attachments_message)
				_pb_offset = _pb_attachments_read.offset
			4:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_checkpoints_length: VarintRead = Wire.read_length(_pb_data, _pb_offset)
				if _pb_checkpoints_length.error != ProtobufError.OK:
					return _pb_checkpoints_length.error
				var _pb_checkpoints_end: int = _pb_checkpoints_length.offset + _pb_checkpoints_length.value
				_pb_offset = _pb_checkpoints_length.offset
				var _pb_checkpoints_key: String = ""
				var _pb_checkpoints_value: Timestamp = Timestamp.new()
				while _pb_offset < _pb_checkpoints_end:
					var _pb_checkpoints_entry_tag: VarintRead = Wire.decode_varint(_pb_data, _pb_offset)
					if _pb_checkpoints_entry_tag.error != ProtobufError.OK:
						return _pb_checkpoints_entry_tag.error
					if _pb_checkpoints_entry_tag.offset > _pb_checkpoints_end:
						return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
					_pb_offset = _pb_checkpoints_entry_tag.offset
					var _pb_checkpoints_entry_wire_type: int = Wire.get_wire_type(_pb_checkpoints_entry_tag.value)
					match Wire.get_field_number(_pb_checkpoints_entry_tag.value):
						1:
							if _pb_checkpoints_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_checkpoints_key_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
							if _pb_checkpoints_key_read.error != ProtobufError.OK:
								return _pb_checkpoints_key_read.error
							if _pb_checkpoints_key_read.offset > _pb_checkpoints_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_checkpoints_key = _pb_checkpoints_key_read.value
							_pb_offset = _pb_checkpoints_key_read.offset
						2:
							if _pb_checkpoints_entry_wire_type != Wire.WIRE_LENGTH_DELIMITED:
								return ProtobufError.WIRE_TYPE_MISMATCH
							var _pb_checkpoints_value_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_checkpoints_value)
							if _pb_checkpoints_value_read.error != ProtobufError.OK:
								return _pb_checkpoints_value_read.error
							if _pb_checkpoints_value_read.offset > _pb_checkpoints_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_checkpoints_value_read.offset
						_:
							var _pb_checkpoints_skip: SkipRead = Wire.skip_field(_pb_data, _pb_offset, _pb_checkpoints_entry_wire_type)
							if _pb_checkpoints_skip.error != ProtobufError.OK:
								return _pb_checkpoints_skip.error
							if _pb_checkpoints_skip.offset > _pb_checkpoints_end:
								return ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH
							_pb_offset = _pb_checkpoints_skip.offset
				checkpoints[_pb_checkpoints_key] = _pb_checkpoints_value
			5:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_detail_payload_message: Any = Any.new()
				var _pb_detail_payload_read: SkipRead = Wire.read_message(_pb_data, _pb_offset, _pb_detail_payload_message)
				if _pb_detail_payload_read.error != ProtobufError.OK:
					return _pb_detail_payload_read.error
				detail = EventDetailCase.Payload(_pb_detail_payload_message)
				_pb_offset = _pb_detail_payload_read.offset
			6:
				if _pb_wire_type != Wire.WIRE_LENGTH_DELIMITED:
					return ProtobufError.WIRE_TYPE_MISMATCH
				var _pb_detail_note_read: StringRead = Wire.read_string(_pb_data, _pb_offset)
				if _pb_detail_note_read.error != ProtobufError.OK:
					return _pb_detail_note_read.error
				detail = EventDetailCase.Note(_pb_detail_note_read.value)
				_pb_offset = _pb_detail_note_read.offset
			_:
				var _pb_skipped: SkipRead = Wire.capture_field(_pb_data, _pb_offset, _pb_tag.value, _pb_wire_type, _pb_unknown_fields)
				if _pb_skipped.error != ProtobufError.OK:
					return _pb_skipped.error
				_pb_offset = _pb_skipped.offset
	return ProtobufError.OK
