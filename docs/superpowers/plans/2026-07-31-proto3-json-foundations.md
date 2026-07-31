# proto3 canonical JSON — foundations

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the two prerequisites for JSON emission — a generation option threaded from both front ends into the generator, and the runtime Foundry Script helpers the JSON forms will call — without emitting any JSON yet.

**Architecture:** A new `Options` value is passed to `Generate`, parsed from the protoc `--foundryscript_opt` parameter and from an anvil flag. Four new runtime `.fs` classes under `foundry.proto` implement base64, RFC-3339, duration text, and field-mask path conversion as pure static functions. Nothing in this plan changes generated output, so every existing golden file is unchanged.

**Tech Stack:** Go 1.x with testify, cobra for the CLI, `google.golang.org/protobuf` for the plugin protocol, Foundry Script for the runtime. Engine checks run through `tests/foundry/run.sh`.

**Spec:** `docs/superpowers/specs/2026-07-31-proto3-canonical-json-design.md`. Issue #44.

---

## Scope

This plan covers steps 1 and 2 of the roadmap in #44. It deliberately stops before
the emitter. Out of scope here: `json_serialize.go`, `json_deserialize.go`, the
well-known JSON forms, and the README limitations section.

## File structure

| File | Responsibility |
|---|---|
| `internal/proto/internal/foundryscript/generator/options.go` | New. The `Options` value and its documentation. |
| `internal/proto/internal/foundryscript/generator/generator.go` | Modify. `Generate` and `GenerateIntoRuntimeNamespace` accept `Options` and carry it into `generateFiles`. |
| `internal/proto/api.go` | Modify. Re-export `Options`; widen the two `Generate` wrappers. |
| `internal/plugin/parameter.go` | New. Parses the protoc parameter string into `Options`. |
| `internal/plugin/plugin.go` | Modify. Calls the parser, reports an unknown key as a plugin error. |
| `internal/proto/command.go` | Modify. `--json` flag on `anvil proto generate`. |
| `internal/proto/wellknown/gen/gen.go` | Modify. Generates the runtime bindings with JSON on. |
| `internal/runtime/data/foundry/proto/protobuf_error.fs` | Modify. Five JSON error cases. |
| `internal/runtime/data/foundry/proto/json_base64.fs` | New. base64 encode and validating decode. |
| `internal/runtime/data/foundry/proto/json_timestamp.fs` | New. RFC-3339 format and parse over seconds/nanos. |
| `internal/runtime/data/foundry/proto/json_duration.fs` | New. Duration text format and parse. |
| `internal/runtime/data/foundry/proto/json_field_mask.fs` | New. snake_case to lowerCamelCase path conversion, both directions. |
| `tests/foundry/main.fs` | Modify. Engine-run assertions for each runtime helper. |

Each runtime helper is its own file with its own class, matching how `wire.fs` and the
read carriers are already organized. They share no state; every function is static.

## Assumptions

Foundry Script's `String`, `PackedByteArray`, and `Marshalls` surfaces match Godot's.
This is already relied on in `wire.fs` (`to_utf8_buffer`, `get_string_from_utf8`) and
was confirmed for `JSON` and `Marshalls`. If a specific method turns out to be
missing, the engine lint in `task foundry:test` reports it by name.

## Things that will break later, not now

Two guards enforce that no `Variant` reaches a public generated surface:

- `internal/runtime/runtime_test.go` asserts `runtime.PublicSource(files)` contains no
  `"Variant"` at all.
- `tests/foundry/run.sh` greps generated output for a public `-> Variant` signature.

The JSON design deliberately breaks both, because `to_json_variant()` is a public
`Variant` signature and the well-known bindings are runtime files. Nothing in this
plan trips either guard — none of the four helpers mention `Variant`. Loosening them
belongs to the emitter task, where the exception can be scoped to the JSON entry
points rather than opened generally.

---

### Task 1: Thread a generation Options value through the generator API

**Goal:** `Generate` takes an `Options` value that reaches `generateFiles`, with every
caller passing an explicit value. No behavior changes.

**Files:**
- Create: `internal/proto/internal/foundryscript/generator/options.go`
- Modify: `internal/proto/internal/foundryscript/generator/generator.go:20-38`
- Modify: `internal/proto/api.go:59-69`
- Modify: `internal/proto/command.go:69`
- Modify: `internal/plugin/plugin.go:73`
- Modify: `internal/proto/wellknown/gen/gen.go:83`
- Test: `internal/proto/internal/foundryscript/generator/generator_test.go`

**Acceptance Criteria:**
- [ ] `fsgenerator.Options` exists with a documented `JSON bool` field
- [ ] `Generate` and `GenerateIntoRuntimeNamespace` take `Options` as a final parameter
- [ ] `proto.Options` aliases it, the way `proto.FileEntry` aliases `fsgenerator.FileEntry`
- [ ] The well-known binding generator passes `Options{JSON: true}`
- [ ] Every existing test passes with no golden file changed

**Verify:** `go build ./... && go test -race -count=1 ./...` → all packages pass

**Steps:**

- [ ] **Step 1: Write the failing test**

Add to `internal/proto/internal/foundryscript/generator/generator_test.go`:

```go
func TestGenerateAcceptsOptions(t *testing.T) {
	file := &protoast.ProtoFile{
		Package: "probe.v1",
		Messages: []*protoast.Message{
			{Name: "Probe", Fields: []*protoast.Field{
				{Name: "label", Number: 1, Type: "string"},
			}},
		},
	}

	withJSON, err := fsgenerator.Generate(file, "probe.proto", nil, fsgenerator.Options{JSON: true})
	require.NoError(t, err)

	withoutJSON, err := fsgenerator.Generate(file, "probe.proto", nil, fsgenerator.Options{})
	require.NoError(t, err)

	// The option is threaded but nothing consumes it yet, so both runs agree.
	// This assertion inverts when the JSON emitter lands.
	require.Equal(t, withoutJSON, withJSON)
}
```

