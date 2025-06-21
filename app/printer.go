package main

import (
	"fmt"
	"strings"
)

// AstPrinter implements the visitor pattern to print AST as S-expressions
type AstPrinter struct{}

// Print converts an expression to its S-expression string representation
func (ap *AstPrinter) Print(expr Expr) string {
	if expr == nil {
		return ""
	}
	return expr.Accept(ap).(string)
}

// VisitBinaryExpr prints binary expressions as (operator left right)
func (ap *AstPrinter) VisitBinaryExpr(expr *Binary) interface{} {
	return ap.parenthesize(expr.Operator.Lexeme, expr.Left, expr.Right)
}

// VisitGroupingExpr prints grouping expressions as (group expression)
func (ap *AstPrinter) VisitGroupingExpr(expr *Grouping) interface{} {
	return ap.parenthesize("group", expr.Expression)
}

// VisitLiteralExpr prints literal values
func (ap *AstPrinter) VisitLiteralExpr(expr *Literal) interface{} {
	if expr.Value == nil {
		return "nil"
	}
	
	switch v := expr.Value.(type) {
	case float64:
		// Format numbers to match expected output
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%g", v)
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", expr.Value)
	}
}

// VisitUnaryExpr prints unary expressions as (operator operand)
func (ap *AstPrinter) VisitUnaryExpr(expr *Unary) interface{} {
	return ap.parenthesize(expr.Operator.Lexeme, expr.Right)
}

// parenthesize wraps expressions in parentheses with the operator/name first
func (ap *AstPrinter) parenthesize(name string, exprs ...Expr) string {
	var builder strings.Builder
	
	builder.WriteString("(")
	builder.WriteString(name)
	
	for _, expr := range exprs {
		builder.WriteString(" ")
		builder.WriteString(expr.Accept(ap).(string))
	}
	
	builder.WriteString(")")
	return builder.String()
}