package fsgenerator

import (
	"strings"
	"testing"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
	"github.com/stretchr/testify/require"
)

func TestGenerateAggregatesCurrentFileCollisions(t *testing.T) {
	file := namespacedFile(
		[]*protoast.Message{{
			Position: protoast.Position{Line: 4, Column: 1},
			Name:     "Node",
			NestedMessages: []*protoast.Message{{
				Position: protoast.Position{Line: 6, Column: 3},
				Name:     "Timer",
			}},
		}},
		[]*protoast.Enum{{
			Position: protoast.Position{Line: 9, Column: 1},
			Name:     "String",
			Values:   []*protoast.EnumValue{{Name: "STRING_UNSPECIFIED", Number: 0}},
		}},
	)

	files, err := Generate(file, "types.proto", nil, Options{})
	require.Error(t, err)
	require.Nil(t, files)

	diagnostic := err.Error()
	node := `types.proto:4:1: message cafecito.game.v1.Node generates Foundry type "Node", which conflicts with native class "Node"`
	timer := `types.proto:6:3: message cafecito.game.v1.Node.Timer generates Foundry type "Timer", which conflicts with native class "Timer"`
	stringType := `types.proto:9:1: enum cafecito.game.v1.String generates Foundry type "String", which conflicts with built-in type "String"`
	require.Contains(t, diagnostic, node)
	require.Contains(t, diagnostic, timer)
	require.Contains(t, diagnostic, stringType)
	require.Less(t, strings.Index(diagnostic, node), strings.Index(diagnostic, timer))
	require.Less(t, strings.Index(diagnostic, timer), strings.Index(diagnostic, stringType))
	require.Contains(t, diagnostic, "set a non-empty file option such as:")
	require.Contains(t, diagnostic, `option (foundrytools.type_prefix) = "Game";`)
}

func TestGenerateAggregatesRawDistinctNormalizedMessageCollisions(t *testing.T) {
	file := namespacedFile([]*protoast.Message{
		{Position: protoast.Position{Line: 8, Column: 1}, Name: "node_"},
		{Position: protoast.Position{Line: 4, Column: 1}, Name: "Node"},
		{Position: protoast.Position{Line: 6, Column: 1}, Name: "node"},
	}, nil)

	files, err := Generate(file, "types.proto", nil, Options{})
	require.Error(t, err)
	require.Nil(t, files)

	diagnostic := err.Error()
	expected := []string{
		`types.proto:4:1: message cafecito.game.v1.Node generates Foundry type "Node"`,
		`types.proto:6:1: message cafecito.game.v1.node generates Foundry type "Node"`,
		`types.proto:8:1: message cafecito.game.v1.node_ generates Foundry type "Node"`,
	}
	last := -1
	for _, declaration := range expected {
		require.Equal(t, 1, strings.Count(diagnostic, declaration))
		index := strings.Index(diagnostic, declaration)
		require.Greater(t, index, last)
		last = index
	}
}

func TestGenerateAggregatesRawDistinctNestedEnumCollisions(t *testing.T) {
	file := namespacedFile([]*protoast.Message{{
		Name: "Holder",
		NestedEnums: []*protoast.Enum{
			{
				Position: protoast.Position{Line: 8, Column: 3},
				Name:     "string_",
				Values:   []*protoast.EnumValue{{Name: "STRING_TRAILING_UNSPECIFIED", Number: 0}},
			},
			{
				Position: protoast.Position{Line: 4, Column: 3},
				Name:     "String",
				Values:   []*protoast.EnumValue{{Name: "STRING_UPPER_UNSPECIFIED", Number: 0}},
			},
			{
				Position: protoast.Position{Line: 6, Column: 3},
				Name:     "string",
				Values:   []*protoast.EnumValue{{Name: "STRING_LOWER_UNSPECIFIED", Number: 0}},
			},
		},
	}}, nil)

	files, err := Generate(file, "types.proto", nil, Options{})
	require.Error(t, err)
	require.Nil(t, files)

	diagnostic := err.Error()
	for _, declaration := range []string{
		`types.proto:4:3: enum cafecito.game.v1.Holder.String generates Foundry type "String"`,
		`types.proto:6:3: enum cafecito.game.v1.Holder.string generates Foundry type "String"`,
		`types.proto:8:3: enum cafecito.game.v1.Holder.string_ generates Foundry type "String"`,
	} {
		require.Equal(t, 1, strings.Count(diagnostic, declaration))
	}
}

