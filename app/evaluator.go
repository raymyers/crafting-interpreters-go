package main

import (
	"fmt"
	"io"
	"time"
)

// Scope represents a variable scope with optional parent scope
type Scope struct {
	envMap map[string]Value
	parent *Scope
}

// NewScope creates a new scope with optional parent
func NewScope(parent *Scope) *Scope {
	return &Scope{
		envMap: make(map[string]Value),
		parent: parent,
	}
}

// lookup searches for a variable in this scope and parent scopes
func (s *Scope) lookup(name string) (Value, bool) {
	if value, exists := s.envMap[name]; exists {
		return value, true
	}
	if s.parent != nil {
		return s.parent.lookup(name)
	}
	return NilValue{}, false
}

// isDefined checks if a variable is defined in this scope or parent scopes
func (s *Scope) isDefined(name string) bool {
	_, exists := s.lookup(name)
	return exists
}

// define adds a variable to the current scope
func (s *Scope) define(name string, value Value) {
	s.envMap[name] = value
}

// assign sets a variable value in the appropriate scope
func (s *Scope) assign(name string, value Value) bool {
	if _, exists := s.envMap[name]; exists {
		s.envMap[name] = value
		return true
	}
	if s.parent != nil {
		return s.parent.assign(name, value)
	}
	return false
}

// Evaluator implements the visitor pattern to evaluate expressions
type Evaluator struct {
	scope  *Scope
	output io.Writer
}

// NewEvaluator creates a new evaluator with the given scope and output writer
func NewEvaluator(scope *Scope, output io.Writer) *Evaluator {
	return &Evaluator{
		scope:  scope,
		output: output,
	}
}

// Helper functions to create Union types for booleans
func trueValue() UnionValue {
	return UnionValue{Constructor: "True", Value: NilValue{}}
}

func falseValue() UnionValue {
	return UnionValue{Constructor: "False", Value: NilValue{}}
}

func boolToUnion(b bool) UnionValue {
	if b {
		return trueValue()
	}
	return falseValue()
}

// Evaluate evaluates an expression and returns its value
func (e *Evaluator) Evaluate(expr Expr) Value {
	if expr == nil {
		return ErrorValue{"expression is nil", 0}
	}
	return expr.Accept(e)
}

// VisitLiteralExpr evaluates literal expressions
func (e *Evaluator) VisitLiteralExpr(expr *Literal) Value {
	return expr.Value
}

// VisitBinaryExpr evaluates binary expressions
func (e *Evaluator) VisitBinaryExpr(expr *Binary) Value {
	if expr.Operator.Type == EQUAL {
		if leftVar, ok := expr.Left.(*Variable); ok {
			right := e.Evaluate(expr.Right)
			if _, ev := right.(ErrorValue); ev {
				return right
			}
			varName := leftVar.Name.Lexeme
			if e.scope.isDefined(varName) {
				if e.scope.assign(varName, right) {
					return right
				}
			} else {
				// Define new variable in current scope
				e.scope.define(varName, right)
				return right
			}
			return ErrorValue{Message: "Assignment failed", Line: expr.Line}
		} else {
			return ErrorValue{Message: "Left of = must be a variable", Line: expr.Line}
		}

	}
	if expr.Operator.Type == OR {
		left := e.Evaluate(expr.Left)
		if _, ev := left.(ErrorValue); ev {
			return left
		}
		if isTruthy(left) {
			return left
		}
		return e.Evaluate(expr.Right)
	}
	if expr.Operator.Type == AND {
		left := e.Evaluate(expr.Left)
		if _, ev := left.(ErrorValue); ev {
			return left
		}
		if !isTruthy(left) {
			return left
		}
		return e.Evaluate(expr.Right)
	}
	left := e.Evaluate(expr.Left)
	if _, ev := left.(ErrorValue); ev {
		return left
	}
	right := e.Evaluate(expr.Right)
	if _, ev := right.(ErrorValue); ev {
		return right
	}
	switch expr.Operator.Type {
	case PLUS:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return NumberValue{Val: leftNum.Val + rightNum.Val}
			}

		}
		if leftStr, ok := left.(StringValue); ok {
			if rightStr, ok := right.(StringValue); ok {
				return StringValue{Val: leftStr.Val + rightStr.Val}
			}
		}
		return ErrorValue{Message: "Operands must be two numbers or two strings", Line: expr.Line}
	case MINUS:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return NumberValue{Val: leftNum.Val - rightNum.Val}
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case STAR:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return NumberValue{Val: leftNum.Val * rightNum.Val}
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case SLASH:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				if rightNum.Val == 0 {
					return ErrorValue{Message: "Division by zero", Line: expr.Line}
				}
				return NumberValue{Val: leftNum.Val / rightNum.Val}
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case LESS:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return boolToUnion(leftNum.Val < rightNum.Val)
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case LESS_EQUAL:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return boolToUnion(leftNum.Val <= rightNum.Val)
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case GREATER:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return boolToUnion(leftNum.Val > rightNum.Val)
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case GREATER_EQUAL:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return boolToUnion(leftNum.Val >= rightNum.Val)
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case EQUAL_EQUAL:
		return boolToUnion(isEqual(left, right))
	case BANG_EQUAL:
		return boolToUnion(!isEqual(left, right))
	}

	return ErrorValue{Message: "Unknown binary operator", Line: expr.Line}
}

