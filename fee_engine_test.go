package feecalc

import (
	"testing"

	"github.com/shopspring/decimal"
)

func findAmountByCurrency(items []FeeItem, currency string) decimal.Decimal {
	for _, item := range items {
		if item.Currency == currency {
			return item.Amount
		}
	}
	return decimal.Zero
}

func TestFeeEngine_BasicExecution(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.02,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(amount * rate, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.ProcessedRules != 1 {
		t.Errorf("Expected 1 processed rule, got %d", result.ProcessedRules)
	}

	if len(result.FeeItems) != 1 {
		t.Errorf("Expected 1 fee item, got %d", len(result.FeeItems))
	}

	expectedAmount := decimal.NewFromFloat(20.0)
	if !result.FeeItems[0].Amount.Equal(expectedAmount) {
		t.Errorf("Expected fee amount 20.0, got %s", result.FeeItems[0].Amount.String())
	}

	usdAmount := findAmountByCurrency(result.Summary, "USD")
	if !usdAmount.Equal(expectedAmount) {
		t.Errorf("Expected USD summary 20.0, got %s", usdAmount.String())
	}
}

func TestFeeEngine_NegativeFee(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(100.0, "USD")`)
	engine.AddRule(`$(-20.0, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.FeeItems) != 2 {
		t.Errorf("Expected 2 fee items, got %d", len(result.FeeItems))
	}

	usdAmount := findAmountByCurrency(result.Summary, "USD")
	expectedAmount := decimal.NewFromFloat(80.0)
	if !usdAmount.Equal(expectedAmount) {
		t.Errorf("Expected USD summary 80.0, got %s", usdAmount.String())
	}
}

func TestFeeEngine_InterruptAndContinue(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	for i := 0; i < 5; i++ {
		engine.AddRule(`$(10.0, "USD")`)
	}

	result1, err := engine.ExecuteN(3)
	if err != nil {
		t.Fatalf("ExecuteN failed: %v", err)
	}

	if result1.ProcessedRules != 3 {
		t.Errorf("Expected 3 processed rules, got %d", result1.ProcessedRules)
	}

	if len(result1.FeeItems) != 3 {
		t.Errorf("Expected 3 fee items, got %d", len(result1.FeeItems))
	}

	result2, err := engine.ExecuteN(2)
	if err != nil {
		t.Fatalf("ExecuteN failed: %v", err)
	}

	if result2.ProcessedRules != 2 {
		t.Errorf("Expected 2 processed rules, got %d", result2.ProcessedRules)
	}

	if len(result2.FeeItems) != 5 {
		t.Errorf("Expected 5 fee items, got %d", len(result2.FeeItems))
	}

	usdAmount := findAmountByCurrency(result2.Summary, "USD")
	expectedAmount := decimal.NewFromFloat(50.0)
	if !usdAmount.Equal(expectedAmount) {
		t.Errorf("Expected USD summary 50.0, got %s", usdAmount.String())
	}
}

func TestFeeEngine_MultipleCurrencies(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`[$(100.0, "USD"), $(200.0, "EUR")]`)
	engine.AddRule(`$(50.0, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	usdAmount := findAmountByCurrency(result.Summary, "USD")
	expectedUSDAmount := decimal.NewFromFloat(150.0)
	if !usdAmount.Equal(expectedUSDAmount) {
		t.Errorf("Expected USD summary 150.0, got %s", usdAmount.String())
	}

	eurAmount := findAmountByCurrency(result.Summary, "EUR")
	expectedEURAmount := decimal.NewFromFloat(200.0)
	if !eurAmount.Equal(expectedEURAmount) {
		t.Errorf("Expected EUR summary 200.0, got %s", eurAmount.String())
	}
}

func TestFeeEngine_NilResult(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`nil`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.FeeItems) != 0 {
		t.Errorf("Expected 0 fee items, got %d", len(result.FeeItems))
	}
}

func TestFeeEngine_ContextUpdate(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"value": 10.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`Set("value", value * 2)`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Context == nil {
		t.Fatal("Expected context to be updated")
	}

	newValue, ok := result.Context.Vars["value"]
	if !ok {
		t.Fatal("Expected value to be set in context")
	}

	if newValue.(float64) != 20.0 {
		t.Errorf("Expected value 20.0, got %v", newValue)
	}
}

func TestFeeEngine_MissingVariable(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(amount * rate, "USD")`)

	_, err := engine.Execute()
	if err == nil {
		t.Fatal("Expected error when variable is missing, but got nil")
	}

	if err.Error() == "" {
		t.Fatal("Expected error message, but got empty string")
	}

	t.Logf("Got expected error: %v", err)
}