func TestPrefixResolvesEngineTypeCollisions(t *testing.T) {
	files, err := Generate(prefixedFile(
		"Game",
		[]*protoast.Message{{Name: "Node"}},
		[]*protoast.Enum{{
			Name:   "String",
			Values: []*protoast.EnumValue{{Name: "STRING_UNSPECIFIED", Number: 0}},
		}},
	), "types.proto", nil, Options{})

	require.NoError(t, err)

	require.Contains(t, files, "cafecito/game/v1/GameNode.pb.fs")
	require.Contains(t, files["cafecito/game/v1/GameNode.pb.fs"], "class_name GameNode")
	require.Contains(t, files, "cafecito/game/v1/GameString.pb.fs")
	require.Contains(t, files["cafecito/game/v1/GameString.pb.fs"], "enum_name GameString")
}

func TestPrefixCanStillProduceEngineTypeCollision(t *testing.T) {
	files, err := Generate(prefixedFile(
		"Animation",
		[]*protoast.Message{{Name: "Node"}},
		nil,
	), "types.proto", nil, Options{})

	require.Error(t, err)
	require.Nil(t, files)
	require.Contains(t, err.Error(),
		`message cafecito.game.v1.Node generates Foundry type "AnimationNode", which conflicts with native class "AnimationNode"`)
	require.Contains(t, err.Error(), `current prefix "Animation" still produces reserved Foundry type names`)
	require.Contains(t, err.Error(), "change it")
}

func TestInternalNonExposedClassRemainsLegal(t *testing.T) {
	files, err := Generate(namespacedFile(
		[]*protoast.Message{{Name: "FSNativeClass"}},
		nil,
	), "types.proto", nil, Options{})

	require.NoError(t, err)
	require.Contains(t, files, "cafecito/game/v1/FSNativeClass.pb.fs")
	require.Contains(t, files["cafecito/game/v1/FSNativeClass.pb.fs"], "class_name FSNativeClass")
}

func TestReferencedDependencyCollisionIsDeduplicatedAndLazy(t *testing.T) {
	inventory := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Messages: []*protoast.Message{{
			Name: "Node",
		}},
	}
	local := namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{FieldType: "Node", Name: "first", Number: 1, SourceFile: "inventory.proto"},
			{FieldType: "Node", Name: "second", Number: 2, SourceFile: "inventory.proto"},
		},
	}}, nil)

	files, err := Generate(local, "player.proto", []FileEntry{{
		File: inventory, Filename: "inventory.proto",
	}}, Options{})

	require.Error(t, err)
	require.Nil(t, files)
	require.Equal(t, 1, strings.Count(err.Error(),
		`inventory.proto: message cafecito.inventory.v1.Node generates Foundry type "Node"`))
	require.Contains(t, err.Error(), "set or change (foundrytools.type_prefix) in inventory.proto")

	unreferenced := namespacedFile([]*protoast.Message{{
		Name:   "Player",
		Fields: []*protoast.Field{{FieldType: "string", Name: "name", Number: 1}},
	}}, nil)
	files, err = Generate(unreferenced, "player.proto", []FileEntry{{
		File: inventory, Filename: "inventory.proto",
	}}, Options{})

	require.NoError(t, err)
	require.Contains(t, files, "cafecito/game/v1/Player.pb.fs")
}

