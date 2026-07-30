# Foundry Member Type Collisions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make generated Foundry Script message members automatically escape Foundry built-in and exposed-native-class names while rejecting deterministic post-escape member collisions.

**Architecture:** Extend the existing centralized field-name planner so it returns both the emitted spelling and a structured escape reason. Carry raw names, final names, positions, and escape metadata through message plans; validate the complete generated member namespace with a dedicated collector before rendering. Keep emitters ignorant of the engine table and verify identical behavior through direct CLI, protoc-plugin, and real Foundry round trips.

**Tech Stack:** Go 1.26, Testify, protobuf descriptors and `CodeGeneratorResponse`, Task, Foundry Script, Foundry alpha.14 integration tooling.

---

## File Map

- Modify `internal/proto/internal/foundryscript/generator/names.go`
  - Own member-name classification and the exact one-underscore policy.
- Modify `internal/proto/internal/foundryscript/generator/names_test.go`
  - Unit-test engine categories, case sensitivity, and existing escape behavior.
- Create `internal/proto/internal/foundryscript/generator/member_collisions.go`
  - Collect generated member claims and render stable aggregated diagnostics.
- Create `internal/proto/internal/foundryscript/generator/member_collisions_test.go`
  - Unit-test collision ordering, positions, escape explanations, and derived members.
- Modify `internal/proto/internal/foundryscript/generator/plan.go`
  - Carry member metadata and submit each planned message to the collision collector.
- Modify `internal/proto/internal/foundryscript/generator/generator.go`
  - Fail on collected member collisions before rendering.
- Modify `internal/proto/internal/foundryscript/generator/fields.go`
  - Use raw protobuf names in fallback documentation.
- Modify `internal/proto/internal/foundryscript/generator/generator_test.go`
  - Prove escaped names flow through every emitter path and enum/oneof cases remain unchanged.
- Create `tests/integration/fixtures/member_collisions/fields.proto`
  - Shared successful direct-CLI and protoc-plugin fixture.
- Create `tests/integration/fixtures/member_collisions/secondary.proto`
  - Direct-CLI post-escape collision fixture.
- Modify `tests/integration/direct_cli_test.go`
  - Verify output, diagnostics, and failure atomicity.
- Modify `tests/integration/protoc_plugin_test.go`
  - Verify descriptor-driven successful escaping.
- Modify `internal/plugin/plugin_test.go`
  - Verify handcrafted descriptor collision diagnostics and empty failed responses.
- Modify `tests/foundry/collisions.proto`
  - Add regular, repeated, map, and oneof-group engine-name members.
- Modify `tests/foundry/main.fs`
  - Exercise escaped APIs and round trips in a real Foundry project.
- Modify `README.md`
  - Document member escaping, the oneof boundary, and secondary-collision behavior.

## Task 1: Centralize Engine-Aware Member Naming

**Files:**

- Modify: `internal/proto/internal/foundryscript/generator/names.go`
- Modify: `internal/proto/internal/foundryscript/generator/names_test.go`

- [ ] **Step 1: Add failing table tests for engine-aware member planning**

Add tests that exercise the structured planner rather than only the compatibility
wrapper:

