package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/EdgarOrtegaRamirez/gqlscope/internal/analyzer"
	"github.com/EdgarOrtegaRamirez/gqlscope/internal/complexity"
	"github.com/EdgarOrtegaRamirez/gqlscope/internal/parser"
	"github.com/EdgarOrtegaRamirez/gqlscope/internal/reporter"

	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	buildDate = "unknown"
	format    = "text"
	output    string

	// Complexity flags
	defaultFieldCost int
	listMultiplier   float64
	argMultiplier    float64
	depthWeight      float64
	maxDepth         int
)

func deepFieldsToStrings(fields []complexity.FieldScore) []string {
	var result []string
	for _, f := range fields {
		result = append(result, fmt.Sprintf("%s (depth %d, score %.0f)", f.Path, f.Depth, f.Score))
	}
	return result
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "gqlscope",
		Short: "GraphQL schema and query analysis CLI",
		Long: `Gqlscope is a CLI tool for analyzing GraphQL schemas and queries.
It parses GraphQL SDL, validates schema structure, scores query complexity,
identifies deprecated fields, finds unused types, and detects common issues.`,
	}

	// Schema analyze command
	var schemaCmd = &cobra.Command{
		Use:   "analyze [schema_file]",
		Short: "Analyze a GraphQL schema file",
		Long: `Parse and analyze a GraphQL SDL schema file. Reports type count,
field count, deprecated items, unused types, and common issues.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			s, err := parser.Parse(string(content))
			if err != nil {
				return fmt.Errorf("parse schema: %w", err)
			}

			info := analyzer.Analyze(s)
			warnings := analyzer.Validate(s)

			result := reporter.FormatSchemaInfo(info, reporter.Format(format))
			if output != "" {
				os.WriteFile(output, []byte(result+"\n"), 0644)
				return nil
			}

			fmt.Print(result)

			if len(warnings) > 0 && format == "text" {
				fmt.Println("\nWarnings:")
				for _, w := range warnings {
					fmt.Printf("  ! %s\n", w)
				}
			}

			fmt.Println()
			return nil
		},
	}

	// Schema inspect command (more detailed)
	var inspectCmd = &cobra.Command{
		Use:   "inspect [schema_file]",
		Short: "Inspect schema types and fields in detail",
		Long:  `Print detailed information about every type, field, and argument in a GraphQL schema.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			s, err := parser.Parse(string(content))
			if err != nil {
				return fmt.Errorf("parse schema: %w", err)
			}

			info := analyzer.Analyze(s)
			result := reporter.FormatSchemaInfo(info, reporter.Format(format))
			if result != "" {
				fmt.Println(result)
			}

			// Show type details
			for name, t := range s.Types {
				if t.Kind == parser.TypeKindScalar {
					continue
				}
				fmt.Printf("\n%s (%s)\n", name, t.Kind)
				if t.Description != "" {
					fmt.Printf("  %s\n", strings.TrimSpace(t.Description))
				}
				for _, f := range t.Fields {
					deprecated := ""
					if f.Deprecated {
						deprecated = " [deprecated]"
						if f.DeprecationReason != "" {
							deprecated += fmt.Sprintf(" (%s)", f.DeprecationReason)
						}
					}
					argsStr := ""
					if len(f.Args) > 0 {
						var argNames []string
						for _, a := range f.Args {
							argNames = append(argNames, a.Name)
						}
						argsStr = "(" + strings.Join(argNames, ", ") + ")"
					}
					fmt.Printf("  - %s%s%s : %s\n", f.Name, argsStr, deprecated, f.TypeName)
				}
				if t.Kind == parser.TypeKindEnum {
					for _, v := range t.EnumValues {
						dep := ""
						if v.Deprecated {
							dep = " [deprecated]"
							if v.Reason != "" {
								dep += fmt.Sprintf(" (%s)", v.Reason)
							}
						}
						fmt.Printf("  . %s%s\n", v.Name, dep)
					}
				}
				if t.Kind == parser.TypeKindUnion {
					fmt.Printf("  Members: %s\n", strings.Join(t.UnionTypes, ", "))
				}
				if len(t.Interfaces) > 0 {
					fmt.Printf("  Implements: %s\n", strings.Join(t.Interfaces, ", "))
				}
			}

			return nil
		},
	}

	// Complexity command
	var scoreCmd = &cobra.Command{
		Use:   "score [query_file]",
		Short: "Score the complexity of a GraphQL query",
		Long:  `Analyze and score the complexity of a GraphQL query based on field count, nesting depth, list fields, and arguments.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			cfg := complexity.ComplexityConfig{
				DefaultFieldCost: defaultFieldCost,
				ListMultiplier:   listMultiplier,
				ArgMultiplier:    argMultiplier,
				DepthWeight:      depthWeight,
				MaxDepth:         maxDepth,
			}

			result, err := complexity.Score(string(content), cfg)
			if err != nil {
				return fmt.Errorf("score query: %w", err)
			}

			var warnings []string
			if maxDepth > 0 && result.MaxDepth >= maxDepth {
				warnings = append(warnings, fmt.Sprintf("Query depth (%d) meets or exceeds max depth (%d)", result.MaxDepth, maxDepth))
			}

			r := reporter.QueryResult{
				TotalScore: result.TotalScore,
				MaxDepth:   result.MaxDepth,
				FieldCount: result.FieldCount,
				ListFields: result.ListFields,
				ArgCount:   result.ArgCount,
				DeepFields: deepFieldsToStrings(result.DeepFields),
				Warnings:   warnings,
			}

			var out string
			if format == "json" {
				out = reporter.FormatQueryJSON(r)
			} else {
				out = reporter.FormatQueryText(r)
			}

			if output != "" {
				os.WriteFile(output, []byte(out+"\n"), 0644)
				return nil
			}

			fmt.Println(out)
			return nil
		},
	}

	// Validate command
	var validateCmd = &cobra.Command{
		Use:   "validate [schema_file]",
		Short: "Validate a GraphQL schema and report issues",
		Long:  `Check a GraphQL SDL schema for common issues: missing roots, self-referencing types, unused types, deprecated fields without reasons.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			s, err := parser.Parse(string(content))
			if err != nil {
				return fmt.Errorf("parse schema: %w", err)
			}

			warnings := analyzer.Validate(s)

			if output != "" {
				if len(warnings) == 0 {
					os.WriteFile(output, []byte("Schema is valid. No issues found.\n"), 0644)
				} else {
					var buf strings.Builder
					buf.WriteString(fmt.Sprintf("Schema issues (%d):\n", len(warnings)))
					for i, w := range warnings {
						buf.WriteString(fmt.Sprintf("  %d. %s\n", i+1, w))
					}
					os.WriteFile(output, []byte(buf.String()), 0644)
				}
				return nil
			}

			if len(warnings) == 0 {
				fmt.Println("Schema is valid. No issues found.")
				return nil
			}

			fmt.Printf("Schema issues (%d):\n", len(warnings))
			for i, w := range warnings {
				fmt.Printf("  %d. %s\n", i+1, w)
			}
			return fmt.Errorf("schema has issues")
		},
	}

	// Version command
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("gqlscope %s (built %s)\n", version, buildDate)
		},
	}

	// Add flags
	schemaCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json, compact")
	schemaCmd.Flags().StringVarP(&output, "output", "o", "", "Write output to file instead of stdout")

	inspectCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json")
	inspectCmd.Flags().StringVarP(&output, "output", "o", "", "Write output to file instead of stdout")

	scoreCmd.Flags().IntVar(&defaultFieldCost, "default-field-cost", 10, "Base cost per field")
	scoreCmd.Flags().Float64Var(&listMultiplier, "list-multiplier", 2.0, "Multiplier for list fields")
	scoreCmd.Flags().Float64Var(&argMultiplier, "arg-multiplier", 1.5, "Per-argument multiplier")
	scoreCmd.Flags().Float64Var(&depthWeight, "depth-weight", 5.0, "Weight per nesting depth level")
	scoreCmd.Flags().IntVar(&maxDepth, "max-depth", 0, "Maximum allowed depth (0 = unlimited)")
	scoreCmd.Flags().StringVarP(&format, "score-format", "", "text", "Output format: text, json")
	scoreCmd.Flags().StringVarP(&output, "output", "o", "", "Write output to file instead of stdout")

	validateCmd.Flags().StringVarP(&output, "output", "o", "", "Write output to file instead of stdout")

	// Add subcommands
	rootCmd.AddCommand(schemaCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(scoreCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
