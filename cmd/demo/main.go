package main

import (
	"fmt"
	"log"

	fee_engine "github.com/noru/fee-engine"
	"github.com/shopspring/decimal"
)

func main() {
	fmt.Println("=== Fee Engine Demo ===")
	fmt.Println()

	// Demo 1: Basic fee calculation
	fmt.Println("Demo 1: Basic Fee Calculation")
	basic()

	// Demo 2: Multiple rules with negative fees
	fmt.Println("\nDemo 2: Multiple Rules with Negative Fees")
	negative()

	// Demo 3: Context updates and assignment syntax
	fmt.Println("\nDemo 3: Context Updates with Assignment Syntax")
	contextUpdate()

	// Demo 4: Multiple currencies
	fmt.Println("\nDemo 4: Multiple Currencies")
	multiCurrencies()

	// Demo 5: Interrupt and continue execution
	fmt.Println("\nDemo 5: Interrupt and Continue Execution")
	resumable()

	// Demo 6: Expression arrays
	fmt.Println("\nDemo 6: Expression Arrays")
	exprArray()

	// Demo 7: Decimal precision
	fmt.Println("\nDemo 7: Decimal Precision")
	decimalPrecision()

	// Demo 8: Assignment with fee calculation
	fmt.Println("\nDemo 8: Assignment with Fee Calculation")
	assignment()

	// Demo 9: Execution trace with multiple rules
	fmt.Println("\nDemo 9: Execution Trace with Multiple Rules")
	executionTrace()

	// Demo 10: OnRamp
	fmt.Println("\nDemo 10: OnRamp")
	OnRamp()
}

func basic() {
	ctx := &fee_engine.Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.02,
		},
		FeeItems: make([]fee_engine.FeeItem, 0),
	}
	engine := fee_engine.New(ctx)

	engine.AddRule(`$(amount * rate, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Amount: %.2f, Rate: %.2f%%\n", ctx.Vars["amount"].(float64), ctx.Vars["rate"].(float64)*100)
	fmt.Printf("  Processed Rules: %d\n", result.ProcessedRules)
	fmt.Printf("  Fee: %s %s\n", result.Summary[0].Amount.String(), result.Summary[0].Currency)
}

func negative() {
	ctx := &fee_engine.Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]fee_engine.FeeItem, 0),
	}
	engine := fee_engine.New(ctx)

	engine.AddRule(`$(100.0, "USD")`) // Base fee
	engine.AddRule(`$(-20.0, "USD")`) // Discount
	engine.AddRule(`$(10.0, "USD")`)  // Additional fee

	result, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Base Fee: $100.00\n")
	fmt.Printf("  Discount: -$20.00\n")
	fmt.Printf("  Additional Fee: $10.00\n")
	fmt.Printf("  Total: %s %s\n", result.Summary[0].Amount.String(), result.Summary[0].Currency)
}

func contextUpdate() {
	ctx := &fee_engine.Context{
		Vars: map[string]interface{}{
			"value": 10.0,
		},
		FeeItems: make([]fee_engine.FeeItem, 0),
	}
	engine := fee_engine.New(ctx)

	// Using assignment syntax: value = value * 2
	engine.AddRule(`value = value * 2`)

	result, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	newValue, _ := result.Context.GetVar("value")
	fmt.Printf("  Original Value: 10.0\n")
	fmt.Printf("  Updated Value: %.1f\n", newValue.(float64))
}

func multiCurrencies() {
	ctx := &fee_engine.Context{
		Vars:     make(map[string]interface{}),
		FeeItems: make([]fee_engine.FeeItem, 0),
	}
	engine := fee_engine.New(ctx)

	engine.AddRule(`[$(100.0, "USD"), $(200.0, "EUR")]`)
	engine.AddRule(`$(50.0, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Fee Items: %d\n", len(result.FeeItems))
	for _, item := range result.Summary {
		fmt.Printf("  %s: %s\n", item.Currency, item.Amount.String())
	}
}

func resumable() {
	ctx := &fee_engine.Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]fee_engine.FeeItem, 0),
	}
	engine := fee_engine.New(ctx)

	// Add 5 rules
	for i := 0; i < 5; i++ {
		engine.AddRule(`$(10.0, "USD")`)
	}

	// Execute first 3 rules
	fmt.Println("  Executing first 3 rules...")
	result1, err := engine.ExecuteN(3)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Processed: %d rules, Total Fee: %s USD\n", result1.ProcessedRules, result1.Summary[0].Amount.String())

	// Continue with remaining rules
	fmt.Println("  Continuing with remaining rules...")
	result2, err := engine.ExecuteN(2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Processed: %d more rules, Total Fee: %s USD\n", result2.ProcessedRules, result2.Summary[0].Amount.String())
}