```go
func TestPlanMemberNameEscapesEngineTypes(t *testing.T) {
	tests := []struct {
		raw       string
		generated string
		kind      memberEscapeKind
		reason    string
	}{
		{raw: "String", generated: "String_", kind: memberEscapeEngineBuiltin, reason: `built-in type "String"`},
		{raw: "AsyncCallable", generated: "AsyncCallable_", kind: memberEscapeEngineBuiltin, reason: `built-in type "AsyncCallable"`},
		{raw: "Node", generated: "Node_", kind: memberEscapeEngineNative, reason: `native class "Node"`},
		{raw: "Timer", generated: "Timer_", kind: memberEscapeEngineNative, reason: `native class "Timer"`},
		{raw: "node", generated: "node", kind: memberEscapeNone},
		{raw: "string", generated: "string", kind: memberEscapeNone},
		{raw: "Node_", generated: "Node_", kind: memberEscapeNone},
	}

	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			got := planMemberName(test.raw)
			require.Equal(t, test.generated, got.Generated)
			require.Equal(t, test.kind, got.Escape.Kind)
			require.Equal(t, test.reason, got.Escape.Description())
			require.Equal(t, test.generated, FieldName(test.raw))
		})
	}
}

func TestPlanMemberNameKeepsExistingEscapePolicies(t *testing.T) {
	tests := []struct {
		raw       string
		generated string
		kind      memberEscapeKind
	}{
		{raw: "var", generated: "var_", kind: memberEscapeKeyword},
		{raw: "to_bytes", generated: "to_bytes_", kind: memberEscapeGenerated},
		{raw: "merge_from_bytes", generated: "merge_from_bytes_", kind: memberEscapeGenerated},
		{raw: "plain", generated: "plain", kind: memberEscapeNone},
	}
	for _, test := range tests {
		got := planMemberName(test.raw)
		require.Equal(t, test.generated, got.Generated)
		require.Equal(t, test.kind, got.Escape.Kind)
	}
}

func TestPlanOneofAlternativeNameSkipsEngineTypes(t *testing.T) {
	require.Equal(t, "Image", planOneofAlternativeName("Image").Generated)
	require.Equal(t, memberEscapeNone, planOneofAlternativeName("Image").Escape.Kind)
	require.Equal(t, "var_", planOneofAlternativeName("var").Generated)
	require.Equal(t, memberEscapeKeyword, planOneofAlternativeName("var").Escape.Kind)
}
```

- [ ] **Step 2: Run the focused tests and verify the RED state**

Run:

```bash
go test ./internal/proto/internal/foundryscript/generator \
  -run 'TestPlanMemberName|TestPlanOneofAlternativeName' -count=1
```

Expected: compilation fails because `planMemberName`,
`planOneofAlternativeName`, `memberEscapeKind`, and the structured escape
metadata do not exist.

- [ ] **Step 3: Add the minimal structured member-name implementation**

In `names.go`, add focused unexported types near `reservedFieldNames`:

```go
type memberEscapeKind uint8

const (
	memberEscapeNone memberEscapeKind = iota
	memberEscapeKeyword
	memberEscapeGenerated
	memberEscapeEngineBuiltin
	memberEscapeEngineNative
)

type memberEscape struct {
	Kind         memberEscapeKind
	ReservedName string
}

func (e memberEscape) Description() string {
	switch e.Kind {
	case memberEscapeKeyword:
		return "Foundry keyword"
	case memberEscapeGenerated:
		return "generated member"
	case memberEscapeEngineBuiltin:
		return fmt.Sprintf("built-in type %q", e.ReservedName)
	case memberEscapeEngineNative:
		return fmt.Sprintf("native class %q", e.ReservedName)
	default:
		return ""
	}
}

type plannedMemberName struct {
	Generated string
	Escape    memberEscape
}
```

Replace the current `FieldName` body with a shared non-engine planner, an
engine-aware member planner, and a compatibility wrapper:

```go
func planNonEngineMemberName(name string) plannedMemberName {
	switch {
	case reservedFieldNames[name]:
		return plannedMemberName{
			Generated: name + "_",
			Escape:    memberEscape{Kind: memberEscapeKeyword, ReservedName: name},
		}
	case generatedMemberNames[name]:
		return plannedMemberName{
			Generated: name + "_",
			Escape:    memberEscape{Kind: memberEscapeGenerated, ReservedName: name},
		}
	}

	return plannedMemberName{Generated: name}
}

func planMemberName(name string) plannedMemberName {
	if planned := planNonEngineMemberName(name); planned.Escape.Kind != memberEscapeNone {
		return planned
	}

	if engineType, reserved := foundryEngineReservedTypes[name]; reserved {
		kind := memberEscapeEngineNative
		if engineType.kind == engineTypeBuiltin {
			kind = memberEscapeEngineBuiltin
		}
		return plannedMemberName{
			Generated: name + "_",
			Escape:    memberEscape{Kind: kind, ReservedName: name},
		}
	}

	return plannedMemberName{Generated: name}
}

func planOneofAlternativeName(name string) plannedMemberName {
	return planNonEngineMemberName(name)
}

func FieldName(name string) string {
	return planMemberName(name).Generated
}
```

