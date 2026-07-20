// Package analyzer provides high-level schema analysis.
package analyzer

import (
	"fmt"
	"strings"

	"github.com/EdgarOrtegaRamirez/gqlscope/internal/parser"
	"github.com/EdgarOrtegaRamirez/gqlscope/internal/reporter"
)

// Analyze returns schema information for reporting.
func Analyze(s *parser.Schema) reporter.SchemaInfo {
	info := reporter.SchemaInfo{
		TypeCount:       len(s.Types),
		DeprecatedCount: 0,
		TotalFields:     0,
		TotalEnums:      0,
		HasQuery:        s.HasOperation("query"),
		HasMutation:     s.HasOperation("mutation"),
		HasSubscription: s.HasOperation("subscription"),
	}

	typeNames := make([]string, 0, len(s.Types))
	for name := range s.Types {
		typeNames = append(typeNames, name)
	}
	info.TypeNames = typeNames

	for _, t := range s.Types {
		info.TotalFields += len(t.Fields)
		info.TotalEnums += len(t.EnumValues)

		for _, f := range t.Fields {
			if f.Deprecated {
				info.DeprecatedCount++
				info.DeprecatedFields = append(info.DeprecatedFields, fmt.Sprintf("%s.%s", t.Name, f.Name))
			}
		}

		if t.Kind == parser.TypeKindEnum {
			for _, v := range t.EnumValues {
				if v.Deprecated {
					info.DeprecatedEnums = append(info.DeprecatedEnums, fmt.Sprintf("%s.%s", t.Name, v.Name))
				}
			}
		}

		if t.Kind == parser.TypeKindDirective {
			info.Directives = append(info.Directives, t.Name)
		}
	}

	if s.Queries != nil {
		info.QueryCount = len(s.Queries)
	}
	if s.Mutations != nil {
		info.MutationCount = len(s.Mutations)
	}
	if s.Subscriptions != nil {
		info.SubscriptionCount = len(s.Subscriptions)
	}

	return info
}

// FindUnreachableTypes finds types that aren't used by any other type.
func FindUnreachableTypes(s *parser.Schema) []string {
	var unused []string
	builtinScalars := map[string]bool{
		"String": true, "Int": true, "Float": true, "Boolean": true, "ID": true,
	}
	for name := range s.Types {
		if name == "Query" || name == "Mutation" || name == "Subscription" {
			continue
		}
		if builtinScalars[name] {
			continue
		}
		if !s.IsTypeUsed(name) {
			unused = append(unused, name)
		}
	}
	return unused
}

// Validate checks for common schema issues and returns warnings.
func Validate(s *parser.Schema) []string {
	var warnings []string

	// Check for missing query root
	if !s.HasOperation("query") {
		warnings = append(warnings, "No Query type defined")
	}

	// Check for self-referencing types (potential infinite recursion)
	type QueueItem struct {
		name string
		path string
	}
	seen := make(map[string]bool)
	queue := []QueueItem{}

	if s.Queries != nil {
		for _, f := range s.Queries {
			queue = append(queue, QueueItem{name: f.TypeName, path: f.Name})
		}
	}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if seen[item.name] {
			continue
		}
		seen[item.name] = true

		t := s.GetType(item.name)
		if t == nil {
			continue
		}

		if t.Kind == parser.TypeKindObject || t.Kind == parser.TypeKindInterface || t.Kind == parser.TypeKindInput {
			for _, f := range t.Fields {
				if f.TypeName == item.name {
					warnings = append(warnings, fmt.Sprintf("Type '%s' is self-referencing via field '%s' (possible infinite recursion)", item.name, f.Name))
					continue
				}
				if !seen[f.TypeName] {
					queue = append(queue, QueueItem{name: f.TypeName, path: f.Name})
				}
			}
		}
	}

	// Check for unused types
	unused := FindUnreachableTypes(s)
	if len(unused) > 0 {
		warnings = append(warnings, fmt.Sprintf("Unused types: %s", strings.Join(unused, ", ")))
	}

	// Check for deprecated fields without reason
	for _, t := range s.Types {
		for _, f := range t.Fields {
			if f.Deprecated && f.DeprecationReason == "" {
				warnings = append(warnings, fmt.Sprintf("Field '%s.%s' is deprecated without a reason", t.Name, f.Name))
			}
		}
	}

	return warnings
}