func exprArray() {
	ctx := &fee_engine.Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
		},
		FeeItems: make([]fee_engine.FeeItem, 0),
	}
	engine := fee_engine.New(ctx)

	// Expression array: returns array of expression strings to execute
	engine.AddRule(`[$(amount * 0.01, "USD"), $(amount * 0.02, "EUR")]`)

	result, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Amount: %.2f\n", ctx.Vars["amount"].(float64))
	fmt.Printf("  Fee Items: %d\n", len(result.FeeItems))
	for _, item := range result.Summary {
		fmt.Printf("  %s: %s\n", item.Currency, item.Amount.String())
	}
}

func decimalPrecision() {
	ctx := &fee_engine.Context{
		Vars: map[string]interface{}{
			"amount": 100.1,
			"rate":   0.015,
		},
		FeeItems: make([]fee_engine.FeeItem, 0),
	}
	engine := fee_engine.New(ctx)

	// Using decimal functions for precision
	engine.AddRule(`$(Mul(amount, rate), "USD")`)

	result, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Amount: %.2f, Rate: %.3f%%\n", ctx.Vars["amount"].(float64), ctx.Vars["rate"].(float64)*100)
	fmt.Printf("  Fee (using decimal): %s %s\n", result.Summary[0].Amount.String(), result.Summary[0].Currency)

	// Compare with float64 calculation
	floatResult := 100.1 * 0.015
	fmt.Printf("  Fee (using float64): %.10f USD\n", floatResult)
	fmt.Printf("  Difference: %s\n", decimal.NewFromFloat(floatResult).Sub(result.Summary[0].Amount).Abs().String())
}

func assignment() {
	ctx := &fee_engine.Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.02,
		},
		FeeItems: make([]fee_engine.FeeItem, 0),
	}
	engine := fee_engine.New(ctx)

	// Update amount and calculate fee in one rule
	engine.AddRule(`amount = amount * 2; $(amount * rate, "USD")`)

	result, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	newAmount, _ := result.Context.GetVar("amount")
	fmt.Printf("  Original Amount: 1000.0\n")
	fmt.Printf("  Updated Amount: %.1f\n", newAmount.(float64))
	fmt.Printf("  Fee: %s %s\n", result.Summary[0].Amount.String(), result.Summary[0].Currency)
}

func executionTrace() {
	ctx := &fee_engine.Context{
		Vars: map[string]interface{}{
			"amount": 1000.0,
			"rate":   0.02,
		},
		FeeItems: make([]fee_engine.FeeItem, 0),
		Logs:     make([]fee_engine.Log, 0),
	}
	engine := fee_engine.New(ctx).EnableLog()

	// Add multiple rules with different operations
	rules := []string{
		`$(amount * rate, "USD")`,          // Rule 1: Calculate base fee
		`amount = amount * 1.1`,            // Rule 2: Increase amount by 10%
		`$(amount * rate, "USD")`,          // Rule 3: Calculate fee with new amount
		`$(-10.0, "USD")`,                  // Rule 4: Apply discount
		`rate = rate * 1.5`,                // Rule 5: Increase rate
		`$(amount * rate, "USD")`,          // Rule 6: Calculate final fee
		`[$(50.0, "EUR"), $(30.0, "EUR")]`, // Rule 7: Add EUR fees
	}

	engine.AddRule(rules...)

	fmt.Println("  Rules to execute:")
	for i, rule := range rules {
		fmt.Printf("    Rule %d: %s\n", i+1, rule)
	}
	fmt.Println()

	result, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("  Summary:")
	for _, item := range result.Summary {
		fmt.Printf("      %s: %s\n", item.Currency, item.Amount.String())
	}

	fmt.Printf("\n  Total Fee Items: %d\n", len(result.FeeItems))
	for _, item := range result.FeeItems {
		fmt.Printf("    %s: %s\n", item.Currency, item.Amount.String())
	}

	// Show logs
	fmt.Printf("\n  Execution Logs of %d:\n", len(result.Logs))
	for i, logEntry := range result.Logs {
		fmt.Printf("    [%d] Rule: %s\n", i+1, logEntry.Rule)
		fmt.Printf("        Vars: %v\n", logEntry.Vars)
		if len(logEntry.FeeItems) > 0 {
			fmt.Printf("        FeeItems: ")
			for j, item := range logEntry.FeeItems {
				if j > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s %s", item.Amount.String(), item.Currency)
			}
			fmt.Println()
		} else {
			fmt.Printf("        FeeItems: (none)\n")
		}
	}
}