func TestFeeEngine_NilVariable(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   nil,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(amount * rate, "USD")`)

	_, err := engine.Execute()
	if err == nil {
		t.Fatal("Expected error when variable is nil, but got nil")
	}

	t.Logf("Got expected error: %v", err)
}

func TestFeeEngine_OptionalVariable(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(amount, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.FeeItems) != 1 {
		t.Errorf("Expected 1 fee item, got %d", len(result.FeeItems))
	}

	expectedAmount := decimal.NewFromFloat(1000.0)
	if !result.FeeItems[0].Amount.Equal(expectedAmount) {
		t.Errorf("Expected fee amount 1000.0, got %s", result.FeeItems[0].Amount.String())
	}
}

func TestFeeEngine_ExpressionArray(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`["$(amount * 0.01, \"USD\")", "$(amount * 0.02, \"EUR\")"]`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.FeeItems) != 2 {
		t.Errorf("Expected 2 fee items, got %d", len(result.FeeItems))
	}

	usdAmount := findAmountByCurrency(result.Summary, "USD")
	expectedUSDAmount := decimal.NewFromFloat(10.0)
	if !usdAmount.Equal(expectedUSDAmount) {
		t.Errorf("Expected USD summary 10.0, got %s", usdAmount.String())
	}

	eurAmount := findAmountByCurrency(result.Summary, "EUR")
	expectedEURAmount := decimal.NewFromFloat(20.0)
	if !eurAmount.Equal(expectedEURAmount) {
		t.Errorf("Expected EUR summary 20.0, got %s", eurAmount.String())
	}
}

func TestFeeEngine_SideEffect(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"counter": 0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`Set("counter", counter + 1)`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.FeeItems) != 0 {
		t.Errorf("Expected 0 fee items, got %d", len(result.FeeItems))
	}

	if result.Context == nil {
		t.Fatal("Expected context to be updated")
	}

	newCounter, ok := result.Context.Vars["counter"]
	if !ok {
		t.Fatal("Expected counter to be set in context")
	}

	if newCounter.(int) != 1 {
		t.Errorf("Expected counter 1, got %v", newCounter)
	}
}

func TestFeeEngine_AssignmentSyntax(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"value": 10.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	// Test assignment syntax: amount = 123
	engine.AddRule(`value = value * 2`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Context == nil {
		t.Fatal("Expected context to be updated")
	}

	newValue, ok := result.Context.Vars["value"]
	if !ok {
		t.Fatal("Expected value to be set in context")
	}

	if newValue.(float64) != 20.0 {
		t.Errorf("Expected value 20.0, got %v", newValue)
	}
}

func TestFeeEngine_AssignmentWithFeeItem(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.02,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	// Test assignment followed by fee item creation
	engine.AddRule(`amount = amount * 2; $(amount * rate, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check context update
	newAmount, ok := result.Context.Vars["amount"]
	if !ok {
		t.Fatal("Expected amount to be set in context")
	}
	if newAmount.(float64) != 2000.0 {
		t.Errorf("Expected amount 2000.0, got %v", newAmount)
	}

	// Check fee item (2000 * 0.02 = 40)
	if len(result.FeeItems) != 1 {
		t.Errorf("Expected 1 fee item, got %d", len(result.FeeItems))
	}
	expectedAmount := decimal.NewFromFloat(40.0)
	if !result.FeeItems[0].Amount.Equal(expectedAmount) {
		t.Errorf("Expected fee amount 40.0, got %s", result.FeeItems[0].Amount.String())
	}
}

func TestFeeEngine_MultipleAssignments(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.02,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	// Test multiple assignments
	engine.AddRule(`amount = amount * 2; rate = rate * 1.5`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Context == nil {
		t.Fatal("Expected context to be updated")
	}

	newAmount, ok := result.Context.Vars["amount"]
	if !ok {
		t.Fatal("Expected amount to be set in context")
	}
	if newAmount.(float64) != 2000.0 {
		t.Errorf("Expected amount 2000.0, got %v", newAmount)
	}

	newRate, ok := result.Context.Vars["rate"]
	if !ok {
		t.Fatal("Expected rate to be set in context")
	}
	if newRate.(float64) != 0.03 {
		t.Errorf("Expected rate 0.03, got %v", newRate)
	}
}

func TestFeeEngine_EnableLog(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.02,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx).EnableLog()

	engine.AddRule(`$(amount * rate, "USD")`)
	engine.AddRule(`Set("amount", amount * 2)`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.Logs) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(result.Logs))
	}

	if result.Logs[0].Rule != `$(amount * rate, "USD")` {
		t.Errorf("Expected first log rule to match, got %s", result.Logs[0].Rule)
	}

	if len(result.Logs[0].FeeItems) != 1 {
		t.Errorf("Expected 1 fee item in first log, got %d", len(result.Logs[0].FeeItems))
	}
}

