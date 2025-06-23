package main

import "fmt"

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
	scope *Scope
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
			}
			return ErrorValue{Message: "Assigned variable must be defined", Line: expr.Line}
		} else {
			return ErrorValue{Message: "Left of = must be a variable", Line: expr.Line}
		}

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
				return BoolValue{Val: leftNum.Val < rightNum.Val}
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case LESS_EQUAL:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return BoolValue{Val: leftNum.Val <= rightNum.Val}
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case GREATER:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return BoolValue{Val: leftNum.Val > rightNum.Val}
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case GREATER_EQUAL:
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return BoolValue{Val: leftNum.Val >= rightNum.Val}
			}
		}
		return ErrorValue{Message: "Operands must be numbers", Line: expr.Line}
	case EQUAL_EQUAL:
		return BoolValue{Val: isEqual(left, right)}
	case BANG_EQUAL:
		return BoolValue{Val: !isEqual(left, right)}
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
		return BoolValue{Val: !isTruthy(right)}
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
		fmt.Printf("%s\n", formatValue(result))
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

	var result Value = NilValue{}
	for _, stmt := range expr.Statements {
		result = e.Evaluate(stmt)
		switch result.(type) {
		case ErrorValue:
			// Restore previous scope on error
			e.scope = previousScope
			return result
		}
	}

	// Restore previous scope (block scoping)
	e.scope = previousScope
	return result
}

// isTruthy determines the truthiness of a value following Lox rules
func isTruthy(value Value) bool {
	switch v := value.(type) {
	case NilValue:
		return false
	case BoolValue:
		return v.Val
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
