package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadReservedTypes(t *testing.T) {
	got, err := loadAPI("testdata/extension_api.json")
	if err != nil {
		t.Fatalf("loadAPI() error = %v", err)
	}

	want := reservedTypes{
		Builtins:      []string{"Array", "AsyncCallable", "String"},
		NativeClasses: []string{"Node", "Timer"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loadAPI() = %#v, want %#v", got, want)
	}
}

func TestRenderGoIsDeterministicAndRecordsVersion(t *testing.T) {
	got, err := renderGo(reservedTypes{
		Version:       "0.1.alpha14.gh.b9a5e66c2",
		Builtins:      []string{"String", "AsyncCallable"},
		NativeClasses: []string{"Timer", "Node"},
	})
	if err != nil {
		t.Fatalf("renderGo() error = %v", err)
	}

	source := string(got)
	if !strings.Contains(source, `const foundryEngineTypeSourceVersion = "0.1.alpha14.gh.b9a5e66c2"`) {
		t.Fatalf("renderGo() did not record the source version:\n%s", source)
	}

	asyncCallableIndex := strings.Index(source, `"AsyncCallable":`)
	stringIndex := strings.Index(source, `"String":`)
	if asyncCallableIndex < 0 || stringIndex < 0 || asyncCallableIndex > stringIndex {
		t.Fatalf("renderGo() did not sort built-in types:\n%s", source)
	}

	nodeIndex := strings.Index(source, `"Node":`)
	timerIndex := strings.Index(source, `"Timer":`)
	if nodeIndex < 0 || timerIndex < 0 || nodeIndex > timerIndex {
		t.Fatalf("renderGo() did not sort native classes:\n%s", source)
	}

	if !strings.Contains(source, `"AsyncCallable": {kind: engineTypeBuiltin},`) {
		t.Fatalf("renderGo() did not emit AsyncCallable as a built-in:\n%s", source)
	}

	equivalent, err := renderGo(reservedTypes{
		Version:       "0.1.alpha14.gh.b9a5e66c2",
		Builtins:      []string{"AsyncCallable", "String"},
		NativeClasses: []string{"Node", "Timer"},
	})
	if err != nil {
		t.Fatalf("renderGo() with equivalent input error = %v", err)
	}
	if !bytes.Equal(got, equivalent) {
		t.Fatalf("renderGo() output differs for equivalent input order:\nfirst:\n%s\nsecond:\n%s", got, equivalent)
	}
}

func TestLoadReservedTypesRejectsDuplicatesAndMissingNames(t *testing.T) {
	tests := []struct {
		name      string
		api       string
		wantError string
	}{
		{
			name:      "duplicate built-in",
			api:       `{"builtin_classes":[{"name":"String"},{"name":"String"}],"classes":[]}`,
			wantError: `duplicate built-in type "String"`,
		},
		{
			name:      "empty built-in",
			api:       `{"builtin_classes":[{"name":""}],"classes":[]}`,
			wantError: "built-in type has an empty name",
		},
		{
			name:      "empty native class",
			api:       `{"builtin_classes":[],"classes":[{"name":""}]}`,
			wantError: "native class has an empty name",
		},
		{
			name:      "duplicate native class",
			api:       `{"builtin_classes":[],"classes":[{"name":"Node"},{"name":"Node"}]}`,
			wantError: `duplicate native class "Node"`,
		},
		{
			name:      "cross-category duplicate",
			api:       `{"builtin_classes":[{"name":"String"}],"classes":[{"name":"String"}]}`,
			wantError: `type "String" appears in both categories`,
		},
		{
			name:      "manual built-in conflicts with native class",
			api:       `{"builtin_classes":[],"classes":[{"name":"AsyncCallable"}]}`,
			wantError: `type "AsyncCallable" appears in both categories`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeAPI(strings.NewReader(test.api))
			if err == nil {
				t.Fatal("decodeAPI() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("decodeAPI() error = %q, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestDecodeAPIRequiresTypeSections(t *testing.T) {
	tests := []struct {
		name      string
		api       string
		wantError string
	}{
		{
			name:      "null document",
			api:       `null`,
			wantError: "builtin_classes is missing or null",
		},
		{
			name:      "empty object",
			api:       `{}`,
			wantError: "builtin_classes is missing or null",
		},
		{
			name:      "schema drift",
			api:       `{"builtins":[],"native_classes":[]}`,
			wantError: "builtin_classes is missing or null",
		},
		{
			name:      "missing built-ins",
			api:       `{"classes":[]}`,
			wantError: "builtin_classes is missing or null",
		},
		{
			name:      "null built-ins",
			api:       `{"builtin_classes":null,"classes":[]}`,
			wantError: "builtin_classes is missing or null",
		},
		{
			name:      "missing native classes",
			api:       `{"builtin_classes":[]}`,
			wantError: "classes is missing or null",
		},
		{
			name:      "null native classes",
			api:       `{"builtin_classes":[],"classes":null}`,
			wantError: "classes is missing or null",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeAPI(strings.NewReader(test.api))
			if err == nil {
				t.Fatal("decodeAPI() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("decodeAPI() error = %q, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestDecodeAPIAcceptsEmptyTypeSections(t *testing.T) {
	got, err := decodeAPI(strings.NewReader(`{"builtin_classes":[],"classes":[]}`))
	if err != nil {
		t.Fatalf("decodeAPI() error = %v", err)
	}

	want := reservedTypes{Builtins: []string{"AsyncCallable"}, NativeClasses: []string{}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decodeAPI() = %#v, want %#v", got, want)
	}
}

func TestRunRequiresFlags(t *testing.T) {
	dir := t.TempDir()
	apiPath := filepath.Join(dir, "extension_api.json")
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "api",
			wantError: "--api is required",
		},
		{
			name:      "version",
			args:      []string{"--api", apiPath},
			wantError: "--version is required",
		},
		{
			name:      "output",
			args:      []string{"--api", apiPath, "--version", "test-version"},
			wantError: "--output is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := run(test.args)
			if err == nil {
				t.Fatal("run() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("run() error = %q, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestRunWritesVersionedOutput(t *testing.T) {
	dir := t.TempDir()
	apiPath := writeTestAPI(t, dir)
	outputPath := filepath.Join(dir, "engine_reserved_types.gen.go")

	err := run([]string{
		"--api", apiPath,
		"--version", "0.1.test-version",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	source, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read generated output: %v", err)
	}
	if !bytes.Contains(source, []byte(`const foundryEngineTypeSourceVersion = "0.1.test-version"`)) {
		t.Fatalf("generated output did not record --version:\n%s", source)
	}
	if !bytes.Contains(source, []byte(`"String":`)) || !bytes.Contains(source, []byte(`"Node":`)) {
		t.Fatalf("generated output did not contain API types:\n%s", source)
	}
}

func TestRunNormalizesExistingOutputPermissions(t *testing.T) {
	dir := t.TempDir()
	apiPath := writeTestAPI(t, dir)
	outputPath := filepath.Join(dir, "engine_reserved_types.gen.go")
	if err := os.WriteFile(outputPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("write existing output: %v", err)
	}

	err := run([]string{
		"--api", apiPath,
		"--version", "0.1.test-version",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("stat generated output: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o644); got != want {
		t.Fatalf("generated output permissions = %04o, want %04o", got, want)
	}
}

func TestRepositoryTaskDeclaresEngineTypeRefresh(t *testing.T) {
	root := findRepoRoot(t)
	taskfile, err := os.ReadFile(filepath.Join(root, "Taskfile.yml"))
	if err != nil {
		t.Fatalf("read repository Taskfile: %v", err)
	}

	source := string(taskfile)
	for _, want := range []string{"gen-engine-types:", "sync-foundry-engine-types.sh write"} {
		if !strings.Contains(source, want) {
			t.Errorf("Taskfile.yml does not contain %q", want)
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat go.mod in %q: %v", dir, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root containing go.mod not found")
		}
		dir = parent
	}
}

func writeTestAPI(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "extension_api.json")
	api := []byte(`{"builtin_classes":[{"name":"String"}],"classes":[{"name":"Node"}]}`)
	if err := os.WriteFile(path, api, 0o644); err != nil {
		t.Fatalf("write extension API: %v", err)
	}
	return path
}
