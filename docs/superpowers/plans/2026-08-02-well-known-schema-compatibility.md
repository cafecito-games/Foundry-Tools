# Well-Known Schema Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reject structurally incompatible caller-supplied well-known schemas before the direct CLI or protoc plugin substitutes the runtime bindings, while keeping compatible newer copies silent.

**Architecture:** Normalize both hand-parsed and descriptor-converted files through one AST comparator in `internal/proto/internal/wktcompat`. The comparator lazily parses the embedded canonical WKT sources once, treats their structure as a required subset, and returns deterministic canonical-path diagnostics. The direct command passes roots plus retained imports; the plugin passes every converted request descriptor.

**Tech Stack:** Go 1.26, the existing protobuf AST/parser/descriptor converter, Cobra, `pluginpb.CodeGeneratorRequest`, Testify, build-tagged integration tests.

**Spec:** `docs/superpowers/specs/2026-08-02-well-known-schema-compatibility-design.md`

---

## File Structure

| Path | Responsibility |
|---|---|
| `internal/proto/internal/wktcompat/compat.go` | Canonical cache, AST normalization, subset comparison, deterministic diagnostics. |
| `internal/proto/internal/wktcompat/compat_test.go` | Structural policy tests, including parser/descriptor normalization parity. |
| `internal/proto/api.go` | Shared `SchemaFile` alias and compatibility-check facade used by both front ends. |
| `internal/proto/command.go` | Collect direct roots and retained imports and validate before skip/output. |
| `internal/proto/command_test.go` | Direct-root, direct-import, public-transitive, compatibility, and atomicity tests. |
| `internal/plugin/plugin.go` | Validate all request descriptors before the file-to-generate loop. |
| `internal/plugin/plugin_test.go` | Dependency/direct WKT rejection and compatible-addition tests. |
| `internal/proto/wellknown/wellknown_test.go` | Exact embedded-source/conformance-fixture synchronization check. |
| `tests/integration/direct_cli_test.go` | Real CLI incompatible-copy diagnostic and atomic-output test. |
| `tests/integration/protoc_plugin_test.go` | Real protoc request with its installed compatible Timestamp descriptor. |
| `README.md` | User-facing compatibility and diagnostic behavior. |

No `.pb.fs` output should change.

## Task 1: Build the shared AST compatibility validator

**Files:**
- Create: `internal/proto/internal/wktcompat/compat_test.go`
- Create: `internal/proto/internal/wktcompat/compat.go`

- [ ] **Step 1: Write the first failing canonical/subset tests**

Create `compat_test.go` with a parser helper and tests that state the public API and the asymmetric policy:

```go
package wktcompat_test

import (
    "strings"
    "testing"

    "github.com/stretchr/testify/require"

    protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
    protoparse "github.com/cafecito-games/foundry-tools/internal/proto/internal/parser"
    "github.com/cafecito-games/foundry-tools/internal/proto/internal/wktcompat"
    "github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
)

func parseFile(t *testing.T, source string) *protoast.ProtoFile {
    t.Helper()
    tokens, err := protoparse.Tokenize(strings.TrimSpace(source), "candidate.proto")
    require.NoError(t, err)
    file, err := protoparse.Parse(tokens, "candidate.proto")
    require.NoError(t, err)
    return file
}

func TestCanonicalWellKnownSourcesAreCompatible(t *testing.T) {
    var candidates []wktcompat.SchemaFile
    for _, path := range wellknown.Files() {
        source, err := wellknown.Source(path)
        require.NoError(t, err)
        candidates = append(candidates, wktcompat.SchemaFile{
            ImportPath: path,
            File:       parseFile(t, string(source)),
        })
    }
    require.NoError(t, wktcompat.Check(candidates))
}

func TestCompatiblePresentationChangesAndAdditionsStayQuiet(t *testing.T) {
    candidate := parseFile(t, `
