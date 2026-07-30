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
	Origin      declarationOrigin
}

type collisionCollector struct {
	byDeclaration map[string]typeCollision
}

type declarationOrigin uint8

const (
	declarationOriginDependency declarationOrigin = iota
	declarationOriginLocal
)

func newCollisionCollector() *collisionCollector {
	return &collisionCollector{
		byDeclaration: map[string]typeCollision{},
	}
}

// Add records a local declaration when its final generated name is reserved by
// the engine. Dependency callers use AddDependency to make origin explicit.
func (c *collisionCollector) Add(info declarationInfo) {
	c.add(info, declarationOriginLocal)
}

// AddLocal records a local declaration independently of its source spelling.
func (c *collisionCollector) AddLocal(info declarationInfo) {
	c.add(info, declarationOriginLocal)
}

// AddDependency records a referenced dependency declaration.
func (c *collisionCollector) AddDependency(info declarationInfo) {
	c.add(info, declarationOriginDependency)
}

func (c *collisionCollector) add(info declarationInfo, origin declarationOrigin) {
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
		Origin:      origin,
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
	localSources := map[string]bool{}
	dependencySources := map[string]bool{}
	for _, collision := range collisions {
		info := collision.Declaration
		if collision.Origin == declarationOriginLocal {
			localSources[info.SourceName] = true
		} else {
			dependencySources[info.SourceName] = true
		}
		fmt.Fprintf(&diagnostic, "\n  %s: %s %s generates Foundry type %q, which conflicts with %s %q",
			declarationLocation(info), info.Kind, info.ProtoName, info.GeneratedName,
			engineKindDescription(collision.EngineKind), collision.EngineName)
	}

	diagnostic.WriteString("\n\nRemediation:")
	if len(localSources) > 0 {
		if prefix == "" {
			sources := sortedSourceNames(localSources)
			diagnostic.WriteString("\n  For declarations in ")
			diagnostic.WriteString(displaySourceName(sources[0]))
			diagnostic.WriteString(", set a non-empty file option such as:\n")
			diagnostic.WriteString(`  option (foundrytools.type_prefix) = "Game";`)
		} else {
			fmt.Fprintf(&diagnostic,
				"\n  The current prefix %q still produces reserved Foundry type names; change it to a prefix that makes each generated name non-reserved.",
				prefix)
		}
	}

	for _, source := range sortedSourceNames(dependencySources) {
		fmt.Fprintf(&diagnostic,
			"\n  For referenced declarations, set or change (foundrytools.type_prefix) in %s.",
			displaySourceName(source))
	}
	return fmt.Errorf("%s", diagnostic.String())
}

func declarationLocation(info declarationInfo) string {
	sourceName := displaySourceName(info.SourceName)
	if info.Position.Line > 0 && info.Position.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", sourceName, info.Position.Line, info.Position.Column)
	}
	if info.Position.Line > 0 {
		return fmt.Sprintf("%s:%d", sourceName, info.Position.Line)
	}
	return sourceName
}

func displaySourceName(sourceName string) string {
	if sourceName == "" {
		return "<unknown source>"
	}
	return sourceName
}

func sortedSourceNames(seen map[string]bool) []string {
	sources := make([]string, 0, len(seen))
	for source := range seen {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	return sources
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