Match the surrounding file's construction style — if the existing tests build a
`*protoast.ProtoFile` through a helper, use that helper instead of the literal above.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/proto/internal/foundryscript/generator/ -run TestGenerateAcceptsOptions`
Expected: FAIL, compile error — "too many arguments in call to fsgenerator.Generate"

- [ ] **Step 3: Add the Options type**

Create `internal/proto/internal/foundryscript/generator/options.go`:

```go
package fsgenerator

// Options selects what a generation run emits beyond the protobuf wire codec.
// The zero value is the wire codec alone, which is what a caller that has not
// opted into anything gets.
type Options struct {
	// JSON emits the proto3 canonical JSON mapping on every generated message:
	// to_json_variant, to_json_string, merge_from_json_variant, and a
	// from_json_string constructor. It roughly doubles the size of a generated
	// file, which is why it is opt-in rather than always on.
	JSON bool
}
```

- [ ] **Step 4: Widen the generator entry points**

In `generator.go`, add the parameter to both exported functions and to `generateFiles`:

```go
// Generate renders top-level message and enum skeletons for a proto file.
func Generate(file *protoast.ProtoFile, sourceName string, imports []FileEntry, options Options) (GeneratedFiles, error) {
	return generateFiles(file, sourceName, imports, options, ValidateNamespace)
}

func GenerateIntoRuntimeNamespace(file *protoast.ProtoFile, sourceName string, imports []FileEntry, options Options) (GeneratedFiles, error) {
	return generateFiles(file, sourceName, imports, options, validateNamespaceShape)
}

