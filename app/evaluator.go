package main

import (
	"fmt"
)

// Evaluator implements the visitor pattern to evaluate expressions
type Evaluator struct{}

// Evaluate evaluates an expression and returns its value
func (e *Evaluator) Evaluate(expr Expr) (interface{}, error) {
	if expr == nil {
		return nil, fmt.Errorf("expression is nil")
	}
	return expr.Accept(e), nil
}

// VisitLiteralExpr evaluates literal expressions
func (e *Evaluator) VisitLiteralExpr(expr *Literal) interface{} {
	return expr.Value
}

// VisitBinaryExpr evaluates binary expressions
func (e *Evaluator) VisitBinaryExpr(expr *Binary) interface{} {
	left, _ := e.Evaluate(expr.Left)
	right, _ := e.Evaluate(expr.Right)

	switch expr.Operator.Type {
	case PLUS:
		if leftNum, ok := left.(float64); ok {
			if rightNum, ok := right.(float64); ok {
				return leftNum + rightNum
			}
		}
		if leftStr, ok := left.(string); ok {
			if rightStr, ok := right.(string); ok {
				return leftStr + rightStr
			}
		}
		return nil
	case MINUS:
		if leftNum, ok := left.(float64); ok {
			if rightNum, ok := right.(float64); ok {
				return leftNum - rightNum
			}
		}
		return nil
	case STAR:
		if leftNum, ok := left.(float64); ok {
			if rightNum, ok := right.(float64); ok {
				return leftNum * rightNum
			}
		}
		return nil
	case SLASH:
		if leftNum, ok := left.(float64); ok {
			if rightNum, ok := right.(float64); ok {
				if rightNum == 0 {
					return nil // Division by zero
				}
				return leftNum / rightNum
			}
		}
		return nil
	case LESS:
		if leftNum, ok := left.(float64); ok {
			if rightNum, ok := right.(float64); ok {
				return leftNum < rightNum
			}
		}
		return nil
	case LESS_EQUAL:
		if leftNum, ok := left.(float64); ok {
			if rightNum, ok := right.(float64); ok {
				return leftNum <= rightNum
			}
		}
		return nil
	case GREATER:
		if leftNum, ok := left.(float64); ok {
			if rightNum, ok := right.(float64); ok {
				return leftNum > rightNum
			}
		}
		return nil
	case GREATER_EQUAL:
		if leftNum, ok := left.(float64); ok {
			if rightNum, ok := right.(float64); ok {
				return leftNum >= rightNum
			}
		}
		return nil
	}

	return nil
}

// VisitGroupingExpr evaluates grouping expressions
func (e *Evaluator) VisitGroupingExpr(expr *Grouping) interface{} {
	result, _ := e.Evaluate(expr.Expression)
	return result
}

// VisitUnaryExpr evaluates unary expressions
func (e *Evaluator) VisitUnaryExpr(expr *Unary) interface{} {
	right, _ := e.Evaluate(expr.Right)

	switch expr.Operator.Type {
	case MINUS:
		if num, ok := right.(float64); ok {
			return -num
		}
		return nil
	case BANG:
		return !isTruthy(right)
	}

	return nil
}

// isTruthy determines the truthiness of a value following Lox rules
func isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}
	if b, ok := value.(bool); ok {
		return b
	}
	return true
}
