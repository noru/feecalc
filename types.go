package feecalc

import (
	"sync"

	"github.com/shopspring/decimal"
)

type Log struct {
	Rule     string                 `json:"rule"`
	Vars     map[string]interface{} `json:"vars"`
	FeeItems []FeeItem              `json:"fee_items"`
}

// Context holds variables and fee items during calculation
type Context struct {
	mu               sync.RWMutex
	ctxJson          []byte                 `json: "-"`
	Vars             map[string]interface{} `json:"vars"`
	FeeItems         []FeeItem              `json:"fee_items"`
	Logs             []Log                  `json:"logs"`
	enableLog        bool
	lastExecutedRule int
}

// FeeItem represents a fee with amount and currency
type FeeItem struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

// RuleResult represents the result of executing a fee rule
type RuleResult struct {
	FeeItems []FeeItem `json:"fee_items,omitempty"`
	Context  *Context  `json:"context,omitempty"`
}

// FeeEngine executes fee calculation rules
type FeeEngine struct {
	ctx   *Context
	rules []string
}

// ExecuteResult represents the result of executing rules
type ExecuteResult struct {
	ProcessedRules int       `json:"processed_rules"`
	Logs           []Log     `json:"logs"`
	FeeItems       []FeeItem `json:"fee_items"`
	Summary        []FeeItem `json:"summary"`
	Context        *Context  `json:"context"`
}