func generateFiles(
	file *protoast.ProtoFile,
	sourceName string,
	imports []FileEntry,
	options Options,
	validateNamespace func(string) error,
) (GeneratedFiles, error) {
```

Leave the `GenerateIntoRuntimeNamespace` doc comment exactly as it is. `options` is
unused inside `generateFiles` for now; Go does not complain about an unused parameter.

- [ ] **Step 5: Widen the public API wrappers**

In `internal/proto/api.go`, add the alias next to the other type aliases:

```go
// Options selects what a generation run emits beyond the wire codec.
type Options = fsgenerator.Options
```

and widen both wrappers:

```go
func Generate(file *File, sourceName string, imports []FileEntry, options Options) (GeneratedFiles, error) {
	return fsgenerator.Generate(file, sourceName, imports, options)
}

func GenerateIntoRuntimeNamespace(file *File, sourceName string, imports []FileEntry, options Options) (GeneratedFiles, error) {
	return fsgenerator.GenerateIntoRuntimeNamespace(file, sourceName, imports, options)
}
```

- [ ] **Step 6: Update the three production call sites**

`internal/proto/command.go:69` — the flag arrives in Task 3, so pass the zero value:

```go
generatedFiles, err := Generate(parsed.File, parsed.Filename, ImportsOf(parsed), Options{})
```

`internal/plugin/plugin.go:73` — the parameter parser arrives in Task 2:

```go
generated, err := foundryproto.Generate(file, name, importsFor(name, dependencies, filesByName), foundryproto.Options{})
```

`internal/proto/wellknown/gen/gen.go:83` — the runtime bindings always carry JSON,
because a consumer that enables JSON needs the well-known types to have it too:

```go
generated, err := proto.GenerateIntoRuntimeNamespace(parsed.File, sourceName, proto.ImportsOf(parsed), proto.Options{JSON: true})
```

- [ ] **Step 7: Update the test call sites**

There are 45 `Generate(` calls across seven test files. Let the compiler list them:

```bash
go build ./... && go vet ./... 2>&1 | grep "not enough arguments"
```

Each takes an added `fsgenerator.Options{}` (or `proto.Options{}` / `foundryproto.Options{}`
depending on the importing package). Mechanical; no test's meaning changes.

- [ ] **Step 8: Run the full suite**

Run: `go build ./... && go test -race -count=1 ./...`
Expected: PASS, including `TestGenerateAcceptsOptions` and every golden test unchanged

- [ ] **Step 9: Commit**

```bash
git add internal/proto internal/plugin
git commit -m "Thread a generation options value through the generator API"
```

---

### Task 2: Parse the protoc plugin parameter

**Goal:** `protoc --foundryscript_opt=json` turns JSON on; an unrecognized key is
reported as a plugin error rather than ignored.

**Files:**
- Create: `internal/plugin/parameter.go`
- Modify: `internal/plugin/plugin.go:20-52`
- Test: `internal/plugin/plugin_test.go`

**Acceptance Criteria:**
- [ ] An empty parameter yields the zero `Options`
- [ ] `json` yields `Options{JSON: true}`
- [ ] Keys are comma-separated, and surrounding whitespace on a key is ignored
- [ ] An unknown key produces a `CodeGeneratorResponse.Error` naming the key
- [ ] The error is written as a response, not returned from `Run` — matching how every
      other failure in `Run` is reported

**Verify:** `go test -race -count=1 ./internal/plugin/` → PASS

**Steps:**

- [ ] **Step 1: Write the failing tests**

Add to `internal/plugin/plugin_test.go`. The existing tests build a request through a
helper in that file; reuse it and set `Parameter` on the result.

```go
func TestParseParameterAcceptsJSON(t *testing.T) {
	options, err := parseParameter("json")
	require.NoError(t, err)
	require.True(t, options.JSON)
}

func TestParseParameterAcceptsAnEmptyString(t *testing.T) {
	options, err := parseParameter("")
	require.NoError(t, err)
	require.False(t, options.JSON)
}

func TestParseParameterIgnoresSurroundingWhitespaceAndEmptyEntries(t *testing.T) {
	options, err := parseParameter(" json , ")
	require.NoError(t, err)
	require.True(t, options.JSON)
}

func TestParseParameterRejectsAnUnknownKey(t *testing.T) {
	_, err := parseParameter("json,jsonn")
	require.Error(t, err)
	require.Contains(t, err.Error(), "jsonn")
}
```

The parser is unexported, so this test must be in package `plugin`, not `plugin_test`.
If `plugin_test.go` declares `package plugin_test`, put these four in a new
`internal/plugin/parameter_test.go` declaring `package plugin`, and keep the
end-to-end test below in `plugin_test.go`.

```go
func TestRunReportsAnUnknownParameterKey(t *testing.T) {
	req := requestWith(t, "probe.proto", "probe.v1", "Probe")
	req.Parameter = proto.String("nonsense")

	resp := runPlugin(t, req)

	require.Contains(t, resp.GetError(), "nonsense")
	require.Empty(t, resp.GetFile())
}
```

Name `requestWith` and `runPlugin` after whatever the existing helpers in
`plugin_test.go` are actually called.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/plugin/ -run 'TestParseParameter|TestRunReportsAnUnknownParameterKey'`
Expected: FAIL — "undefined: parseParameter"

- [ ] **Step 3: Write the parser**

Create `internal/plugin/parameter.go`:

```go
package plugin

import (
	"fmt"
	"strings"

	foundryproto "github.com/cafecito-games/foundry-tools/internal/proto"
)

// parseParameter turns protoc's --foundryscript_opt string into generation
// options. protoc joins repeated _opt flags with commas, so "json" and
// "json,json" both mean the same thing.
//
// An unrecognized key is an error rather than something to ignore: a misspelled
// option that silently does nothing produces output the caller did not ask for
// and has no way to notice.
func parseParameter(parameter string) (foundryproto.Options, error) {
	var options foundryproto.Options
	for _, entry := range strings.Split(parameter, ",") {
		switch key := strings.TrimSpace(entry); key {
		case "":
			continue
		case "json":
			options.JSON = true
		default:
			return foundryproto.Options{}, fmt.Errorf("unknown generator option %q", key)
		}
	}
	return options, nil
}
```

- [ ] **Step 4: Call it from Run**

In `plugin.go`, immediately after the request unmarshals and before
`FromCodeGeneratorRequest`:

```go
	options, err := parseParameter(req.GetParameter())
	if err != nil {
		return writeError(out, err.Error())
	}
```

and pass it at line 73:

```go
		generated, err := foundryproto.Generate(file, name, importsFor(name, dependencies, filesByName), options)
```

- [ ] **Step 5: Run the tests**

Run: `go test -race -count=1 ./internal/plugin/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/plugin
git commit -m "Parse the protoc generator parameter into generation options"
```

---

### Task 3: Add the anvil --json flag

**Goal:** `anvil proto generate --json` turns JSON on for the direct CLI, matching the
plugin.

**Files:**
- Modify: `internal/proto/command.go:37-88`
- Test: `internal/proto/command_test.go`

**Acceptance Criteria:**
- [ ] `--json` is registered on the generate command with help text
- [ ] Its value reaches `Generate`
- [ ] Omitting it leaves output identical to today

**Verify:** `go test -race -count=1 ./internal/proto/` → PASS

**Steps:**

- [ ] **Step 1: Write the failing test**

Add to `internal/proto/command_test.go`. Follow the file's existing pattern for
building a temp proto and invoking the command — `TestGenerateSkipsWellKnownFiles`
at line 67 is the closest model.

```go
func TestGenerateAcceptsTheJSONFlag(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "probe.proto")
	require.NoError(t, os.WriteFile(source, []byte(`
syntax = "proto3";
package probe.v1;
message Probe { string label = 1; }
`), 0o600))

	out := filepath.Join(dir, "out")
	var stdout bytes.Buffer
	cmd := NewCommand(&stdout)
	cmd.SetArgs([]string{"generate", "--json", "-I", dir, "-o", out, source})

	require.NoError(t, cmd.Execute())
	require.FileExists(t, filepath.Join(out, "probe", "v1", "Probe.pb.fs"))
}
```

Confirm the generated output path against an existing test before asserting on it —
the namespace-to-path mapping is `NamespaceFor`, not the proto package verbatim.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/proto/ -run TestGenerateAcceptsTheJSONFlag`
Expected: FAIL — "unknown flag: --json"

- [ ] **Step 3: Add the flag**

In `newGenerateCommand`, extend the options struct:

```go
	var opts struct {
		outDir     string
		importPath []string
		json       bool
	}
```

pass it at the `Generate` call:

```go
				generatedFiles, err := Generate(parsed.File, parsed.Filename, ImportsOf(parsed), Options{JSON: opts.json})
```

and register it beside the others:

```go
	cmd.Flags().BoolVar(&opts.json, "json", false, "emit proto3 canonical JSON methods")
```

- [ ] **Step 4: Run the tests**

Run: `go test -race -count=1 ./internal/proto/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/proto
git commit -m "Add the anvil --json generation flag"
```

---

### Task 4: Add the JSON error cases to ProtobufError

**Goal:** The five error cases the JSON paths return exist in the runtime enum, with
existing numbering untouched.

**Files:**
- Modify: `internal/runtime/data/foundry/proto/protobuf_error.fs`
- Test: `internal/runtime/runtime_test.go`

**Acceptance Criteria:**
- [ ] Cases 7 through 11 are `JSON_PARSE_FAILED`, `JSON_TYPE_MISMATCH`,
      `JSON_UNKNOWN_FIELD`, `JSON_VALUE_OUT_OF_RANGE`, `JSON_ANY_UNSUPPORTED`
- [ ] Cases 0 through 6 keep their existing numbers
- [ ] The enum still lints in the engine

**Verify:** `go test -race -count=1 ./internal/runtime/` → PASS, then
`task foundry:test` → exit 0

**Steps:**

- [ ] **Step 1: Write the failing test**

Add to `internal/runtime/runtime_test.go`:

```go
func TestProtobufErrorCarriesTheJSONCases(t *testing.T) {
	source := runtime.Files()["foundry/proto/protobuf_error.fs"]

	require.Contains(t, source, "UNKNOWN_REQUIRED_FEATURE = 6")
	require.Contains(t, source, "JSON_PARSE_FAILED = 7")
	require.Contains(t, source, "JSON_TYPE_MISMATCH = 8")
	require.Contains(t, source, "JSON_UNKNOWN_FIELD = 9")
	require.Contains(t, source, "JSON_VALUE_OUT_OF_RANGE = 10")
	require.Contains(t, source, "JSON_ANY_UNSUPPORTED = 11")
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/runtime/ -run TestProtobufErrorCarriesTheJSONCases`
Expected: FAIL — the JSON cases are absent

- [ ] **Step 3: Extend the enum**

`internal/runtime/data/foundry/proto/protobuf_error.fs` becomes:

```
namespace foundry.proto

enum_name ProtobufError:
	OK = 0
	VARINT_NOT_FOUND = 1
	VARINT_TOO_LONG = 2
	WIRE_TYPE_MISMATCH = 3
	LENGTH_DELIMITED_SIZE_NOT_FOUND = 4
	LENGTH_DELIMITED_SIZE_MISMATCH = 5
	UNKNOWN_REQUIRED_FEATURE = 6
	## The document is not well-formed JSON.
	JSON_PARSE_FAILED = 7
	## A well-formed JSON value has the wrong shape for the field it is being
	## read into: an object where a string belongs, a string where a number does.
	JSON_TYPE_MISMATCH = 8
	## A JSON object member matches no field. Canonical JSON has no unknown-field
	## buffer, so there is nothing to preserve it in and it is refused instead.
	JSON_UNKNOWN_FIELD = 9
	## A number falls outside its field's domain, which includes a timestamp
	## outside the range RFC 3339 can express.
	JSON_VALUE_OUT_OF_RANGE = 10
	## google.protobuf.Any has no JSON form yet; it needs a type-URL registry.
	JSON_ANY_UNSUPPORTED = 11
```

- [ ] **Step 4: Run the test and the engine check**

Run: `go test -race -count=1 ./internal/runtime/`
Expected: PASS

Run: `task foundry:test`
Expected: exit 0 — the enum lints and the existing round-trip checks still pass

- [ ] **Step 5: Commit**

```bash
git add internal/runtime
git commit -m "Add the JSON error cases to ProtobufError"
```

---

### Task 5: Add the base64 runtime helper

**Goal:** `bytes` fields can move to and from a JSON base64 string, accepting both the
standard and URL-safe alphabets and tolerating missing padding, as the canonical
mapping requires.

**Files:**
- Create: `internal/runtime/data/foundry/proto/json_base64.fs`
- Modify: `tests/foundry/main.fs`

**Acceptance Criteria:**
- [ ] `JsonBase64.encode` produces standard-alphabet, padded base64
- [ ] `JsonBase64.decode` accepts standard and URL-safe input, with or without padding
- [ ] `decode` returns `JSON_TYPE_MISMATCH` for a character outside both alphabets
- [ ] Round-tripping an arbitrary byte string returns the same bytes

**Verify:** `task foundry:test` → exit 0

**Steps:**

- [ ] **Step 1: Write the failing engine checks**

In `tests/foundry/main.fs`, inside `_init()`, alongside the existing checks:

```
	var base64_source: PackedByteArray = PackedByteArray([0, 1, 250, 255, 16, 32])
	var base64_text: String = JsonBase64.encode(base64_source)
	var (base64_decoded, base64_error) = JsonBase64.decode(base64_text)
	check(base64_error == ProtobufError.OK, "base64 round trip decodes")
	check(base64_decoded == base64_source, "base64 round trip preserves bytes")

	var (base64_padded, base64_padded_error) = JsonBase64.decode("AAH6")
	check(base64_padded_error == ProtobufError.OK, "base64 accepts a full quantum")
	check(base64_padded == PackedByteArray([0, 1, 250]), "base64 decodes a full quantum")

	var (base64_unpadded, base64_unpadded_error) = JsonBase64.decode("AAE")
	check(base64_unpadded_error == ProtobufError.OK, "base64 accepts missing padding")
	check(base64_unpadded == PackedByteArray([0, 1]), "base64 decodes missing padding")

	var (base64_url_safe, base64_url_safe_error) = JsonBase64.decode("-_8")
	check(base64_url_safe_error == ProtobufError.OK, "base64 accepts the URL-safe alphabet")

	var (_base64_bad, base64_bad_error) = JsonBase64.decode("not base64!")
	check(base64_bad_error == ProtobufError.JSON_TYPE_MISMATCH, "base64 rejects a stray character")
```

Add `import foundry.proto` only if `main.fs` does not already have it — it does, at
line 3.

- [ ] **Step 2: Run to verify it fails**

Run: `task foundry:test`
Expected: FAIL — the engine lint reports `JsonBase64` as an unresolved identifier

- [ ] **Step 3: Write the helper**

Create `internal/runtime/data/foundry/proto/json_base64.fs`:

```
namespace foundry.proto

## Base64 for the canonical JSON mapping of a bytes field.
##
## Output is always the standard alphabet with padding, which is what the
## canonical form specifies. Input is looser on purpose: the mapping requires a
## parser to accept the URL-safe alphabet as well, and padding is optional in
## practice, so both are normalized before the engine decoder sees them.
class_name JsonBase64 extends RefCounted

static func encode(value: PackedByteArray) -> String:
	return Marshalls.raw_to_base64(value)

static func decode(text: String) -> (PackedByteArray, ProtobufError):
	var normalized: String = text.replace("-", "+").replace("_", "/")
	var remainder: int = normalized.length() % 4
	if remainder == 1:
		return (PackedByteArray(), ProtobufError.JSON_TYPE_MISMATCH)
	if remainder == 2:
		normalized += "=="
	elif remainder == 3:
		normalized += "="
	if not _is_base64(normalized):
		return (PackedByteArray(), ProtobufError.JSON_TYPE_MISMATCH)
	return (Marshalls.base64_to_raw(normalized), ProtobufError.OK)

## The engine decoder does not report a bad character, so the alphabet is
## checked here rather than being told about a silently truncated result.
static func _is_base64(text: String) -> bool:
	var index: int = 0
	var padding_started: bool = false
	while index < text.length():
		var character: String = text.substr(index, 1)
		if character == "=":
			padding_started = true
		elif padding_started:
			return false
		elif not _is_base64_character(character):
			return false
		index += 1
	return true

static func _is_base64_character(character: String) -> bool:
	if character >= "A" and character <= "Z":
		return true
	if character >= "a" and character <= "z":
		return true
	if character >= "0" and character <= "9":
		return true
	return character == "+" or character == "/"
```

- [ ] **Step 4: Run the engine checks**

Run: `task foundry:test`
Expected: exit 0

If the engine rejects `String` comparison with `>=`, replace `_is_base64_character`
with a lookup against a constant alphabet string:
`return "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/".find(character) >= 0`

- [ ] **Step 5: Confirm the runtime tests still hold**

Run: `go test -race -count=1 ./internal/runtime/`
Expected: PASS — in particular the assertion that no runtime source mentions `Variant`

- [ ] **Step 6: Commit**

```bash
git add internal/runtime tests/foundry/main.fs
git commit -m "Add the base64 runtime helper for JSON bytes fields"
```

---

### Task 6: Add the RFC-3339 timestamp helper

**Goal:** A `Timestamp`'s seconds and nanos convert to and from an RFC-3339 string, in
the canonical form on the way out and accepting offsets on the way in.

**Files:**
- Create: `internal/runtime/data/foundry/proto/json_timestamp.fs`
- Modify: `tests/foundry/main.fs`

**Acceptance Criteria:**
- [ ] `format` emits UTC with a `Z` suffix and 0, 3, 6, or 9 fractional digits
- [ ] `format` returns `JSON_VALUE_OUT_OF_RANGE` outside
      `[-62135596800, 253402300799]` seconds or `[0, 999999999]` nanos
- [ ] `parse` accepts `Z`, `z`, and a `±HH:MM` offset
- [ ] `parse` accepts any fractional digit count from 1 to 9
- [ ] `parse` returns `JSON_TYPE_MISMATCH` for a malformed string
- [ ] A pre-epoch instant round-trips

**Verify:** `task foundry:test` → exit 0

**Steps:**

- [ ] **Step 1: Write the failing engine checks**

In `tests/foundry/main.fs`:

```
	var (epoch_text, epoch_error) = JsonTimestamp.format(0, 0)
	check(epoch_error == ProtobufError.OK, "epoch formats")
	check(epoch_text == "1970-01-01T00:00:00Z", "epoch formats canonically")

	var (fraction_text, fraction_error) = JsonTimestamp.format(1136214245, 10000000)
	check(fraction_error == ProtobufError.OK, "fractional timestamp formats")
	check(fraction_text == "2006-01-02T15:04:05.010Z", "fraction uses three digits")

	var (nano_text, _nano_error) = JsonTimestamp.format(0, 1)
	check(nano_text == "1970-01-01T00:00:00.000000001Z", "a single nanosecond uses nine digits")

	var (pre_epoch_text, _pre_epoch_error) = JsonTimestamp.format(-62135596800, 0)
	check(pre_epoch_text == "0001-01-01T00:00:00Z", "the lower bound formats")

	var (_range_text, range_error) = JsonTimestamp.format(253402300800, 0)
	check(range_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "an out-of-range second is refused")

	var (parsed_seconds, parsed_nanos, parse_error) = JsonTimestamp.parse("2006-01-02T15:04:05.010Z")
	check(parse_error == ProtobufError.OK, "an RFC 3339 string parses")
	check(parsed_seconds == 1136214245, "parsed seconds")
	check(parsed_nanos == 10000000, "parsed nanos")

	var (offset_seconds, _offset_nanos, offset_error) = JsonTimestamp.parse("2006-01-02T16:04:05+01:00")
	check(offset_error == ProtobufError.OK, "an offset parses")
	check(offset_seconds == 1136214245, "an offset is folded into UTC")

	var (lowercase_seconds, _lowercase_nanos, lowercase_error) = JsonTimestamp.parse("1970-01-01t00:00:00z")
	check(lowercase_error == ProtobufError.OK, "lowercase designators parse")
	check(lowercase_seconds == 0, "lowercase designators give the epoch")

	var (_bad_seconds, _bad_nanos, bad_error) = JsonTimestamp.parse("2006-01-02")
	check(bad_error == ProtobufError.JSON_TYPE_MISMATCH, "a date alone is refused")
```

- [ ] **Step 2: Run to verify it fails**

Run: `task foundry:test`
Expected: FAIL — `JsonTimestamp` is unresolved

- [ ] **Step 3: Write the helper**

Create `internal/runtime/data/foundry/proto/json_timestamp.fs`:

```
namespace foundry.proto

## RFC 3339 conversion for the canonical JSON mapping of google.protobuf.Timestamp.
##
## The calendar math is the proleptic Gregorian days-from-civil algorithm, which
## is closed-form in both directions and needs no lookup table. Both functions
## work in seconds since the Unix epoch plus a non-negative nanosecond remainder,
## which is exactly how the message stores them.
class_name JsonTimestamp extends RefCounted

## 0001-01-01T00:00:00Z, the earliest instant RFC 3339 can express.
const MINIMUM_SECONDS: int = -62135596800

## 9999-12-31T23:59:59Z, the latest.
const MAXIMUM_SECONDS: int = 253402300799

const SECONDS_PER_DAY: int = 86400

static func format(seconds: int, nanos: int) -> (String, ProtobufError):
	if seconds < MINIMUM_SECONDS or seconds > MAXIMUM_SECONDS:
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	if nanos < 0 or nanos > 999999999:
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)

	var days: int = _floor_divide(seconds, SECONDS_PER_DAY)
	var second_of_day: int = seconds - days * SECONDS_PER_DAY
	var (year, month, day) = _civil_from_days(days)

	var text: String = _pad(year, 4) + "-" + _pad(month, 2) + "-" + _pad(day, 2)
	text += "T" + _pad(second_of_day / 3600, 2)
	text += ":" + _pad((second_of_day / 60) % 60, 2)
	text += ":" + _pad(second_of_day % 60, 2)
	text += _format_fraction(nanos)
	return (text + "Z", ProtobufError.OK)

## The canonical form uses whichever of 0, 3, 6, or 9 fractional digits
## represents the value exactly.
static func _format_fraction(nanos: int) -> String:
	if nanos == 0:
		return ""
	if nanos % 1000000 == 0:
		return "." + _pad(nanos / 1000000, 3)
	if nanos % 1000 == 0:
		return "." + _pad(nanos / 1000, 6)
	return "." + _pad(nanos, 9)

static func parse(text: String) -> (int, int, ProtobufError):
	if text.length() < 20:
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
	if text.substr(4, 1) != "-" or text.substr(7, 1) != "-":
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
	var designator: String = text.substr(10, 1)
	if designator != "T" and designator != "t":
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
	if text.substr(13, 1) != ":" or text.substr(16, 1) != ":":
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)

	var (year, year_ok) = _digits(text, 0, 4)
	var (month, month_ok) = _digits(text, 5, 2)
	var (day, day_ok) = _digits(text, 8, 2)
	var (hour, hour_ok) = _digits(text, 11, 2)
	var (minute, minute_ok) = _digits(text, 14, 2)
	var (second, second_ok) = _digits(text, 17, 2)
	if not (year_ok and month_ok and day_ok and hour_ok and minute_ok and second_ok):
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
	if month < 1 or month > 12 or day < 1 or day > 31:
		return (0, 0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	if hour > 23 or minute > 59 or second > 59:
		return (0, 0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)

	var cursor: int = 19
	var nanos: int = 0
	if cursor < text.length() and text.substr(cursor, 1) == ".":
		cursor += 1
		var digits: int = 0
		while cursor + digits < text.length() and _is_digit(text.substr(cursor + digits, 1)):
			digits += 1
		if digits == 0 or digits > 9:
			return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
		var (fraction, fraction_ok) = _digits(text, cursor, digits)
		if not fraction_ok:
			return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
		nanos = fraction
		var scale: int = 9 - digits
		while scale > 0:
			nanos *= 10
			scale -= 1
		cursor += digits

	var (offset_seconds, offset_ok) = _parse_offset(text, cursor)
	if not offset_ok:
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)

	var days: int = _days_from_civil(year, month, day)
	var seconds: int = days * SECONDS_PER_DAY + hour * 3600 + minute * 60 + second
	seconds -= offset_seconds
	if seconds < MINIMUM_SECONDS or seconds > MAXIMUM_SECONDS:
		return (0, 0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	return (seconds, nanos, ProtobufError.OK)

## Returns the offset in seconds east of UTC, and whether the suffix was
## well-formed. A `Z` is a zero offset.
static func _parse_offset(text: String, cursor: int) -> (int, bool):
	if cursor >= text.length():
		return (0, false)
	var designator: String = text.substr(cursor, 1)
	if designator == "Z" or designator == "z":
		if cursor + 1 != text.length():
			return (0, false)
		return (0, true)
	if designator != "+" and designator != "-":
		return (0, false)
	if cursor + 6 != text.length() or text.substr(cursor + 3, 1) != ":":
		return (0, false)
	var (hours, hours_ok) = _digits(text, cursor + 1, 2)
	var (minutes, minutes_ok) = _digits(text, cursor + 4, 2)
	if not (hours_ok and minutes_ok) or hours > 23 or minutes > 59:
		return (0, false)
	var total: int = hours * 3600 + minutes * 60
	if designator == "-":
		return (-total, true)
	return (total, true)

static func _digits(text: String, offset: int, length: int) -> (int, bool):
	if offset + length > text.length():
		return (0, false)
	var value: int = 0
	var index: int = 0
	while index < length:
		var character: String = text.substr(offset + index, 1)
		if not _is_digit(character):
			return (0, false)
		value = value * 10 + (character.unicode_at(0) - 48)
		index += 1
	return (value, true)

static func _is_digit(character: String) -> bool:
	return character >= "0" and character <= "9"

static func _pad(value: int, width: int) -> String:
	var text: String = str(value)
	while text.length() < width:
		text = "0" + text
	return text

## Integer division that rounds toward negative infinity. The engine truncates
## toward zero, which would put a pre-epoch instant on the wrong day.
static func _floor_divide(numerator: int, denominator: int) -> int:
	var quotient: int = numerator / denominator
	if numerator % denominator != 0 and (numerator < 0) != (denominator < 0):
		quotient -= 1
	return quotient

## Howard Hinnant's civil-from-days, shifted to an era beginning on 0000-03-01
## so that the leap day falls at the end of a year and needs no special case.
static func _civil_from_days(days: int) -> (int, int, int):
	var shifted: int = days + 719468
	var era: int = _floor_divide(shifted, 146097)
	var day_of_era: int = shifted - era * 146097
	var year_of_era: int = (day_of_era - day_of_era / 1460 + day_of_era / 36524 - day_of_era / 146096) / 365
	var year: int = year_of_era + era * 400
	var day_of_year: int = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100)
	var month_prime: int = (5 * day_of_year + 2) / 153
	var day: int = day_of_year - (153 * month_prime + 2) / 5 + 1
	var month: int = month_prime + 3
	if month_prime >= 10:
		month = month_prime - 9
	if month <= 2:
		year += 1
	return (year, month, day)

## The inverse of _civil_from_days.
static func _days_from_civil(year: int, month: int, day: int) -> int:
	var shifted_year: int = year
	if month <= 2:
		shifted_year -= 1
	var era: int = _floor_divide(shifted_year, 400)
	var year_of_era: int = shifted_year - era * 400
	var month_prime: int = month + 9
	if month > 2:
		month_prime = month - 3
	var day_of_year: int = (153 * month_prime + 2) / 5 + day - 1
	var day_of_era: int = year_of_era * 365 + year_of_era / 4 - year_of_era / 100 + day_of_year
	return era * 146097 + day_of_era - 719468
```

- [ ] **Step 4: Run the engine checks**

Run: `task foundry:test`
Expected: exit 0

If `str(value)` is not available, use `String.num_int64(value)`. If `unicode_at` is
not available, subtract using `"0123456789".find(character)` instead.

- [ ] **Step 5: Commit**

```bash
git add internal/runtime tests/foundry/main.fs
git commit -m "Add the RFC 3339 timestamp runtime helper"
```

---

### Task 7: Add the duration text helper

**Goal:** A `Duration`'s seconds and nanos convert to and from the canonical
`"1.000000001s"` form, including negative durations.

**Files:**
- Create: `internal/runtime/data/foundry/proto/json_duration.fs`
- Modify: `tests/foundry/main.fs`

**Acceptance Criteria:**
- [ ] `format` emits 0, 3, 6, or 9 fractional digits and always the `s` suffix
- [ ] A negative duration formats with a single leading `-`, whether the whole
      seconds are zero or not
- [ ] `format` refuses a seconds/nanos pair whose signs disagree
- [ ] `parse` requires the `s` suffix and returns `JSON_TYPE_MISMATCH` without it
- [ ] `parse("-0.5s")` yields seconds 0 and nanos -500000000

**Verify:** `task foundry:test` → exit 0

**Steps:**

- [ ] **Step 1: Write the failing engine checks**

In `tests/foundry/main.fs`:

```
	var (duration_text, duration_error) = JsonDuration.format(3, 1)
	check(duration_error == ProtobufError.OK, "a duration formats")
	check(duration_text == "3.000000001s", "a duration uses nine digits when it must")

	var (whole_text, _whole_error) = JsonDuration.format(3, 0)
	check(whole_text == "3s", "a whole duration has no fraction")

	var (negative_text, _negative_error) = JsonDuration.format(0, -500000000)
	check(negative_text == "-0.500s", "a sub-second negative duration keeps its sign")

	var (_mixed_text, mixed_error) = JsonDuration.format(1, -1)
	check(mixed_error == ProtobufError.JSON_VALUE_OUT_OF_RANGE, "disagreeing signs are refused")

	var (duration_seconds, duration_nanos, duration_parse_error) = JsonDuration.parse("-0.5s")
	check(duration_parse_error == ProtobufError.OK, "a negative duration parses")
	check(duration_seconds == 0, "a sub-second duration has zero seconds")
	check(duration_nanos == -500000000, "a sub-second duration carries the sign in nanos")

	var (_no_suffix_seconds, _no_suffix_nanos, no_suffix_error) = JsonDuration.parse("3")
	check(no_suffix_error == ProtobufError.JSON_TYPE_MISMATCH, "a missing suffix is refused")
```

- [ ] **Step 2: Run to verify it fails**

Run: `task foundry:test`
Expected: FAIL — `JsonDuration` is unresolved

- [ ] **Step 3: Write the helper**

Create `internal/runtime/data/foundry/proto/json_duration.fs`:

```
namespace foundry.proto

## Text conversion for the canonical JSON mapping of google.protobuf.Duration.
##
## A Duration stores a signed second count and a signed nanosecond remainder
## whose signs must agree, so the sign is pulled out once and both components
## are formatted from their magnitudes. That is also why a sub-second negative
## duration needs care: its seconds are zero, and only the nanos carry the sign.
class_name JsonDuration extends RefCounted

const MAXIMUM_SECONDS: int = 315576000000

static func format(seconds: int, nanos: int) -> (String, ProtobufError):
	if seconds > MAXIMUM_SECONDS or seconds < -MAXIMUM_SECONDS:
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	if nanos > 999999999 or nanos < -999999999:
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	if (seconds > 0 and nanos < 0) or (seconds < 0 and nanos > 0):
		return ("", ProtobufError.JSON_VALUE_OUT_OF_RANGE)

	var sign: String = ""
	if seconds < 0 or nanos < 0:
		sign = "-"
	var whole: int = seconds
	if whole < 0:
		whole = -whole
	var fraction: int = nanos
	if fraction < 0:
		fraction = -fraction
	return (sign + str(whole) + _format_fraction(fraction) + "s", ProtobufError.OK)

static func _format_fraction(nanos: int) -> String:
	if nanos == 0:
		return ""
	if nanos % 1000000 == 0:
		return "." + _pad(nanos / 1000000, 3)
	if nanos % 1000 == 0:
		return "." + _pad(nanos / 1000, 6)
	return "." + _pad(nanos, 9)

static func parse(text: String) -> (int, int, ProtobufError):
	if text.length() < 2 or text.substr(text.length() - 1, 1) != "s":
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
	var body: String = text.substr(0, text.length() - 1)

	var negative: bool = false
	if body.substr(0, 1) == "-":
		negative = true
		body = body.substr(1, body.length() - 1)
	elif body.substr(0, 1) == "+":
		body = body.substr(1, body.length() - 1)

	var point: int = body.find(".")
	var whole_text: String = body
	var fraction_text: String = ""
	if point >= 0:
		whole_text = body.substr(0, point)
		fraction_text = body.substr(point + 1, body.length() - point - 1)
		if fraction_text.length() == 0 or fraction_text.length() > 9:
			return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)

	var (whole, whole_ok) = _digits(whole_text)
	if not whole_ok:
		return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)

	var nanos: int = 0
	if fraction_text.length() > 0:
		var (fraction, fraction_ok) = _digits(fraction_text)
		if not fraction_ok:
			return (0, 0, ProtobufError.JSON_TYPE_MISMATCH)
		nanos = fraction
		var scale: int = 9 - fraction_text.length()
		while scale > 0:
			nanos *= 10
			scale -= 1

	if whole > MAXIMUM_SECONDS:
		return (0, 0, ProtobufError.JSON_VALUE_OUT_OF_RANGE)
	if negative:
		return (-whole, -nanos, ProtobufError.OK)
	return (whole, nanos, ProtobufError.OK)