func TestFeeEngine_EmptyRules(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	_, err := engine.Execute()
	if err == nil {
		t.Fatal("Expected error when no rules, but got nil")
	}

	if err.Error() == "" {
		t.Fatal("Expected error message, but got empty string")
	}
}

func TestFeeEngine_ExecuteN_ZeroCount(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)
	engine.AddRule(`$(100.0, "USD")`)

	_, err := engine.ExecuteN(0)
	if err == nil {
		t.Fatal("Expected error for zero count, but got nil")
	}
}

func TestFeeEngine_ExecuteN_NegativeCount(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)
	engine.AddRule(`$(100.0, "USD")`)

	_, err := engine.ExecuteN(-1)
	if err == nil {
		t.Fatal("Expected error for negative count, but got nil")
	}
}

func TestFeeEngine_ExecuteN_ExceedsRules(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)
	engine.AddRule(`$(10.0, "USD")`)
	engine.AddRule(`$(20.0, "USD")`)

	result, err := engine.ExecuteN(10)
	if err != nil {
		t.Fatalf("ExecuteN failed: %v", err)
	}

	if result.ProcessedRules != 2 {
		t.Errorf("Expected 2 processed rules, got %d", result.ProcessedRules)
	}

	usdAmount := findAmountByCurrency(result.Summary, "USD")
	expectedAmount := decimal.NewFromFloat(30.0)
	if !usdAmount.Equal(expectedAmount) {
		t.Errorf("Expected USD summary 30.0, got %s", usdAmount.String())
	}
}

func TestFeeEngine_ExecuteN_NoMoreRules(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)
	engine.AddRule(`$(10.0, "USD")`)

	result1, err := engine.ExecuteN(1)
	if err != nil {
		t.Fatalf("ExecuteN failed: %v", err)
	}

	if result1.ProcessedRules != 1 {
		t.Errorf("Expected 1 processed rule, got %d", result1.ProcessedRules)
	}

	result2, err := engine.ExecuteN(1)
	if err != nil {
		t.Fatalf("ExecuteN failed: %v", err)
	}

	if result2.ProcessedRules != 0 {
		t.Errorf("Expected 0 processed rules when no more rules, got %d", result2.ProcessedRules)
	}
}

func TestFeeEngine_NilContext(t *testing.T) {
	engine := New(nil)

	if engine.GetContext() == nil {
		t.Fatal("Expected context to be created when nil is passed")
	}

	engine.AddRule(`$(100.0, "USD")`)
	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.ProcessedRules != 1 {
		t.Errorf("Expected 1 processed rule, got %d", result.ProcessedRules)
	}
}

func TestFeeEngine_GetRules(t *testing.T) {
	engine := New(nil)
	engine.AddRule(`$(100.0, "USD")`)
	engine.AddRule(`$(200.0, "EUR")`)

	rules := engine.GetRules()
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	if rules[0] != `$(100.0, "USD")` {
		t.Errorf("Expected first rule to match, got %s", rules[0])
	}

	if rules[1] != `$(200.0, "EUR")` {
		t.Errorf("Expected second rule to match, got %s", rules[1])
	}
}

func TestFeeEngine_GetRuleCount(t *testing.T) {
	engine := New(nil)
	if engine.GetRuleCount() != 0 {
		t.Errorf("Expected 0 rules, got %d", engine.GetRuleCount())
	}

	engine.AddRule(`$(100.0, "USD")`)
	if engine.GetRuleCount() != 1 {
		t.Errorf("Expected 1 rule, got %d", engine.GetRuleCount())
	}

	engine.AddRule(`$(200.0, "EUR")`)
	if engine.GetRuleCount() != 2 {
		t.Errorf("Expected 2 rules, got %d", engine.GetRuleCount())
	}
}