syntax = "proto3";
package google.protobuf;
option deprecated = true;
message Extra {}
message Timestamp {
  int64 renamed_seconds = 1;
  int32 renamed_nanos = 2;
  string future = 3;
}
enum ExtraEnum { EXTRA = 0; }
`)
    require.NoError(t, wktcompat.Check([]wktcompat.SchemaFile{{
        ImportPath: "google/protobuf/timestamp.proto",
        File:       candidate,
    }}))
}

func TestIncompatibleFieldsReportCanonicalIdentityInStableOrder(t *testing.T) {
    candidate := parseFile(t, `
syntax = "proto3";
package google.protobuf;
message Timestamp {
  repeated string seconds = 1;
}
`)
    err := wktcompat.Check([]wktcompat.SchemaFile{{
        ImportPath: "google/protobuf/timestamp.proto",
        File:       candidate,
    }})
    require.EqualError(t, err, strings.Join([]string{
        "google/protobuf/timestamp.proto: google.protobuf.Timestamp.seconds (#1): expected singular int64; found repeated string",
        "google/protobuf/timestamp.proto: google.protobuf.Timestamp.nanos (#2): missing canonical field",
    }, "\n"))
}
```

- [ ] **Step 2: Run the focused package and verify RED**

Run:

```bash
go test ./internal/proto/internal/wktcompat -count=1
```

Expected: FAIL because `internal/proto/internal/wktcompat` and its `SchemaFile`/`Check` API do not exist.

- [ ] **Step 3: Implement the canonical cache and normalized subset model**

Create `compat.go` with these public types and entry point:

```go
package wktcompat

import (
    "fmt"
    "path"
    "sort"
    "strings"
    "sync"

    protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
    protoparse "github.com/cafecito-games/foundry-tools/internal/proto/internal/parser"
    "github.com/cafecito-games/foundry-tools/internal/proto/wellknown"
)

type SchemaFile struct {
    ImportPath string
    File       *protoast.ProtoFile
}

type mismatch struct {
    path    string
    subject string
    detail  string
}

func (m mismatch) String() string {
    if m.subject == "" {
        return m.path + ": " + m.detail
    }
    return m.path + ": " + m.subject + ": " + m.detail
}

type mismatchList []mismatch

func (m mismatchList) Error() string {
    lines := make([]string, len(m))
    for i := range m {
        lines[i] = m[i].String()
    }
    return strings.Join(lines, "\n")
}

var canonicalOnce = sync.OnceValues(loadCanonical)

func Check(files []SchemaFile) error {
    canonical, err := canonicalOnce()
    if err != nil {
        return err
    }
    candidates := append([]SchemaFile(nil), files...)
    sort.SliceStable(candidates, func(i, j int) bool {
        return normalizePath(candidates[i].ImportPath) < normalizePath(candidates[j].ImportPath)
    })
    var mismatches mismatchList
    for _, candidate := range candidates {
        name := normalizePath(candidate.ImportPath)
        expected, ok := canonical[name]
        if !ok {
            continue
        }
        actual, normalizationErrors := normalize(candidate.File, name)
        mismatches = append(mismatches, normalizationErrors...)
        if len(normalizationErrors) == 0 {
            mismatches = append(mismatches, compare(expected, actual, name)...)
        }
    }
    if len(mismatches) == 0 {
        return nil
    }
    return mismatches
}

func loadCanonical() (map[string]schemaShape, error) {
    out := make(map[string]schemaShape, len(wellknown.Files()))
    for _, name := range wellknown.Files() {
        source, err := wellknown.Source(name)
        if err != nil {
            return nil, err
        }
        tokens, err := protoparse.Tokenize(string(source), name)
        if err != nil {
            return nil, fmt.Errorf("parse canonical %s: %w", name, err)
        }
        file, err := protoparse.Parse(tokens, name)
        if err != nil {
            return nil, fmt.Errorf("parse canonical %s: %w", name, err)
        }
        shape, normalizationErrors := normalize(file, name)
        if len(normalizationErrors) != 0 {
            return nil, fmt.Errorf("normalize canonical %s: %s", name, normalizationErrors.Error())
        }
        out[name] = shape
    }
    return out, nil
}

func normalizePath(name string) string {
    return path.Clean(strings.ReplaceAll(name, `\`, "/"))
}
```

