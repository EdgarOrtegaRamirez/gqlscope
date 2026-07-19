package parser

import "testing"

func TestParse_BasicSchema(t *testing.T) {
	schema := `
type Query {
  users: [User!]!
  user(id: ID!): User
}

type User {
  id: ID!
  name: String!
}
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(s.Queries) == 0 {
		t.Error("Expected queries, got none")
	}

	if s.GetType("User") == nil {
		t.Error("Expected User type to exist")
	}
}

func TestParse_Mutation(t *testing.T) {
	schema := `
type Mutation {
  createUser(name: String!): User
}
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(s.Mutations) == 0 {
		t.Error("Expected mutations, got none")
	}
}

func TestParse_Subscription(t *testing.T) {
	schema := `
type Subscription {
  onMessage: Message
}
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(s.Subscriptions) == 0 {
		t.Error("Expected subscriptions, got none")
	}
}

func TestParse_Enum(t *testing.T) {
	schema := `
enum Status {
  ACTIVE
  INACTIVE
  PENDING
}
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	enumType := s.GetType("Status")
	if enumType == nil {
		t.Fatal("Expected Status enum to exist")
	}

	if len(enumType.EnumValues) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(enumType.EnumValues))
	}
}

func TestParse_Union(t *testing.T) {
	schema := `
type User { id: ID! name: String! }
type Post { id: ID! title: String! }
union SearchResult = User | Post
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	unionType := s.GetType("SearchResult")
	if unionType == nil {
		t.Fatal("Expected SearchResult union to exist")
	}

	if len(unionType.UnionTypes) != 2 {
		t.Errorf("Expected 2 union members, got %d", len(unionType.UnionTypes))
	}
}

func TestParse_EnumDeprecation(t *testing.T) {
	schema := `
enum Role {
  ADMIN
  USER
  BANNED @deprecated
}
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	enumType := s.GetType("Role")
	if enumType == nil {
		t.Fatal("Expected Role enum")
	}

	if len(enumType.EnumValues) != 3 {
		t.Errorf("Expected 3 values, got %d", len(enumType.EnumValues))
	}
}

func TestParse_InputType(t *testing.T) {
	schema := `
input CreateUserInput {
  name: String!
  email: String!
}
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	inputType := s.GetType("CreateUserInput")
	if inputType == nil {
		t.Fatal("Expected CreateUserInput type")
	}

	if len(inputType.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(inputType.Fields))
	}
}

func TestParse_Interface(t *testing.T) {
	schema := `
interface Node {
  id: ID!
}

type User implements Node {
  id: ID!
  name: String!
}
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	userType := s.GetType("User")
	if userType == nil {
		t.Fatal("Expected User type")
	}
}

func TestParse_SchemaDef(t *testing.T) {
	schema := `
schema {
  query: Query
  mutation: Mutation
}

type Query {
  hello: String
}

type Mutation {
  add(x: Int): Int
}
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if s.SchemaDef == nil {
		t.Fatal("Expected schema definition")
	}
	if s.SchemaDef.Query != "Query" {
		t.Errorf("Expected Query root, got %s", s.SchemaDef.Query)
	}
	if s.SchemaDef.Mutation != "Mutation" {
		t.Errorf("Expected Mutation root, got %s", s.SchemaDef.Mutation)
	}
}

func TestParse_EmptySchema(t *testing.T) {
	s, err := Parse("")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if s == nil {
		t.Fatal("Expected non-nil schema")
	}
}

func TestParse_FieldArgs(t *testing.T) {
	schema := `
type Query {
  user(id: ID!, name: String, role: UserRole = ADMIN): User
}
`
	s, err := Parse(schema)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(s.Queries) != 1 {
		t.Fatalf("Expected 1 query, got %d", len(s.Queries))
	}

	userField := s.Queries[0]
	if len(userField.Args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(userField.Args))
	}
}