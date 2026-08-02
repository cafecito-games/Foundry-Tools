# Well-Known Native Semantics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Emit regenerable native-value helpers for `Struct`, `Value`, `ListValue`, `Timestamp`, and `Duration` as real members of the checked-in well-known bindings.

**Architecture:** The existing `wellKnownJSONForms` table remains the declaration-site discriminator for well-known semantics. A focused emitter module maps its Struct/List/Value and Timestamp/Duration forms to additional `fsast.Func` members; ordinary messages, `Any`, and wrappers receive none. Native conversion failures use appended `ProtobufError` cases and tuple results, matching the generated wire factories, while the existing JSON surface keeps its `JsonResult[T]` contracts. The public-`Variant` exception additionally requires the literal canonical `google/protobuf/struct.proto` path and declaration name.

**Tech Stack:** Go 1.26, Foundry Script alpha19, Task, testify.

---

## File Structure

- Create `internal/proto/internal/foundryscript/generator/wellknown_semantics.go`: build the five messages' additional generated members and their recursive conversion bodies.
- Modify `internal/proto/internal/foundryscript/generator/generator.go`: append the selected semantics to each message class independently of the JSON option.
- Modify `internal/proto/internal/foundryscript/generator/generator_test.go`: pin declaration-site selection, method signatures, strict rejection branches, recursion, normalization, and carry logic.
- Modify `internal/proto/internal/foundryscript/generator/public_api.go` and `public_api_test.go`: preserve the public-`Variant` ban except for the exact five Struct/Value/ListValue bridge signatures forced by this well-known API.
- Modify `internal/runtime/data/foundry/proto/protobuf_error.fs` and `internal/runtime/runtime_test.go`: append and pin `STRUCT_KEY_NOT_STRING`, `STRUCT_VALUE_UNREPRESENTABLE`, and `WELL_KNOWN_TIME_OUT_OF_RANGE` without renumbering existing cases.
- Modify `tests/foundry/main.fs`: exercise native recursion/cycle rejections and time normalization/range handling against alpha19.
- Regenerate `internal/runtime/data/foundry/proto/wkt/{Struct,Value,ListValue,Timestamp,Duration}.pb.fs`: checked-in generator output carrying the real class members.
- Modify `README.md`: replace the obsolete blocked/plain-message note with the delivered API and its documented float/int precision behavior.

## Task 1: Pin the emitter and runtime contract

- [ ] Add generator tests whose desired signatures are:

```foundryscript
func to_dictionary() -> Dictionary[String, Variant]
static func from_dictionary(_pb_value: Dictionary) -> (Struct?, ProtobufError)
func to_variant() -> Variant
static func from_variant(_pb_value: Variant) -> (Value?, ProtobufError)
func to_array() -> Array[Variant]
static func from_array(_pb_value: Array[Variant]) -> (ListValue?, ProtobufError)
static func from_unix_time(_pb_value: float) -> (Timestamp?, ProtobufError)
static func now() -> Timestamp
func to_unix_time() -> float
static func from_seconds(_pb_value: float) -> (Duration?, ProtobufError)
func to_seconds() -> float
```

- [ ] Assert the Struct body checks every dictionary key with `typeof(key) != TYPE_STRING`, the Value body matches only nil/bool/int/float/string/dictionary/array, and nested errors return `(null, error)` before assigning to the result.
- [ ] Assert a caller-owned message named `Value` or `Timestamp` receives none of these members, and WKT members are emitted even with the JSON option off.
- [ ] Add runtime tests for the three appended error numbers and test that the narrow public-Variant exception requires an exact bridge signature, canonical WKT declaration, and literal canonical path.
- [ ] Run `go test ./internal/proto/internal/foundryscript/generator ./internal/runtime -count=1`; expect failures because the members/errors and exception do not exist.

## Task 2: Pin Struct/Value/ListValue behavior in the engine