func TestReferencedDependencyCollisionUsesExactRawDeclaration(t *testing.T) {
	inventory := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Messages: []*protoast.Message{
			{Position: protoast.Position{Line: 4, Column: 1}, Name: "Node"},
			{Position: protoast.Position{Line: 6, Column: 1}, Name: "node"},
			{Position: protoast.Position{Line: 8, Column: 1}, Name: "node_"},
		},
	}
	local := namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{{
			FieldType: "node", Name: "held", Number: 1, SourceFile: "inventory.proto",
		}},
	}}, nil)

	files, err := Generate(local, "player.proto", []FileEntry{{
		File: inventory, Filename: "inventory.proto",
	}}, Options{})

	require.Error(t, err)
	require.Nil(t, files)

	diagnostic := err.Error()
	require.Equal(t, 1, strings.Count(diagnostic,
		`inventory.proto:6:1: message cafecito.inventory.v1.node generates Foundry type "Node"`))
	require.NotContains(t, diagnostic, "cafecito.inventory.v1.Node generates")
	require.NotContains(t, diagnostic, "cafecito.inventory.v1.node_ generates")
}

func TestDeclarationIndexResolvesExactRelativeAndQualifiedRawPaths(t *testing.T) {
	declaration := declarationInfo{
		SourceName: "inventory.proto",
		Kind:       "message",
		ProtoName:  "cafecito.inventory.v1.Outer.node",
	}
	index := newDeclarationIndex("cafecito.inventory.v1", []declarationInfo{declaration})

	resolved, found := index.resolve("", "Outer.node")
	require.True(t, found)
	require.Equal(t, declaration, resolved)
	for _, reference := range []string{
		"cafecito.inventory.v1.Outer.node",
		".cafecito.inventory.v1.Outer.node",
	} {
		resolved, found := index.resolve(reference, "ignored.relative.path")
		require.True(t, found, reference)
		require.Equal(t, declaration, resolved)
	}
	resolved, found = index.resolve("", ".cafecito.inventory.v1.Outer.node")
	require.True(t, found)
	require.Equal(t, declaration, resolved)

	_, found = index.resolve("", ".Outer.node")
	require.False(t, found, "leading dot must select only the canonical full-name index")
	_, found = index.resolve("Outer.node", "cafecito.inventory.v1.Outer.node")
	require.False(t, found, "a supplied full path must not fall through to the relative index")
	_, found = index.resolve("", "Outer.Node")
	require.False(t, found)
}

func TestReferencedDependencyCollisionPrefersFullPathAcrossPackageAliasOverlap(t *testing.T) {
	inventory := packageAliasOverlapFile()
	local := namespacedFile([]*protoast.Message{{
		Name: "Player",
		Fields: []*protoast.Field{
			{
				FieldType: "p.Node", FullTypePath: ".p.p.Node",
				Name: "nested", Number: 1, SourceFile: "inventory.proto",
			},
			{
				FieldType: "Node", FullTypePath: "p.Node",
				Name: "top", Number: 2, SourceFile: "inventory.proto",
			},
		},
		Maps: []*protoast.MapField{{
			KeyType: "string", ValueType: "p.Node", FullValueTypePath: "p.p.Node",
			Name: "nested_map", Number: 3, ValueSourceFile: "inventory.proto",
		}},
		Oneofs: []*protoast.Oneof{{
			Name: "choice",
			Fields: []*protoast.Field{{
				FieldType: "Node", FullTypePath: "p.Node",
				Name: "top_choice", Number: 4, SourceFile: "inventory.proto",
			}},
		}},
	}}, nil)

	files, err := Generate(local, "player.proto", []FileEntry{{
		File: inventory, Filename: "inventory.proto",
	}}, Options{})

	require.Error(t, err)
	require.Nil(t, files)

	diagnostic := err.Error()
	nested := `inventory.proto:5:3: message p.p.Node generates Foundry type "Node"`
	top := `inventory.proto:8:1: message p.Node generates Foundry type "Node"`
	require.Equal(t, 1, strings.Count(diagnostic, nested))
	require.Equal(t, 1, strings.Count(diagnostic, top))
	require.Less(t, strings.Index(diagnostic, top), strings.Index(diagnostic, nested))
}

