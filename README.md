# gqlscope

GraphQL schema and query analysis CLI tool. Parse GraphQL SDL, validate schemas, score query complexity, identify deprecated fields, and detect common issues.

## Features

- **Schema Analysis** - Parse GraphQL SDL and extract type information, fields, operations, directives
- **Schema Inspection** - Detailed view of all types, fields, arguments, and enum values
- **Query Complexity Scoring** - Score queries based on field count, nesting depth, list fields, and arguments
- **Schema Validation** - Detect missing roots, self-referencing types, unused types, and deprecated fields without reasons
- **Multiple Output Formats** - Human-readable text, JSON, and compact output

## Installation

```bash
# Go install
go install github.com/EdgarOrtegaRamirez/gqlscope/cmd/gqlscope@latest

# Or build from source
git clone https://github.com/EdgarOrtegaRamirez/gqlscope.git
cd gqlscope
go build -o gqlscope ./cmd/gqlscope/
```

## Usage

### Analyze a Schema

```bash
# Basic analysis
gqlscope analyze schema.graphql

# JSON output for CI/CD
gqlscope analyze schema.graphql --format json

# Compact output
gqlscope analyze schema.graphql --format compact
```

### Inspect Schema in Detail

```bash
gqlscope inspect schema.graphql

# Shows all types, fields, arguments, deprecations
```

### Score Query Complexity

```bash
gqlscope score query.graphql

# Customize scoring thresholds
gqlscope score query.graphql \
  --default-field-cost 5 \
  --depth-weight 10 \
  --list-multiplier 3.0 \
  --arg-multiplier 2.0 \
  --max-depth 10
```

### Validate Schema

```bash
gqlscope validate schema.graphql
```

## Query Complexity Scoring

The complexity scoring system assigns weights to different aspects of a query:

| Factor | Weight | Description |
|--------|--------|-------------|
| Base field cost | 10 (default) | Base score for each field |
| List multiplier | 2.0 (default) | Multiplier for fields returning lists |
| Arg multiplier | 1.5 (default) | Multiplier per argument on a field |
| Depth weight | 5.0 (default) | Additional weight per nesting level |

## Example

Given this schema:

```graphql
type Query {
  users: [User!]!
  user(id: ID!): User
}

type User {
  id: ID!
  name: String!
  email: String!
  posts: [Post]
  friends: [User]
}

type Post {
  id: ID!
  title: String!
  author: User!
}
```

Running `gqlscope analyze schema.graphql`:

```
Schema Analysis
========================================

Total Types:       4
Total Fields:      9
Total Enum Values: 0
Deprecated:        0
Directives:        0

Operations:
  Queries:         2
  Mutations:       0
  Subscriptions:   0

Types:
  - User
  - Post
  - Query
  - String
  - Int
  - Float
  - Boolean
  - ID
```

## Project Structure

```
gqlscope/
├── cmd/gqlscope/main.go    # CLI entry point
├── internal/
│   ├── parser/             # GraphQL SDL parser
│   ├── analyzer/           # Schema analysis engine
│   ├── complexity/         # Query complexity scorer
│   └── reporter/           # Output formatting
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── AGENTS.md
└── .gitignore
```

## License

MIT