In the same file, implement one normalized model used for both AST origins:

```go
type declarationKind string

const (
    kindMessage declarationKind = "message"
    kindEnum    declarationKind = "enum"
)

type schemaShape struct {
    packageName string
    messages    map[string]messageShape
    messageOrder []string
    enums       map[string]enumShape
    enumOrder   []string
    kinds       map[string]declarationKind
}

type messageShape struct {
    fields map[int]fieldShape
    fieldNumbers []int
}

type enumShape struct {
    numbers map[int]bool
    numberOrder []int
}

type fieldShape struct {
    canonicalName string
    cardinality string
    value typeShape
    mapKey *typeShape
    oneof string
}

type typeShape struct {
    kind declarationKind
    name string
}
```

Implement `normalize` in two passes: first recursively register every full
message/enum name and reject duplicates; then collect regular, map, and oneof
fields by number and reject duplicate numbers. Sort each message's field-number
slice and each enum's unique numeric values. Resolve a type from
`FullTypePath`/`FullValueTypePath` first; otherwise search the current message
scope outward through the package. Scalars are the exact fifteen spellings and
carry no declaration kind.

Implement `compare` as a canonical-subset walk:

```go
func compare(expected, actual schemaShape, importPath string) mismatchList {
    var out mismatchList
    if actual.packageName != expected.packageName {
        out = append(out, mismatch{path: importPath,
            detail: fmt.Sprintf("expected package %s; found %s", expected.packageName, actual.packageName)})
        return out
    }
    for _, fullName := range expected.messageOrder {
        want := expected.messages[fullName]
        got, ok := actual.messages[fullName]
        if !ok {
            detail := "missing canonical message"
            if actual.kinds[fullName] == kindEnum {
                detail = "expected message; found enum"
            }
            out = append(out, mismatch{path: importPath, subject: fullName, detail: detail})
            continue
        }
        out = append(out, compareMessage(importPath, fullName, want, got)...)
    }
    for _, fullName := range expected.enumOrder {
        want := expected.enums[fullName]
        got, ok := actual.enums[fullName]
        if !ok {
            detail := "missing canonical enum"
            if actual.kinds[fullName] == kindMessage {
                detail = "expected enum; found message"
            }
            out = append(out, mismatch{path: importPath, subject: fullName, detail: detail})
            continue
        }
        for _, number := range want.numberOrder {
            if !got.numbers[number] {
                out = append(out, mismatch{path: importPath, subject: fullName,
                    detail: fmt.Sprintf("missing canonical enum number %d", number)})
            }
        }
    }
    return out
}
```

`compareMessage` must use the canonical field's name only in the subject,
format field shapes as `singular int64`, `repeated google.protobuf.Value`, or
`map<string, google.protobuf.Value>`, and compare oneof partitions after
restricting candidate group membership to canonical field numbers. Renamed
oneof groups then compare equal, additive members disappear from the restricted
set, and split/merge/membership changes fail once per canonical group.

- [ ] **Step 4: Run the focused tests and verify GREEN**

Run:

```bash
go test ./internal/proto/internal/wktcompat -count=1
```

Expected: PASS.

- [ ] **Step 5: Add the full structural matrix as failing tests**

Extend `compat_test.go` with table cases over `timestamp.proto` and
`struct.proto`. Load the complete embedded source with this helper, apply one
mutation, and assert either an empty error or an exact diagnostic substring:

```go
func canonicalSource(t *testing.T, name string) string {
    t.Helper()
    source, err := wellknown.Source(name)
    require.NoError(t, err)
    return string(source)
}

func replaceOnce(t *testing.T, source, old, replacement string) string {
    t.Helper()
    require.Contains(t, source, old)
    return strings.Replace(source, old, replacement, 1)
}

func TestStructuralCompatibilityMatrix(t *testing.T) {
    timestamp := canonicalSource(t, "google/protobuf/timestamp.proto")
    structProto := canonicalSource(t, "google/protobuf/struct.proto")
    splitOneof := replaceOnce(t, structProto,
        "    string string_value = 3;\n    // Represents a boolean value.",
        "    string string_value = 3;\n  }\n  oneof second {\n    // Represents a boolean value.")
    tests := []struct {
        name, path, source, want string
    }{
        {"package", "google/protobuf/timestamp.proto", `syntax="proto3"; package custom; message Timestamp { int64 seconds=1; int32 nanos=2; }`, "expected package google.protobuf; found custom"},
        {"moved field", "google/protobuf/timestamp.proto", replaceOnce(t, timestamp, "int64 seconds = 1;", "int64 seconds = 9;"), "Timestamp.seconds (#1): missing canonical field"},
        {"optional", "google/protobuf/timestamp.proto", replaceOnce(t, timestamp, "int64 seconds = 1;", "optional int64 seconds = 1;"), "expected singular int64; found optional int64"},
        {"map key", "google/protobuf/struct.proto", replaceOnce(t, structProto, "map<string, Value> fields = 1;", "map<int64, Value> renamed = 1;"), "expected map<string, google.protobuf.Value>; found map<int64, google.protobuf.Value>"},
        {"map becomes repeated", "google/protobuf/struct.proto", replaceOnce(t, structProto, "map<string, Value> fields = 1;", "repeated Value renamed = 1;"), "expected map<string, google.protobuf.Value>; found repeated google.protobuf.Value"},
        {"oneof renamed", "google/protobuf/struct.proto", replaceOnce(t, structProto, "oneof kind {", "oneof renamed {"), ""},
        {"oneof split", "google/protobuf/struct.proto", splitOneof, "canonical oneof fields [1 2 3 4 5 6]"},
        {"enum value renamed", "google/protobuf/struct.proto", replaceOnce(t, structProto, "NULL_VALUE = 0;", "RENAMED = 0;"), ""},
        {"enum number missing", "google/protobuf/struct.proto", replaceOnce(t, structProto, "NULL_VALUE = 0;", "RENAMED = 1;"), "missing canonical enum number 0"},
    }
    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            err := wktcompat.Check([]wktcompat.SchemaFile{{ImportPath: test.path, File: parseFile(t, test.source)}})
            if test.want == "" {
                require.NoError(t, err)
            } else {
                require.ErrorContains(t, err, test.want)
            }
        })
    }
}
```

Add separate complete-source or `replaceOnce` cases for missing type,
message/enum kind swap, scalar type change, referenced type change, repeated
change, oneof insertion/removal/merge/split, map value kind, additive type/enum
number, and deterministic multi-file ordering. Do not mutate embedded files on
disk.

- [ ] **Step 6: Run the matrix and verify RED**

Run:

```bash
go test ./internal/proto/internal/wktcompat -count=1
```

Expected: FAIL on the first structural rule not yet implemented, with the test's
expected canonical-path diagnostic.

- [ ] **Step 7: Complete normalization and comparison minimally**

Complete `compareMessage` by walking `want.fieldNumbers`. Emit one missing-field
diagnostic when `got.fields[number]` is absent; otherwise compare the complete
formatted cardinality/type or map shape. Then derive each canonical oneof as a
sorted set of canonical field numbers and compare it with the candidate group
restricted to those same canonical numbers. Track the smallest canonical field
number already used for a group so split/merge differences produce one stable
diagnostic per canonical group. A canonical non-oneof field whose candidate is
in any oneof gets its own membership diagnostic. Do not compare field names,
oneof names, enum value names, options, imports, comments, positions,
reservations, JSON names, packed options, syntax, or additive declarations.