Use only one branch, so a spelling reserved by multiple categories can receive
only one trailing underscore. Oneof alternatives use the non-engine planner:
they are union cases and payload identifiers rather than members on the
generated message class, but still retain existing keyword/generated-name
protection.

- [ ] **Step 4: Run focused and package tests**

Run:

```bash
go test ./internal/proto/internal/foundryscript/generator \
  -run 'TestPlanMemberName|TestPlanOneofAlternativeName' -count=1
go test ./internal/proto/internal/foundryscript/generator -count=1
```

Expected: both commands pass.

- [ ] **Step 5: Format and commit Task 1**

Run:

```bash
gofmt -w internal/proto/internal/foundryscript/generator/names.go \
  internal/proto/internal/foundryscript/generator/names_test.go
git add internal/proto/internal/foundryscript/generator/names.go \
  internal/proto/internal/foundryscript/generator/names_test.go
git commit -m "feat: escape Foundry engine member names"
```

Expected: one focused commit containing the naming implementation and its unit
tests.

## Task 2: Carry Member Metadata and Aggregate Secondary Collisions

**Files:**

- Create: `internal/proto/internal/foundryscript/generator/member_collisions.go`
- Create: `internal/proto/internal/foundryscript/generator/member_collisions_test.go`
- Modify: `internal/proto/internal/foundryscript/generator/plan.go`
- Modify: `internal/proto/internal/foundryscript/generator/generator.go`
- Modify: `internal/proto/internal/foundryscript/generator/fields.go`
- Modify: `internal/proto/internal/foundryscript/generator/generator_test.go`

- [ ] **Step 1: Add failing generator tests for complete emitted-name propagation**

Add a test whose one message covers the supported member forms:

```go
func TestGenerateUsesEscapedEngineMemberNamesEverywhere(t *testing.T) {
	files := generate(t, namespacedFile(
		[]*protoast.Message{{
			Name: "Probe",
			Fields: []*protoast.Field{
				{FieldType: "string", Name: "Node", Number: 1},
				{FieldType: "string", Name: "String", Number: 2, Optional: true},
				{FieldType: "string", Name: "Timer", Number: 3, Repeated: true},
			},
			Maps: []*protoast.MapField{{
				KeyType: "string", ValueType: "int32", Name: "Resource", Number: 4,
			}},
			Oneofs: []*protoast.Oneof{{
				Name: "Object",
				Fields: []*protoast.Field{{
					FieldType: "string", Name: "Image", Number: 5,
				}},
			}},
			NestedMessages: []*protoast.Message{{
				Name: "Nested",
				Fields: []*protoast.Field{{
					FieldType: "string", Name: "Node", Number: 1,
				}},
			}},
		}},
		nil,
	))

	source := files["cafecito/game/v1/Probe.pb.fs"]
	require.Contains(t, source, "var Node_: String = \"\"")
	require.Contains(t, source, "if Node_ != \"\":")
	require.Contains(t, source, "Node_ = _pb_Node_read.value")
	require.Contains(t, source, "var String_: String? = null")
	require.Contains(t, source, "var Timer_: Array[String] = []")
	require.Contains(t, source, "var Resource_: Dictionary[String, int] = {}")
	require.Contains(t, source, "var Object_: ProbeObjectCase? = null")
	require.Contains(t, source, "Object_ = ProbeObjectCase.Image(")
	require.Contains(t, source, "final class Nested")
	require.Contains(t, source, "var Node_: String = \"\"")

	union := files["cafecito/game/v1/ProbeObjectCase.pb.fs"]
	require.Contains(t, union, "Image(Image: String)")
	require.NotContains(t, union, "Image_")

	require.Contains(t, source, "## The Node protobuf field.")
	require.NotContains(t, source, "## The Node_ protobuf field.")
	require.Contains(t, source, "## The Object protobuf oneof;")
}
```

Use the exact rendered read expression present after inspecting the failing
output; the assertion must prove assignment targets the escaped member rather
than merely proving the declaration changed.

