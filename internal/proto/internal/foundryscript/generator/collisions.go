package fsgenerator

import (
	"fmt"
	"sort"
	"strings"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
)

type declarationInfo struct {
	SourceName    string
	Position      protoast.Position
	Kind          string
	ProtoName     string
	GeneratedName string
}

type typeCollision struct {
	Declaration declarationInfo
	EngineName  string
	EngineKind  engineTypeKind
}

type collisionCollector struct {
	byDeclaration map[string]typeCollision
	localSource   string
}

func newCollisionCollector(localSource string) *collisionCollector {
	return &collisionCollector{
		byDeclaration: map[string]typeCollision{},
		localSource:   localSource,
	}
}

// Add records info when its final generated name is reserved by the engine.
func (c *collisionCollector) Add(info declarationInfo) {
	if c == nil {
		return
	}
	engineType, reserved := foundryEngineReservedTypes[info.GeneratedName]
	if !reserved {
		return
	}
	if c.byDeclaration == nil {
		c.byDeclaration = map[string]typeCollision{}
	}
	key := strings.Join([]string{info.SourceName, info.Kind, info.ProtoName}, "\x00")
	c.byDeclaration[key] = typeCollision{
		Declaration: info,
		EngineName:  info.GeneratedName,
		EngineKind:  engineType.kind,
	}
}

// Err returns one stable diagnostic containing every recorded collision.
func (c *collisionCollector) Err(prefix string) error {
	if c == nil || len(c.byDeclaration) == 0 {
		return nil
	}

	collisions := make([]typeCollision, 0, len(c.byDeclaration))
	for _, collision := range c.byDeclaration {
		collisions = append(collisions, collision)
	}
	sort.Slice(collisions, func(i, j int) bool {
		left, right := collisions[i].Declaration, collisions[j].Declaration
		if left.SourceName != right.SourceName {
			return left.SourceName < right.SourceName
		}
		if left.ProtoName != right.ProtoName {
			return left.ProtoName < right.ProtoName
		}
		return left.Kind < right.Kind
	})

	var diagnostic strings.Builder
	diagnostic.WriteString("generated Foundry type names conflict with reserved engine types:")
	dependencySources := map[string]bool{}
	hasLocalCollision := false
	for _, collision := range collisions {
		info := collision.Declaration
		if info.SourceName == c.localSource {
			hasLocalCollision = true
		} else {
			dependencySources[info.SourceName] = true
		}
		fmt.Fprintf(&diagnostic, "\n  %s: %s %s generates Foundry type %q, which conflicts with %s %q",
			declarationLocation(info), info.Kind, info.ProtoName, info.GeneratedName,
			engineKindDescription(collision.EngineKind), collision.EngineName)
	}

	diagnostic.WriteString("\n\nRemediation:")
	if hasLocalCollision {
		if prefix == "" {
			diagnostic.WriteString("\n  For declarations in ")
			diagnostic.WriteString(c.localSource)
			diagnostic.WriteString(", set a non-empty file option such as:\n")
			diagnostic.WriteString(`  option (foundrytools.type_prefix) = "Game";`)
		} else {
			fmt.Fprintf(&diagnostic,
				"\n  The current prefix %q still produces reserved Foundry type names; change it to a prefix that makes each generated name non-reserved.",
				prefix)
		}
	}

	sources := make([]string, 0, len(dependencySources))
	for source := range dependencySources {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	for _, source := range sources {
		fmt.Fprintf(&diagnostic,
			"\n  For referenced declarations, set or change (foundrytools.type_prefix) in %s.",
			source)
	}
	return fmt.Errorf("%s", diagnostic.String())
}

func declarationLocation(info declarationInfo) string {
	if info.Position.Line > 0 && info.Position.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", info.SourceName, info.Position.Line, info.Position.Column)
	}
	return info.SourceName
}

func engineKindDescription(kind engineTypeKind) string {
	switch kind {
	case engineTypeBuiltin:
		return "built-in type"
	case engineTypeNativeClass:
		return "native class"
	default:
		return "engine type"
	}
}