func TestFeeEngine_GetContext(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	retrievedCtx := engine.GetContext()
	if retrievedCtx != ctx {
		t.Error("Expected GetContext to return the same context")
	}

	amount, ok := engine.GetVar("amount")
	if !ok {
		t.Fatal("Expected amount to be in context")
	}
	if amount.(float64) != 1000.0 {
		t.Errorf("Expected amount 1000.0, got %v", amount)
	}
}

func TestFeeEngine_ContextCopy(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: []FeeItem{
			{Amount: decimal.NewFromFloat(100.0), Currency: "USD"},
		},
		Logs: []Log{
			{Rule: "test", Vars: make(map[string]interface{})},
		},
	}

	copy := ctx.Copy()
	if copy == ctx {
		t.Fatal("Expected copy to be a different instance")
	}

	if copy.Vars["amount"].(float64) != 1000.0 {
		t.Errorf("Expected amount 1000.0, got %v", copy.Vars["amount"])
	}

	if len(copy.FeeItems) != 1 {
		t.Errorf("Expected 1 fee item, got %d", len(copy.FeeItems))
	}

	if len(copy.Logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(copy.Logs))
	}

	engine := New(ctx)
	engine.SetVar("amount", 2000.0)
	if copy.Vars["amount"].(float64) != 1000.0 {
		t.Error("Expected copy to be independent of original")
	}
}

func TestFeeEngine_ContextSetVar(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.SetVar("test", 123)
	val, ok := engine.GetVar("test")
	if !ok {
		t.Fatal("Expected test variable to be set")
	}
	if val.(int) != 123 {
		t.Errorf("Expected test value 123, got %v", val)
	}
}

func TestFeeEngine_IntAmount(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(100, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expectedAmount := decimal.NewFromInt(100)
	if !result.FeeItems[0].Amount.Equal(expectedAmount) {
		t.Errorf("Expected fee amount 100, got %s", result.FeeItems[0].Amount.String())
	}
}

func TestFeeEngine_StringAmount(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$("123.45", "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expectedAmount := decimal.NewFromFloat(123.45)
	if !result.FeeItems[0].Amount.Equal(expectedAmount) {
		t.Errorf("Expected fee amount 123.45, got %s", result.FeeItems[0].Amount.String())
	}
}

func TestFeeEngine_EmptyStringExpression(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(``)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.FeeItems) != 0 {
		t.Errorf("Expected 0 fee items, got %d", len(result.FeeItems))
	}
}

func TestFeeEngine_MultipleAddRuleCalls(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(10.0, "USD")`)
	engine.AddRule(`$(20.0, "USD")`)
	engine.AddRule(`$(30.0, "USD")`)

	if engine.GetRuleCount() != 3 {
		t.Errorf("Expected 3 rules, got %d", engine.GetRuleCount())
	}

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.ProcessedRules != 3 {
		t.Errorf("Expected 3 processed rules, got %d", result.ProcessedRules)
	}

	usdAmount := findAmountByCurrency(result.Summary, "USD")
	expectedAmount := decimal.NewFromFloat(60.0)
	if !usdAmount.Equal(expectedAmount) {
		t.Errorf("Expected USD summary 60.0, got %s", usdAmount.String())
	}
}

func TestFeeEngine_AddRuleMultipleArgs(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(10.0, "USD")`, `$(20.0, "USD")`, `$(30.0, "USD")`)

	if engine.GetRuleCount() != 3 {
		t.Errorf("Expected 3 rules, got %d", engine.GetRuleCount())
	}

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.ProcessedRules != 3 {
		t.Errorf("Expected 3 processed rules, got %d", result.ProcessedRules)
	}
}

