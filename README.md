# Fee Engine

A fee calculation engine that supports expression-based fee rule computation.

## Features

- Expression-driven fee calculation rules
- Context variable updates
- Execution logging
- Interruptible and resumable execution
- High-precision calculations (using decimal)

## Quick Start

### Installation

```bash
go get github.com/ethan/fee-engine
```

### Basic Usage

```go
import fee_engine "github.com/ethan/fee-engine/src"

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

fmt.Printf("Fee: %s %s\n", result.Summary[0].Amount.String(), result.Summary[0].Currency)
```

## Rule Syntax

### Fee Calculation

Use `$(amount, currency)` syntax to calculate fees:

```go
engine.AddRule(`$(100.0, "USD")`)           // Fixed fee
engine.AddRule(`$(amount * rate, "USD")`)   // Variable-based fee
engine.AddRule(`$(-20.0, "USD")`)           // Negative fee (discount)
```

### Variable Assignment

Use assignment syntax to update context variables:

```go
engine.AddRule(`amount = amount * 2`)
engine.AddRule(`rate = 0.03`)
```

### Multi-statement Rules

Use semicolons to separate multiple statements:

```go
engine.AddRule(`amount = amount * 2; $(amount * rate, "USD")`)
```

### Expression Arrays

Return expression arrays to execute multiple fee calculations:

```go
engine.AddRule(`[$(100.0, "USD"), $(200.0, "EUR")]`)
```

### High-precision Calculations

Use decimal functions to ensure precision:

```go
engine.AddRule(`$(Mul(amount, rate), "USD")`)
```

Supported functions: `Add`, `Sub`, `Mul`, `Div`, `Neg`

## Execution Control

### Execute All Rules

```go
result, err := engine.Execute()
```

### Execute N Rules

```go
result, err := engine.ExecuteN(3)  // Execute first 3 rules
```

### Interrupt and Resume

```go
// Execute first 3 rules
result1, _ := engine.ExecuteN(3)

// Continue with remaining rules
result2, _ := engine.ExecuteN(2)
```

## Execution Logging

Enable logging to track execution:

```go
engine := fee_engine.New(ctx).EnableLog()
result, _ := engine.Execute()

// Access logs
for _, log := range result.Logs {
    fmt.Printf("Rule: %s\n", log.Rule)
    fmt.Printf("Vars: %v\n", log.Vars)
    fmt.Printf("FeeItems: %v\n", log.FeeItems)
}
```

## Result Structure

```go
type ExecuteResult struct {
    ProcessedRules int              // Number of processed rules
    FeeItems       []FeeItem        // All fee items
    Summary        []FeeItem        // Fees summarized by currency
    Context        *Context         // Updated context
    Logs           []Log            // Execution logs (if enabled)
}
```

## Examples

See `cmd/demo/main.go` for more examples, including:

- Basic fee calculation
- Negative fee handling
- Context updates
- Multi-currency calculations
- Interrupt and resume execution
- Expression arrays
- High-precision calculations
- Execution logging

Run examples:

```bash
go run cmd/demo/main.go
```

## Dependencies