- [ ] Add `check_well_known_native_values()` to `tests/foundry/main.fs` and call it from `_init()`.
- [ ] Build a nested native dictionary containing null, bool, int, float, string, dictionary, and array values; call `Struct.from_dictionary`, then `to_dictionary`, and assert the complete recursive shape survives while the input int returns as float.
- [ ] Pass `{1: "bad"}`, `Vector2(1.0, 2.0)`, and a nested array containing that vector; assert `STRUCT_KEY_NOT_STRING` or `STRUCT_VALUE_UNREPRESENTABLE`, a null result, and no partially built tree.
- [ ] Pass self-referential and mutually recursive Dictionary and Array graphs; assert `STRUCT_VALUE_UNREPRESENTABLE`. Reuse one acyclic Dictionary or Array in sibling positions and assert it is accepted.
- [ ] Run `task foundry:test`; expect lint failure because `Struct.from_dictionary` is absent.

## Task 3: Emit Struct/Value/ListValue members

- [ ] Append two errors after `JSON_ANY_UNSUPPORTED = 11`:

```foundryscript
STRUCT_KEY_NOT_STRING = 12
STRUCT_VALUE_UNREPRESENTABLE = 13
WELL_KNOWN_TIME_OUT_OF_RANGE = 14
```

- [ ] Emit native outbound methods by recursively matching `ValueKindCase`; treat an unset `Value` like its JSON form and return null.
- [ ] Emit strict inbound methods. Use `typeof` rather than `is` for builtins, `float(int_value)` for the documented narrowing, typed output containers, and local construction so a failing branch returns no partial object. Private recursive helpers carry an ancestor path, compare containers with `is_same`, and pop every container after pushing it on every success or failure return; this rejects cycles without rejecting shared acyclic siblings.
- [ ] Restrict `CheckPublicAPI`'s exception to exact whole-line spellings of `to_variant`, `from_variant`, `to_dictionary`, `to_array`, and `from_array` shown in Task 1, and require their exact canonical WKT declaration/form/path; keep every other public Variant signature rejected.
- [ ] Run the focused Go tests, regenerate with `task gen-wkt`, then run `task foundry:test`; expect all focused Struct checks to pass.

## Task 4: Pin and emit Timestamp/Duration behavior

- [ ] Add `check_well_known_time_helpers()` before production changes. Cover Timestamp `1700000000.5`, `-0.25`, a sub-nanosecond input, and a fractional value whose rounded nanos carry; cover Duration `1.25`, `-1.25`, sub-second negative and sub-nanosecond values, and rounded positive/negative carries. Assert Timestamp nanos are `[0, 999999999]`, Duration seconds/nanos have compatible signs, and `now()` is near `Time.get_unix_time_from_system()`.
- [ ] Assert NaN and both infinities fail before integer conversion. Cover exact minimum/maximum seconds, values just outside each range, and rounding that carries beyond the maximum. Document that finite input rounds to the nearest nanosecond, so sub-nanosecond input is not lossless; outbound float conversion can also lose precision.
- [ ] Run `task foundry:test`; expect lint failure because `Timestamp.from_unix_time` is absent.
- [ ] Emit fallible tuple factories that reject non-finite input before integer conversion and validate the normalized protobuf ranges. Use type-safe `floori`/`roundi` Timestamp construction with positive carry and canonical nonnegative nanos. Use truncation-toward-zero Duration construction with `roundi`, both carry directions, nanos in `[-999999999, 999999999]`, and sign-compatible fields. Both outbound methods return `float(seconds) + float(nanos) / 1000000000.0`; `now()` remains infallible by calling a private helper on system time.
- [ ] Run focused Go tests, `task gen-wkt`, and `task foundry:test`; expect all time checks to pass.

## Task 5: Reproducibility, documentation, and full verification

- [ ] Run `task gen-wkt`, verify a second `task gen-wkt` leaves `git diff` unchanged, and run `go test ./internal/runtime -run TestWellKnownBindingsAreUpToDate -count=1`.
- [ ] Update README with fallible method signatures, cycle and strict rejection cases, int-to-float loss beyond 2^53, nearest-nanosecond rounding, Timestamp's canonical negative representation/range, Duration sign normalization/range, and outbound float precision loss. State that `Any` remains unsupported and wrappers remain ordinary messages.
- [ ] Run sequentially: focused generator/runtime tests, `task ci`, `task integration`, and `task foundry:test`.
- [ ] Review `git diff --check`, `git diff --stat`, and the complete diff against issue #43 and the amended design; confirm no `Any`, wrapper, reflection, or unrelated JSON behavior changed.
- [ ] Commit with `feat: add native well-known type conversions`.