func TestFeeEngine_DecimalPrecision(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.015,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(amount * rate, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expectedAmount := decimal.NewFromFloat(15.0)
	if !result.FeeItems[0].Amount.Equal(expectedAmount) {
		t.Errorf("Expected fee amount 15.0, got %s", result.FeeItems[0].Amount.String())
	}
}

func TestFeeEngine_DecimalArithmeticFunctions(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"a": 10.0,
			"b": 3.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(Add(a, b), "USD")`)
	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	expected := decimal.NewFromFloat(13.0)
	if !result.FeeItems[0].Amount.Equal(expected) {
		t.Errorf("Expected Add result 13.0, got %s", result.FeeItems[0].Amount.String())
	}

	engine2 := New(&Context{
		Vars: map[string]interface{}{
			"a": 10.0,
			"b": 3.0,
		},
		FeeItems: make([]FeeItem, 0),
	})
	engine2.AddRule(`$(Sub(a, b), "USD")`)
	result2, err := engine2.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	expected2 := decimal.NewFromFloat(7.0)
	if !result2.FeeItems[0].Amount.Equal(expected2) {
		t.Errorf("Expected Sub result 7.0, got %s", result2.FeeItems[0].Amount.String())
	}

	engine3 := New(&Context{
		Vars: map[string]interface{}{
			"a": 10.0,
			"b": 3.0,
		},
		FeeItems: make([]FeeItem, 0),
	})
	engine3.AddRule(`$(Mul(a, b), "USD")`)
	result3, err := engine3.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	expected3 := decimal.NewFromFloat(30.0)
	if !result3.FeeItems[0].Amount.Equal(expected3) {
		t.Errorf("Expected Mul result 30.0, got %s", result3.FeeItems[0].Amount.String())
	}

	engine4 := New(&Context{
		Vars: map[string]interface{}{
			"a": 10.0,
			"b": 2.0,
		},
		FeeItems: make([]FeeItem, 0),
	})
	engine4.AddRule(`$(Div(a, b), "USD")`)
	result4, err := engine4.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	expected4 := decimal.NewFromFloat(5.0)
	if !result4.FeeItems[0].Amount.Equal(expected4) {
		t.Errorf("Expected Div result 5.0, got %s", result4.FeeItems[0].Amount.String())
	}

	engine5 := New(&Context{
		Vars: map[string]interface{}{
			"a": 10.0,
		},
		FeeItems: make([]FeeItem, 0),
	})
	engine5.AddRule(`$(Neg(a), "USD")`)
	result5, err := engine5.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	expected5 := decimal.NewFromFloat(-10.0)
	if !result5.FeeItems[0].Amount.Equal(expected5) {
		t.Errorf("Expected Neg result -10.0, got %s", result5.FeeItems[0].Amount.String())
	}
}

func TestFeeEngine_InvalidExpression(t *testing.T) {
	ctx := &Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`invalid syntax here!!!`)

	_, err := engine.Execute()
	if err == nil {
		t.Fatal("Expected error for invalid expression, but got nil")
	}
}

func TestFeeEngine_ComplexExpression(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate1":  0.01,
			"rate2":  0.02,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(amount * rate1 + amount * rate2, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expectedAmount := decimal.NewFromFloat(30.0)
	if !result.FeeItems[0].Amount.Equal(expectedAmount) {
		t.Errorf("Expected fee amount 30.0, got %s", result.FeeItems[0].Amount.String())
	}
}

func TestFeeEngine_ContextWithExistingFeeItems(t *testing.T) {
	ctx := &Context{
		Vars: make(map[string]interface{}),
		FeeItems: []FeeItem{
			{Amount: decimal.NewFromFloat(50.0), Currency: "USD"},
		},
	}
	engine := New(ctx)

	engine.AddRule(`$(100.0, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result.FeeItems) != 2 {
		t.Errorf("Expected 2 fee items, got %d", len(result.FeeItems))
	}

	usdAmount := findAmountByCurrency(result.Summary, "USD")
	expectedAmount := decimal.NewFromFloat(150.0)
	if !usdAmount.Equal(expectedAmount) {
		t.Errorf("Expected USD summary 150.0, got %s", usdAmount.String())
	}
}

func TestFeeEngine_Reset(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.02,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(amount * rate, "USD")`)
	engine.AddRule(`amount = amount * 2`)
	engine.AddRule(`$(amount * rate, "USD")`)

	// Execute first 2 rules
	result1, err := engine.ExecuteN(2)
	if err != nil {
		t.Fatalf("ExecuteN failed: %v", err)
	}

	if len(result1.FeeItems) != 1 {
		t.Errorf("Expected 1 fee item before reset, got %d", len(result1.FeeItems))
	}

	// Check that amount was modified
	amountAfter, _ := engine.GetVar("amount")
	if amountAfter.(float64) != 2000.0 {
		t.Errorf("Expected amount 2000.0 after execution, got %v", amountAfter)
	}

	// Reset
	engine.Reset()

	// Check that Vars are restored to initial values
	amountAfterReset, _ := engine.GetVar("amount")
	if amountAfterReset.(float64) != 1000.0 {
		t.Errorf("Expected amount 1000.0 after reset, got %v", amountAfterReset)
	}

	rateAfterReset, _ := engine.GetVar("rate")
	if rateAfterReset.(float64) != 0.02 {
		t.Errorf("Expected rate 0.02 after reset, got %v", rateAfterReset)
	}

	// Check that FeeItems are cleared
	if len(ctx.FeeItems) != 0 {
		t.Errorf("Expected 0 fee items after reset, got %d", len(ctx.FeeItems))
	}

	// Check that lastExecutedRule is reset
	if ctx.lastExecutedRule != 0 {
		t.Errorf("Expected lastExecutedRule 0 after reset, got %d", ctx.lastExecutedRule)
	}

	// Check that rules are preserved
	if engine.GetRuleCount() != 3 {
		t.Errorf("Expected 3 rules after reset, got %d", engine.GetRuleCount())
	}

	// Execute again from the beginning
	result2, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed after reset: %v", err)
	}

	if result2.ProcessedRules != 3 {
		t.Errorf("Expected 3 processed rules after reset, got %d", result2.ProcessedRules)
	}

	if len(result2.FeeItems) != 2 {
		t.Errorf("Expected 2 fee items after reset execution, got %d", len(result2.FeeItems))
	}

	// Check that amount is modified again
	amountAfterReexec, _ := engine.GetVar("amount")
	if amountAfterReexec.(float64) != 2000.0 {
		t.Errorf("Expected amount 2000.0 after re-execution, got %v", amountAfterReexec)
	}
}

func TestFeeEngine_ResetWithLogs(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.02,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx).EnableLog()

	engine.AddRule(`$(amount * rate, "USD")`)
	engine.AddRule(`amount = amount * 2`)

	result1, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if len(result1.Logs) != 2 {
		t.Errorf("Expected 2 log entries before reset, got %d", len(result1.Logs))
	}

	// Reset
	engine.Reset()

	// Check that Logs are cleared
	if len(ctx.Logs) != 0 {
		t.Errorf("Expected 0 log entries after reset, got %d", len(ctx.Logs))
	}

	// Execute again
	result2, err := engine.Execute()
	if err != nil {
		t.Fatalf("Execute failed after reset: %v", err)
	}

	if len(result2.Logs) != 2 {
		t.Errorf("Expected 2 log entries after reset execution, got %d", len(result2.Logs))
	}
}

func TestFeeEngine_ResetPreservesRules(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`$(10.0, "USD")`)
	engine.AddRule(`$(20.0, "USD")`)
	engine.AddRule(`$(30.0, "USD")`)

	rulesBefore := engine.GetRules()

	// Execute and reset
	engine.Execute()
	engine.Reset()

	rulesAfter := engine.GetRules()

	if len(rulesBefore) != len(rulesAfter) {
		t.Errorf("Expected %d rules after reset, got %d", len(rulesBefore), len(rulesAfter))
	}

	for i, rule := range rulesBefore {
		if i >= len(rulesAfter) || rulesAfter[i] != rule {
			t.Errorf("Rule at index %d changed after reset: expected %s, got %s", i, rule, rulesAfter[i])
		}
	}
}

func TestFeeEngine_ResetMultipleTimes(t *testing.T) {
	ctx := &Context{
		Vars: map[string]interface{}{
			"counter": 0,
		},
		FeeItems: make([]FeeItem, 0),
	}
	engine := New(ctx)

	engine.AddRule(`counter = counter + 1; $(counter, "USD")`)

	// Execute and reset multiple times
	for i := 0; i < 3; i++ {
		result, err := engine.Execute()
		if err != nil {
			t.Fatalf("Execute failed on iteration %d: %v", i, err)
		}

		if len(result.FeeItems) != 1 {
			t.Errorf("Expected 1 fee item on iteration %d, got %d", i, len(result.FeeItems))
		}

		// Reset
		engine.Reset()

		// Check that counter is reset
		counter, _ := engine.GetVar("counter")
		var counterVal int
		switch v := counter.(type) {
		case int:
			counterVal = v
		case float64:
			counterVal = int(v)
		default:
			t.Fatalf("Unexpected type for counter: %T", counter)
		}
		if counterVal != 0 {
			t.Errorf("Expected counter 0 after reset on iteration %d, got %d", i, counterVal)
		}

		// Check that fee items are cleared
		if len(ctx.FeeItems) != 0 {
			t.Errorf("Expected 0 fee items after reset on iteration %d, got %d", i, len(ctx.FeeItems))
		}
	}
}
