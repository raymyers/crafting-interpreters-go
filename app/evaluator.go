package main

import (
	"fmt"
	"io"
	"strconv"
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

// NewDefaultScope creates a new scope with default built-in effects
func NewDefaultScope(output io.Writer) *Scope {
	scope := NewScope(nil)

	// Define the Log effect
	logEffect := LambdaValue{
		Parameters: []string{"value"},
		Body:       nil, // Builtin function, no body
		Closure:    nil,
		Builtin: func(args []Value) Value {
			if len(args) != 1 {
				return ErrorValue{Message: "Log expects exactly 1 argument", Line: 0}
			}

			// Print the value
			fmt.Fprintf(output, "%s\n", formatValue(args[0]))

			// Return empty record (unit value)
			return RecordValue{Fields: make(map[string]Value)}
		},
	}

	scope.define("Log", logEffect)
	return scope
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

func valuesEqual(a, b Value) bool {
	switch va := a.(type) {
	case NumberValue:
		if vb, ok := b.(NumberValue); ok {
			return va.Val == vb.Val
		}
	case StringValue:
		if vb, ok := b.(StringValue); ok {
			return va.Val == vb.Val
		}
	case BoolValue:
		if vb, ok := b.(BoolValue); ok {
			return va.Val == vb.Val
		}
	case NilValue:
		_, ok := b.(NilValue)
		return ok
	case UnionValue:
		if vb, ok := b.(UnionValue); ok {
			return va.Constructor == vb.Constructor && valuesEqual(va.Value, vb.Value)
		}
	}
	return false
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
	case NOT:
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
		} else if lv, ok := lookup.(LambdaValue); ok {
			// Handle lambda function call
			if len(lv.Parameters) != len(expr.Arguments) {
				return ErrorValue{
					Message: fmt.Sprintf("Expected %d arguments but got %d", len(lv.Parameters), len(expr.Arguments)),
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

			// Check if this is a builtin function
			if lv.Builtin != nil {
				return lv.Builtin(argValues)
			}

			// Create new scope for lambda execution (based on closure)
			previousScope := e.scope
			e.scope = NewScope(lv.Closure)

			// Bind parameters to arguments in the new scope
			for i, paramName := range lv.Parameters {
				e.scope.define(paramName, argValues[i])
			}

			// Execute lambda body
			result := e.Evaluate(lv.Body)

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

	// Handle different types of callable values
	if fv, ok := callee.(FunValue); ok {
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
	} else if lv, ok := callee.(LambdaValue); ok {
		// Handle lambda function call
		if len(lv.Parameters) != len(expr.Arguments) {
			return ErrorValue{
				Message: fmt.Sprintf("Expected %d arguments but got %d", len(lv.Parameters), len(expr.Arguments)),
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

		// Check if this is a builtin function
		if lv.Builtin != nil {
			return lv.Builtin(argValues)
		}

		// Create new scope for lambda execution (based on closure)
		previousScope := e.scope
		e.scope = NewScope(lv.Closure)

		// Bind parameters to arguments in the new scope
		for i, paramName := range lv.Parameters {
			e.scope.define(paramName, argValues[i])
		}

		// Execute lambda body
		result := e.Evaluate(lv.Body)

		// Restore previous scope
		e.scope = previousScope
		return result
	} else {
		return ErrorValue{Message: "cannot call a non-function", Line: expr.Line}
	}
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
	fields := make(map[string]Value)

	// First pass: process all spread fields
	for _, field := range expr.Fields {
		if field.Name == "" {
			// This is a spread field
			if spread, ok := field.Value.(*Spread); ok {
				// Evaluate the spread expression
				spreadValue := e.Evaluate(spread.Expression)
				if _, ev := spreadValue.(ErrorValue); ev {
					return spreadValue
				}

				// Spread must be a record
				if record, ok := spreadValue.(RecordValue); ok {
					// Add all fields from the spread record
					for name, value := range record.Fields {
						fields[name] = value
					}
				} else {
					return ErrorValue{Message: "Can only spread records", Line: spread.Line}
				}
			}
		}
	}

	// Second pass: process explicit fields (these override spread fields)
	for _, field := range expr.Fields {
		if field.Name != "" {
			// Regular field
			value := e.Evaluate(field.Value)
			if _, ev := value.(ErrorValue); ev {
				return value
			}
			fields[field.Name] = value
		}
	}

	return RecordValue{Fields: fields}
}

func (e *Evaluator) VisitEmptyRecord(expr *EmptyRecord) Value {
	return NilValue{}
}

func (e *Evaluator) VisitList(expr *List) Value {
	var elements []Value
	for _, element := range expr.Elements {
		if spread, ok := element.(*Spread); ok {
			// Handle spread operator
			spreadValue := e.Evaluate(spread.Expression)
			if _, ev := spreadValue.(ErrorValue); ev {
				return spreadValue
			}
			if list, ok := spreadValue.(ListValue); ok {
				elements = append(elements, list.Elements...)
			} else {
				return ErrorValue{Message: "Can only spread lists", Line: spread.Line}
			}
		} else {
			value := e.Evaluate(element)
			if _, ev := value.(ErrorValue); ev {
				return value
			}
			elements = append(elements, value)
		}
	}
	return ListValue{Elements: elements}
}

func (e *Evaluator) VisitAccess(expr *Access) Value {
	object := e.Evaluate(expr.Object)
	if _, ev := object.(ErrorValue); ev {
		return object
	}

	if record, ok := object.(RecordValue); ok {
		if value, exists := record.Fields[expr.Name]; exists {
			return value
		}
		return ErrorValue{Message: "Undefined property '" + expr.Name + "'", Line: expr.Line}
	}

	return ErrorValue{Message: "Only records have properties", Line: expr.Line}
}

func (e *Evaluator) VisitBuiltin(expr *Builtin) Value {
	switch expr.Name {
	case "list_fold":
		if len(expr.Arguments) != 3 {
			return ErrorValue{Message: "list_fold expects 3 arguments", Line: expr.Line}
		}

		// Evaluate list
		listValue := e.Evaluate(expr.Arguments[0])
		if _, ev := listValue.(ErrorValue); ev {
			return listValue
		}
		list, ok := listValue.(ListValue)
		if !ok {
			return ErrorValue{Message: "First argument to list_fold must be a list", Line: expr.Line}
		}

		// Evaluate initial value
		accumulator := e.Evaluate(expr.Arguments[1])
		if _, ev := accumulator.(ErrorValue); ev {
			return accumulator
		}

		// Evaluate function
		funcValue := e.Evaluate(expr.Arguments[2])
		if _, ev := funcValue.(ErrorValue); ev {
			return funcValue
		}
		lambda, ok := funcValue.(LambdaValue)
		if !ok {
			return ErrorValue{Message: "Third argument to list_fold must be a function", Line: expr.Line}
		}

		// Fold over the list
		for _, element := range list.Elements {
			// Call lambda with accumulator and element
			previousScope := e.scope
			e.scope = NewScope(lambda.Closure)

			// Bind parameters
			if len(lambda.Parameters) != 2 {
				e.scope = previousScope
				return ErrorValue{Message: "Fold function must take exactly 2 parameters", Line: expr.Line}
			}
			e.scope.define(lambda.Parameters[0], accumulator)
			e.scope.define(lambda.Parameters[1], element)

			// Execute lambda body
			result := e.Evaluate(lambda.Body)
			e.scope = previousScope

			if _, ev := result.(ErrorValue); ev {
				return result
			}
			accumulator = result
		}

		return accumulator

	case "int_parse":
		if len(expr.Arguments) != 1 {
			return ErrorValue{Message: "int_parse expects 1 argument", Line: expr.Line}
		}

		// Evaluate string argument
		strValue := e.Evaluate(expr.Arguments[0])
		if _, ev := strValue.(ErrorValue); ev {
			return strValue
		}
		str, ok := strValue.(StringValue)
		if !ok {
			return ErrorValue{Message: "int_parse expects a string argument", Line: expr.Line}
		}

		// Parse the string to integer
		if val, err := strconv.ParseFloat(str.Val, 64); err == nil {
			return NumberValue{Val: val}
		} else {
			return ErrorValue{Message: "Cannot parse string as integer", Line: expr.Line}
		}

	case "clock":
		if len(expr.Arguments) != 1 {
			return ErrorValue{Message: "clock expects 1 argument (empty record)", Line: expr.Line}
		}

		// Evaluate the argument (should be an empty record)
		argValue := e.Evaluate(expr.Arguments[0])
		if _, ev := argValue.(ErrorValue); ev {
			return argValue
		}

		// Check if it's an empty record (NilValue)
		if _, ok := argValue.(NilValue); !ok {
			return ErrorValue{Message: "clock expects an empty record argument", Line: expr.Line}
		}

		epochSeconds := float64(time.Now().Unix())
		return NumberValue{Val: epochSeconds}

	default:
		return ErrorValue{Message: fmt.Sprintf("Unknown builtin function: %s", expr.Name), Line: expr.Line}
	}
}

func (e *Evaluator) VisitUnion(expr *Union) Value {
	value := e.Evaluate(expr.Value)
	if _, ev := value.(ErrorValue); ev {
		return value
	}
	return UnionValue{Constructor: expr.Constructor, Value: value}
}

func (e *Evaluator) VisitLambda(expr *Lambda) Value {
	return LambdaValue{
		Parameters: expr.Parameters,
		Body:       expr.Body,
		Closure:    e.scope,
	}
}

func (e *Evaluator) VisitMatch(expr *Match) Value {
	return ErrorValue{Message: "Match not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitPerform(expr *Perform) Value {
	// Look up the effect in the current scope
	effectValue, ok := e.scope.lookup(expr.Effect)
	if !ok {
		return ErrorValue{Message: fmt.Sprintf("Undefined effect '%s'", expr.Effect), Line: expr.Line}
	}

	// The effect should be a lambda function
	lambda, ok := effectValue.(LambdaValue)
	if !ok {
		return ErrorValue{Message: fmt.Sprintf("'%s' is not an effect function", expr.Effect), Line: expr.Line}
	}

	// Check argument count
	if len(expr.Arguments) != len(lambda.Parameters) {
		return ErrorValue{
			Message: fmt.Sprintf("Effect '%s' expects %d arguments but got %d",
				expr.Effect, len(lambda.Parameters), len(expr.Arguments)),
			Line: expr.Line,
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

	// If it's a builtin effect, call it
	if lambda.Builtin != nil {
		return lambda.Builtin(argValues)
	}

	// Otherwise, execute the lambda with the arguments
	previousScope := e.scope
	e.scope = NewScope(lambda.Closure)

	// Bind parameters to arguments
	for i, paramName := range lambda.Parameters {
		e.scope.define(paramName, argValues[i])
	}

	// Execute lambda body
	result := e.Evaluate(lambda.Body)

	// Restore previous scope
	e.scope = previousScope
	return result
}

func (e *Evaluator) VisitHandle(expr *Handle) Value {
	// Evaluate the handler expression
	handlerValue := e.Evaluate(expr.Handler)
	if _, isError := handlerValue.(ErrorValue); isError {
		return handlerValue
	}

	// Create a new scope with the effect temporarily overridden
	previousScope := e.scope
	e.scope = NewScope(previousScope)

	// Define the effect in the new scope to override any existing definition
	e.scope.define(expr.Effect, handlerValue)

	// Evaluate the fallback expression with the new effect handler
	result := e.Evaluate(expr.Fallback)

	// Restore the previous scope
	e.scope = previousScope

	return result
}

func (e *Evaluator) VisitNamedRef(expr *NamedRef) Value {
	// For now, implement basic std library
	if expr.Module == "std" && expr.Index == 1 {
		// Create a std library with list.contains function
		// Use LambdaValue to represent the builtin function
		containsFunc := LambdaValue{
			Parameters: []string{"list", "item"},
			Body:       nil, // Special marker for builtin
			Closure:    nil,
			Builtin: func(args []Value) Value {
				if len(args) != 2 {
					return ErrorValue{Message: "contains expects 2 arguments", Line: expr.Line}
				}

				list, ok := args[0].(ListValue)
				if !ok {
					return falseValue()
				}

				target := args[1]
				for _, elem := range list.Elements {
					if valuesEqual(elem, target) {
						return trueValue()
					}
				}
				return falseValue()
			},
		}

		listRecord := RecordValue{
			Fields: map[string]Value{
				"contains": containsFunc,
			},
		}

		return RecordValue{
			Fields: map[string]Value{
				"list": listRecord,
			},
		}
	}

	return ErrorValue{Message: fmt.Sprintf("Unknown named reference @%s:%d", expr.Module, expr.Index), Line: expr.Line}
}

func (e *Evaluator) VisitThunk(expr *Thunk) Value {
	return ErrorValue{Message: "Thunk not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitSpread(expr *Spread) Value {
	// Spread is handled in the context where it's used (e.g., List, Record)
	// This should not be called directly
	return ErrorValue{Message: "Spread can only be used in lists or records", Line: expr.Line}
}

func (e *Evaluator) VisitDestructure(expr *Destructure) Value {
	return ErrorValue{Message: "Destructure not implemented", Line: expr.Line}
}

func (e *Evaluator) VisitSeq(expr *Seq) Value {
	return ErrorValue{Message: "Seq not implemented", Line: expr.Line}
}
