package main

import (
	"fmt"
	"io"
	"strconv"
	"strings"
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

// isDefinedInCurrentScope checks if a variable is defined only in the current scope
func (s *Scope) isDefinedInCurrentScope(name string) bool {
	_, exists := s.envMap[name]
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

// EffectHandler represents an active effect handler
type EffectHandler struct {
	EffectName string
	Handler    LambdaValue
	Line       uint
}

// Evaluator implements the visitor pattern to evaluate expressions
type Evaluator struct {
	scope            *Scope
	output           io.Writer
	effectHandlers   []EffectHandler // Stack of active effect handlers
	collectedEffects []EffectValue   // Effects collected during execution
}

// NewEvaluator creates a new evaluator with the given scope and output writer
func NewEvaluator(scope *Scope, output io.Writer) *Evaluator {
	return &Evaluator{
		scope:            scope,
		output:           output,
		effectHandlers:   make([]EffectHandler, 0),
		collectedEffects: make([]EffectValue, 0),
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
	result := expr.Accept(e)

	// Check if the result is an effect that can be handled
	if effect, isEffect := result.(EffectValue); isEffect {
		// First check for Log effect (built-in handler) - handle immediately
		if effect.Name == "Log" {
			if logHandler, exists := e.scope.lookup("Log"); exists {
				if lambda, ok := logHandler.(LambdaValue); ok {
					return e.callLambdaWithValues(lambda, effect.Arguments, 0)
				}
			}
		}

		// Only handle other effects that have a proper continuation
		// Effects without continuation should bubble up to be captured first
		if effect.Continuation.Body == nil {
			return effect
		}

		// Check the effect handler stack for matching handlers
		for i := len(e.effectHandlers) - 1; i >= 0; i-- {
			handler := e.effectHandlers[i]
			if handler.EffectName == effect.Name {
				// Found a matching handler - call it
				resumeFunc := LambdaValue{
					Parameters: []string{"value"},
					Builtin: func(args []Value) Value {
						if len(args) != 1 {
							return ErrorValue{Message: "resume expects 1 argument", Line: handler.Line}
						}

						// Execute the captured continuation
						// resumeValue := args[0] // TODO: Use this value in continuation

						// Save current scope and switch to continuation scope
						previousScope := e.scope
						e.scope = effect.Continuation.Scope

						// Execute the continuation body
						result := e.Evaluate(effect.Continuation.Body)

						// Restore previous scope
						e.scope = previousScope

						// Debug: Check if continuation produces another effect
						if _, isEffect := result.(EffectValue); isEffect {
							// If continuation produces another effect, propagate it
							return result
						}

						return result
					},
				}

				// Call the handler with (value, resume)
				handlerArgs := append(effect.Arguments, resumeFunc)
				return e.callLambdaWithValues(handler.Handler, handlerArgs, handler.Line)
			}
		}
	}

	return result
}

// VisitLiteralExpr evaluates literal expressions
func (e *Evaluator) VisitLiteralExpr(expr *Literal) Value {
	return expr.Value
}

// VisitBinaryExpr evaluates binary expressions
func (e *Evaluator) VisitBinaryExpr(expr *Binary) Value {
	if expr.Operator.Type == EQUAL {
		// Evaluate the right side first
		right := e.Evaluate(expr.Right)
		if _, ev := right.(ErrorValue); ev {
			return right
		}
		// Check if right side produces an effect
		if _, isEffect := right.(EffectValue); isEffect {
			return right // Propagate effect
		}

		// Handle different left-hand side patterns
		switch left := expr.Left.(type) {
		case *Variable:
			// Simple variable assignment - always define in current scope
			varName := left.Name.Lexeme
			e.scope.define(varName, right)
			return right

		case *Destructure:
			// Destructuring assignment
			// Right side must be a record
			record, ok := right.(RecordValue)
			if !ok {
				return ErrorValue{Message: "Cannot destructure non-record value", Line: expr.Line}
			}

			// Process each field in the destructure pattern
			for _, field := range left.Fields {
				// Get the value from the record
				value, exists := record.Fields[field.Name]
				if !exists {
					return ErrorValue{Message: fmt.Sprintf("Field '%s' not found in record", field.Name), Line: expr.Line}
				}

				// The field.Value should be a Variable that we bind to
				if varExpr, ok := field.Value.(*Variable); ok {
					varName := varExpr.Name.Lexeme
					e.scope.define(varName, value)
				} else {
					return ErrorValue{Message: "Destructure pattern must contain variables", Line: expr.Line}
				}
			}

			return right

		default:
			return ErrorValue{Message: "Left of = must be a variable or destructure pattern", Line: expr.Line}
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
	case EffectValue:
		return result // Propagate effects immediately
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
	for i, stmt := range statements {
		result = e.Evaluate(stmt)
		switch v := result.(type) {
		case ErrorValue:
			return result
		case EffectValue:
			// Capture continuation: remaining statements
			remainingStatements := statements[i+1:]
			if len(remainingStatements) > 0 {
				// Create a block with remaining statements as the continuation
				continuationBody := &Block{Statements: remainingStatements}
				v.Continuation = ContinuationValue{
					Scope: e.scope, // Capture current scope
					Body:  continuationBody,
				}
			} else {
				// No remaining statements, continuation returns NilValue
				v.Continuation = ContinuationValue{
					Scope: e.scope,
					Body:  &Literal{Value: NilValue{}},
				}
			}
			return v // Propagate effect with continuation
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
	if varExpr, ok := expr.Callee.(*Variable); ok {
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
			// Handle lambda function call with currying support
			return e.callLambda(lv, expr.Arguments, expr.Line)
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
		// Handle lambda function call with currying support
		return e.callLambda(lv, expr.Arguments, expr.Line)
	} else {
		return ErrorValue{Message: "cannot call a non-function", Line: expr.Line}
	}
}

// callLambda handles lambda function calls with currying support
// callLambdaWithValues calls a lambda with already-evaluated values
func (e *Evaluator) callLambdaWithValues(lv LambdaValue, argValues []Value, line uint) Value {
	// Get partial arguments
	partialArgs := lv.PartialArgs

	// Combine partial arguments with new arguments
	allArgs := append(partialArgs, argValues...)

	// Check if we have enough arguments to call the function
	if len(allArgs) < len(lv.Parameters) {
		// Not enough arguments - return a partially applied function
		remainingParams := lv.Parameters[len(allArgs):]
		return LambdaValue{
			Parameters:    lv.Parameters, // Keep original parameters
			Body:          lv.Body,
			Closure:       lv.Closure,
			Builtin:       lv.Builtin,
			PartialArgs:   allArgs,         // Store all arguments so far
			PartialParams: remainingParams, // Store remaining parameters
		}
	}

	// We have enough arguments - call the function
	if lv.Builtin != nil {
		return lv.Builtin(allArgs)
	}

	// Create new scope for function execution
	previousScope := e.scope
	e.scope = NewScope(lv.Closure)

	// Bind parameters to arguments
	for i, paramName := range lv.Parameters {
		e.scope.define(paramName, allArgs[i])
	}

	// Execute function body
	result := e.Evaluate(lv.Body)

	// Restore previous scope
	e.scope = previousScope

	return result
}

func (e *Evaluator) callLambda(lv LambdaValue, arguments []Expr, line uint) Value {
	// Get partial arguments
	partialArgs := lv.PartialArgs

	// Evaluate the new arguments
	newArgValues := make([]Value, len(arguments))
	for i, arg := range arguments {
		argValue := e.Evaluate(arg)
		if _, isError := argValue.(ErrorValue); isError {
			return argValue
		}
		newArgValues[i] = argValue
	}

	// Combine partial arguments with new arguments
	allArgs := append(partialArgs, newArgValues...)

	// Check if we have enough arguments to call the function
	if len(allArgs) < len(lv.Parameters) {
		// Not enough arguments - return a partially applied function
		remainingParams := lv.Parameters[len(allArgs):]
		return LambdaValue{
			Parameters:    lv.Parameters, // Keep original parameters
			Body:          lv.Body,
			Closure:       lv.Closure,
			Builtin:       lv.Builtin,
			PartialArgs:   allArgs,         // Store all arguments so far
			PartialParams: remainingParams, // Store remaining parameters
		}
	}

	// We have enough arguments - check for exact match or too many
	if len(allArgs) > len(lv.Parameters) {
		return ErrorValue{
			Message: fmt.Sprintf("Too many arguments: expected %d but got %d", len(lv.Parameters), len(allArgs)),
			Line:    line,
		}
	}

	// Check if this is a builtin function
	if lv.Builtin != nil {
		return lv.Builtin(allArgs)
	}

	// Create new scope for lambda execution (based on closure)
	previousScope := e.scope
	e.scope = NewScope(lv.Closure)

	// Bind parameters to arguments in the new scope
	for i, paramName := range lv.Parameters {
		e.scope.define(paramName, allArgs[i])
	}

	// Execute lambda body
	result := e.Evaluate(lv.Body)

	// Restore previous scope
	e.scope = previousScope
	return result
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
	return RecordValue{Fields: make(map[string]Value)}
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
		// list_fold takes 3 arguments: list, initial value, fold function
		return LambdaValue{
			Parameters: []string{"list", "init", "fn"},
			Builtin: func(args []Value) Value {
				if len(args) != 3 {
					return ErrorValue{Message: "list_fold expects 3 arguments", Line: expr.Line}
				}

				list, ok := args[0].(ListValue)
				if !ok {
					return ErrorValue{Message: "First argument to list_fold must be a list", Line: expr.Line}
				}

				accumulator := args[1]

				lambda, ok := args[2].(LambdaValue)
				if !ok {
					return ErrorValue{Message: "Third argument to list_fold must be a function", Line: expr.Line}
				}

				// Fold over the list
				for _, element := range list.Elements {
					// Call lambda with accumulator and element
					result := e.callLambdaWithValues(lambda, []Value{accumulator, element}, expr.Line)
					if _, ev := result.(ErrorValue); ev {
						return result
					}
					accumulator = result
				}

				return accumulator
			},
		}

	case "int_parse":
		// int_parse takes 1 argument: string
		return LambdaValue{
			Parameters: []string{"str"},
			Builtin: func(args []Value) Value {
				if len(args) != 1 {
					return ErrorValue{Message: "int_parse expects 1 argument", Line: expr.Line}
				}

				str, ok := args[0].(StringValue)
				if !ok {
					return ErrorValue{Message: "int_parse expects a string argument", Line: expr.Line}
				}

				// Parse the string to integer
				if val, err := strconv.ParseFloat(str.Val, 64); err == nil {
					// Return Ok(value) union type
					return UnionValue{Constructor: "Ok", Value: NumberValue{Val: val}}
				} else {
					// Return Error(message) union type
					return UnionValue{Constructor: "Error", Value: StringValue{Val: err.Error()}}
				}
			},
		}

	case "clock":
		// clock takes 1 argument: empty record
		return LambdaValue{
			Parameters: []string{"_"},
			Builtin: func(args []Value) Value {
				if len(args) != 1 {
					return ErrorValue{Message: "clock expects 1 argument", Line: expr.Line}
				}

				// Check if it's an empty record
				if _, ok := args[0].(RecordValue); ok {
					epochSeconds := float64(time.Now().Unix())
					return NumberValue{Val: epochSeconds}
				}

				return ErrorValue{Message: "clock expects an empty record argument", Line: expr.Line}
			},
		}

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
		Parameters:    expr.Parameters,
		Body:          expr.Body,
		Closure:       e.scope,
		Builtin:       nil,
		PartialArgs:   nil,
		PartialParams: nil,
	}
}

func (e *Evaluator) VisitMatch(expr *Match) Value {
	// Evaluate the value to match against
	value := e.Evaluate(expr.Value)
	if errorVal, ok := value.(ErrorValue); ok {
		return errorVal
	}

	// Try each case in order
	for _, matchCase := range expr.Cases {
		bindings, matches := e.matchPattern(matchCase.Pattern, value)
		if matches {
			// Create new scope with pattern bindings
			e.scope = NewScope(e.scope)
			for name, val := range bindings {
				e.scope.define(name, val)
			}

			// Evaluate the body
			result := e.Evaluate(matchCase.Body)

			// Restore previous scope
			e.scope = e.scope.parent

			return result
		}
	}

	return ErrorValue{Message: "No matching pattern found", Line: expr.Line}
}

// matchPattern attempts to match a pattern against a value
// Returns (bindings, matches) where bindings is a map of variable names to values
func (e *Evaluator) matchPattern(pattern Expr, value Value) (map[string]Value, bool) {
	bindings := make(map[string]Value)

	switch p := pattern.(type) {
	case *Wildcard:
		// Wildcard matches anything
		return bindings, true

	case *Variable:
		// Variable pattern binds the value to the variable name
		bindings[p.Name.Lexeme] = value
		return bindings, true

	case *Union:
		// Constructor pattern: Constructor(params)
		if unionVal, ok := value.(UnionValue); ok {
			// Check if constructors match
			if p.Constructor == unionVal.Constructor {
				// Extract parameters from the pattern
				if varPattern, ok := p.Value.(*Variable); ok {
					paramNames := strings.Split(varPattern.Name.Lexeme, ",")

					// Handle empty parameter list
					if len(paramNames) == 1 && paramNames[0] == "" {
						return bindings, true
					}

					// For single parameter patterns, bind directly
					if len(paramNames) == 1 && paramNames[0] != "_" {
						bindings[paramNames[0]] = unionVal.Value
						return bindings, true
					}

					// For multiple parameters, we'd need to destructure
					// For now, handle simple cases
					if len(paramNames) == 1 && paramNames[0] == "_" {
						// Wildcard parameter, don't bind
						return bindings, true
					}
				}
				return bindings, true
			}
		}
		return bindings, false

	default:
		// Unknown pattern type
		return bindings, false
	}
}

func (e *Evaluator) VisitPerform(expr *Perform) Value {
	// Evaluate arguments
	argValues := make([]Value, len(expr.Arguments))
	for i, arg := range expr.Arguments {
		argValue := e.Evaluate(arg)
		if _, isError := argValue.(ErrorValue); isError {
			return argValue
		}
		argValues[i] = argValue
	}

	// Create an effect that will bubble up to be caught by a handler
	// The continuation will be set when the effect bubbles up through evalStatements
	return EffectValue{
		Name:         expr.Effect,
		Arguments:    argValues,
		Continuation: ContinuationValue{}, // Empty continuation, will be set later
	}
}

func (e *Evaluator) VisitHandle(expr *Handle) Value {
	// Evaluate the handler expression
	handlerValue := e.Evaluate(expr.Handler)
	if _, isError := handlerValue.(ErrorValue); isError {
		return handlerValue
	}

	// Convert handler to LambdaValue
	handler, ok := handlerValue.(LambdaValue)
	if !ok {
		return ErrorValue{Message: "Handler must be a function", Line: expr.Line}
	}

	// Push the handler onto the effect handler stack
	effectHandler := EffectHandler{
		EffectName: expr.Effect,
		Handler:    handler,
		Line:       expr.Line,
	}
	e.effectHandlers = append(e.effectHandlers, effectHandler)

	// Evaluate the fallback expression with the handler active
	fallbackValue := e.Evaluate(expr.Fallback)
	if _, isError := fallbackValue.(ErrorValue); isError {
		// Pop the handler before returning error
		e.effectHandlers = e.effectHandlers[:len(e.effectHandlers)-1]
		return fallbackValue
	}

	// If the fallback is a lambda, call it with unit argument
	var result Value
	if lambda, isLambda := fallbackValue.(LambdaValue); isLambda {
		// Call the fallback lambda with unit argument
		unitArg := RecordValue{Fields: make(map[string]Value)}
		result = e.callLambdaWithValues(lambda, []Value{unitArg}, expr.Line)
	} else {
		result = fallbackValue
	}

	// Pop the handler from the stack
	e.effectHandlers = e.effectHandlers[:len(e.effectHandlers)-1]

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
	// Evaluate left expression first
	leftResult := e.Evaluate(expr.Left)

	// If left produces an error or effect, propagate it immediately
	if _, isError := leftResult.(ErrorValue); isError {
		return leftResult
	}
	if _, isEffect := leftResult.(EffectValue); isEffect {
		return leftResult
	}

	// Then evaluate right expression
	rightResult := e.Evaluate(expr.Right)

	// Return the result of the right expression (sequence returns last value)
	return rightResult
}

func (e *Evaluator) VisitWildcard(expr *Wildcard) Value {
	// Wildcards are only used in patterns, not as expressions
	return ErrorValue{Message: "Wildcard can only be used in match patterns", Line: expr.Line}
}