// VisitGroupingExpr evaluates grouping expressions
func (e *Evaluator) VisitGroupingExpr(expr *Grouping) Value {
	return e.Evaluate(expr.Expression)
}

// VisitUnaryExpr evaluates unary expressions
func (e *Evaluator) VisitUnaryExpr(expr *Unary) Value {
	right := e.Evaluate(expr.Right)
	if _, ev := right.(ErrorValue); ev {
		return right
	}
	switch expr.Operator.Type {
	case MINUS:
		if num, ok := right.(NumberValue); ok {
			return NumberValue{Val: -num.Val}
		}
		return ErrorValue{Message: "Operand must be a number", Line: expr.Line}
	case BANG:
		return boolToUnion(!isTruthy(right))
	}

	return ErrorValue{Message: "Unknown unary operator", Line: expr.Line}
}

// VisitVariableExpr evaluates variable expressions
func (e *Evaluator) VisitVariableExpr(expr *Variable) Value {
	if value, ok := e.scope.lookup(expr.Name.Lexeme); ok {
		return value
	}
	return ErrorValue{Message: fmt.Sprintf("Undefined variable '%s'", expr.Name.Lexeme), Line: expr.Line}
}

func (e *Evaluator) VisitPrintStatement(expr *PrintStatement) Value {
	result := e.Evaluate(expr.Expression)
	switch result.(type) {
	case ErrorValue:
		return result
	default:
		_, err := fmt.Fprintf(e.output, "%s\n", formatValue(result))
		if err != nil {
			return ErrorValue{Message: "Print failed"}
		}
		return NilValue{}
	}
}

func (e *Evaluator) VisitStatements(expr *Statements) Value {
	var result Value = NilValue{}
	for _, v := range expr.Exprs {
		result = e.Evaluate(v)
		switch result.(type) {
		case ErrorValue:
			return result
		}
	}
	return result
}

func (e *Evaluator) VisitVarStatement(expr *VarStatement) Value {
	result := e.Evaluate(expr.Expression)
	switch result.(type) {
	case ErrorValue:
		return result
	default:
		e.scope.define(expr.name, result)
		return NilValue{}
	}
}

func (e *Evaluator) VisitBlock(expr *Block) Value {
	// Create new scope for block
	previousScope := e.scope
	e.scope = NewScope(previousScope)

	result := e.evalStatements(expr.Statements)
	// Restore previous scope (block scoping)
	e.scope = previousScope
	return result
}

func (e *Evaluator) evalStatements(statements []Expr) Value {
	var result Value = NilValue{}
	for _, stmt := range statements {
		result = e.Evaluate(stmt)
		switch result.(type) {
		case ErrorValue:
			return result
		}
	}
	return result
}

func (e *Evaluator) VisitIfStatement(expr *IfStatement) Value {
	conditionValue := e.Evaluate(expr.Condition)
	if _, isError := conditionValue.(ErrorValue); isError {
		return conditionValue
	}

	if isTruthy(conditionValue) {
		return e.Evaluate(expr.ThenBranch)
	} else if expr.ElseBranch != nil {
		return e.Evaluate(expr.ElseBranch)
	}

	return NilValue{}
}

func (e *Evaluator) VisitWhileStatement(expr *WhileStatement) Value {
	for {
		conditionValue := e.Evaluate(expr.Condition)
		if _, isError := conditionValue.(ErrorValue); isError {
			return conditionValue
		}

		if !isTruthy(conditionValue) {
			break
		}

		bodyResult := e.Evaluate(expr.Body)
		if _, isError := bodyResult.(ErrorValue); isError {
			return bodyResult
		}
	}

	return NilValue{}
}

func (e *Evaluator) VisitForStatement(expr *ForStatement) Value {
	if nil != expr.Initializer {
		_ = e.Evaluate(expr.Initializer)
	}
	for {

		conditionValue := e.Evaluate(expr.Condition)
		if _, isError := conditionValue.(ErrorValue); isError {
			return conditionValue
		}

		if !isTruthy(conditionValue) {
			break
		}

		bodyResult := e.Evaluate(expr.Body)
		if _, isError := bodyResult.(ErrorValue); isError {
			return bodyResult
		}
		if nil != expr.Increment {
			_ = e.Evaluate(expr.Increment)
		}
	}

	return NilValue{}
}