- [ ] **Step 2: Add failing tests for aggregated, positioned collision diagnostics**

Create `member_collisions_test.go` with generation-level tests:

```go
func TestGenerateAggregatesEscapedMemberCollisions(t *testing.T) {
	file := namespacedFile([]*protoast.Message{
		{
			Name: "Alpha",
			Fields: []*protoast.Field{
				{Position: protoast.Position{Line: 5, Column: 3}, FieldType: "string", Name: "Node", Number: 1},
				{Position: protoast.Position{Line: 6, Column: 3}, FieldType: "string", Name: "Node_", Number: 2},
			},
		},
		{
			Name: "Beta",
			Fields: []*protoast.Field{{
				Position: protoast.Position{Line: 10, Column: 3},
				FieldType: "string", Name: "String", Number: 1,
			}},
			Oneofs: []*protoast.Oneof{{
				Position: protoast.Position{Line: 11, Column: 3},
				Name: "String_",
				Fields: []*protoast.Field{{FieldType: "int32", Name: "value", Number: 2}},
			}},
		},
	}, nil)

	files, err := Generate(file, "members.proto", nil)

	require.Nil(t, files)
	require.Error(t, err)
	message := err.Error()
	require.Contains(t, message, "generated Foundry member names collide:")
	require.Contains(t, message, "members.proto:5:3")
	require.Contains(t, message, "members.proto:6:3")
	require.Contains(t, message, `field cafecito.game.v1.Alpha.Node`)
	require.Contains(t, message, `Foundry member "Node_"`)
	require.Contains(t, message, `native class "Node"`)
	require.Contains(t, message, "members.proto:10:3")
	require.Contains(t, message, "members.proto:11:3")
	require.Contains(t, message, `oneof cafecito.game.v1.Beta.String_`)
	require.Less(t, strings.Index(message, "Alpha.Node"), strings.Index(message, "Beta.String"))
}
```

Add focused cases for:

- `var` plus `var_`, retaining existing coverage under the new diagnostic;
- an escaped field plus escaped oneof group;
- a retained enum companion collision;
- the generated unknown-field buffer claim;
- a descriptor-style claim with zero position, which must retain filename and
  fully qualified identity.

- [ ] **Step 3: Run tests and verify the RED state**

Run:

```bash
go test ./internal/proto/internal/foundryscript/generator \
  -run 'TestGenerateUsesEscapedEngineMemberNamesEverywhere|TestGenerateAggregatesEscapedMemberCollisions' \
  -count=1
```

Expected:

- emitted names may partially pass because Task 1 changed `FieldName`;
- fallback documentation still uses escaped names;
- collision tests fail because validation returns the old first-error format
  without aggregate metadata or positions.

- [ ] **Step 4: Add raw/final metadata to field and oneof plans**

Extend `fieldPlan`:

```go
type fieldPlan struct {
	Doc      []string
	Position protoast.Position
	Kind     string
	Name     string
	RawName  string
	Escape   memberEscape
	// existing Number, Cardinality, Value, Key, Packed, and oneof fields follow.
}
```

Extend `oneofPlan`:

```go
type oneofPlan struct {
	Doc      []string
	Position protoast.Position
	Field    string
	RawField string
	Escape   memberEscape
	Type     string
	Members  []fieldPlan
}
```

For regular fields, call `planMemberName` once and store both results:

```go
member := planMemberName(field.Name)
plan := fieldPlan{
	Doc:      field.Doc,
	Position: field.Position,
	Kind:     "field",
	Name:     member.Generated,
	RawName:  field.Name,
	Escape:   member.Escape,
	Number:   field.Number,
	Value:    value,
}
```

Map fields use the same engine-aware planner with `Kind: "map field"` and
`mapField.Position`.

For oneof alternatives, use `planOneofAlternativeName` when constructing their
`fieldPlan`. Preserve all existing raw wire-path bookkeeping. This keeps an
engine name such as `Image` unchanged in both the union case and its payload
identifier while continuing to escape keywords and generated helper names.

