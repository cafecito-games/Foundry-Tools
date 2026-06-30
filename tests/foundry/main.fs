import cafecito.game.v1
import foundry.proto

extends SceneTree

func _init() -> void:
	var player: Player = Player.new()
	player.set_name("Ava")
	player.set_level(7)
	var data: PackedByteArray = player.to_bytes()
	var decoded: DecodeResult[Player] = Player.from_bytes(data)
	if not decoded.is_ok():
		printerr("decode failed")
		quit(1)
		return
	if decoded.value == null:
		printerr("decoded value missing")
		quit(1)
		return
	if decoded.value.get_name() != "Ava":
		printerr("decoded name mismatch")
		quit(1)
		return
	quit(0)
