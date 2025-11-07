package fee_engine

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Copy creates a deep copy of the context
func (c *Context) Copy() *Context {
	c.mu.RLock()
	defer c.mu.RUnlock()

	newVars := make(map[string]interface{})
	for k, v := range c.Vars {
		newVars[k] = v
	}

	newFeeItems := make([]FeeItem, len(c.FeeItems))
	copy(newFeeItems, c.FeeItems)

	newLogs := make([]Log, len(c.Logs))
	copy(newLogs, c.Logs)

	return &Context{
		Vars:             newVars,
		FeeItems:         newFeeItems,
		Logs:             newLogs,
		lastExecutedRule: c.lastExecutedRule,
	}
}

// SetVar sets a variable in the context
func (c *Context) SetVar(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Vars[key] = value
}

// GetVar gets a variable from the context
func (c *Context) GetVar(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.Vars[key]
	return val, ok
}

// addFeeItem adds a fee item to the context
func (c *Context) addFeeItem(item FeeItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.FeeItems = append(c.FeeItems, item)
}

// addLog adds a log entry to the context
func (c *Context) addLog(log Log) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Logs = append(c.Logs, log)
}

// New creates a new instance of FeeEngine with the given context
func New(ctx *Context) *FeeEngine {
	if ctx == nil {
		ctx = &Context{
			Vars:             make(map[string]interface{}),
			FeeItems:         make([]FeeItem, 0),
			Logs:             make([]Log, 0),
			lastExecutedRule: 0,
			enableLog:        false,
		}
	}
	return &FeeEngine{
		ctx:   ctx,
		rules: make([]string, 0),
	}
}

func (e *FeeEngine) EnableLog() *FeeEngine {
	e.ctx.enableLog = true
	return e
}

// AddRule adds one or more fee rules to the engine
func (e *FeeEngine) AddRule(rules ...string) *FeeEngine {
	e.rules = append(e.rules, rules...)
	return e
}

// Execute executes all remaining rules from the current position
func (e *FeeEngine) Execute() (*ExecuteResult, error) {
	remaining := len(e.rules) - e.ctx.lastExecutedRule
	return e.ExecuteN(remaining)
}

// ExecuteN executes N rules starting from the last executed position
func (e *FeeEngine) ExecuteN(count int) (*ExecuteResult, error) {
	if e.ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}

	startIndex := e.ctx.lastExecutedRule
	if startIndex >= len(e.rules) {
		return e.buildExecuteResult(0)
	}

	endIndex := startIndex + count
	if endIndex > len(e.rules) {
		endIndex = len(e.rules)
	}

	processed := 0
	for i := startIndex; i < endIndex; i++ {
		rule := e.rules[i]

		result, err := e.executeRule(rule)
		if err != nil {
			return nil, fmt.Errorf("error executing rule at index %d: %w", i, err)
		}

		// Process rule result: add fee items and update context
		var ruleFeeItems []FeeItem
		if result != nil {
			if len(result.FeeItems) > 0 {
				ruleFeeItems = make([]FeeItem, len(result.FeeItems))
				copy(ruleFeeItems, result.FeeItems)
				for _, item := range result.FeeItems {
					e.ctx.addFeeItem(item)
				}
			}
			if result.Context != nil {
				for k, v := range result.Context.Vars {
					e.ctx.SetVar(k, v)
				}
			}
		}

		// Log entry (only if logging is enabled)
		if e.ctx.enableLog {
			e.ctx.mu.RLock()
			varsAfter := make(map[string]interface{})
			for k, v := range e.ctx.Vars {
				varsAfter[k] = v
			}
			e.ctx.mu.RUnlock()

			e.ctx.addLog(Log{
				Rule:     rule,
				Vars:     varsAfter,
				FeeItems: ruleFeeItems,
			})
		}

		processed++
	}

	e.ctx.lastExecutedRule = endIndex
	return e.buildExecuteResult(processed)
}

// buildExecuteResult builds an ExecuteResult from current context state
func (e *FeeEngine) buildExecuteResult(processed int) (*ExecuteResult, error) {
	e.ctx.mu.RLock()
	defer e.ctx.mu.RUnlock()

	summary := e.summarizeFeeItems(e.ctx.FeeItems)
	feeItems := make([]FeeItem, len(e.ctx.FeeItems))
	copy(feeItems, e.ctx.FeeItems)
	logs := make([]Log, len(e.ctx.Logs))
	copy(logs, e.ctx.Logs)

	return &ExecuteResult{
		ProcessedRules: processed,
		FeeItems:       feeItems,
		Summary:        summary,
		Context:        e.ctx,
		Logs:           logs,
	}, nil
}

// executeRule executes a single rule and returns the result
func (e *FeeEngine) executeRule(rule string) (*RuleResult, error) {
	return executeExpression(rule, e.ctx)
}

// summarizeFeeItems summarizes fee items by currency
func (e *FeeEngine) summarizeFeeItems(items []FeeItem) []FeeItem {
	currencyMap := make(map[string]decimal.Decimal)
	for _, item := range items {
		currencyMap[item.Currency] = currencyMap[item.Currency].Add(item.Amount)
	}

	summary := make([]FeeItem, 0, len(currencyMap))
	for currency, amount := range currencyMap {
		summary = append(summary, FeeItem{
			Amount:   amount,
			Currency: currency,
		})
	}
	return summary
}

// GetRules returns all rules
func (e *FeeEngine) GetRules() []string {
	return e.rules
}

// GetRuleCount returns the number of rules
func (e *FeeEngine) GetRuleCount() int {
	return len(e.rules)
}

// GetContext returns the context
func (e *FeeEngine) GetContext() *Context {
	return e.ctx
}