For each oneof group, compute `oneofMember := planMemberName(oneof.Name)` once.
Use `oneofMember.Generated` for `plan.OneofField` and `oneofPlan.Field`, and
store the raw name, position, and escape metadata on `oneofPlan`. Continue to
derive `OneofCaseName` with `TypeName(field.Name)` so engine-name oneof
alternatives remain unchanged.

- [ ] **Step 5: Implement the focused member collision collector**

Create `member_collisions.go` with:

```go
type memberClaim struct {
	SourceName    string
	Position      protoast.Position
	MessageName   string
	Kind          string
	RawName       string
	GeneratedName string
	Escape        memberEscape
}

type memberCollisionCollector struct {
	claims map[string]map[string][]memberClaim
}

func newMemberCollisionCollector() *memberCollisionCollector {
	return &memberCollisionCollector{
		claims: map[string]map[string][]memberClaim{},
	}
}
```

Use a private `add` method keyed first by fully scoped message, then by final
member spelling. Add one message at a time:

```go
func (c *memberCollisionCollector) AddMessage(
	sourceName, messageName string,
	plans []fieldPlan,
	oneofs []oneofPlan,
) {
	c.add(memberClaim{
		SourceName: sourceName, MessageName: messageName,
		Kind: "generated unknown-field buffer",
		RawName: unknownFieldsMember, GeneratedName: unknownFieldsMember,
	})

	for i := range plans {
		plan := &plans[i]
		if plan.RetainsUnknownEnum() {
			c.add(memberClaim{
				SourceName: sourceName, Position: plan.Position,
				MessageName: messageName, Kind: "retained enum value",
				RawName: plan.RawName, GeneratedName: plan.UnknownMember(),
			})
		}
		if plan.OneofCase != "" {
			continue
		}
		c.add(memberClaim{
			SourceName: sourceName, Position: plan.Position,
			MessageName: messageName, Kind: plan.Kind,
			RawName: plan.RawName, GeneratedName: plan.Name,
			Escape: plan.Escape,
		})
	}

	for i := range oneofs {
		oneof := &oneofs[i]
		c.add(memberClaim{
			SourceName: sourceName, Position: oneof.Position,
			MessageName: messageName, Kind: "oneof",
			RawName: oneof.RawField, GeneratedName: oneof.Field,
			Escape: oneof.Escape,
		})
	}
}
```

`Err` must:

1. discard buckets with fewer than two claims;
2. sort collisions by message, generated name, kind, then raw name;
3. render both source locations when available;
4. identify each declaration by kind and fully scoped name;
5. append an escape explanation for transformed declarations; and
6. end each collision with `rename one protobuf declaration`.

Use existing `declarationLocation` behavior as the location-format precedent,
but keep member diagnostics in the new file so type-collision and
member-collision responsibilities remain separate.

- [ ] **Step 6: Wire collection into planning and generation**

Add `memberCollisions *memberCollisionCollector` to `resolver` and initialize it
in `newResolver`.

At the end of each successful `planMessage`, before returning its plan, call:

```go
resolve.memberCollisions.AddMessage(
	resolve.sourceName,
	protoOwnerIdentity,
	plans,
	oneofs,
)
```

Remove the old fail-fast `validateMemberNames` call and function after its
retained-buffer and oneof behavior is represented by the collector.

In `Generate`, after `resolve.collisions.Err(localNamer.prefix)` and before any
render loop, add:

```go
if err := resolve.memberCollisions.Err(); err != nil {
	return nil, err
}
```

This preserves type-collision precedence and ensures member collisions fail
before any `GeneratedFiles` content is rendered.

- [ ] **Step 7: Preserve raw names in fallback documentation**

In `fields.go`, change only the fallback inputs:

```go
Lines: docOrFallback(plan.Doc, fieldDoc(plan.RawName)),
```

and:

```go
Lines: docOrFallback(oneof.Doc, oneofDoc(oneof.RawField)),
```

In `oneofUnion`, use `oneof.RawField` for `oneofUnionDoc`. Do not change
user-authored `Doc` content.

- [ ] **Step 8: Run focused tests, then the generator package**

Run:

