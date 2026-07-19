// Package parser provides a GraphQL SDL schema parser.
package parser

import (
	"strings"
)

// Schema represents a parsed GraphQL schema.
type Schema struct {
	Types         map[string]*Type
	Queries       []*Field
	Mutations     []*Field
	Subscriptions []*Field
	SchemaDef     *SchemaDef
	Directives    []string
}

// SchemaDef represents the schema definition block.
type SchemaDef struct {
	Query, Mutation, Subscription string
}

// TypeKind represents a GraphQL type kind.
type TypeKind string

const (
	TypeKindObject    TypeKind = "OBJECT"
	TypeKindInterface TypeKind = "INTERFACE"
	TypeKindUnion     TypeKind = "UNION"
	TypeKindEnum      TypeKind = "ENUM"
	TypeKindInput     TypeKind = "INPUT_OBJECT"
	TypeKindScalar    TypeKind = "SCALAR"
	TypeKindDirective TypeKind = "DIRECTIVE"
)

// Type represents a GraphQL type definition.
type Type struct {
	Name        string
	Kind        TypeKind
	Description string
	Fields      []*Field
	Interfaces  []string
	UnionTypes  []string
	EnumValues  []EnumValue
}

// EnumValue represents an enum value.
type EnumValue struct {
	Name       string
	Deprecated bool
	Reason     string
}

// Field represents a field in a GraphQL type.
type Field struct {
	Name              string
	TypeName          string
	Args              []Argument
	Deprecated        bool
	DeprecationReason string
}

// Argument represents a field argument.
type Argument struct {
	Name     string
	TypeName string
	Default  string
}

// Parser holds parsing state.
type Parser struct {
	lines []string
	pos   int
}

// Parse returns a Schema from SDL content.
func Parse(content string) (*Schema, error) {
	p := &Parser{lines: strings.Split(content, "\n"), pos: 0}
	s := &Schema{Types: make(map[string]*Type)}
	// built-in scalars
	for _, n := range []string{"String", "Int", "Float", "Boolean", "ID"} {
		s.Types[n] = &Type{Name: n, Kind: TypeKindScalar}
	}
	p.parseDefinitions(s)
	return s, nil
}

func (p *Parser) parseDefinitions(s *Schema) {
	for p.pos < len(p.lines) {
		line := p.nextNonEmptyOrComment()
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "schema "):
			p.parseSchemaDef(s, line)
		case strings.HasPrefix(line, "type "):
			if t := p.parseTypeDef(s, TypeKindObject, line, 5); t != nil {
				s.Types[t.Name] = t
				if t.Name == "Query" {
					s.Queries = t.Fields
				} else if t.Name == "Mutation" {
					s.Mutations = t.Fields
				} else if t.Name == "Subscription" {
					s.Subscriptions = t.Fields
				}
			}
		case strings.HasPrefix(line, "interface "):
			if t := p.parseTypeDef(s, TypeKindInterface, line, 10); t != nil {
				s.Types[t.Name] = t
			}
		case strings.HasPrefix(line, "input "):
			if t := p.parseTypeDef(s, TypeKindInput, line, 6); t != nil {
				s.Types[t.Name] = t
			}
		case strings.HasPrefix(line, "union "):
			p.parseUnion(s, line)
		case strings.HasPrefix(line, "enum "):
			p.parseEnum(s, line)
		case strings.HasPrefix(line, "scalar "):
			n := extractIdent(line, 7)
			s.Types[n] = &Type{Name: n, Kind: TypeKindScalar}
		case strings.HasPrefix(line, "directive @"):
			p.parseDirective(s, line)
		}
	}
}

func (p *Parser) nextNonEmptyOrComment() string {
	for p.pos < len(p.lines) {
		line := strings.TrimSpace(p.lines[p.pos])
		p.pos++
		if line == "" || len(line) > 0 && line[0] == '#' {
			continue
		}
		return line
	}
	return ""
}

func (p *Parser) parseTypeDef(s *Schema, kind TypeKind, line string, skip int) *Type {
	name := extractIdent(line, skip)
	t := &Type{Name: name, Kind: kind}

	// Check implements clause
	if idx := strings.Index(line, "implements"); idx > 0 {
		rest := strings.TrimSpace(line[idx+len("implements"):])
		rest = strings.Split(rest, "{")[0]
		for _, iface := range strings.Split(rest, "&") {
			iface = strings.TrimSpace(iface)
			if iface != "" {
				t.Interfaces = append(t.Interfaces, iface)
			}
		}
	}

	t.Fields = p.parseFieldsForType(kind)
	return t
}

