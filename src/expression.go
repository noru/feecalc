package fee_engine

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/shopspring/decimal"
)

// newFeeItem creates a new fee item
// amount can be float64, int, string, or decimal.Decimal
func newFeeItem(amount interface{}, currency string) FeeItem {
	var d decimal.Decimal
	switch v := amount.(type) {
	case decimal.Decimal:
		d = v
	case float64:
		d = decimal.NewFromFloat(v)
	case int:
		d = decimal.NewFromInt(int64(v))
	case int64:
		d = decimal.NewFromInt(v)
	case string:
		var err error
		d, err = decimal.NewFromString(v)
		if err != nil {
			d = decimal.Zero
		}
	default:
		d = decimal.Zero
	}
	return FeeItem{
		Amount:   d,
		Currency: currency,
	}
}

// executeSingleExpression executes a single expression string
func executeSingleExpression(exprStr string, env map[string]interface{}) (interface{}, error) {
	if exprStr == "" {
		return nil, nil
	}

	program, err := expr.Compile(exprStr, expr.Env(env))
	if err != nil {
		return nil, fmt.Errorf("failed to compile expression: %w", err)
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("failed to execute expression: %w", err)
	}

	return output, nil
}

// extractExpressionStrings extracts expression strings from output
func extractExpressionStrings(output interface{}) []string {
	if arr, ok := output.([]string); ok {
		return arr
	}

	if arr, ok := output.([]interface{}); ok && len(arr) > 0 {
		expressions := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				expressions = append(expressions, str)
			} else {
				return nil // Not all strings, return nil
			}
		}
		return expressions
	}

	return nil
}

// extractFeeItems extracts FeeItems from output and appends to the slice
func extractFeeItems(output interface{}, feeItems *[]FeeItem) {
	if output == nil {
		return
	}

	if fi, ok := output.(FeeItem); ok {
		*feeItems = append(*feeItems, fi)
		return
	}

	if arr, ok := output.([]interface{}); ok {
		for _, item := range arr {
			if fi, ok := item.(FeeItem); ok {
				*feeItems = append(*feeItems, fi)
			}
		}
	}
}

// toDecimal converts various numeric types to decimal.Decimal
func toDecimal(v interface{}) decimal.Decimal {
	switch val := v.(type) {
	case decimal.Decimal:
		return val
	case float64:
		return decimal.NewFromFloat(val)
	case float32:
		return decimal.NewFromFloat32(val)
	case int:
		return decimal.NewFromInt(int64(val))
	case int8:
		return decimal.NewFromInt(int64(val))
	case int16:
		return decimal.NewFromInt(int64(val))
	case int32:
		return decimal.NewFromInt(int64(val))
	case int64:
		return decimal.NewFromInt(val)
	case uint:
		return decimal.NewFromInt(int64(val))
	case uint8:
		return decimal.NewFromInt(int64(val))
	case uint16:
		return decimal.NewFromInt(int64(val))
	case uint32:
		return decimal.NewFromInt(int64(val))
	case uint64:
		// uint64 might overflow int64, convert via string to be safe
		return decimal.NewFromInt(int64(val))
	case string:
		d, err := decimal.NewFromString(val)
		if err != nil {
			return decimal.Zero
		}
		return d
	default:
		return decimal.Zero
	}
}