```bash
go test ./internal/proto/internal/foundryscript/generator \
  -run 'TestGenerateUsesEscapedEngineMemberNamesEverywhere|TestGenerateAggregatesEscapedMemberCollisions|TestGenerateRejectsCollidingMemberNames|TestGenerateRejectsOneofCollidingWithAField|TestGenerateRejectsCollidingRetentionMembers' \
  -count=1
go test ./internal/proto/internal/foundryscript/generator -count=1
```

Expected: all focused and package tests pass. If exact rendered expressions
differ from the test draft, update assertions only to match intentional
existing emitter syntax; do not weaken assertions to declaration-only checks.

- [ ] **Step 9: Format and commit Task 2**

Run:

```bash
gofmt -w internal/proto/internal/foundryscript/generator/member_collisions.go \
  internal/proto/internal/foundryscript/generator/member_collisions_test.go \
  internal/proto/internal/foundryscript/generator/plan.go \
  internal/proto/internal/foundryscript/generator/generator.go \
  internal/proto/internal/foundryscript/generator/fields.go \
  internal/proto/internal/foundryscript/generator/generator_test.go
git add internal/proto/internal/foundryscript/generator
git commit -m "fix: reject escaped Foundry member collisions"
```

Expected: one commit for plan metadata, aggregation, diagnostics, and emitter
consistency.

## Task 3: Verify Direct CLI and Protoc-Plugin Entry Points

**Files:**

- Create: `tests/integration/fixtures/member_collisions/fields.proto`
- Create: `tests/integration/fixtures/member_collisions/secondary.proto`
- Modify: `tests/integration/direct_cli_test.go`
- Modify: `tests/integration/protoc_plugin_test.go`
- Modify: `internal/plugin/plugin_test.go`

- [ ] **Step 1: Add shared source fixtures**

Create `fields.proto`:

```protobuf
syntax = "proto3";

package probe.members.v1;

message MemberProbe {
  string Node = 1;
  optional string String = 2;
  repeated string Timer = 3;
  map<string, int32> Resource = 4;

  oneof Object {
    string Image = 5;
  }
}
```

Create `secondary.proto`:

```protobuf
syntax = "proto3";

package probe.members.v1;

message SecondaryCollision {
  string Node = 1;
  string Node_ = 2;
}
```

The second fixture is for the direct parser only; normal protoc rejects it
earlier because both fields have the same default JSON name.

- [ ] **Step 2: Add failing direct-CLI integration tests**

Add tests that build `anvil`, generate `fields.proto`, and assert the generated
file contains:

```text
var Node_: String = ""
var String_: String? = null
var Timer_: Array[String] = []
var Resource_: Dictionary[String, int] = {}
var Object_: MemberProbeObjectCase? = null
```

Add a failure-atomic test that passes `fields.proto` and `secondary.proto` in
one command, then asserts:

- stderr contains both source positions;
- stderr names `Node`, `Node_`, `Node_`, and native class `Node`;
- the output directory contains neither `MemberProbe.pb.fs` nor any
  `foundry/proto` runtime files.

Use existing `runFailure`, `repoRoot`, temporary directory, and binary-building
patterns from `direct_cli_test.go`.

- [ ] **Step 3: Add a failing protoc-plugin success test**

Run protoc against `fields.proto` with the built plugin and assert the same
member spellings in `MemberProbe.pb.fs`. Reuse the executable setup and output
search patterns already present in `protoc_plugin_test.go`.

- [ ] **Step 4: Add a handcrafted descriptor failure test**

In `internal/plugin/plugin_test.go`, build a
`CodeGeneratorRequest` with one message containing fields `Node` and `Node_`.
Give them source paths:

```go
[]int32{4, 0, 2, 0}
[]int32{4, 0, 2, 1}
```

and spans corresponding to lines 5 and 6. Assert:

```go
resp := runPlugin(t, req)
require.Contains(t, resp.GetError(), "members.proto:5:3")
require.Contains(t, resp.GetError(), "members.proto:6:3")
require.Contains(t, resp.GetError(), `Foundry member "Node_"`)
require.Contains(t, resp.GetError(), `native class "Node"`)
require.Empty(t, resp.GetFile())
```

