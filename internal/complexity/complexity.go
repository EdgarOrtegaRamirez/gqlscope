// Package complexity provides query complexity scoring.
package complexity

import (
	"strings"
)

// ComplexityConfig holds configuration for complexity scoring.
type ComplexityConfig struct {
	DefaultFieldCost int     // Default cost for each field
	ListMultiplier   float64 // Multiplier for list fields
	ArgMultiplier    float64 // Multiplier per argument
	DepthWeight      float64 // Weight for nesting depth
	MaxDepth         int     // Maximum allowed depth (0 = unlimited)
}

// ComplexityResult holds the result of complexity analysis.
type ComplexityResult struct {
	TotalScore   float64
	MaxDepth     int
	FieldCount   int
	ListFields   int
	ArgCount     int
	DeepFields   []FieldScore // Fields at or beyond max depth
}

// FieldScore represents the complexity of a single field.
type FieldScore struct {
	Path  string
	Score float64
	Depth int
}

// Score analyzes a query string and returns its complexity.
func Score(query string, config ComplexityConfig) (*ComplexityResult, error) {
	result := &ComplexityResult{}

	lines := strings.Split(query, "\n")
	currentPath := []string{}
	currentDepth := 0
	currentScore := 0.0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comments, aliases, fragments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip fragment definitions
		if strings.HasPrefix(line, "fragment") {
			continue
		}

		// Count braces for depth tracking
		for _, ch := range line {
			if ch == '{' {
				currentDepth++
				result.MaxDepth = max(result.MaxDepth, currentDepth)
			}
			if ch == '}' {
				currentDepth--
				// Pop the current field from path
				if len(currentPath) > 0 {
					currentPath = currentPath[:len(currentPath)-1]
				}
			}
		}

		// Extract field names (not within braces)
		fields := extractFieldNames(line)
		for _, field := range fields {
			if field == "..." || field == "on" || field == "fragment" {
				continue
			}

			path := strings.Join(currentPath, ".")
			if path != "" {
				path += "."
			}
			fieldPath := path + field

			// Calculate field score
			fieldScore := float64(config.DefaultFieldCost)

			// Add depth weight
			fieldScore += float64(currentDepth) * config.DepthWeight

			// Check if field is a list (has [ ])
			if strings.Contains(line, "["+field) {
				fieldScore *= config.ListMultiplier
				result.ListFields++
			}

			// Check for arguments
			if idx := strings.Index(line, "("); idx > 0 && strings.Index(line[idx:], field) < 0 {
				// Arguments exist on this line
				if parenIdx := strings.Index(line, "("); parenIdx > 0 {
					parenEnd := strings.LastIndex(line, ")")
					if parenEnd > parenIdx {
						args := line[parenIdx+1 : parenEnd]
						if args != "" && args != field {
							argCount := strings.Count(args, ":") + 1
							fieldScore *= config.ArgMultiplier * float64(argCount)
							result.ArgCount += argCount
						}
					}
				}
			}

			currentScore += fieldScore
			result.FieldCount++

			if config.MaxDepth > 0 && currentDepth >= config.MaxDepth {
				result.DeepFields = append(result.DeepFields, FieldScore{
					Path:  fieldPath,
					Score: fieldScore,
					Depth: currentDepth,
				})
			}

			currentPath = append(currentPath, field)
		}
	}

	result.TotalScore = currentScore
	return result, nil
}

func extractFieldNames(line string) []string {
	// Remove strings and comments
	line = strings.TrimRight(line, "{} ")

	var names []string
	// Split by whitespace and punctuation
	for _, part := range splitToken(line) {
		part = strings.Trim(part, "()[]{}:,")
		if part != "" && part != "on" && part != "..." && part != "query" && part != "mutation" && part != "subscription" {
			names = append(names, part)
		}
	}
	return names
}

func splitToken(s string) []string {
	var parts []string
	var current string
	for _, ch := range s {
		if ch == '(' || ch == ')' || ch == '[' || ch == ']' || ch == '{' || ch == '}' || ch == ',' || ch == ':' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}