- [expr-lang/expr](https://github.com/expr-lang/expr) - Expression evaluation
- [shopspring/decimal](https://github.com/shopspring/decimal) - High-precision numeric calculations

## Real-world Example: OnRamp Fee Calculation

The OnRamp example demonstrates a complex fee calculation scenario for cryptocurrency on-ramp transactions. It shows how to:

1. Convert network fees between currencies
2. Calculate multiple fee types (fiat fee, wello fee, merchant fee)
3. Apply discounts (coupons)
4. Update context variables during execution
5. Return fees in multiple currencies

### Code Example

```go
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
        "fiat_fee":           0.0,
        "wello_fee":          0.0,
        "merchant_fee":       0.0,
        "total_fee":          0.0,
        "fee_in_usd":         0.0,
    },
}
engine := fee_engine.New(ctx).EnableLog()

result, err := engine.AddRule(
    // Convert network fee from crypto to fiat currency
    `network_fee = network_fee * crypto_to_usd_rate / kes_to_usd_rate; $(network_fee, fiat_currency)`,
    
    // Add network fee to base amount
    `amount = amount + network_fee`,
    
    // Calculate fiat fee (rate * amount + fixed)
    `fiat_fee = amount * fiat_fee_rate + fiat_fee_fixed; $(fiat_fee, fiat_currency)`,
    
    // Calculate wello fee
    `wello_fee = amount * wello_fee_rate + wello_fee_fixed; $(wello_fee, fiat_currency)`,
    
    // Calculate merchant fee
    `merchant_fee = amount * merchant_fee_rate + merchant_fee_fixed; $(merchant_fee, fiat_currency)`,
    
    // Calculate total fee in KES
    `total_fee = fiat_fee + wello_fee + merchant_fee + network_fee`,
    
    // Apply coupon discount
    `total_fee = total_fee - coupon; coupon > 0 ? $(-coupon, coupon_currency) : nil`,
    
    // Convert total fee to USD
    `fee_in_usd = total_fee * kes_to_usd_rate`,
    
    // Return fees in both currencies (negative for deduction)
    `[$(-total_fee, fiat_currency), $(fee_in_usd, "USD")]`,
).Execute()
```

### Output

```
Updated Amount: 5854.79
Summary:
    KES: -0.000000000000095
    USD: 6.024361411000001
Fee Items:
    KES: 26.792370000000005
    KES: 158.5479237
    KES: 258.5479237
    KES: 358.5479237
    KES: -200
    KES: -602.4361411000001
    USD: 6.024361411000001

Execution Logs of 9:
  [1] Rule: network_fee = network_fee * crypto_to_usd_rate / kes_to_usd_rate; $(network_fee, fiat_currency)
      Vars: map[amount:5828 network_fee:26.79 ...]
      FeeItems: 26.792370000000005 KES
  [2] Rule: amount = amount + network_fee
      Vars: map[amount:5854.79 ...]
      FeeItems: (none)
  [3] Rule: fiat_fee = amount * fiat_fee_rate + fiat_fee_fixed; $(fiat_fee, fiat_currency)
      Vars: map[amount:5854.79 fiat_fee:158.55 ...]
      FeeItems: 158.5479237 KES
  [4] Rule: wello_fee = amount * wello_fee_rate + wello_fee_fixed; $(wello_fee, fiat_currency)
      Vars: map[amount:5854.79 wello_fee:258.55 ...]
      FeeItems: 258.5479237 KES
  [5] Rule: merchant_fee = amount * merchant_fee_rate + merchant_fee_fixed; $(merchant_fee, fiat_currency)
      Vars: map[amount:5854.79 merchant_fee:358.55 ...]
      FeeItems: 358.5479237 KES
  [6] Rule: total_fee = fiat_fee + wello_fee + merchant_fee + network_fee
      Vars: map[total_fee:802.44 ...]
      FeeItems: (none)
  [7] Rule: total_fee = total_fee - coupon; coupon > 0 ? $(-coupon, coupon_currency) : nil
      Vars: map[total_fee:602.44 ...]
      FeeItems: -200 KES
  [8] Rule: fee_in_usd = total_fee * kes_to_usd_rate
      Vars: map[fee_in_usd:6.02 ...]
      FeeItems: (none)
  [9] Rule: [$(-total_fee, fiat_currency), $(fee_in_usd, "USD")]
      Vars: map[total_fee:602.44 fee_in_usd:6.02 ...]
      FeeItems: -602.4361411000001 KES, 6.024361411000001 USD
```

### Output Explanation

**Updated Amount**: The final amount after adding the network fee (5828.0 + 26.79 = 5854.79 KES)

**Summary**: Fees aggregated by currency
- `KES: -0.000000000000095` - Floating-point precision issue. Ignore or use decimal functions to avoid this
- `USD: 6.024361411000001` - Total fee in USD

**Fee Items**: All individual fee items in execution order
1. `26.79 KES` - Network fee (converted from crypto)
2. `158.55 KES` - Fiat fee (1% of amount + 100 fixed)
3. `258.55 KES` - Wello fee (1% of amount + 200 fixed)
4. `358.55 KES` - Merchant fee (1% of amount + 300 fixed)
5. `-200 KES` - Coupon discount (negative fee)
6. `-602.44 KES` - Total fee deduction in KES
7. `6.02 USD` - Total fee in USD

**Execution Logs**: Detailed trace of each rule execution showing:
- Rule expression
- Variable state after execution
- Fee items generated (if any)

## License

MIT