- [ ] **Step 8: Verify the complete shared validator is GREEN**

Run:

```bash
go test ./internal/proto/internal/wktcompat -count=1
```

Expected: PASS.

- [ ] **Step 9: Add descriptor-converter parity**

Construct a canonical `timestamp.proto` `FileDescriptorProto`, convert it with
`protodesc.FromFileDescriptorProto`, and pass it to the same `wktcompat.Check`.
Then mutate field #1 to `TYPE_STRING` and assert the same `expected singular
int64; found singular string` diagnostic produced for parsed source. This test
must import the converter, not duplicate descriptor normalization in
`wktcompat`.

- [ ] **Step 10: Run focused tests and commit**

Run:

```bash
go test ./internal/proto/internal/wktcompat -count=1
git diff --check
```

Expected: PASS and no whitespace errors.

Commit:

```bash
git add internal/proto/internal/wktcompat
git commit -m "feat: compare well-known schema structure"
```

## Task 2: Enforce compatibility in the direct command

**Files:**
- Modify: `internal/proto/api.go`
- Modify: `internal/proto/command.go`
- Modify: `internal/proto/command_test.go`

- [ ] **Step 1: Add failing direct-root/import/public-transitive tests**

In `command_test.go`, add a helper that writes a proto under a temporary include
root. Add three tests:

```go
func TestGenerateRejectsIncompatibleWellKnownRootBeforeOutput(t *testing.T) {
    root := t.TempDir()
    timestamp := writeProto(t, root, "google/protobuf/timestamp.proto", `
syntax = "proto3";
package google.protobuf;
message Timestamp { string seconds = 1; int32 nanos = 2; }
`)
    out := t.TempDir()
    var stdout bytes.Buffer
    cmd := NewCommand(&stdout)
    cmd.SetArgs([]string{"generate", "-I", root, "-o", out, timestamp})
    err := cmd.Execute()
    require.ErrorContains(t, err, "google/protobuf/timestamp.proto")
    require.ErrorContains(t, err, "Timestamp.seconds (#1)")
    entries, readErr := os.ReadDir(out)
    require.NoError(t, readErr)
    require.Empty(t, entries)
}
```

The direct-import test writes `event.proto` importing the incompatible
Timestamp and generates only `event.proto`. The public-transitive test writes
`bridge.proto` with `import public "google/protobuf/timestamp.proto";`, then a
root importing `bridge.proto`; both must report the same mismatch and leave the
output empty. Add one green caller-copy test that renames both canonical fields
and adds field #3 while preserving their shapes.

- [ ] **Step 2: Run the direct tests and verify RED**

Run:

```bash
go test ./internal/proto -run 'TestGenerate.*WellKnown|TestGenerate.*Public' -count=1
```

Expected: FAIL because all incompatible copies are still skipped/substituted
without a compatibility error.

- [ ] **Step 3: Expose the shared facade**

In `api.go`, add:

```go
import wktcompat "github.com/cafecito-games/foundry-tools/internal/proto/internal/wktcompat"

type SchemaFile = wktcompat.SchemaFile

func CheckWellKnownCompatibility(files []SchemaFile) error {
    return wktcompat.Check(files)
}
```

- [ ] **Step 4: Validate parsed roots and retained imports before generation**

In `command.go`, immediately after `ParseFiles` succeeds and before allocating
generated output, call a helper:

```go
func parsedSchemaFiles(parsedFiles []ParsedFile) []SchemaFile {
    var out []SchemaFile
    for _, parsed := range parsedFiles {
        out = append(out, SchemaFile{ImportPath: parsed.ImportPath, File: parsed.File})
        for _, imported := range parsed.Imports {
            out = append(out, SchemaFile{ImportPath: imported.Filename, File: imported.File})
        }
    }
    return out
}
```

```go
if err := CheckWellKnownCompatibility(parsedSchemaFiles(parsedFiles)); err != nil {
    return err
}
```

Do not add another filesystem traversal and do not move the existing
well-known skip.

- [ ] **Step 5: Run direct tests and verify GREEN**

Run:

```bash
go test ./internal/proto -run 'TestGenerate.*WellKnown|TestGenerate.*Public' -count=1
go test ./internal/proto/... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the direct front end**

