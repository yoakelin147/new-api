package billingexpr

import (
	"math"
	"strings"

	"github.com/expr-lang/expr/ast"
)

// UnitPrices contains the linear $/1M-token coefficients declared by one
// tier in a billing expression.
type UnitPrices struct {
	Input        float64
	Output       float64
	CacheRead    float64
	CacheWrite   float64
	CacheWrite1h float64
}

var unitPriceVariables = map[string]struct{}{
	"p":    {},
	"c":    {},
	"cr":   {},
	"cc":   {},
	"cc1h": {},
}

type linearUnitPrice struct {
	constant     float64
	coefficients map[string]float64
}

type cachedUnitPrices struct {
	prices UnitPrices
	ok     bool
}

// ExtractTierUnitPrices returns the linear token prices declared by the
// requested tier. It rejects non-linear tier bodies rather than guessing
// prices that cannot be represented as fixed unit rates.
func ExtractTierUnitPrices(exprStr string, matchedTier string) (UnitPrices, bool) {
	tierKey := strings.ToLower(strings.TrimSpace(matchedTier))
	if strings.TrimSpace(exprStr) == "" || tierKey == "" {
		return UnitPrices{}, false
	}
	hash := ExprHashString(exprStr)
	prog, err := CompileFromCacheByHash(exprStr, hash)
	if err != nil {
		return UnitPrices{}, false
	}
	cacheMu.RLock()
	entry := cache[hash]
	if entry != nil {
		if cached, ok := entry.unitPrices[tierKey]; ok {
			cacheMu.RUnlock()
			return cached.prices, cached.ok
		}
	}
	cacheMu.RUnlock()

	var prices UnitPrices
	found := false
	ast.Find(prog.Node(), func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallNode)
		if !ok || len(call.Arguments) != 2 {
			return false
		}
		callee, ok := call.Callee.(*ast.IdentifierNode)
		if !ok || callee.Value != "tier" {
			return false
		}
		label, ok := call.Arguments[0].(*ast.StringNode)
		if !ok || !strings.EqualFold(strings.TrimSpace(label.Value), strings.TrimSpace(matchedTier)) {
			return false
		}
		linear, ok := extractLinearUnitPrice(call.Arguments[1])
		if !ok {
			return false
		}
		prices = UnitPrices{
			Input:        linear.coefficients["p"],
			Output:       linear.coefficients["c"],
			CacheRead:    linear.coefficients["cr"],
			CacheWrite:   linear.coefficients["cc"],
			CacheWrite1h: linear.coefficients["cc1h"],
		}
		found = validUnitPrices(prices)
		return found
	})
	cacheMu.Lock()
	if entry := cache[hash]; entry != nil {
		if entry.unitPrices == nil {
			entry.unitPrices = make(map[string]cachedUnitPrices)
		}
		entry.unitPrices[tierKey] = cachedUnitPrices{prices: prices, ok: found}
	}
	cacheMu.Unlock()
	return prices, found
}

func extractLinearUnitPrice(node ast.Node) (linearUnitPrice, bool) {
	switch typed := node.(type) {
	case *ast.IdentifierNode:
		if _, ok := unitPriceVariables[typed.Value]; !ok {
			return linearUnitPrice{}, false
		}
		return linearUnitPrice{coefficients: map[string]float64{typed.Value: 1}}, true
	case *ast.IntegerNode:
		return constantLinearUnitPrice(float64(typed.Value)), true
	case *ast.FloatNode:
		return constantLinearUnitPrice(typed.Value), true
	case *ast.ConstantNode:
		value, ok := numericConstant(typed.Value)
		if !ok {
			return linearUnitPrice{}, false
		}
		return constantLinearUnitPrice(value), true
	case *ast.UnaryNode:
		value, ok := extractLinearUnitPrice(typed.Node)
		if !ok || typed.Operator != "-" {
			return linearUnitPrice{}, false
		}
		return scaleLinearUnitPrice(value, -1), true
	case *ast.BinaryNode:
		left, leftOK := extractLinearUnitPrice(typed.Left)
		right, rightOK := extractLinearUnitPrice(typed.Right)
		if !leftOK || !rightOK {
			return linearUnitPrice{}, false
		}
		switch typed.Operator {
		case "+":
			return addLinearUnitPrices(left, right, 1), true
		case "-":
			return addLinearUnitPrices(left, right, -1), true
		case "*":
			if isConstantLinearUnitPrice(left) {
				return scaleLinearUnitPrice(right, left.constant), true
			}
			if isConstantLinearUnitPrice(right) {
				return scaleLinearUnitPrice(left, right.constant), true
			}
		case "/":
			if isConstantLinearUnitPrice(right) && right.constant != 0 {
				return scaleLinearUnitPrice(left, 1/right.constant), true
			}
		}
	}
	return linearUnitPrice{}, false
}

func numericConstant(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return 0, false
	}
}

func constantLinearUnitPrice(value float64) linearUnitPrice {
	return linearUnitPrice{constant: value, coefficients: map[string]float64{}}
}

func isConstantLinearUnitPrice(value linearUnitPrice) bool {
	return len(value.coefficients) == 0
}

func scaleLinearUnitPrice(value linearUnitPrice, factor float64) linearUnitPrice {
	result := constantLinearUnitPrice(value.constant * factor)
	for variable, coefficient := range value.coefficients {
		result.coefficients[variable] = coefficient * factor
	}
	return result
}

func addLinearUnitPrices(left linearUnitPrice, right linearUnitPrice, rightFactor float64) linearUnitPrice {
	result := constantLinearUnitPrice(left.constant + right.constant*rightFactor)
	for variable, coefficient := range left.coefficients {
		result.coefficients[variable] = coefficient
	}
	for variable, coefficient := range right.coefficients {
		result.coefficients[variable] += coefficient * rightFactor
	}
	return result
}

func validUnitPrices(prices UnitPrices) bool {
	values := []float64{prices.Input, prices.Output, prices.CacheRead, prices.CacheWrite, prices.CacheWrite1h}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return false
		}
	}
	return true
}