Also add a positionless variant by omitting `SourceCodeInfo`; require the
filename and fully scoped field identities remain present.

- [ ] **Step 5: Run the focused tests and verify RED**

Run:

```bash
go test -tags=integration ./tests/integration \
  -run 'Test.*MemberCollision' -count=1
go test ./internal/plugin -run 'TestRun.*MemberCollision' -count=1
```

Expected: the new tests fail before their assertions are satisfied if Task 2
missed either entry point, descriptor positions, or failure atomicity. If all
behavior already flows correctly from Task 2, use `git show HEAD^` to confirm
the tests would fail against the pre-Task-2 implementation before accepting
the green result.

- [ ] **Step 6: Make only entry-point-specific corrections**

If the direct CLI writes files before all input generation succeeds, preserve
its current collect-then-write structure and correct the regression without
moving writes earlier.

If descriptor positions are missing, correct field and map-field conversion in
`internal/proto/internal/desc/converter.go` by passing
`source.position(pathAppend(path, 2, int32(i)))` into each converted field.
Add the matching converter unit assertion in
`internal/proto/internal/desc/converter_test.go`.

Do not duplicate member naming or collision logic in CLI or plugin packages;
both must continue to call the shared generator.

- [ ] **Step 7: Run integration and plugin suites**

Run:

```bash
go test ./internal/plugin -count=1
task build
task integration
```

Expected: plugin tests and all direct CLI, protoc, and Buf integration tests
pass.

- [ ] **Step 8: Format and commit Task 3**

Run:

```bash
gofmt -w tests/integration/direct_cli_test.go \
  tests/integration/protoc_plugin_test.go \
  internal/plugin/plugin_test.go
git add tests/integration/fixtures/member_collisions \
  tests/integration/direct_cli_test.go \
  tests/integration/protoc_plugin_test.go \
  internal/plugin/plugin_test.go
git add internal/proto/internal/desc/converter.go \
  internal/proto/internal/desc/converter_test.go 2>/dev/null || true
git commit -m "test: cover Foundry member collision entry points"
```

Expected: integration fixtures and entry-point verification in one focused
commit.

## Task 4: Prove Foundry Behavior and Document the Contract

**Files:**

- Modify: `tests/foundry/collisions.proto`
- Modify: `tests/foundry/main.fs`
- Modify: `README.md`

- [ ] **Step 1: Extend the Foundry fixture before rebuilding**

In `tests/foundry/collisions.proto`, add distinct fields to the existing
prefixed `Node` message:

```protobuf
  string Node = 6;
  string String = 7;
  repeated string Timer = 8;
  map<string, int32> Resource = 9;

  oneof Object {
    string Image = 10;
    int32 count = 11;
  }
```

This proves `type_prefix` does not rename members, the engine list covers both
built-ins and native classes, and the oneof alternative `Image` remains
unchanged.

- [ ] **Step 2: Run the old generator and verify the RED lint state**

Build the pre-feature binary from the task's starting commit or temporarily
invoke the binary built before Task 1, generate the extended fixture, and run
Foundry lint. The expected failure includes member-shadowing diagnostics for at
least `Node`, `String`, `Timer`, `Resource`, or `Object`.

If the current worktree binary already contains Tasks 1–3, prove RED by running
the same fixture against `git worktree` or a temporary build at the base commit.
Do not treat a green-only fixture run as a valid regression test.

- [ ] **Step 3: Exercise escaped members in the Foundry runtime script**

In `tests/foundry/main.fs`, set:

```foundryscript
	collision.Node_ = "node member"
	collision.String_ = "string member"
	collision.Timer_ = ["first", "second"]
	collision.Resource_ = {"ore": 3}
	collision.Object_ = GameNodeObjectCase.Image("image case")
```

After the existing encode/decode round trip, assert:

```foundryscript
		check(collision_decoded.Node_ == "node member", "native-class member")
		check(collision_decoded.String_ == "string member", "built-in member")
		check(collision_decoded.Timer_ == ["first", "second"], "repeated escaped member")
		check(collision_decoded.Resource_ == {"ore": 3}, "map escaped member")
		match collision_decoded.Object_:
			GameNodeObjectCase.Image(var image):
				check(image == "image case", "oneof alternative stays unchanged")
			_:
				printerr("FAIL: escaped oneof member did not round trip")
				failures += 1
```