// parseFieldsForType reads the block { ... } and extracts fields (or enum values).
func (p *Parser) parseFieldsForType(kind TypeKind) []*Field {
	// Find the block. It may be on the same line or the next.
	// Find first { starting from position 5 (after "type " etc)
	content := ""
	line := p.lines[p.pos-1]
	start := strings.Index(line, "{")
	if start < 0 {
		start = 5
	}

	// Check if { is present on the definition line
	foundBrace := false
	for i := start; i < len(line); i++ {
		if line[i] == '{' {
			foundBrace = true
			break
		}
	}

	if foundBrace {
		// Content after { on same line
		after := line[start+1:]
		// Check if } is also on this line
		closeIdx := strings.Index(after, "}")
		if closeIdx >= 0 {
			content = after[:closeIdx]
		} else {
			// Multi-line block, read subsequent lines
			content = after
			for p.pos < len(p.lines) {
				ln := p.lines[p.pos]
				p.pos++
				for _, ch := range ln {
					if ch == '{' {
						// nested block - skip to }
						// not common in SDL but handle gracefully
					}
					if ch == '}' {
						goto done
					}
				}
				content += "\n" + ln
			}
		done:
		}
	} else {
		// { is on a later line - skip empty/comment lines, then read block
		for p.pos < len(p.lines) {
			ln := strings.TrimSpace(p.lines[p.pos])
			if ln == "" || (len(ln) > 0 && ln[0] == '#') {
				p.pos++
				continue
			}
			// Check for opening brace
			if strings.Contains(ln, "{") {
				// Content after {
				si := strings.Index(ln, "{")
				content = ln[si+1:]
				if ci := strings.Index(content, "}"); ci >= 0 {
					content = content[:ci]
				}
				p.pos++
			} else {
				// Not a brace line, could be content before brace
				p.pos++
				continue
			}
			// Continue reading block content
			for p.pos < len(p.lines) {
				ln := p.lines[p.pos]
				p.pos++
				for _, ch := range ln {
					if ch == '}' {
						goto done2
					}
				}
				content += "\n" + ln
			}
		done2:
		}
	}

	return p.parseBlockContent(content, kind)
}

func (p *Parser) parseBlockContent(content string, kind TypeKind) []*Field {
	var fields []*Field
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if f := p.tryParseFieldOrEnumValue(line, kind); f != nil {
			fields = append(fields, f)
		}
	}
	return fields
}

func (p *Parser) tryParseFieldOrEnumValue(line string, kind TypeKind) *Field {
	if kind == TypeKindEnum {
		// For enums, check @deprecated first (reason clause has : inside parens)
		if idx := strings.Index(line, "@deprecated"); idx >= 0 {
			// This is an enum value with deprecation
			name := strings.TrimSpace(line[:idx])
			f := &Field{Name: name, Deprecated: true}
			afterDep := strings.TrimSpace(line[idx+len("@deprecated"):])
			f.DeprecationReason = extractQuotedString(afterDep)
			return f
		}
		// Plain enum value (no colon)
		if !strings.Contains(line, ":") {
			return &Field{Name: strings.TrimSpace(line)}
		}
	}
	return p.parseField(line)
}

