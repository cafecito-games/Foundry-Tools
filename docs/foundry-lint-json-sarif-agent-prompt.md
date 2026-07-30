# Prompt: Foundry Script Lint JSON/SARIF CLI

You are working in the Foundry Engine repository at:

`/Users/christian/CafecitoGames/godot`

Read the repository `AGENTS.md` first and follow its build, style, and testing guidance. This task is for the Foundry Script module, primarily under `modules/foundry_script/`.

## Goal

Implement a headless Foundry Script lint command that emits machine-readable diagnostics in JSON and SARIF, and make sure the CLI argument is available in a release build, not only in test builds.

The command should use Foundry's real parser/analyzer rather than reimplementing syntax in external tooling. It should be suitable for CI and for external tools such as `anvil` to wrap.

## Important Context

Foundry already has these relevant pieces:

- `modules/foundry_script/fs_parser.{h,cpp}`: parses Foundry Script and records parser errors.
- `modules/foundry_script/fs_analyzer.{h,cpp}`: runs semantic analysis and emits analyzer errors / warnings.
- `modules/foundry_script/fs_warning.{h,cpp}`: warning IDs, names, messages, and severities.
- `modules/foundry_script/fs_format.{h,cpp}`: existing formatter and formatter CLI patterns.
- `modules/foundry_script/language_server/fs_extend_parser.{h,cpp}`: converts parser/analyzer errors and warnings into LSP diagnostics.
- `modules/foundry_script/register_types.cpp`: currently registers some Foundry Script command hooks.
- `main/main.cpp`: has special CLI detection for Foundry Script format commands.

The existing formatter CLI is useful as a reference, but do not copy its test-only registration model if that prevents release-build availability. The lint command must be available when `TESTS_ENABLED` is off.

## Desired CLI

Add a command-line entry point named:

```bash
foundry --headless --path /path/to/project --foundry_script-lint [options] [paths...]
```

Support at least:

```bash
--format=json
--format=sarif
--out <path>
--fail-on=error
--fail-on=warning
```

Recommended behavior:

- If no paths are provided, lint all relevant `.fs` files in the current project.
- If paths are provided, accept files and directories.
- Directory traversal should find `.fs` files deterministically.
- Hidden/cache directories such as `.godot` should not be linted.
- The command should run in headless mode.
- The command should not require `--test`.
- The command should be available in release/editor builds where `TESTS_ENABLED` is disabled.

If pure non-debug release builds currently compile out warning support via `DEBUG_ENABLED`, handle that explicitly. Prefer making the lint reporting path available in release builds with parser/analyzer diagnostics. If full warning support cannot be moved safely in this change, document the limitation in code comments and tests, but the CLI flag itself must still exist and return a clear diagnostic rather than being unknown.

## Diagnostic Model

Collect diagnostics from:

1. Parser errors from `FSParser`.
2. Analyzer errors from `FSAnalyzer`.
3. Existing `FSWarning` entries when available.

Do not invent many new lint rules in this pass. The first implementation can treat the existing parser/analyzer/warning diagnostics as the lint output. Keep the design open for later rule passes.

Each diagnostic should include:

- file path, preferably project-relative `res://...` when possible.
- start line and column.
- end line and column when available, otherwise a best-effort single-token/single-line range.
- severity: `error`, `warning`, `note`, or equivalent.
- source: `foundry_script`.
- rule ID:
  - parser errors can use something stable like `parse-error`.
  - analyzer errors can use `analyzer-error` unless a better existing code exists.
  - warnings should use `FSWarning::get_name()`.
- message.

## JSON Output

Emit stable JSON with a version field. Suggested shape:

```json
{
  "version": 1,
  "diagnostics": [
    {
      "path": "res://scripts/player.fs",
      "range": {
        "startLine": 12,
        "startColumn": 5,
        "endLine": 12,
        "endColumn": 20
      },
      "severity": "warning",
      "source": "foundry_script",
      "ruleId": "unused_variable",
      "message": "The local variable \"value\" is declared but never used."
    }
  ]
}
```

