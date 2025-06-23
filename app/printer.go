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
	result := expr.Accept(ap)
	if str, ok := result.(StringValue); ok {
		return str.Val
	}
	return ""
}

// VisitBinaryExpr prints binary expressions as (operator left right)
func (ap *AstPrinter) VisitBinaryExpr(expr *Binary) Value {
	return StringValue{Val: ap.parenthesize(expr.Operator.Lexeme, expr.Left, expr.Right)}
}

// VisitGroupingExpr prints grouping expressions as (group expression)
func (ap *AstPrinter) VisitGroupingExpr(expr *Grouping) Value {
	return StringValue{Val: ap.parenthesize("group", expr.Expression)}
}

// VisitLiteralExpr prints literal values
func (ap *AstPrinter) VisitLiteralExpr(expr *Literal) Value {
	switch v := expr.Value.(type) {
	case NilValue:
		return StringValue{Val: "nil"}
	case NumberValue:
		// Format numbers to match expected output
		if v.Val == float64(int64(v.Val)) {
			return StringValue{Val: fmt.Sprintf("%.1f", v.Val)}
		}
		return StringValue{Val: fmt.Sprintf("%g", v.Val)}
	case StringValue:
		return StringValue{Val: v.Val}
	case BoolValue:
		if v.Val {
			return StringValue{Val: "true"}
		}
		return StringValue{Val: "false"}
	default:
		return StringValue{Val: fmt.Sprintf("%v", expr.Value)}
	}
}

// VisitUnaryExpr prints unary expressions as (operator operand)
func (ap *AstPrinter) VisitUnaryExpr(expr *Unary) Value {
	return StringValue{Val: ap.parenthesize(expr.Operator.Lexeme, expr.Right)}
}

// VisitVariableExpr prints variable names
func (ap *AstPrinter) VisitVariableExpr(expr *Variable) Value {
	return StringValue{Val: expr.Name.Lexeme}
}

func (ap *AstPrinter) VisitPrintStatement(expr *PrintStatement) Value {
	return StringValue{Val: ap.parenthesize("print", expr.Expression)}
}

func (ap *AstPrinter) VisitStatements(expr *Statements) Value {
	return StringValue{Val: ap.parenthesize("seq", expr.Exprs...)}
}

func (ap *AstPrinter) VisitVarStatement(expr *VarStatement) Value {
	var strVal string
	if str, ok := expr.Expression.Accept(ap).(StringValue); ok {
		strVal = str.Val
	} else {
		strVal = "?"
	}

	return StringValue{Val: ap.parenthesizeStrings("var", expr.name, strVal)}
}

func (ap *AstPrinter) VisitBlock(expr *Block) Value {
	return StringValue{Val: ap.parenthesize("block", expr.Statements...)}
}

func (ap *AstPrinter) VisitIfStatement(expr *IfStatement) Value {
	if expr.ElseBranch != nil {
		return StringValue{Val: ap.parenthesize("if", expr.Condition, expr.ThenBranch, expr.ElseBranch)}
	}
	return StringValue{Val: ap.parenthesize("if", expr.Condition, expr.ThenBranch)}
}

func (ap *AstPrinter) VisitWhileStatement(expr *WhileStatement) Value {
	return StringValue{Val: ap.parenthesize("while", expr.Condition, expr.Body)}
}

func (ap *AstPrinter) VisitForStatement(expr *ForStatement) Value {
	return StringValue{Val: ap.parenthesize("for", expr.Initializer, expr.Condition, expr.Increment, expr.Body)}
}

// VisitCallExpr prints function call expressions as (call callee arg1 arg2 ...)
func (ap *AstPrinter) VisitCallExpr(expr *Call) Value {
	args := append([]Expr{expr.Callee}, expr.Arguments...)
	return StringValue{Val: ap.parenthesize("call", args...)}
}

// parenthesize wraps expressions in parentheses with the operator/name first
func (ap *AstPrinter) parenthesize(name string, exprs ...Expr) string {
	var builder strings.Builder

	builder.WriteString("(")
	builder.WriteString(name)

	for _, expr := range exprs {
		builder.WriteString(" ")
		if nil == expr {
			builder.WriteString("nil")
		} else {
			result := expr.Accept(ap)
			if str, ok := result.(StringValue); ok {
				builder.WriteString(str.Val)
			}
		}

	}

	builder.WriteString(")")
	return builder.String()
}

func (ap *AstPrinter) parenthesizeStrings(first string, rest ...string) string {
	var builder strings.Builder

	builder.WriteString("(")
	builder.WriteString(first)

	for _, item := range rest {
		builder.WriteString(" ")
		builder.WriteString(item)
	}

	builder.WriteString(")")
	return builder.String()
}