```bash
git add internal/proto/api.go internal/proto/command.go internal/proto/command_test.go
git commit -m "fix: reject incompatible well-known CLI schemas"
```

## Task 3: Enforce compatibility for every protoc descriptor

**Files:**
- Modify: `internal/plugin/plugin.go`
- Modify: `internal/plugin/plugin_test.go`

- [ ] **Step 1: Write failing dependency and direct-output plugin tests**

Add a helper that finds Timestamp field #1 in `wellKnownRequest` and changes it
to `TYPE_STRING`. Test with `file_to_generate = ["event.proto"]` and with
`["google/protobuf/timestamp.proto"]`. Both responses must have no files and an
error containing:

```text
google/protobuf/timestamp.proto
google.protobuf.Timestamp.seconds (#1)
expected singular int64; found singular string
```

Also add a green case that appends `string future = 3` and renames fields #1/#2
inside the descriptor while retaining number/type/cardinality.

- [ ] **Step 2: Run plugin tests and verify RED**

Run:

```bash
go test ./internal/plugin -run 'TestRun.*WellKnown.*Compat|TestRun.*Incompatible' -count=1
```

Expected: FAIL because dependency descriptors are converted but never checked,
and direct WKT outputs are skipped before validation.

- [ ] **Step 3: Pass the entire converted request through the facade**

In `plugin.Run`, after `FromCodeGeneratorRequest` succeeds and before building
the generation response, build entries by descriptor/file index:

```go
schemaFiles := make([]foundryproto.SchemaFile, 0, len(files))
for i, file := range files {
    if i >= len(req.GetProtoFile()) {
        break
    }
    schemaFiles = append(schemaFiles, foundryproto.SchemaFile{
        ImportPath: req.GetProtoFile()[i].GetName(),
        File:       file,
    })
}
if err := foundryproto.CheckWellKnownCompatibility(schemaFiles); err != nil {
    return writeError(out, err.Error())
}
```

This intentionally checks all `proto_file` entries, not `file_to_generate` and
not only the dependency walk of one output.

- [ ] **Step 4: Run plugin tests and verify GREEN**

Run:

```bash
go test ./internal/plugin -count=1
go test ./internal/proto/internal/desc -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the plugin front end**

```bash
git add internal/plugin/plugin.go internal/plugin/plugin_test.go
git commit -m "fix: reject incompatible well-known descriptors"
```

## Task 4: Add integration, synchronization, and user documentation

**Files:**
- Modify: `internal/proto/wellknown/wellknown_test.go`
- Modify: `tests/integration/direct_cli_test.go`
- Modify: `tests/integration/protoc_plugin_test.go`
- Modify: `README.md`

- [ ] **Step 1: Pin the two vendored source trees together**

Add this test to `wellknown_test.go`:

```go
func TestEmbeddedSourcesMatchConformanceFixture(t *testing.T) {
    fixtureRoot := filepath.Join("..", "..", "..", "tests", "integration", "fixtures", "conformance")
    for _, name := range Files() {
        embedded, err := Source(name)
        require.NoError(t, err)
        fixture, err := os.ReadFile(filepath.Join(fixtureRoot, filepath.FromSlash(name)))
        require.NoError(t, err)
        require.Equal(t, string(embedded), string(fixture), "%s drifted from the shared upstream pin", name)
    }
}
```

Add `os` to the test imports. Run:

```bash
go test ./internal/proto/wellknown -run TestEmbeddedSourcesMatchConformanceFixture -count=1
```

Expected: PASS; the test locks an existing documented invariant.

- [ ] **Step 2: Add a real direct CLI incompatible-copy integration test**

In `direct_cli_test.go`, write an event and incompatible Timestamp beneath a
temporary include root, run `go run ./cmd/anvil proto generate`, and assert the
canonical path/type/field diagnostic plus an empty output directory. This uses
the built CLI path rather than calling `NewCommand` in-process.

- [ ] **Step 3: Add a real protoc bundled-WKT compatibility test**

In `protoc_plugin_test.go`, write a temporary `event.proto` that imports
`google/protobuf/timestamp.proto`, build the plugin, and run installed `protoc`
with only the temporary directory as an explicit include path. Assert plugin
generation succeeds and the emitted Event imports `foundry.proto.wkt` with a
`Timestamp?` field. Protoc's own include installation supplies the routine WKT
descriptor; do not copy the repository pin into this test.

- [ ] **Step 4: Run both new integrations**

Run:

```bash
go test -tags=integration ./tests/integration -run 'TestDirectCLIRejectsIncompatibleWellKnown|TestProtocPluginAcceptsBundledWellKnown' -count=1 -v
```

Expected: PASS.

- [ ] **Step 5: Document the boundary**

In README's well-known-type section, add a paragraph stating that a
caller-supplied canonical `google/protobuf` path is structurally checked before
runtime substitution; field names/comments/options and additive declarations
are compatible, while missing/moved/type/cardinality/map/oneof changes are
errors naming the canonical declaration.

- [ ] **Step 6: Run focused packages and commit**

```bash
go test ./internal/proto/internal/wktcompat ./internal/proto/wellknown ./internal/proto ./internal/plugin -count=1
go test -tags=integration ./tests/integration -run 'TestDirectCLIRejectsIncompatibleWellKnown|TestProtocPluginAcceptsBundledWellKnown|TestConformanceSchemaGenerates' -count=1
git diff --check
git add README.md internal/proto/wellknown/wellknown_test.go tests/integration/direct_cli_test.go tests/integration/protoc_plugin_test.go
git commit -m "test: cover well-known compatibility front ends"
```

Expected: all focused tests pass and no whitespace errors.

## Task 5: Rebase, regenerate, and verify the complete branch

**Files:**
- Verify only: all branch changes and generated/runtime artifacts

- [ ] **Step 1: Fetch and rebase onto current main**

```bash
git fetch origin main
git rebase origin/main
```

Expected: branch is based on `origin/main` at `2ad3ecd` or newer. Resolve any
overlap by preserving current-main behavior and the compatibility calls; do not
drop unrelated upstream tests.

- [ ] **Step 2: Prove generated WKT bindings do not change**

```bash
task gen-wkt
git status --short
```

Expected: no generated `.pb.fs` diff. If generation changes output because of
an upstream main change, inspect and commit only an intentional regeneration.

- [ ] **Step 3: Run focused tests after rebase**

```bash
go test ./internal/proto/internal/wktcompat ./internal/proto/wellknown ./internal/proto ./internal/plugin -count=1
```

Expected: PASS.

- [ ] **Step 4: Run full repository verification**

Run sequentially, avoiding concurrent golangci-lint processes:

```bash
task ci
task integration
task foundry:test
```

Expected: `task ci` reports `0 issues`; every integration test passes; Foundry
lint has empty diagnostics and the runtime fixtures report their success lines.

- [ ] **Step 5: Audit the final diff and artifact cleanup**

```bash
git diff --check origin/main...HEAD
git diff --stat origin/main...HEAD
git diff origin/main...HEAD
git status --short --branch
find tests/foundry -maxdepth 1 \( -name '*.uid' -o -name '.foundry' -o -name 'generated' \) -print
```

Expected: only the spec, plan, compatibility package, front-end wiring, tests,
and README are changed; the worktree is clean; no Foundry artifacts remain.

- [ ] **Step 6: Stop for independent reviews**

Report exact rebased commits and verification evidence. Do not push or open a PR
until the root agent's independent specification and quality reviews approve
the branch.