func (p *Parser) parseSchemaDef(s *Schema, line string) {
	def := &SchemaDef{}

	// Check inline format: schema(query: Query)
	if idx := strings.Index(line, "("); idx > 0 {
		closeIdx := strings.Index(line[idx:], ")")
		if closeIdx > 0 {
			// Parse arguments between ( and )
			for _, part := range splitComma(line[idx+1 : idx+closeIdx]) {
				p.parseSchemaOption(def, strings.TrimSpace(part))
			}
			s.SchemaDef = def
			return
		}
	}

	// Block format: schema { ... }
	// Try to find block content
	if idx := strings.Index(line, "{"); idx > 0 {
		content := line[idx+1:]
		if ci := strings.Index(content, "}"); ci >= 0 {
			content = content[:ci]
			for _, part := range strings.Split(content, ";") {
				p.parseSchemaOption(def, strings.TrimSpace(part))
			}
			s.SchemaDef = def
			return
		}
		// Multi-line block
		for _, part := range strings.Split(content, ";") {
			p.parseSchemaOption(def, strings.TrimSpace(part))
		}
		// Read subsequent lines
		for p.pos < len(p.lines) {
			ln := strings.TrimSpace(p.lines[p.pos])
			p.pos++
			if ln == "" {
				continue
			}
			if strings.HasPrefix(ln, "query:") || strings.HasPrefix(ln, "mutation:") || strings.HasPrefix(ln, "subscription:") {
				p.parseSchemaOption(def, ln)
			} else if strings.Contains(ln, "}") {
				break
			}
		}
	} else {
		// Find { on next non-empty line
		for p.pos < len(p.lines) {
			ln := strings.TrimSpace(p.lines[p.pos])
			p.pos++
			if ln == "" || ln[0] == '#' {
				continue
			}
			if strings.Contains(ln, "{") {
				si := strings.Index(ln, "{")
				after := ln[si+1:]
				if ci := strings.Index(after, "}"); ci >= 0 {
					for _, part := range strings.Split(after[:ci], ";") {
						p.parseSchemaOption(def, strings.TrimSpace(part))
					}
				}
				p.pos++
				break
			}
		}
	}

	s.SchemaDef = def
}

func (p *Parser) parseSchemaOption(def *SchemaDef, part string) {
	part = strings.TrimSpace(part)
	if strings.HasPrefix(part, "query:") {
		def.Query = strings.TrimSpace(strings.TrimPrefix(part, "query:"))
	} else if strings.HasPrefix(part, "mutation:") {
		def.Mutation = strings.TrimSpace(strings.TrimPrefix(part, "mutation:"))
	} else if strings.HasPrefix(part, "subscription:") {
		def.Subscription = strings.TrimSpace(strings.TrimPrefix(part, "subscription:"))
	}
}

func (p *Parser) parseUnion(s *Schema, line string) {
	name := extractIdent(line, 6)
	t := &Type{Name: name, Kind: TypeKindUnion}
	if idx := strings.Index(line, "="); idx > 0 {
		members := strings.Split(strings.TrimSpace(line[idx+1:]), "|")
		for _, m := range members {
			m = strings.TrimSpace(m)
			if m != "" {
				t.UnionTypes = append(t.UnionTypes, m)
			}
		}
	}
	// skip block
	p.parseFieldsForType(TypeKindUnion)
	s.Types[name] = t
}

func (p *Parser) parseEnum(s *Schema, line string) {
	name := extractIdent(line, 5)
	t := &Type{Name: name, Kind: TypeKindEnum}
	for _, f := range p.parseFieldsForType(TypeKindEnum) {
		ev := EnumValue{Name: f.Name, Deprecated: f.Deprecated, Reason: f.DeprecationReason}
		t.EnumValues = append(t.EnumValues, ev)
	}
	s.Types[name] = t
}

func (p *Parser) parseDirective(s *Schema, line string) {
	rest := line[len("directive @"):]
	name := extractIdent(rest, 0)
	s.Types["@"+name] = &Type{Name: "@" + name, Kind: TypeKindDirective}
	p.parseFieldsForType(TypeKindDirective)
}

func (p *Parser) parseField(line string) *Field {
	if strings.Contains(line, "@deprecated") {
		f := p.parseFieldNoDepr(line)
		if f == nil {
			return nil
		}
		f.Deprecated = true
		if idx := strings.Index(line, "reason:"); idx >= 0 {
			f.DeprecationReason = extractQuotedString(line[idx+len("reason"):])
		}
		return f
	}
	return p.parseFieldNoDepr(line)
}

func (p *Parser) parseFieldNoDepr(line string) *Field {
	// name(args): Type
	if idx := strings.Index(line, "("); idx > 0 {
		closeIdx := strings.LastIndex(line[idx:], ")")
		if closeIdx > 0 {
			name := strings.TrimSpace(line[:idx])
			argsStr := line[idx+1 : idx+closeIdx]
			typePart := strings.TrimSpace(strings.TrimLeft(line[idx+closeIdx+1:], ":"))
			f := &Field{Name: name, TypeName: typePart}
			f.Args = parseArgsString(argsStr)
			return f
		}
	}

	// name: Type
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	name := strings.TrimSpace(parts[0])
	typeName := strings.TrimSpace(parts[1])
	if name == "" || typeName == "" {
		return nil
	}
	return &Field{Name: name, TypeName: typeName}
}

