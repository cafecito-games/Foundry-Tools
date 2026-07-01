# Foundry Package Manager And Anvil Rename Design

## Summary

Add a Foundry-specific package manager to this tooling and rename the primary
CLI binary from `foundry-tools` to `anvil`.

The package manager copies the proven non-AssetLib behavior from
`/Users/christian/CafecitoGames/godot-addon-manager/`, adapted to Foundry
project discovery and file naming. It discovers projects by `project.foundry`,
stores editable dependencies in `packages.toml`, stores reproducible pins in
`packages.lock`, and installs package contents into the project `addons/`
directory.

Godot AssetLib browsing/search/add and the interactive TUI are out of scope for
this Foundry pass. All other core package-manager features come over: init,
add, install, update, remove, list, git/github-release/archive sources,
checksums, lockfile pinning, JSON output, quiet/verbose output, and download
size limits.

## Goals

- Add `anvil pkg ...` commands for Foundry package management.
- Keep package-manager implementation isolated behind
  `internal/packagemanager`.
- Preserve source fetching, install safety, checksum, and lockfile semantics
  from the Godot package manager.
- Use Foundry project conventions:
  - project marker: `project.foundry`
  - manifest: `packages.toml`
  - lockfile: `packages.lock`
  - install root: `<project-root>/addons/`
- Rename the main CLI binary and command identity from `foundry-tools` to
  `anvil`.
- Keep `protoc-gen-foundryscript` unchanged.

## Non-Goals

- No Godot AssetLib integration.
- No interactive add wizard in the first Foundry package-manager pass.
- No separate package-manager binary.
- No compatibility shim for the old `foundry-tools` command.
- No registry service or Foundry-specific package index.

## Architecture

Use an isolated package-manager boundary:

```text
internal/
  foundryscript/
  packagemanager/
    api.go
    internal/
      installer/
      lockfile/
      manifest/
      output/
      project/
      source/
```

`internal/packagemanager` is the only package the rest of the repository uses.
It exposes the public API for project discovery and package operations. Its
nested `internal/` packages hold implementation details copied and adapted from
`gpm`.

The CLI layer stays thin. `internal/cli` wires `anvil pkg` subcommands, parses
flags, invokes `packagemanager`, and renders output. It must not import
`internal/packagemanager/internal/...`.

This keeps package-management concerns separate from protobuf generation and
shared Foundry Script functionality. Future shared Foundry Script parsers or
helpers should live under a separate package such as `internal/foundryscript`,
not inside package-manager internals.

## Commands

The root command becomes `anvil`.

Existing commands move under the renamed binary:

```bash
anvil version
anvil proto generate -I proto -o foundry/generated proto/player.proto
anvil proto print-options-proto
```

New package commands:

```bash
anvil pkg init
anvil pkg add --name my_package \
  --source git \
  --url https://github.com/org/my_package.git \
  --version v1.0.0 \
  --source-path addons/my_package
anvil pkg install
anvil pkg update
anvil pkg update my_package
anvil pkg remove my_package
anvil pkg list
```

Source types:

```text
git             clone a Git ref and lock the resolved commit SHA
github-release  download one GitHub release asset by tag and optional glob
archive         download and extract a direct zip, .tar.gz, or .tgz URL
```

Shared package-manager flags:

```text
--dir                 start directory for project discovery
--json                emit machine-readable JSON for supported commands
--verbose, -v         print project, manifest, and lock paths
--quiet, -q           suppress non-error text output
--max-download-size   cap compressed downloads, for example 512MB or 1GiB
--max-extract-size    cap total extracted archive size
```

`pkg init` creates `packages.toml` in the current directory or the directory
passed with `--dir`. It fails if the file already exists.

`pkg add` saves a new manifest entry and installs it immediately. If
installation fails, the command removes the manifest entry it added and saves
the manifest again.

`pkg install` installs every package declared in `packages.toml`. When an
existing lock entry still matches the package spec hash, it honors that pin.

`pkg update` ignores selected lock pins and re-resolves packages. With no
package names, it updates all packages. With names, it updates only those
packages.

`pkg remove` removes the installed package directory, removes the manifest
entry, and removes the lock entry.

`pkg list` lists configured packages and whether each install directory exists.

## Manifest

`packages.toml` keeps the `gpm` schema shape with Foundry package terminology:

```toml
[packages.some_package]
source = "git"
url = "https://github.com/org/some_package.git"
version = "v1.0.0"
source_path = "addons/some_package"
install_as = "some_package"
exclude = ["editor_only"]
```

Supported fields:

```text
source       required for every package: git, github-release, or archive
url          required for git and archive
repo         required for github-release, in owner/repo form
version      required for git and github-release; advisory for archive
asset        optional GitHub release asset name or path.Match glob
source_path  optional relative subdirectory inside the fetched source
install_as   optional directory name under the project addons/ directory
exclude      optional directories below the selected install root to skip
checksum     optional SHA-256 for archive and github-release sources
```

`install_as` defaults to the manifest table key. Package names and `install_as`
values must be single directory names; absolute paths, path separators, `.`,
and `..` are rejected.

`source_path` must be relative and cannot escape the fetched source root.
`exclude` entries must be relative directories below the selected install root
and cannot escape it.

`checksum` must be a lowercase 64-character SHA-256 digest and is valid only
for archive and GitHub release sources.

