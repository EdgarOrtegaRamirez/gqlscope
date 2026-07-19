// Package reporter provides output formatting.
package reporter

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Format specifies the output format.
type Format string

const (
	FormatText    Format = "text"
	FormatJSON    Format = "json"
	FormatCompact Format = "compact"
)

// SchemaInfo holds schema statistics.
type SchemaInfo struct {
	TypeCount       int      `json:"type_count"`
	TypeNames       []string `json:"type_names"`
	DeprecatedCount int      `json:"deprecated_count"`
	DeprecatedFields []string `json:"deprecated_fields"`
	DeprecatedEnums []string `json:"deprecated_enum_values"`
	TotalFields     int      `json:"total_fields"`
	TotalEnums      int      `json:"total_enum_values"`
	HasQuery        bool     `json:"has_query"`
	HasMutation     bool     `json:"has_mutation"`
	HasSubscription bool     `json:"has_subscription"`
	QueryCount      int      `json:"query_count"`
	MutationCount   int      `json:"mutation_count"`
	SubscriptionCount int    `json:"subscription_count"`
	Directives      []string `json:"directives"`
}

// QueryResult holds query analysis results.
type QueryResult struct {
	TotalScore float64  `json:"total_score"`
	MaxDepth   int      `json:"max_depth"`
	FieldCount int      `json:"field_count"`
	ListFields int      `json:"list_fields"`
	ArgCount   int      `json:"arg_count"`
	DeepFields []string `json:"deep_fields"`
	Warnings   []string `json:"warnings"`
}

// FormatSchemaInfo formats schema info according to the specified format.
func FormatSchemaInfo(info SchemaInfo, format Format) string {
	switch format {
	case FormatJSON:
		data, _ := json.MarshalIndent(info, "", "  ")
		return string(data)
	case FormatCompact:
		return fmt.Sprintf("types=%d fields=%d deprecated=%d queries=%d mutations=%d subscriptions=%d",
			info.TypeCount, info.TotalFields, info.DeprecatedCount,
			info.QueryCount, info.MutationCount, info.SubscriptionCount)
	default:
		return formatSchemaText(info)
	}
}

// FormatQueryJSON formats a query result as JSON.
func FormatQueryJSON(result QueryResult) string {
	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data)
}

// FormatQueryText formats a query result as text.
func FormatQueryText(result QueryResult) string {
	var sb strings.Builder

	sb.WriteString("Query Complexity Analysis\n")
	sb.WriteString(strings.Repeat("=", 40) + "\n\n")

	sb.WriteString(fmt.Sprintf("Total Score:    %.0f\n", result.TotalScore))
	sb.WriteString(fmt.Sprintf("Max Depth:      %d\n", result.MaxDepth))
	sb.WriteString(fmt.Sprintf("Fields:         %d\n", result.FieldCount))
	sb.WriteString(fmt.Sprintf("List Fields:    %d\n", result.ListFields))
	sb.WriteString(fmt.Sprintf("Arguments:      %d\n", result.ArgCount))

	if len(result.DeepFields) > 0 {
		sb.WriteString("\nDeep Fields:\n")
		for _, f := range result.DeepFields {
			sb.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}

	if len(result.Warnings) > 0 {
		sb.WriteString("\nWarnings:\n")
		for _, w := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  ! %s\n", w))
		}
	}

	return sb.String()
}

func formatSchemaText(info SchemaInfo) string {
	var sb strings.Builder

	sb.WriteString("Schema Analysis\n")
	sb.WriteString(strings.Repeat("=", 40) + "\n\n")

	sb.WriteString(fmt.Sprintf("Total Types:       %d\n", info.TypeCount))
	sb.WriteString(fmt.Sprintf("Total Fields:      %d\n", info.TotalFields))
	sb.WriteString(fmt.Sprintf("Total Enum Values: %d\n", info.TotalEnums))
	sb.WriteString(fmt.Sprintf("Deprecated:        %d\n", info.DeprecatedCount))
	sb.WriteString(fmt.Sprintf("Directives:        %d\n", len(info.Directives)))

	sb.WriteString("\nOperations:\n")
	sb.WriteString(fmt.Sprintf("  Queries:         %d\n", info.QueryCount))
	sb.WriteString(fmt.Sprintf("  Mutations:       %d\n", info.MutationCount))
	sb.WriteString(fmt.Sprintf("  Subscriptions:   %d\n", info.SubscriptionCount))

	if len(info.TypeNames) > 0 {
		sb.WriteString("\nTypes:\n")
		for _, name := range info.TypeNames {
			sb.WriteString(fmt.Sprintf("  - %s\n", name))
		}
	}

	if len(info.DeprecatedFields) > 0 {
		sb.WriteString("\nDeprecated Fields:\n")
		for _, f := range info.DeprecatedFields {
			sb.WriteString(fmt.Sprintf("  - %s\n", f))
		}
	}

	return sb.String()
}