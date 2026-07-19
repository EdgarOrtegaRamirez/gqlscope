# gqlscope — AI Agent Notes

## Project Overview
GraphQL schema and query analysis CLI. Parses GraphQL SDL, analyzes schemas, scores query complexity, and validates schemas.

## Build & Test
```bash
go build ./cmd/gqlscope/
go test ./... -v
go vet ./...
```

## Key Files
- `cmd/gqlscope/main.go` — CLI entry point (cobra)
- `internal/parser/schema.go` — GraphQL SDL parser
- `internal/analyzer/analyzer.go` — Schema analysis engine
- `internal/complexity/complexity.go` — Query complexity scorer
- `internal/reporter/reporter.go` — Output formatting

## Adding New Commands
1. Create a new cobra.Command in main.go
2. Add flags with appropriate types
3. Add the command as a subcommand with `rootCmd.AddCommand()`

## Common Extensions
- Add new parser rules for additional GraphQL constructs
- Extend complexity scoring with new metrics
- Add new output formats in reporter package