static func _digits(text: String) -> (int, bool):
	if text.length() == 0:
		return (0, false)
	var value: int = 0
	var index: int = 0
	while index < text.length():
		var character: String = text.substr(index, 1)
		if character < "0" or character > "9":
			return (0, false)
		value = value * 10 + (character.unicode_at(0) - 48)
		index += 1
	return (value, true)

static func _pad(value: int, width: int) -> String:
	var text: String = str(value)
	while text.length() < width:
		text = "0" + text
	return text
```

`_pad` here is identical to `JsonTimestamp._pad`, and `_digits` is the whole-string
form of it. Both stay private to each class rather than being shared: Foundry Script
has no free functions, and a shared class for two small helpers is not yet worth the
indirection. If a third caller appears, promote them to a `JsonText` class rather than
growing a third copy.

- [ ] **Step 4: Run the engine checks**

Run: `task foundry:test`
Expected: exit 0

- [ ] **Step 5: Commit**

```bash
git add internal/runtime tests/foundry/main.fs
git commit -m "Add the duration text runtime helper"
```

---

### Task 8: Add the field-mask path helper

**Goal:** A `FieldMask`'s paths convert between proto `snake_case` and JSON
`lowerCamelCase`, refusing a path that cannot round-trip.

**Files:**
- Create: `internal/runtime/data/foundry/proto/json_field_mask.fs`
- Modify: `tests/foundry/main.fs`

**Acceptance Criteria:**
- [ ] `to_json` joins paths with `,` and camelCases each dot-separated segment
- [ ] `to_json` returns `JSON_TYPE_MISMATCH` for a path containing an uppercase
      letter, which cannot round-trip
- [ ] `from_json` splits on `,` and reverses the conversion
- [ ] `from_json("")` yields an empty array, not one empty path
- [ ] `foo_bar.baz_qux` and `fooBar.bazQux` convert to each other

**Verify:** `task foundry:test` → exit 0

**Steps:**

- [ ] **Step 1: Write the failing engine checks**

In `tests/foundry/main.fs`:

```
	var (mask_text, mask_error) = JsonFieldMask.to_json(["foo_bar.baz_qux", "user"])
	check(mask_error == ProtobufError.OK, "a field mask converts to JSON")
	check(mask_text == "fooBar.bazQux,user", "a field mask camelCases each segment")

	var (_upper_text, upper_error) = JsonFieldMask.to_json(["fooBar"])
	check(upper_error == ProtobufError.JSON_TYPE_MISMATCH, "an uppercase path is refused")

	var (mask_paths, mask_paths_error) = JsonFieldMask.from_json("fooBar.bazQux,user")
	check(mask_paths_error == ProtobufError.OK, "a field mask parses")
	check(mask_paths == ["foo_bar.baz_qux", "user"], "a field mask restores snake_case")

	var (empty_paths, empty_error) = JsonFieldMask.from_json("")
	check(empty_error == ProtobufError.OK, "an empty mask parses")
	check(empty_paths.size() == 0, "an empty mask has no paths")