## Lockfile

`packages.lock` records reproducible pins:

```toml
[packages.some_package]
resolved_version = "..."
source_path = "addons/some_package"
checksum = "..."
spec_hash = "..."
```

Fields:

```text
resolved_version  Git commit SHA, GitHub release tag, or archive version marker
source_path       selected source subdirectory that was installed
checksum          SHA-256 for archive and github-release downloads
spec_hash         hash of manifest fields that affect resolution
```

`pkg install` compares each manifest entry to its lock entry. If the spec hash
matches, the lock pin is honored. If the hash differs or the lock entry is
missing, the package is re-resolved and the lockfile is updated.

On a full install, lock entries for packages no longer present in
`packages.toml` are removed.

## Install Semantics

Project discovery walks upward from `--dir` or the current working directory
until it finds a regular `project.foundry` file. For a discovered root:

```text
manifest path: <root>/packages.toml
lock path:     <root>/packages.lock
install root:  <root>/addons
```

Install flow:

1. Validate the manifest.
2. Select packages by command arguments.
3. Fetch each source into a temporary directory.
4. Verify manifest and lockfile checksums when present.
5. Select the installed source directory:
   - use `source_path` when set
   - otherwise use the only directory under fetched `addons/` when exactly one
     exists
   - otherwise use the fetched root when no fetched `addons/` directory exists
   - otherwise fail and ask the user to set `source_path`
6. Copy to a staging directory under `<project-root>/addons`.
7. Reject symlinks in source trees and archive entries.
8. Atomically swap staging into `<project-root>/addons/<install_as-or-name>`,
   using a backup directory for rollback and crash recovery.
9. Save `packages.lock` after successful package installs.
10. Remove temporary fetch directories.

Archive downloads keep the `gpm` safety limits:

- compressed download cap defaults to 512 MiB
- extracted size cap defaults to 1 GiB
- archive entry count cap defaults to 20,000 entries
- zip-slip path traversal is rejected
- symlinks and hard links are rejected

Git sources use the system `git` binary, inherit normal user credentials, and
disable unsafe `ext::` remote-helper behavior. Private GitHub release assets use
`GITHUB_TOKEN` or `GH_TOKEN` when present.

## Public Package API

`internal/packagemanager` should expose a small API shaped around operations,
not implementation packages. A caller should be able to:

- initialize a manifest
- discover a Foundry project
- add one package and install it
- install all or selected packages
- update all or selected packages
- remove one package
- list configured packages

The API should accept explicit streams/options only where behavior requires it.
It should return structured results for CLI rendering and tests.

Implementation packages under `internal/packagemanager/internal/` are free to
mirror `gpm` package boundaries, but they are not part of the rest of the
repository's contract.

## Binary Rename

Rename the primary CLI binary from `foundry-tools` to `anvil`.

Required changes:

```text
cmd/foundry-tools/ -> cmd/anvil/
root command Use: "anvil"
version output: "anvil dev"
Taskfile build output: bin/anvil
Taskfile install command installs ./cmd/anvil
README install and usage examples use anvil
tests expect anvil command text
```

The `protoc-gen-foundryscript` binary stays unchanged because protoc and Buf
integrations depend on that plugin name.

The old `foundry-tools` command is removed from this repository's build output.

## Error Handling

Use typed package-manager errors equivalent to `gpm`:

```text
usage errors      bad flags or arguments
manifest errors   discovery, manifest, or lockfile problems
fetch errors      network, auth, git, source resolution, checksum mismatch
install errors    extraction and filesystem install problems
```

The CLI renders non-JSON errors to stderr prefixed with `anvil:`. In JSON mode,
supported commands emit a JSON object with `error` and `code` for failures.

Command exit codes should preserve the `gpm` meanings where possible:

```text
0  success
1  generic error
2  usage error
3  manifest or lockfile error
4  fetch error
5  install error
```

## Testing

Tests should be package-local and adapted from `gpm` where behavior is copied.

Core unit coverage:

- manifest load/save round trip for `packages.toml`
- manifest validation by source type
- package name, `install_as`, `source_path`, `exclude`, checksum validation
- lockfile load/save and spec hash matching
- project discovery via `project.foundry`
- installer staging, atomic replacement, backup recovery, exclusion handling,
  source path auto-detection, and symlink rejection
- git, archive, and GitHub release source fetchers with fakes or local test
  servers
- command behavior for init, add, install, update, remove, list
- JSON/quiet/verbose output behavior
- binary rename expectations in existing CLI tests

Integration-level coverage should stay focused on command behavior that crosses
package boundaries. Existing protobuf integration tests should continue to pass
under the renamed `anvil` binary while `protoc-gen-foundryscript` remains
unchanged.

## Documentation

Update README examples to use `anvil`.

Add package-manager documentation covering:

- project layout
- quickstart
- commands
- manifest format
- lockfile behavior
- source types
- authentication for private git and GitHub release sources
- troubleshooting common fetch/install errors

## Rollout

This is a breaking CLI rename. The implementation should update all in-repo
references in the same branch so tests, docs, and build scripts agree on
`anvil`.

Because the package manager is new for Foundry projects, there is no migration
path for existing Foundry package manifests. `packages.toml` and
`packages.lock` are introduced as new files.