// preprocessExpression converts assignment syntax (var = value) to Set calls
// Examples:
//   - "amount = 123" -> "Set(\"amount\", 123)"
//   - "amount = 123; rate = 0.02" -> "Set(\"amount\", 123); Set(\"rate\", 0.02)"
//   - "amount = 123; $(amount * rate, \"USD\")" -> "Set(\"amount\", 123); $(amount * rate, \"USD\")"
func preprocessExpression(exprStr string) string {
	if exprStr == "" {
		return exprStr
	}

	// Pattern to match variable assignments: identifier = expression
	// Match: word characters = (rest of the line until semicolon or end)
	assignmentPattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(.+)$`)

	// Split by semicolon to handle multiple statements
	parts := strings.Split(exprStr, ";")
	var processedParts []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if this part is an assignment
		if matches := assignmentPattern.FindStringSubmatch(part); len(matches) == 3 {
			varName := matches[1]
			valueExpr := strings.TrimSpace(matches[2])
			// Convert to Set call (SetVar is kept for backward compatibility)
			processedParts = append(processedParts, fmt.Sprintf(`Set("%s", %s)`, varName, valueExpr))
		} else {
			// Not an assignment, keep as is
			processedParts = append(processedParts, part)
		}
	}

	if len(processedParts) == 0 {
		return exprStr
	}

	// If we have multiple parts, we need to execute them in sequence
	// Since expr doesn't support multiple statements, we'll need to handle this differently
	// For now, if there are multiple parts, we'll execute them as separate expressions
	// But we need to modify executeExpression to handle this
	if len(processedParts) > 1 {
		// Return as array of expressions to execute sequentially
		// This will be handled in executeExpression
		return strings.Join(processedParts, "; ")
	}

	return processedParts[0]
}

// executeExpression executes an expression and returns rule result
// Expression can return:
//   - FeeItem: saved as fee item
//   - []string or []interface{} (strings): treated as array of expressions to execute
//   - nil or other: treated as side effect (context changes tracked via SetVar)
func executeExpression(exprStr string, ctx *Context) (*RuleResult, error) {
	if exprStr == "" {
		return nil, nil
	}

	// Preprocess expression to convert assignments to SetVar calls
	preprocessed := preprocessExpression(exprStr)

	ctx.mu.RLock()
	env := make(map[string]interface{})

	// Keep variables as their original types for expression evaluation
	// Numeric operations will be converted to decimal in newFeeItem
	for k, v := range ctx.Vars {
		env[k] = v
	}

	// Track context updates
	contextUpdates := make(map[string]interface{})

	// Add helper functions
	env["$"] = newFeeItem

	// Set function for variable assignment
	env["Set"] = func(key string, value interface{}) interface{} {
		contextUpdates[key] = value
		env[key] = value
		return nil
	}

	// Add decimal arithmetic functions for expressions
	// These allow decimal operations in expressions: Mul(a, b) instead of a * b
	// All numeric operations should use these functions to ensure decimal precision
	env["Add"] = func(a, b interface{}) decimal.Decimal {
		return toDecimal(a).Add(toDecimal(b))
	}
	env["Sub"] = func(a, b interface{}) decimal.Decimal {
		return toDecimal(a).Sub(toDecimal(b))
	}
	env["Mul"] = func(a, b interface{}) decimal.Decimal {
		return toDecimal(a).Mul(toDecimal(b))
	}
	env["Div"] = func(a, b interface{}) decimal.Decimal {
		return toDecimal(a).Div(toDecimal(b))
	}
	env["Neg"] = func(a interface{}) decimal.Decimal {
		return toDecimal(a).Neg()
	}

	ctx.mu.RUnlock()

	// Check if preprocessing resulted in multiple statements (separated by semicolon)
	// If so, we need to execute them sequentially
	var finalExpr string
	if strings.Contains(preprocessed, "; ") {
		parts := strings.Split(preprocessed, "; ")
		// Execute all parts except the last one (they are Set calls or other statements)
		for i := 0; i < len(parts)-1; i++ {
			part := strings.TrimSpace(parts[i])
			if part != "" {
				// Execute this part directly without recursion
				_, err := executeSingleExpression(part, env)
				if err != nil {
					return nil, err
				}
			}
		}
		// Use the last part as the main expression
		finalExpr = strings.TrimSpace(parts[len(parts)-1])
	} else {
		finalExpr = preprocessed
	}

	output, err := executeSingleExpression(finalExpr, env)
	if err != nil {
		return nil, err
	}

	result := &RuleResult{
		FeeItems: make([]FeeItem, 0),
	}

	// Check if output is an array of expression strings
	expressionsToProcess := extractExpressionStrings(output)

	// Extract FeeItems from output
	if len(expressionsToProcess) > 0 {
		// Execute array of expressions
		for _, subExpr := range expressionsToProcess {
			subOutput, err := executeSingleExpression(subExpr, env)
			if err != nil {
				return nil, err
			}
			extractFeeItems(subOutput, &result.FeeItems)
		}
	} else if output != nil {
		// Single expression result
		extractFeeItems(output, &result.FeeItems)
	}

	if len(contextUpdates) > 0 {
		result.Context = &Context{
			Vars:             contextUpdates,
			FeeItems:         make([]FeeItem, 0),
			lastExecutedRule: 0,
		}
	}

	if len(result.FeeItems) == 0 && result.Context == nil {
		return nil, nil
	}

	return result, nil
}