func TestReferencedDependencyCollisionUsesFullPathForEveryFieldShape(t *testing.T) {
	tests := map[string]func(*protoast.Message){
		"regular field": func(message *protoast.Message) {
			message.Fields = []*protoast.Field{{
				FieldType: "p.Node", FullTypePath: "p.p.Node",
				Name: "nested", Number: 1, SourceFile: "inventory.proto",
			}}
		},
		"map value": func(message *protoast.Message) {
			message.Maps = []*protoast.MapField{{
				KeyType: "string", ValueType: "p.Node", FullValueTypePath: ".p.p.Node",
				Name: "nested", Number: 1, ValueSourceFile: "inventory.proto",
			}}
		},
		"oneof field": func(message *protoast.Message) {
			message.Oneofs = []*protoast.Oneof{{
				Name: "choice",
				Fields: []*protoast.Field{{
					FieldType: "p.Node", FullTypePath: ".p.p.Node",
					Name: "nested", Number: 1, SourceFile: "inventory.proto",
				}},
			}}
		},
	}

	for name, configure := range tests {
		t.Run(name, func(t *testing.T) {
			message := &protoast.Message{Name: "Player"}
			configure(message)

			files, err := Generate(
				namespacedFile([]*protoast.Message{message}, nil),
				"player.proto",
				[]FileEntry{{File: packageAliasOverlapFile(), Filename: "inventory.proto"}}, Options{})

			require.Error(t, err)
			require.Nil(t, files)

			diagnostic := err.Error()
			require.Equal(t, 1, strings.Count(diagnostic,
				`inventory.proto:5:3: message p.p.Node generates Foundry type "Node"`))
			require.NotContains(t, diagnostic,
				`inventory.proto:8:1: message p.Node generates Foundry type "Node"`)
		})
	}
}

func packageAliasOverlapFile() *protoast.ProtoFile {
	return &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "p",
		Messages: []*protoast.Message{
			{
				Position: protoast.Position{Line: 4, Column: 1},
				Name:     "p",
				NestedMessages: []*protoast.Message{{
					Position: protoast.Position{Line: 5, Column: 3},
					Name:     "Node",
				}},
			},
			{
				Position: protoast.Position{Line: 8, Column: 1},
				Name:     "Node",
			},
		},
	}
}

func TestReferencedDependencyCollisionsSortAcrossSourceFiles(t *testing.T) {
	inventory := &protoast.ProtoFile{
		Syntax:  "proto3",
		Package: "cafecito.inventory.v1",
		Messages: []*protoast.Message{{
			Name: "Node",
		}},
	}
	local := namespacedFile(
		[]*protoast.Message{
			{
				Name:   "Player",
				Fields: []*protoast.Field{{FieldType: "Node", Name: "node", Number: 1, SourceFile: "inventory.proto"}},
			},
			{Name: "Timer"},
		},
		nil,
	)

	files, err := Generate(local, "player.proto", []FileEntry{{
		File: inventory, Filename: "inventory.proto",
	}}, Options{})

	require.Error(t, err)
	require.Nil(t, files)

	diagnostic := err.Error()
	imported := `inventory.proto: message cafecito.inventory.v1.Node generates Foundry type "Node"`
	localCollision := `player.proto: message cafecito.game.v1.Timer generates Foundry type "Timer"`
	require.Contains(t, diagnostic, imported)
	require.Contains(t, diagnostic, localCollision)
	require.Less(t, strings.Index(diagnostic, imported), strings.Index(diagnostic, localCollision))
}

func TestGenerateReportsBuiltInCollisionWithoutUnknownCoordinates(t *testing.T) {
	files, err := Generate(namespacedFile(nil, []*protoast.Enum{{
		Name:   "AsyncCallable",
		Values: []*protoast.EnumValue{{Name: "ASYNC_CALLABLE_UNSPECIFIED", Number: 0}},
	}}), "types.proto", nil, Options{})

	require.Error(t, err)
	require.Nil(t, files)
	require.Contains(t, err.Error(),
		`types.proto: enum cafecito.game.v1.AsyncCallable generates Foundry type "AsyncCallable", which conflicts with built-in type "AsyncCallable"`)
	require.NotContains(t, err.Error(), "types.proto:0:0")
}