func OnRamp() {
	ctx := &fee_engine.Context{
		Vars: map[string]interface{}{
			"amount":             5828.0,
			"fiat_currency":      "KES",
			"crypto_currency":    "USDT",
			"network_fee":        0.27,
			"kes_to_usd_rate":    0.01,
			"crypto_to_usd_rate": 0.99231,
			"fiat_fee_rate":      0.01,
			"fiat_fee_fixed":     100.0,
			"wello_fee_rate":     0.01,
			"wello_fee_fixed":    200.0,
			"merchant_fee_rate":  0.01,
			"merchant_fee_fixed": 300.0,
			"coupon":             200.0,
			"coupon_currency":    "KES",
			// intermediate variables
			"fiat_fee":     0.0,
			"wello_fee":    0.0,
			"merchant_fee": 0.0,
			"total_fee":    0.0,
			"fee_in_usd":   0.0,
		},
	}
	engine := fee_engine.New(ctx).EnableLog()

	result, err := engine.AddRule(
		`network_fee = network_fee * crypto_to_usd_rate / kes_to_usd_rate; $(network_fee, fiat_currency)`, // calculate the network fee in KES
		`amount = amount + network_fee`, // add the network fee to the base amount
		`fiat_fee = amount * fiat_fee_rate + fiat_fee_fixed; $(fiat_fee, fiat_currency)`,                 // fiat fee
		`wello_fee = amount * wello_fee_rate + wello_fee_fixed; $(wello_fee, fiat_currency)`,             // wello fee
		`merchant_fee = amount * merchant_fee_rate + merchant_fee_fixed; $(merchant_fee, fiat_currency)`, // merchant fee
		`total_fee = fiat_fee + wello_fee + merchant_fee + network_fee`,                                  // total fee in KES
		`total_fee = total_fee - coupon; coupon > 0 ? $(-coupon, coupon_currency) : nil`,                 // apply coupon if it is greater than 0
		`fee_in_usd = total_fee * kes_to_usd_rate`,                                                       // total fee in USD
		`[$(-total_fee, fiat_currency), $(fee_in_usd, "USD")]`,                                           // return the total fee in USD and KES
	).Execute()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Updated Amount: %.2f\n", ctx.Vars["amount"].(float64))
	fmt.Println("  Summary:")
	for _, item := range result.Summary {
		fmt.Printf("      %s: %s\n", item.Currency, item.Amount.String())
	}
	fmt.Println("  Fee Items:")
	for _, item := range result.FeeItems {
		fmt.Printf("      %s: %s\n", item.Currency, item.Amount.String())
	}

	fmt.Printf("\n  Execution Logs of %d:\n", len(result.Logs))
	for i, logEntry := range result.Logs {
		fmt.Printf("    [%d] Rule: %s\n", i+1, logEntry.Rule)
		fmt.Printf("        Vars: %v\n", logEntry.Vars)
		if len(logEntry.FeeItems) > 0 {
			fmt.Printf("        FeeItems: ")
			for j, item := range logEntry.FeeItems {
				if j > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s %s", item.Amount.String(), item.Currency)
			}
			fmt.Println()
		} else {
			fmt.Printf("        FeeItems: (none)\n")
		}
	}
}
