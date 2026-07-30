package main

import (
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

	nodeIndex := strings.Index(source, `"Node":`)
	timerIndex := strings.Index(source, `"Timer":`)
	if nodeIndex < 0 || timerIndex < 0 || nodeIndex > timerIndex {
		t.Fatalf("renderGo() did not sort native classes:\n%s", source)
	}

	if !strings.Contains(source, `"AsyncCallable": {kind: engineTypeBuiltin},`) {
		t.Fatalf("renderGo() did not emit AsyncCallable as a built-in:\n%s", source)
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
			api:       `{"builtin_classes":[{"name":"String"},{"name":"String"}]}`,
			wantError: `duplicate built-in type "String"`,
		},
		{
			name:      "empty native class",
			api:       `{"classes":[{"name":""}]}`,
			wantError: "native class has an empty name",
		},
		{
			name:      "cross-category duplicate",
			api:       `{"builtin_classes":[{"name":"String"}],"classes":[{"name":"String"}]}`,
			wantError: `type "String" appears in both categories`,
		},
		{
			name:      "manual built-in conflicts with native class",
			api:       `{"classes":[{"name":"AsyncCallable"}]}`,
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