Use tabs and the existing script's failure-reporting style.

- [ ] **Step 4: Rebuild and verify GREEN in real Foundry**

Run:

```bash
task build
FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test
```

Expected:

- engine-table drift check passes;
- Foundry lint emits no errors;
- runtime output ends successfully with all checks passing.

- [ ] **Step 5: Document member escaping**

Add a `### Field member collisions` subsection near the existing
`type_prefix` documentation in `README.md`:

```markdown
### Field member collisions

Field names remain exactly as written unless Foundry reserves that spelling as
a keyword, generated member, built-in type, or exposed native class. The
generator appends one underscore to the Foundry Script member:

| Protobuf field | Foundry Script member |
| --- | --- |
| `Node` | `Node_` |
| `String` | `String_` |
| `node` | `node` |

The mapped name is used consistently for reads, writes, serialization, and
deserialization; protobuf field numbers and wire encoding do not change.
Oneof group members follow this rule, while oneof alternatives and enum values
keep their existing names.

If two declarations map to the same member, such as `Node` and `Node_`,
generation fails and asks you to rename one protobuf declaration. The generator
does not search for a second suffix.
```

Do not describe `type_prefix` as affecting fields.

- [ ] **Step 6: Run documentation-adjacent and full local CI checks**

Run:

```bash
task fmt
task ci
task integration
task build
FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test
```

Expected: all commands pass. `task fmt` must leave no uncommitted mechanical
changes outside the files in this plan.

- [ ] **Step 7: Commit Task 4**

Run:

```bash
git add README.md tests/foundry/collisions.proto tests/foundry/main.fs
git commit -m "test: verify escaped members in Foundry"
```

Expected: the real-engine fixture and public contract are committed together.

## Task 5: Final Verification and Branch Review

**Files:**

- Verify all files changed by Tasks 1–4.
- Update generated examples only if verification proves an intentional output
  change in an existing golden fixture; otherwise leave `examples/golden/`
  untouched.

- [ ] **Step 1: Inspect the complete branch diff**

Run:

```bash
git status --short
git diff --check origin/main...HEAD
git diff --stat origin/main...HEAD
git log --oneline origin/main..HEAD
```

Expected:

- only issue #32 design, plan, implementation, tests, fixtures, and README
  changes are present;
- the two pre-existing untracked user documents are not staged or committed;
- no whitespace errors are reported.

- [ ] **Step 2: Run the complete verification matrix**

Run:

```bash
task ci
task integration
task build
FOUNDRY_BIN=/Users/christian/bin/foundry task foundry:test
```

Expected: every command exits zero; Foundry lint is empty and the runtime
fixture passes.

- [ ] **Step 3: Review acceptance criteria line by line**

Confirm against
`docs/superpowers/specs/2026-07-30-foundry-member-type-collisions-design.md`:

- every engine type maps to one trailing underscore as a member;
- case variants remain unchanged;
- oneof groups escape while alternatives and enum values do not;
- every emitter reference uses the planned name;
- secondary collisions aggregate before emission;
- direct CLI, plugin, and Buf behavior agree;
- raw protobuf identity and wire behavior remain unchanged;
- fallback docs use raw names;
- real Foundry lint and runtime behavior pass.

If any item cannot be tied to a passing test, add the missing test using a
red-green cycle before proceeding.

- [ ] **Step 4: Run whole-branch spec and code-quality review**

Use the subagent-driven workflow's final reviewer against `origin/main...HEAD`.
Provide the approved spec, this plan, base SHA, head SHA, and verification
output. Fix every in-scope Critical or Important issue, rerun affected tests,
and request re-review until approved.

- [ ] **Step 5: Commit review fixes if necessary**

For each coherent review fix:

```bash
git add <exact files changed by the fix>
git commit -m "fix: address member collision review"
```

If review is clean, do not create an empty commit.