func TestOneofCollisionKeepsEachRawOwnerIdentity(t *testing.T) {
	const generatedName = "OwnerChoiceCase"
	previous, existed := foundryEngineReservedTypes[generatedName]
	foundryEngineReservedTypes[generatedName] = engineTypeEntry{kind: engineTypeBuiltin}
	t.Cleanup(func() {
		if existed {
			foundryEngineReservedTypes[generatedName] = previous
		} else {
			delete(foundryEngineReservedTypes, generatedName)
		}
	})

	file := namespacedFile([]*protoast.Message{
		{
			Name: "Owner",
			Oneofs: []*protoast.Oneof{{
				Position: protoast.Position{Line: 5, Column: 3},
				Name:     "choice",
			}},
		},
		{
			Name: "owner",
			Oneofs: []*protoast.Oneof{{
				Position: protoast.Position{Line: 9, Column: 3},
				Name:     "choice",
			}},
		},
	}, nil)

	files, err := Generate(file, "types.proto", nil, Options{})
	require.Error(t, err)
	require.Nil(t, files)
	require.Equal(t, 1, strings.Count(err.Error(),
		`types.proto:5:3: oneof enum cafecito.game.v1.Owner.choice generates Foundry type "OwnerChoiceCase"`))
	require.Equal(t, 1, strings.Count(err.Error(),
		`types.proto:9:3: oneof enum cafecito.game.v1.owner.choice generates Foundry type "OwnerChoiceCase"`))
}

func TestCollisionCollectorTracksOriginIndependentlyOfSourceName(t *testing.T) {
	collector := newCollisionCollector()
	collector.AddLocal(declarationInfo{
		SourceName: "shared.proto", Kind: "message",
		ProtoName: "cafecito.game.v1.Node", GeneratedName: "Node",
	})
	collector.AddDependency(declarationInfo{
		SourceName: "shared.proto", Kind: "message",
		ProtoName: "cafecito.inventory.v1.Timer", GeneratedName: "Timer",
	})

	err := collector.Err("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "For declarations in shared.proto, set a non-empty file option")
	require.Contains(t, err.Error(), "set or change (foundrytools.type_prefix) in shared.proto")
}

func TestCollisionCollectorFormatsUnknownAndPartialPositions(t *testing.T) {
	collector := newCollisionCollector()
	collector.AddLocal(declarationInfo{
		Position: protoast.Position{Line: 7},
		Kind:     "message", ProtoName: "cafecito.game.v1.Node", GeneratedName: "Node",
	})
	collector.AddLocal(declarationInfo{
		Position: protoast.Position{Column: 5},
		Kind:     "enum", ProtoName: "cafecito.game.v1.String", GeneratedName: "String",
	})

	err := collector.Err("")
	require.Error(t, err)
	require.Contains(t, err.Error(), `<unknown source>:7: message cafecito.game.v1.Node`)
	require.Contains(t, err.Error(), `<unknown source>: enum cafecito.game.v1.String`)
	require.Contains(t, err.Error(), "For declarations in <unknown source>,")
	require.NotContains(t, err.Error(), ":0")
	require.NotContains(t, err.Error(), "For declarations in ,")
}

func TestCollisionCollectorHandlesNilAndUnknownEngineKind(t *testing.T) {
	var collector *collisionCollector
	require.NotPanics(t, func() {
		collector.AddLocal(declarationInfo{GeneratedName: "Node"})
	})
	require.NoError(t, collector.Err(""))

	const generatedName = "SyntheticReserved"
	previous, existed := foundryEngineReservedTypes[generatedName]
	foundryEngineReservedTypes[generatedName] = engineTypeEntry{kind: engineTypeKind(255)}
	t.Cleanup(func() {
		if existed {
			foundryEngineReservedTypes[generatedName] = previous
		} else {
			delete(foundryEngineReservedTypes, generatedName)
		}
	})

	collector = newCollisionCollector()
	collector.AddLocal(declarationInfo{
		SourceName: "types.proto", Kind: "message",
		ProtoName: "cafecito.game.v1.SyntheticReserved", GeneratedName: generatedName,
	})
	err := collector.Err("")
	require.Error(t, err)
	require.Contains(t, err.Error(), `conflicts with engine type "SyntheticReserved"`)
}
