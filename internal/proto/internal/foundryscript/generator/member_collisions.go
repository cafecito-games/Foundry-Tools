package fsgenerator

import (
	"fmt"
	"sort"
	"strings"

	protoast "github.com/cafecito-games/foundry-tools/internal/proto/internal/ast"
)

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
	byMessage map[string]map[string][]memberClaim
}

func newMemberCollisionCollector() *memberCollisionCollector {
	return &memberCollisionCollector{
		byMessage: map[string]map[string][]memberClaim{},
	}
}

func (c *memberCollisionCollector) AddMessage(
	sourceName, messageName string,
	fields []fieldPlan,
	oneofs []oneofPlan,
) {
	if c == nil {
		return
	}
	c.add(memberClaim{
		SourceName:    sourceName,
		MessageName:   messageName,
		Kind:          "generated unknown-field buffer",
		RawName:       unknownFieldsMember,
		GeneratedName: unknownFieldsMember,
	})
	for i := range fields {
		field := &fields[i]
		if field.RetainsUnknownEnum() {
			c.add(memberClaim{
				SourceName:    sourceName,
				Position:      field.Position,
				MessageName:   messageName,
				Kind:          "retained enum companion",
				RawName:       field.RawName,
				GeneratedName: field.UnknownMember(),
			})
		}
		if field.OneofCase != "" {
			continue
		}
		c.add(memberClaim{
			SourceName:    sourceName,
			Position:      field.Position,
			MessageName:   messageName,
			Kind:          field.Kind,
			RawName:       field.RawName,
			GeneratedName: field.Name,
			Escape:        field.Escape,
		})
	}
	for i := range oneofs {
		oneof := &oneofs[i]
		c.add(memberClaim{
			SourceName:    sourceName,
			Position:      oneof.Position,
			MessageName:   messageName,
			Kind:          "oneof",
			RawName:       oneof.RawField,
			GeneratedName: oneof.Field,
			Escape:        oneof.Escape,
		})
	}
}

func (c *memberCollisionCollector) add(claim memberClaim) {
	if c.byMessage == nil {
		c.byMessage = map[string]map[string][]memberClaim{}
	}
	if c.byMessage[claim.MessageName] == nil {
		c.byMessage[claim.MessageName] = map[string][]memberClaim{}
	}
	c.byMessage[claim.MessageName][claim.GeneratedName] = append(
		c.byMessage[claim.MessageName][claim.GeneratedName],
		claim,
	)
}

type memberCollision struct {
	MessageName   string
	GeneratedName string
	Claims        []memberClaim
}

func (c *memberCollisionCollector) Err() error {
	if c == nil {
		return nil
	}
	var collisions []memberCollision
	for messageName, byGeneratedName := range c.byMessage {
		for generatedName, claims := range byGeneratedName {
			if len(claims) < 2 {
				continue
			}
			sort.SliceStable(claims, func(i, j int) bool {
				return memberClaimLess(claims[i], claims[j])
			})
			collisions = append(collisions, memberCollision{
				MessageName:   messageName,
				GeneratedName: generatedName,
				Claims:        claims,
			})
		}
	}
	if len(collisions) == 0 {
		return nil
	}
	sort.Slice(collisions, func(i, j int) bool {
		if collisions[i].MessageName != collisions[j].MessageName {
			return collisions[i].MessageName < collisions[j].MessageName
		}
		return collisions[i].GeneratedName < collisions[j].GeneratedName
	})

	var diagnostic strings.Builder
	diagnostic.WriteString("generated Foundry member names collide:")
	for _, collision := range collisions {
		for _, claim := range collision.Claims {
			fmt.Fprintf(
				&diagnostic,
				"\n  %s: %s %s.%s generates Foundry member %q",
				memberClaimLocation(claim),
				claim.Kind,
				claim.MessageName,
				claim.RawName,
				claim.GeneratedName,
			)
			if description := claim.Escape.Description(); description != "" {
				fmt.Fprintf(&diagnostic, " after escaping %s", description)
			}
		}
		fmt.Fprintf(
			&diagnostic,
			"\n  rename one protobuf declaration in %s",
			collision.MessageName,
		)
	}
	return fmt.Errorf("%s", diagnostic.String())
}

func memberClaimLess(left, right memberClaim) bool {
	if left.Kind != right.Kind {
		return left.Kind < right.Kind
	}
	if left.RawName != right.RawName {
		return left.RawName < right.RawName
	}
	if left.SourceName != right.SourceName {
		return left.SourceName < right.SourceName
	}
	if left.Position.Line != right.Position.Line {
		return left.Position.Line < right.Position.Line
	}
	if left.Position.Column != right.Position.Column {
		return left.Position.Column < right.Position.Column
	}
	if left.Escape.Kind != right.Escape.Kind {
		return left.Escape.Kind < right.Escape.Kind
	}
	return left.Escape.ReservedName < right.Escape.ReservedName
}

func memberClaimLocation(claim memberClaim) string {
	sourceName := displaySourceName(claim.SourceName)
	if claim.Position.Line > 0 && claim.Position.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", sourceName, claim.Position.Line, claim.Position.Column)
	}
	if claim.Position.Line > 0 {
		return fmt.Sprintf("%s:%d", sourceName, claim.Position.Line)
	}
	return sourceName
}
