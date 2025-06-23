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

// VisitCallExpr prints function call expressions as (call callee arg1 arg2 ...)
func (ap *AstPrinter) VisitCallExpr(expr *Call) Value {
	args := append([]Expr{expr.Callee}, expr.Arguments...)
	return StringValue{Val: ap.parenthesize("call", args...)}
}

func (ap *AstPrinter) VisitFun(expr *Fun) Value {
	args := ap.parenthesizeStrings("args", expr.Parameters...)
	return StringValue{Val: ap.parenthesizeStrings("fun", expr.Name, args, ap.Print(&expr.Block))}
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

// Placeholder implementations for new EYG visitor methods
func (ap *AstPrinter) VisitRecord(expr *Record) Value {
	var fields []string
	for _, field := range expr.Fields {
		fieldStr := fmt.Sprintf("(field %s %s)", field.Name, field.Value.Accept(ap).(StringValue).Val)
		fields = append(fields, fieldStr)
	}
	return StringValue{Val: fmt.Sprintf("(record %s)", strings.Join(fields, " "))}
}

func (ap *AstPrinter) VisitEmptyRecord(expr *EmptyRecord) Value {
	return StringValue{Val: "{}"}
}

func (ap *AstPrinter) VisitList(expr *List) Value {
	var elements []string
	for _, elem := range expr.Elements {
		elements = append(elements, elem.Accept(ap).(StringValue).Val)
	}
	return StringValue{Val: fmt.Sprintf("(list %s)", strings.Join(elements, " "))}
}

func (ap *AstPrinter) VisitAccess(expr *Access) Value {
	return StringValue{Val: fmt.Sprintf("(access %s %s)", expr.Object.Accept(ap).(StringValue).Val, expr.Name)}
}

func (ap *AstPrinter) VisitBuiltin(expr *Builtin) Value {
	var args []string
	for _, arg := range expr.Arguments {
		args = append(args, arg.Accept(ap).(StringValue).Val)
	}
	return StringValue{Val: fmt.Sprintf("(builtin %s %s)", expr.Name, strings.Join(args, " "))}
}

func (ap *AstPrinter) VisitUnion(expr *Union) Value {
	return StringValue{Val: fmt.Sprintf("(union %s %s)", expr.Constructor, expr.Value.Accept(ap).(StringValue).Val)}
}

func (ap *AstPrinter) VisitLambda(expr *Lambda) Value {
	return StringValue{Val: fmt.Sprintf("(lambda (args %s) %s)", strings.Join(expr.Parameters, " "), expr.Body.Accept(ap).(StringValue).Val)}
}

func (ap *AstPrinter) VisitMatch(expr *Match) Value {
	var cases []string
	for _, c := range expr.Cases {
		// Special handling for patterns - convert Union to pattern format
		var patternStr string
		if union, ok := c.Pattern.(*Union); ok {
			patternStr = fmt.Sprintf("(pattern %s %s)", union.Constructor, union.Value.Accept(ap).(StringValue).Val)
		} else {
			patternStr = c.Pattern.Accept(ap).(StringValue).Val
		}
		cases = append(cases, fmt.Sprintf("(case %s %s)", patternStr, c.Body.Accept(ap).(StringValue).Val))
	}
	return StringValue{Val: fmt.Sprintf("(match %s %s)", expr.Value.Accept(ap).(StringValue).Val, strings.Join(cases, " "))}
}

func (ap *AstPrinter) VisitPerform(expr *Perform) Value {
	var args []string
	for _, arg := range expr.Arguments {
		args = append(args, arg.Accept(ap).(StringValue).Val)
	}
	return StringValue{Val: fmt.Sprintf("(perform %s %s)", expr.Effect, strings.Join(args, " "))}
}

func (ap *AstPrinter) VisitHandle(expr *Handle) Value {
	return StringValue{Val: fmt.Sprintf("(handle %s %s %s)", expr.Effect, expr.Handler.Accept(ap).(StringValue).Val, expr.Fallback.Accept(ap).(StringValue).Val)}
}

func (ap *AstPrinter) VisitNamedRef(expr *NamedRef) Value {
	return StringValue{Val: fmt.Sprintf("(named_ref %s %d)", expr.Module, expr.Index)}
}

func (ap *AstPrinter) VisitThunk(expr *Thunk) Value {
	return StringValue{Val: fmt.Sprintf("(thunk %s)", expr.Body.Accept(ap).(StringValue).Val)}
}

func (ap *AstPrinter) VisitSpread(expr *Spread) Value {
	return StringValue{Val: fmt.Sprintf("(spread %s)", expr.Expression.Accept(ap).(StringValue).Val)}
}

func (ap *AstPrinter) VisitDestructure(expr *Destructure) Value {
	var fields []string
	for _, field := range expr.Fields {
		fieldStr := fmt.Sprintf("(field %s %s)", field.Name, field.Value.Accept(ap).(StringValue).Val)
		fields = append(fields, fieldStr)
	}
	return StringValue{Val: fmt.Sprintf("(destructure %s)", strings.Join(fields, " "))}
}

func (ap *AstPrinter) VisitSeq(expr *Seq) Value {
	return StringValue{Val: fmt.Sprintf("(seq %s %s)", expr.Left.Accept(ap).(StringValue).Val, expr.Right.Accept(ap).(StringValue).Val)}
}
