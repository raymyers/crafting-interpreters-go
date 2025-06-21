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

// VisitBinaryExpr evaluates binary expressions (placeholder for now)
func (e *Evaluator) VisitBinaryExpr(expr *Binary) interface{} {
	// TODO: Implement binary expression evaluation
	return nil
}

// VisitGroupingExpr evaluates grouping expressions (placeholder for now)
func (e *Evaluator) VisitGroupingExpr(expr *Grouping) interface{} {
	// TODO: Implement grouping expression evaluation
	return nil
}

// VisitUnaryExpr evaluates unary expressions (placeholder for now)
func (e *Evaluator) VisitUnaryExpr(expr *Unary) interface{} {
	// TODO: Implement unary expression evaluation
	return nil
}