func extractQuotedString(s string) string {
	s = strings.TrimSpace(s)
	// Handle (reason: "value") format
	if strings.HasPrefix(s, "(") {
		if ri := strings.Index(s, "reason:"); ri >= 0 {
			s = strings.TrimSpace(s[ri+len("reason:"):])
		}
	}
	if len(s) == 0 {
		return ""
	}
	if s[0] == '"' {
		for i := 1; i < len(s); i++ {
			if s[i] == '"' && s[i-1] != '\\' {
				return s[1:i]
			}
		}
		return s[1:]
	}
	if s[0] == '`' {
		for i := 1; i < len(s); i++ {
			if s[i] == '`' {
				return s[1:i]
			}
		}
		return s[1:]
	}
	// Unquoted - single word
	words := strings.Fields(s)
	if len(words) > 0 {
		return strings.Trim(words[0], `"'`)
	}
	return ""
}

func extractIdent(s string, skip int) string {
	s = strings.TrimSpace(s[skip:])
	// Remove leading {
	for len(s) > 0 && (s[0] == '{' || s[0] == '}') {
		s = strings.TrimSpace(s[1:])
	}
	end := 0
	for end < len(s) && !isDelim(s[end]) {
		end++
	}
	if end > 0 {
		return s[:end]
	}
	return s
}

func isDelim(ch byte) bool {
	return ch == '{' || ch == '}' || ch == '(' || ch == ')' || ch == ' ' || ch == ':' || ch == '=' || ch == '|'
}

// Argument helpers
func parseArgsString(s string) []Argument {
	var args []Argument
	for _, p := range splitComma(s) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		defVal := ""
		if idx := strings.Index(p, "="); idx > 0 {
			defVal = strings.TrimSpace(p[idx+1:])
			p = p[:idx]
		}
		ci := strings.Index(p, ":")
		if ci > 0 {
			name := strings.TrimSpace(p[:ci])
			typ := strings.TrimSpace(p[ci+1:])
			if name != "" && typ != "" {
				args = append(args, Argument{Name: name, TypeName: typ, Default: defVal})
			}
		}
	}
	return args
}

func splitComma(s string) []string {
	var parts []string
	cur := strings.Builder{}
	depth := 0
	for _, ch := range s {
		switch ch {
		case '[', '(', '{':
			depth++
			cur.WriteRune(ch)
		case ']', ')', '}':
			depth--
			cur.WriteRune(ch)
		case ',':
			if depth == 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			} else {
				cur.WriteRune(ch)
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

// GetType returns a type by name.
func (s *Schema) GetType(name string) *Type {
	return s.Types[name]
}

// HasOperation checks if the schema has a specific operation kind.
func (s *Schema) HasOperation(op string) bool {
	switch op {
	case "query":
		return len(s.Queries) > 0 || (s.SchemaDef != nil && s.SchemaDef.Query != "")
	case "mutation":
		return len(s.Mutations) > 0 || (s.SchemaDef != nil && s.SchemaDef.Mutation != "")
	case "subscription":
		return len(s.Subscriptions) > 0 || (s.SchemaDef != nil && s.SchemaDef.Subscription != "")
	}
	return false
}

// TypeNames returns all type names.
func (s *Schema) TypeNames() []string {
	names := make([]string, 0, len(s.Types))
	for name := range s.Types {
		names = append(names, name)
	}
	return names
}

// DeprecatedFields returns all deprecated fields.
func (s *Schema) DeprecatedFields() []Field {
	var fields []Field
	for _, t := range s.Types {
		for _, f := range t.Fields {
			if f.Deprecated {
				fields = append(fields, *f)
			}
		}
	}
	return fields
}

// IsTypeUsed checks if a type is referenced by any other type.
func (s *Schema) IsTypeUsed(typeName string) bool {
	for _, t := range s.Types {
		if t.Name == typeName {
			continue
		}
		for _, f := range t.Fields {
			if f.TypeName == typeName || strings.Contains(f.TypeName, typeName) {
				return true
			}
		}
		if t.Kind == TypeKindUnion {
			for _, ut := range t.UnionTypes {
				if ut == typeName {
					return true
				}
			}
		}
	}
	return false
}