```

- [ ] **Step 2: Run to verify it fails**

Run: `task foundry:test`
Expected: FAIL — `JsonFieldMask` is unresolved

- [ ] **Step 3: Write the helper**

Create `internal/runtime/data/foundry/proto/json_field_mask.fs`:

```
namespace foundry.proto

## Path conversion for the canonical JSON mapping of google.protobuf.FieldMask.
##
## A mask serializes as one string of comma-joined paths, each path a
## dot-separated chain of field names carried in lowerCamelCase. The conversion
## is only reversible when the proto field names are lower_snake_case, so a path
## that already contains an uppercase letter is refused rather than emitted as
## something that would come back different.
class_name JsonFieldMask extends RefCounted

static func to_json(paths: Array[String]) -> (String, ProtobufError):
	var converted: Array[String] = []
	for path in paths:
		if not _is_lower_snake_case(path):
			return ("", ProtobufError.JSON_TYPE_MISMATCH)
		converted.append(_to_camel_case(path))
	return (",".join(converted), ProtobufError.OK)

static func from_json(text: String) -> (Array[String], ProtobufError):
	var paths: Array[String] = []
	if text.length() == 0:
		return (paths, ProtobufError.OK)
	for entry in text.split(","):
		paths.append(_to_snake_case(entry))
	return (paths, ProtobufError.OK)