func (e *Evaluator) VisitCallExpr(expr *Call) Value {
	// Check if it's a variable reference to "clock"
	if varExpr, ok := expr.Callee.(*Variable); ok && varExpr.Name.Lexeme == "clock" {
		// Check that clock() is called with no arguments
		if len(expr.Arguments) != 0 {
			return ErrorValue{Message: "clock() takes no arguments", Line: expr.Line}
		}

		// Return current time in epoch seconds
		epochSeconds := float64(time.Now().Unix())
		return NumberValue{Val: epochSeconds}
	} else if varExpr, ok := expr.Callee.(*Variable); ok {
		lookup, ok := e.scope.lookup(varExpr.Name.Lexeme)
		if !ok {
			return ErrorValue{Message: "undefined function", Line: expr.Line}
		}
		if fv, ok := lookup.(FunValue); ok {
			// Check argument count
			if len(expr.Arguments) != len(fv.Val.Parameters) {
				return ErrorValue{
					Message: fmt.Sprintf("Expected %d arguments but got %d", len(fv.Val.Parameters), len(expr.Arguments)),
					Line:    expr.Line,
				}
			}

			// Evaluate arguments
			argValues := make([]Value, len(expr.Arguments))
			for i, arg := range expr.Arguments {
				argValue := e.Evaluate(arg)
				if _, isError := argValue.(ErrorValue); isError {
					return argValue
				}
				argValues[i] = argValue
			}

			// Create new scope for function execution
			previousScope := e.scope
			e.scope = NewScope(previousScope)

			// Bind parameters to arguments in the new scope
			for i, paramName := range fv.Val.Parameters {
				e.scope.define(paramName, argValues[i])
			}

			// Execute function body
			result := e.evalStatements(fv.Val.Block.Statements)

			// Restore previous scope
			e.scope = previousScope
			return result
		} else {
			return ErrorValue{Message: "cannot call a non-function", Line: expr.Line}
		}
	}

	// Evaluate the callee for other function calls
	callee := e.Evaluate(expr.Callee)
	if _, isError := callee.(ErrorValue); isError {
		return callee
	}

	// Any other function call is an error
	return ErrorValue{Message: "Undefined function", Line: expr.Line}
}
func (e *Evaluator) VisitFun(expr *Fun) Value {
	val := FunValue{Val: *expr}
	e.scope.define(expr.Name, val)
	return val
}

// isTruthy determines the truthiness of a value following Lox rules
func isTruthy(value Value) bool {
	switch v := value.(type) {
	case NilValue:
		return false
	case BoolValue:
		return v.Val
	case UnionValue:
		// True({}) is truthy, False({}) is falsy
		return v.Constructor == "True"
	default:
		return true
	}
}

// isEqual determines if two values are equal following Lox rules
func isEqual(left, right Value) bool {
	switch l := left.(type) {
	case NilValue:
		_, ok := right.(NilValue)
		return ok
	case BoolValue:
		if r, ok := right.(BoolValue); ok {
			return l.Val == r.Val
		}
	case NumberValue:
		if r, ok := right.(NumberValue); ok {
			return l.Val == r.Val
		}
	case StringValue:
		if r, ok := right.(StringValue); ok {
			return l.Val == r.Val
		}
	}
	return false
}

// Placeholder implementations for new EYG visitor methods
func (e *Evaluator) VisitRecord(expr *Record) Value {
	return ErrorValue{Message: "Record not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitEmptyRecord(expr *EmptyRecord) Value {
	return NilValue{}
}

func (e *Evaluator) VisitList(expr *List) Value {
	return ErrorValue{Message: "List not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitAccess(expr *Access) Value {
	return ErrorValue{Message: "Access not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitBuiltin(expr *Builtin) Value {
	return ErrorValue{Message: "Builtin not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitUnion(expr *Union) Value {
	value := e.Evaluate(expr.Value)
	if _, ev := value.(ErrorValue); ev {
		return value
	}
	return UnionValue{Constructor: expr.Constructor, Value: value}
}

func (e *Evaluator) VisitLambda(expr *Lambda) Value {
	return ErrorValue{Message: "Lambda not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitMatch(expr *Match) Value {
	return ErrorValue{Message: "Match not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitPerform(expr *Perform) Value {
	return ErrorValue{Message: "Perform not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitHandle(expr *Handle) Value {
	return ErrorValue{Message: "Handle not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitNamedRef(expr *NamedRef) Value {
	return ErrorValue{Message: "NamedRef not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitThunk(expr *Thunk) Value {
	return ErrorValue{Message: "Thunk not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitSpread(expr *Spread) Value {
	return ErrorValue{Message: "Spread not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitDestructure(expr *Destructure) Value {
	return ErrorValue{Message: "Destructure not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitSeq(expr *Seq) Value {
	return ErrorValue{Message: "Seq not implemented", Line: expr.Line}
}