Use 1-based line/column values in this JSON format unless there is an established project convention that strongly argues otherwise. If you choose 0-based positions, document it and test it.

## SARIF Output

Emit SARIF 2.1.0 suitable for GitHub code scanning.

Required basics:

- top-level `version: "2.1.0"`.
- `$schema: "https://json.schemastore.org/sarif-2.1.0.json"`.
- one run with tool driver name such as `Foundry Script Lint`.
- `rules` entries for each emitted rule ID.
- `results` entries with:
  - `ruleId`.
  - message text.
  - level mapped from severity.
  - physical location with artifact URI and region.

Prefer project-relative or `res://` paths consistently. If SARIF consumers behave better with filesystem-relative paths, use those in SARIF and keep `res://` in JSON, but document the choice.

## Exit Codes

Use clear CI-friendly exit behavior:

- `0`: no diagnostics at or above the configured failure threshold.
- `1`: lint diagnostics were found at or above the configured failure threshold.
- `2` or another non-zero code: command/config/internal error, such as unreadable path or invalid `--format`.

`--fail-on=error` should fail only on errors.

`--fail-on=warning` should fail on warnings and errors.

## Implementation Notes

Suggested approach:

1. Add a small lint/reporting module in `modules/foundry_script/`, for example `fs_lint.{h,cpp}` or similar.
2. Keep the command-line parsing and file traversal close to the formatter CLI patterns, but do not gate it behind `TESTS_ENABLED`.
3. Reuse `FSParser` + `FSAnalyzer` directly.
4. Reuse or mirror the diagnostic conversion logic from `ExtendFSParser::update_diagnostics()` where appropriate.
5. Add JSON and SARIF serialization using existing Godot JSON utilities.
6. Wire command detection in `main/main.cpp` so the process initializes the needed runtime and recognizes `--foundry_script-lint`.
7. Register or invoke the command from module initialization in a way that works without `TESTS_ENABLED`.

Be careful with build guards:

- The formatter currently depends on `TOOLS_ENABLED` for comment-preserving formatting. Linting may not need that same guard.
- Existing warnings may be under `DEBUG_ENABLED`. Do not accidentally make the CLI disappear in release builds because warning support is unavailable.
- Avoid introducing editor-only dependencies unless the command is explicitly only for release editor/headless editor binaries. The user requirement is that the CLI argument is present in release builds.

## Tests

Add focused tests for:

- JSON output contains parser diagnostics for an invalid `.fs` file.
- JSON output contains analyzer/warning diagnostics for a valid but problematic `.fs` file, when warnings are compiled in.
- SARIF output is valid enough structurally: version, schema, runs, tool, rules, results, locations.
- `--fail-on=error` and `--fail-on=warning` exit behavior.
- The CLI argument is recognized without `--test` in a non-test/release-style build configuration if the repository test harness can cover that. If not, add the closest automated coverage and document the manual verification command.

Also run existing Foundry Script parser/analyzer/formatter tests relevant to the touched code.

## Acceptance Criteria

- `foundry --headless --path <project> --foundry_script-lint --format=json ...` emits stable JSON diagnostics.
- `foundry --headless --path <project> --foundry_script-lint --format=sarif ...` emits SARIF 2.1.0 diagnostics.
- `--out <path>` writes output to a file.
- Failure threshold behavior is deterministic and documented by tests.
- The CLI argument is not hidden behind `TESTS_ENABLED`.
- Existing formatter, parser, analyzer, and LSP behavior is not regressed.

## Non-Goals For This Pass

- Do not build a separate Go parser.
- Do not implement a large custom lint-rule framework unless it naturally falls out of the diagnostic model.
- Do not require the Foundry editor UI to be open.
- Do not make `anvil` call this yet; this task is Foundry-side support only.