static func _to_camel_case(path: String) -> String:
	var result: String = ""
	var capitalize_next: bool = false
	var index: int = 0
	while index < path.length():
		var character: String = path.substr(index, 1)
		if character == "_":
			capitalize_next = true
		elif capitalize_next:
			result += character.to_upper()
			capitalize_next = false
		else:
			result += character
		index += 1
	return result

static func _to_snake_case(path: String) -> String:
	var result: String = ""
	var index: int = 0
	while index < path.length():
		var character: String = path.substr(index, 1)
		if character >= "A" and character <= "Z":
			result += "_" + character.to_lower()
		else:
			result += character
		index += 1
	return result

## A dot separating segments and a digit inside a name are both fine; an
## uppercase letter is what makes a path unable to survive the round trip.
static func _is_lower_snake_case(path: String) -> bool:
	var index: int = 0
	while index < path.length():
		var character: String = path.substr(index, 1)
		if character >= "A" and character <= "Z":
			return false
		index += 1
	return true
```

- [ ] **Step 4: Run the engine checks**

Run: `task foundry:test`
Expected: exit 0

If `",".join(array)` is not the engine's spelling, use the `Array` method the codebase
already relies on — check `tests/foundry/main.fs` for an existing join before
substituting `String.join`.

- [ ] **Step 5: Run the full local pipeline**

Run: `task test && task integration && task foundry:test`
Expected: all pass

- [ ] **Step 6: Commit**

```bash
git add internal/runtime tests/foundry/main.fs
git commit -m "Add the field mask path runtime helper"
```

---

## Done when

- `anvil proto generate --json` and `protoc --foundryscript_opt=json` both run and
  produce output identical to today, because nothing consumes the option yet
- An unknown `--foundryscript_opt` key is a plugin error naming the key
- The four helper classes exist, lint in the engine, and pass their assertions in
  `tests/foundry/main.fs`
- No golden file changed

The next plan starts at the emitter: `json_serialize.go`, then `json_deserialize.go`,
then the well-known forms. That is where the two `Variant` guards described above need
revisiting.
