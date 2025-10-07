package main

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	golang "github.com/benstrobel/sqlc-gen-go/internal"
	"github.com/sqlc-dev/plugin-sdk-go/codegen"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func CheckTypesThenGenerate(ctx context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	enumValues := make(map[string][]string)
	for _, s := range req.Catalog.Schemas {
		for _, e := range s.Enums {
			enumValues[e.Name] = e.Vals
		}
	}

	re := regexp.MustCompile(`safe\.enum\(\s*'([\w_-]+)+'\s*,\s*'([\w_-]+)'\s*\)`)
	query_errors := []error{}
	for _, q := range req.Queries {
		result := re.ReplaceAllStringFunc(q.Text, func(match string) string {
			submatches := re.FindStringSubmatch(match)
			if len(submatches) < 3 {
				query_errors = append(query_errors, fmt.Errorf("failed to parse safe.enum call: %s", match))
				return match
			}
			enumType := submatches[1]
			enumValue := submatches[2]
			if vals, ok := enumValues[enumType]; ok {
				if slices.Contains(vals, enumValue) {
					return fmt.Sprintf("'%s'", enumValue)
				}
				query_errors = append(query_errors, fmt.Errorf("enum '%s' does not contain the following value: '%s'", enumType, enumValue))
				return match
			}
			query_errors = append(query_errors, fmt.Errorf("unknown enum type: '%s'", enumType))
			return match
		})
		q.Text = result
	}

	if len(query_errors) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("\n")
		for _, err := range query_errors {
			errorMsg.WriteString(fmt.Sprintf(" - %s\n", err))
		}

		return nil, errors.New(errorMsg.String())
	}

	return golang.Generate(ctx, req)
}

func main() {
	codegen.Run(CheckTypesThenGenerate)
}
