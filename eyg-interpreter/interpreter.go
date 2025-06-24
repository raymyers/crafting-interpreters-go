package main

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Expression type constants
const (
	VAR     = "v"
	LAMBDA  = "f"
	APPLY   = "a"
	LET     = "l"
	VACANT  = "z"
	BINARY  = "x"
	INT     = "i"
	STRING  = "s"
	TAIL    = "ta"
	CONS    = "c"
	EMPTY   = "u"
	EXTEND  = "e"
	SELECT  = "g"
	OVERWRITE = "o"
	TAG     = "t"
	CASE    = "m"
	NOCASES = "n"
	PERFORM = "p"
	HANDLE  = "h"
	BUILTIN = "b"
)

// Value represents any value in the EYG language
type Value interface{}

// Expression represents an expression in the EYG language
type Expression map[string]interface{}

// Environment maps variable names to values
type Environment map[string]Value

// Stack represents the continuation stack
type Stack []Continuation

// Continuation represents a continuation on the stack
type Continuation interface {
	isContinuation()
}

// State represents the interpreter state
type State struct {
	Control Value       // Current expression or value
	Env     Environment // Variable environment
	Stack   Stack       // Continuation stack
	IsValue bool        // Whether control is a value or expression
	Break   interface{} // Error/break condition
}

// Closure represents a function closure
type Closure struct {
	Lambda   Expression
	Captured Environment
}

// Partial represents a partially applied function
type Partial struct {
	Exp     Expression
	Applied []Value
	Impl    func(*State, ...Value)
}

// Tagged represents a tagged value
type Tagged struct {
	Tag   string
	Value Value
}

// Effect represents an unhandled effect
type Effect struct {
	Label string
	Lift  Value
}

// Resume represents a resumption continuation
type Resume struct {
	Reversed Stack
}

// Continuation types
type ArgCont struct {
	Arg Expression
	Env Environment
}

func (a ArgCont) isContinuation() {}

type ApplyCont struct {
	Func Value
	Env  Environment
}

func (a ApplyCont) isContinuation() {}

type CallCont struct {
	Arg Value
	Env Environment
}

func (c CallCont) isContinuation() {}

type AssignCont struct {
	Label string
	Then  Expression
	Env   Environment
}

func (a AssignCont) isContinuation() {}

type DelimitCont struct {
	Label  string
	Handle Value
}

func (d DelimitCont) isContinuation() {}

// NewState creates a new interpreter state
func NewState(src Expression) *State {
	return &State{
		Control: src,
		Env:     make(Environment),
		Stack:   make(Stack, 0),
		IsValue: false,
	}
}

// SetValue sets the control to a value
func (s *State) SetValue(value Value) {
	s.IsValue = true
	s.Control = value
}

// SetExpression sets the control to an expression
func (s *State) SetExpression(expr Expression) {
	s.IsValue = false
	s.Control = expr
}

// GetVariable gets a variable from the environment
func (s *State) GetVariable(label string) Value {
	if value, ok := s.Env[label]; ok {
		return value
	}
	s.Break = map[string]interface{}{"UndefinedVariable": label}
	return nil
}

// Push pushes a continuation onto the stack
func (s *State) Push(cont Continuation) {
	s.Stack = append(s.Stack, cont)
}

// Pop pops a continuation from the stack
func (s *State) Pop() Continuation {
	if len(s.Stack) == 0 {
		return nil
	}
	cont := s.Stack[len(s.Stack)-1]
	s.Stack = s.Stack[:len(s.Stack)-1]
	return cont
}

// Step performs one step of evaluation
func (s *State) Step() {
	if s.IsValue {
		s.apply()
	} else {
		s.eval()
	}
}

// eval evaluates an expression
func (s *State) eval() {
	expr, ok := s.Control.(Expression)
	if !ok {
		s.Break = fmt.Errorf("expected expression, got %T", s.Control)
		return
	}

	exprType, ok := expr["0"].(string)
	if !ok {
		s.Break = fmt.Errorf("expression missing type field")
		return
	}

	switch exprType {
	case VAR:
		label, ok := expr["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("variable missing label")
			return
		}
		s.SetValue(s.GetVariable(label))

	case LAMBDA:
		s.SetValue(&Closure{Lambda: expr, Captured: s.copyEnv()})

	case APPLY:
		arg, ok := expr["a"]
		if !ok {
			s.Break = fmt.Errorf("apply missing argument")
			return
		}
		f, ok := expr["f"]
		if !ok {
			s.Break = fmt.Errorf("apply missing function")
			return
		}
		argExpr := s.toExpression(arg)
		fExpr := s.toExpression(f)
		s.Push(ArgCont{Arg: argExpr, Env: s.copyEnv()})
		s.SetExpression(fExpr)

	case LET:
		label, ok := expr["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("let missing label")
			return
		}
		then, ok := expr["t"]
		if !ok {
			s.Break = fmt.Errorf("let missing then")
			return
		}
		value, ok := expr["v"]
		if !ok {
			s.Break = fmt.Errorf("let missing value")
			return
		}
		thenExpr := s.toExpression(then)
		valueExpr := s.toExpression(value)
		s.Push(AssignCont{Label: label, Then: thenExpr, Env: s.copyEnv()})
		s.SetExpression(valueExpr)

	case VACANT:
		s.Break = map[string]interface{}{"NotImplemented": ""}

	case BINARY:
		v, ok := expr["v"]
		if !ok {
			s.Break = fmt.Errorf("binary missing value")
			return
		}
		s.SetValue(v)

	case INT:
		v, ok := expr["v"]
		if !ok {
			s.Break = fmt.Errorf("int missing value")
			return
		}
		s.SetValue(v)

	case STRING:
		v, ok := expr["v"]
		if !ok {
			s.Break = fmt.Errorf("string missing value")
			return
		}
		s.SetValue(v)

	case TAIL:
		s.SetValue([]Value{})

	case CONS:
		s.SetValue(&Partial{Exp: expr, Applied: []Value{}, Impl: func(s *State, args ...Value) { s.cons(args...) }})

	case EMPTY:
		s.SetValue(make(map[string]Value))

	case EXTEND, OVERWRITE:
		label, ok := expr["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("extend/overwrite missing label")
			return
		}
		s.SetValue(&Partial{Exp: expr, Applied: []Value{}, Impl: s.extend(label)})

	case SELECT:
		label, ok := expr["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("select missing label")
			return
		}
		s.SetValue(&Partial{Exp: expr, Applied: []Value{}, Impl: s.selectField(label)})

	case TAG:
		label, ok := expr["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("tag missing label")
			return
		}
		s.SetValue(&Partial{Exp: expr, Applied: []Value{}, Impl: s.tag(label)})

	case CASE:
		label, ok := expr["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("case missing label")
			return
		}
		s.SetValue(&Partial{Exp: expr, Applied: []Value{}, Impl: s.caseMatch(label)})

	case NOCASES:
		s.SetValue(&Partial{Exp: expr, Applied: []Value{}, Impl: func(s *State, args ...Value) { s.nocases(args...) }})

	case PERFORM:
		label, ok := expr["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("perform missing label")
			return
		}
		s.SetValue(&Partial{Exp: expr, Applied: []Value{}, Impl: s.perform(label)})

	case HANDLE:
		label, ok := expr["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("handle missing label")
			return
		}
		s.SetValue(&Partial{Exp: expr, Applied: []Value{}, Impl: s.handle(label)})

	case BUILTIN:
		label, ok := expr["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("builtin missing label")
			return
		}
		builtin := s.getBuiltin(label)
		if builtin == nil {
			s.Break = map[string]interface{}{"UndefinedBuiltin": label}
			return
		}
		s.SetValue(&Partial{Exp: expr, Applied: []Value{}, Impl: builtin})

	default:
		s.Break = fmt.Errorf("unrecognized expression type: %s", exprType)
	}
}

// apply applies a value to continuations
func (s *State) apply() {
	value := s.Control
	cont := s.Pop()
	if cont == nil {
		return // Done
	}

	switch c := cont.(type) {
	case AssignCont:
		s.Env[c.Label] = value
		s.SetExpression(c.Then)

	case ArgCont:
		s.Push(ApplyCont{Func: value, Env: c.Env})
		s.Env = c.Env
		s.SetExpression(c.Arg)

	case ApplyCont:
		s.Env = c.Env
		s.call(c.Func, value)

	case CallCont:
		s.Env = c.Env
		s.call(value, c.Arg)

	case DelimitCont:
		// Handle delimit continuation
		break

	default:
		s.Break = fmt.Errorf("invalid continuation type: %T", cont)
	}
}

// call calls a function with an argument
func (s *State) call(fn Value, arg Value) {
	switch f := fn.(type) {
	case *Closure:
		label, ok := f.Lambda["l"].(string)
		if !ok {
			s.Break = fmt.Errorf("lambda missing parameter label")
			return
		}
		body, ok := f.Lambda["b"]
		if !ok {
			s.Break = fmt.Errorf("lambda missing body")
			return
		}
		s.Env = s.copyEnvFrom(f.Captured)
		s.Env[label] = arg
		bodyExpr := s.toExpression(body)
		s.SetExpression(bodyExpr)

	case *Partial:
		applied := append(f.Applied, arg)
		// Determine required argument count based on the expression type
		requiredArgs := s.getRequiredArgs(f.Exp)
		if len(applied) >= requiredArgs {
			f.Impl(s, applied...)
		} else {
			s.SetValue(&Partial{Exp: f.Exp, Applied: applied, Impl: f.Impl})
		}

	case *Resume:
		// Handle resume
		for i := len(f.Reversed) - 1; i >= 0; i-- {
			s.Push(f.Reversed[i])
		}
		s.SetValue(arg)

	default:
		s.Break = fmt.Errorf("not a function: %T", fn)
	}
}

// Loop runs the interpreter until completion
func (s *State) Loop() Value {
	for {
		s.Step()
		if s.Break != nil || (s.IsValue && len(s.Stack) == 0) {
			return s.Control
		}
	}
}

// Resume resumes execution with a value
func (s *State) Resume(value Value) {
	s.SetValue(value)
	s.Break = nil
	s.Loop()
}

// Helper functions
func (s *State) copyEnv() Environment {
	env := make(Environment)
	for k, v := range s.Env {
		env[k] = v
	}
	return env
}

// toExpression converts an interface{} to Expression
func (s *State) toExpression(v interface{}) Expression {
	if expr, ok := v.(Expression); ok {
		return expr
	}
	if m, ok := v.(map[string]interface{}); ok {
		return Expression(m)
	}
	s.Break = fmt.Errorf("cannot convert %T to Expression", v)
	return nil
}

// getRequiredArgs returns the number of arguments required for a partial application
func (s *State) getRequiredArgs(expr Expression) int {
	exprType, ok := expr["0"].(string)
	if !ok {
		return 1
	}
	
	switch exprType {
	case CONS, EXTEND, OVERWRITE:
		return 2
	case SELECT, TAG, PERFORM:
		return 1
	case CASE:
		return 3
	case HANDLE:
		return 2
	case BUILTIN:
		label, ok := expr["l"].(string)
		if !ok {
			return 1
		}
		return s.getBuiltinArgCount(label)
	default:
		return 1
	}
}

// getBuiltinArgCount returns the number of arguments required for a builtin
func (s *State) getBuiltinArgCount(name string) int {
	argCounts := map[string]int{
		"equal":        2,
		"fix":          1,
		"fixed":        2,
		"int_compare":  2,
		"int_add":      2,
		"int_subtract": 2,
		"int_multiply": 2,
		"int_divide":   2,
		"int_absolute": 1,
		"int_parse":    1,
		"int_to_string": 1,
		"string_append": 2,
		"string_split": 2,
		"string_split_once": 2,
		"string_replace": 3,
		"string_uppercase": 1,
		"string_lowercase": 1,
		"string_ends_with": 2,
		"string_starts_with": 2,
		"string_length": 1,
		"list_pop": 1,
		"list_fold": 3,
		"string_to_binary": 1,
		"string_from_binary": 1,
		"binary_from_integers": 1,
		"binary_fold": 3,
	}
	
	if count, exists := argCounts[name]; exists {
		return count
	}
	return 1
}

func (s *State) copyEnvFrom(src Environment) Environment {
	env := make(Environment)
	for k, v := range src {
		env[k] = v
	}
	return env
}

// Builtin implementations (stubs for now)
func (s *State) cons(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("cons expects 2 arguments, got %d", len(args))
		return
	}
	item := args[0]
	tail, ok := args[1].([]Value)
	if !ok {
		s.Break = fmt.Errorf("cons tail must be a list")
		return
	}
	result := append([]Value{item}, tail...)
	s.SetValue(result)
}

func (s *State) extend(label string) func(*State, ...Value) {
	return func(s *State, args ...Value) {
		if len(args) != 2 {
			s.Break = fmt.Errorf("extend expects 2 arguments, got %d", len(args))
			return
		}
		value := args[0]
		rest, ok := args[1].(map[string]Value)
		if !ok {
			s.Break = fmt.Errorf("extend rest must be a record")
			return
		}
		result := make(map[string]Value)
		for k, v := range rest {
			result[k] = v
		}
		result[label] = value
		s.SetValue(result)
	}
}

func (s *State) selectField(label string) func(*State, ...Value) {
	return func(s *State, args ...Value) {
		if len(args) != 1 {
			s.Break = fmt.Errorf("select expects 1 argument, got %d", len(args))
			return
		}
		record, ok := args[0].(map[string]Value)
		if !ok {
			s.Break = fmt.Errorf("select argument must be a record")
			return
		}
		if value, exists := record[label]; exists {
			s.SetValue(value)
		} else {
			s.Break = fmt.Errorf("missing label: %s", label)
		}
	}
}

func (s *State) tag(label string) func(*State, ...Value) {
	return func(s *State, args ...Value) {
		if len(args) != 1 {
			s.Break = fmt.Errorf("tag expects 1 argument, got %d", len(args))
			return
		}
		s.SetValue(&Tagged{Tag: label, Value: args[0]})
	}
}

func (s *State) caseMatch(label string) func(*State, ...Value) {
	return func(s *State, args ...Value) {
		if len(args) != 3 {
			s.Break = fmt.Errorf("case expects 3 arguments, got %d", len(args))
			return
		}
		branch := args[0]
		otherwise := args[1]
		value := args[2]
		
		if tagged, ok := value.(*Tagged); ok {
			if tagged.Tag == label {
				s.call(branch, tagged.Value)
			} else {
				s.call(otherwise, value)
			}
		} else {
			s.Break = fmt.Errorf("case value must be tagged")
		}
	}
}

func (s *State) nocases(args ...Value) {
	s.Break = fmt.Errorf("no cases matched")
}

func (s *State) perform(label string) func(*State, ...Value) {
	return func(s *State, args ...Value) {
		if len(args) != 1 {
			s.Break = fmt.Errorf("perform expects 1 argument, got %d", len(args))
			return
		}
		lift := args[0]
		s.Break = &Effect{Label: label, Lift: lift}
	}
}

func (s *State) handle(label string) func(*State, ...Value) {
	return func(s *State, args ...Value) {
		if len(args) != 2 {
			s.Break = fmt.Errorf("handle expects 2 arguments, got %d", len(args))
			return
		}
		handle := args[0]
		exec := args[1]
		s.Push(DelimitCont{Label: label, Handle: handle})
		s.call(exec, make(map[string]Value))
	}
}

func (s *State) getBuiltin(name string) func(*State, ...Value) {
	builtins := map[string]func(*State, ...Value){
		"equal":        func(s *State, args ...Value) { s.builtinEqual(args...) },
		"fix":          func(s *State, args ...Value) { s.builtinFix(args...) },
		"fixed":        func(s *State, args ...Value) { s.builtinFixed(args...) },
		"int_compare":  func(s *State, args ...Value) { s.builtinIntCompare(args...) },
		"int_add":      func(s *State, args ...Value) { s.builtinIntAdd(args...) },
		"int_subtract": func(s *State, args ...Value) { s.builtinIntSubtract(args...) },
		"int_multiply": func(s *State, args ...Value) { s.builtinIntMultiply(args...) },
		"int_divide":   func(s *State, args ...Value) { s.builtinIntDivide(args...) },
		"int_absolute": func(s *State, args ...Value) { s.builtinIntAbsolute(args...) },
		"int_parse":    func(s *State, args ...Value) { s.builtinIntParse(args...) },
		"int_to_string": func(s *State, args ...Value) { s.builtinIntToString(args...) },
		"string_append": func(s *State, args ...Value) { s.builtinStringAppend(args...) },
		"string_split": func(s *State, args ...Value) { s.builtinStringSplit(args...) },
		"string_split_once": func(s *State, args ...Value) { s.builtinStringSplitOnce(args...) },
		"string_replace": func(s *State, args ...Value) { s.builtinStringReplace(args...) },
		"string_uppercase": func(s *State, args ...Value) { s.builtinStringUppercase(args...) },
		"string_lowercase": func(s *State, args ...Value) { s.builtinStringLowercase(args...) },
		"string_ends_with": func(s *State, args ...Value) { s.builtinStringEndsWith(args...) },
		"string_starts_with": func(s *State, args ...Value) { s.builtinStringStartsWith(args...) },
		"string_length": func(s *State, args ...Value) { s.builtinStringLength(args...) },
		"list_pop": func(s *State, args ...Value) { s.builtinListPop(args...) },
		"list_fold": func(s *State, args ...Value) { s.builtinListFold(args...) },
		"string_to_binary": func(s *State, args ...Value) { s.builtinStringToBinary(args...) },
		"string_from_binary": func(s *State, args ...Value) { s.builtinStringFromBinary(args...) },
		"binary_from_integers": func(s *State, args ...Value) { s.builtinBinaryFromIntegers(args...) },
		"binary_fold": func(s *State, args ...Value) { s.builtinBinaryFold(args...) },
	}
	
	return builtins[name]
}

// Eval evaluates an expression and returns the final state
func Eval(src Expression) *State {
	state := NewState(src)
	state.Loop()
	return state
}

// Builtin function implementations
func (s *State) builtinEqual(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("equal expects 2 arguments, got %d", len(args))
		return
	}
	
	a, b := args[0], args[1]
	isEqual := s.valuesEqual(a, b)
	
	if isEqual {
		s.SetValue(&Tagged{Tag: "True", Value: make(map[string]Value)})
	} else {
		s.SetValue(&Tagged{Tag: "False", Value: make(map[string]Value)})
	}
}

func (s *State) builtinFix(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("fix expects 1 argument, got %d", len(args))
		return
	}
	
	builder := args[0]
	// Create a fixed point combinator
	// This is a simplified implementation
	s.Push(CallCont{Arg: &Partial{
		Exp: Expression{"0": BUILTIN, "l": "fixed"},
		Applied: []Value{builder},
		Impl: func(s *State, args ...Value) { s.builtinFixed(args...) },
	}, Env: s.copyEnv()})
	s.SetValue(builder)
}

func (s *State) builtinFixed(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("fixed expects 2 arguments, got %d", len(args))
		return
	}
	
	builder := args[0]
	arg := args[1]
	
	s.Push(CallCont{Arg: arg, Env: s.copyEnv()})
	s.Push(CallCont{Arg: &Partial{
		Exp: Expression{"0": BUILTIN, "l": "fixed"},
		Applied: []Value{builder},
		Impl: func(s *State, args ...Value) { s.builtinFixed(args...) },
	}, Env: s.copyEnv()})
	s.SetValue(builder)
}

func (s *State) builtinIntCompare(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("int_compare expects 2 arguments, got %d", len(args))
		return
	}
	
	a, okA := args[0].(float64)
	b, okB := args[1].(float64)
	if !okA || !okB {
		s.Break = fmt.Errorf("int_compare expects integer arguments")
		return
	}
	
	var result *Tagged
	if a < b {
		result = &Tagged{Tag: "Lt", Value: make(map[string]Value)}
	} else if a > b {
		result = &Tagged{Tag: "Gt", Value: make(map[string]Value)}
	} else {
		result = &Tagged{Tag: "Eq", Value: make(map[string]Value)}
	}
	
	s.SetValue(result)
}

func (s *State) builtinIntAdd(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("int_add expects 2 arguments, got %d", len(args))
		return
	}
	
	a, okA := args[0].(float64)
	b, okB := args[1].(float64)
	if !okA || !okB {
		s.Break = fmt.Errorf("int_add expects integer arguments")
		return
	}
	
	s.SetValue(a + b)
}

func (s *State) builtinIntSubtract(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("int_subtract expects 2 arguments, got %d", len(args))
		return
	}
	
	a, okA := args[0].(float64)
	b, okB := args[1].(float64)
	if !okA || !okB {
		s.Break = fmt.Errorf("int_subtract expects integer arguments")
		return
	}
	
	s.SetValue(a - b)
}

func (s *State) builtinIntMultiply(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("int_multiply expects 2 arguments, got %d", len(args))
		return
	}
	
	a, okA := args[0].(float64)
	b, okB := args[1].(float64)
	if !okA || !okB {
		s.Break = fmt.Errorf("int_multiply expects integer arguments")
		return
	}
	
	s.SetValue(a * b)
}

func (s *State) builtinIntDivide(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("int_divide expects 2 arguments, got %d", len(args))
		return
	}
	
	a, okA := args[0].(float64)
	b, okB := args[1].(float64)
	if !okA || !okB {
		s.Break = fmt.Errorf("int_divide expects integer arguments")
		return
	}
	
	if b == 0 {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
	} else {
		result := float64(int(a) / int(b)) // Integer division
		s.SetValue(&Tagged{Tag: "Ok", Value: result})
	}
}

func (s *State) builtinIntAbsolute(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("int_absolute expects 1 argument, got %d", len(args))
		return
	}
	
	a, ok := args[0].(float64)
	if !ok {
		s.Break = fmt.Errorf("int_absolute expects integer argument")
		return
	}
	
	if a < 0 {
		s.SetValue(-a)
	} else {
		s.SetValue(a)
	}
}

func (s *State) builtinIntParse(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("int_parse expects 1 argument, got %d", len(args))
		return
	}
	
	str, ok := args[0].(string)
	if !ok {
		s.Break = fmt.Errorf("int_parse expects string argument")
		return
	}
	
	// Try to parse as integer
	var n float64
	if _, err := fmt.Sscanf(str, "%f", &n); err != nil {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
		return
	}
	
	// Check if it's actually an integer (no decimal part)
	if n != float64(int(n)) {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
		return
	}
	
	// Check if the string representation matches exactly (no extra characters)
	expected := fmt.Sprintf("%.0f", n)
	if str != expected {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
		return
	}
	
	s.SetValue(&Tagged{Tag: "Ok", Value: n})
}

func (s *State) builtinIntToString(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("int_to_string expects 1 argument, got %d", len(args))
		return
	}
	
	a, ok := args[0].(float64)
	if !ok {
		s.Break = fmt.Errorf("int_to_string expects integer argument")
		return
	}
	
	result := fmt.Sprintf("%.0f", a)
	s.SetValue(result)
}

func (s *State) builtinStringAppend(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("string_append expects 2 arguments, got %d", len(args))
		return
	}
	
	a, okA := args[0].(string)
	b, okB := args[1].(string)
	if !okA || !okB {
		s.Break = fmt.Errorf("string_append expects string arguments")
		return
	}
	
	s.SetValue(a + b)
}

func (s *State) builtinStringSplit(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("string_split expects 2 arguments, got %d", len(args))
		return
	}
	
	str, okA := args[0].(string)
	sep, okB := args[1].(string)
	if !okA || !okB {
		s.Break = fmt.Errorf("string_split expects string arguments")
		return
	}
	
	result := make(map[string]Value)
	
	if sep == "" {
		// Special case: splitting on empty string means split into characters
		if str == "" {
			result["head"] = ""
			result["tail"] = []Value{}
		} else {
			runes := []rune(str)
			result["head"] = string(runes[0])
			tail := make([]Value, len(runes)-1)
			for i := 1; i < len(runes); i++ {
				tail[i-1] = string(runes[i])
			}
			result["tail"] = tail
		}
	} else {
		// Find the first occurrence of separator
		index := strings.Index(str, sep)
		
		if index == -1 {
			// Separator not found, entire string is head, empty tail
			result["head"] = str
			result["tail"] = []Value{}
		} else {
			// Split at first occurrence
			head := str[:index]
			remainder := str[index+len(sep):]
			
			result["head"] = head
			
			// Split the remainder by separator for tail
			if remainder == "" {
				result["tail"] = []Value{}
			} else {
				tailParts := strings.Split(remainder, sep)
				tail := make([]Value, len(tailParts))
				for i, part := range tailParts {
					tail[i] = part
				}
				result["tail"] = tail
			}
		}
	}
	
	s.SetValue(result)
}

func (s *State) builtinStringSplitOnce(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("string_split_once expects 2 arguments, got %d", len(args))
		return
	}
	
	str, okA := args[0].(string)
	sep, okB := args[1].(string)
	if !okA || !okB {
		s.Break = fmt.Errorf("string_split_once expects string arguments")
		return
	}
	
	idx := strings.Index(str, sep)
	if idx == -1 {
		// No split occurred - return Error
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
	} else {
		// Split at first occurrence - return Ok with record
		before := str[:idx]
		after := str[idx+len(sep):]
		
		record := make(map[string]Value)
		record["pre"] = before
		record["post"] = after
		
		s.SetValue(&Tagged{Tag: "Ok", Value: record})
	}
}

func (s *State) builtinStringReplace(args ...Value) {
	if len(args) != 3 {
		s.Break = fmt.Errorf("string_replace expects 3 arguments, got %d", len(args))
		return
	}
	
	str, okA := args[0].(string)
	old, okB := args[1].(string)
	new, okC := args[2].(string)
	if !okA || !okB || !okC {
		s.Break = fmt.Errorf("string_replace expects string arguments")
		return
	}
	
	result := strings.ReplaceAll(str, old, new)
	s.SetValue(result)
}

func (s *State) builtinStringUppercase(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("string_uppercase expects 1 argument, got %d", len(args))
		return
	}
	
	str, ok := args[0].(string)
	if !ok {
		s.Break = fmt.Errorf("string_uppercase expects string argument")
		return
	}
	
	s.SetValue(strings.ToUpper(str))
}

func (s *State) builtinStringLowercase(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("string_lowercase expects 1 argument, got %d", len(args))
		return
	}
	
	str, ok := args[0].(string)
	if !ok {
		s.Break = fmt.Errorf("string_lowercase expects string argument")
		return
	}
	
	s.SetValue(strings.ToLower(str))
}

func (s *State) builtinStringEndsWith(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("string_ends_with expects 2 arguments, got %d", len(args))
		return
	}
	
	str, okA := args[0].(string)
	suffix, okB := args[1].(string)
	if !okA || !okB {
		s.Break = fmt.Errorf("string_ends_with expects string arguments")
		return
	}
	
	if strings.HasSuffix(str, suffix) {
		s.SetValue(&Tagged{Tag: "True", Value: make(map[string]Value)})
	} else {
		s.SetValue(&Tagged{Tag: "False", Value: make(map[string]Value)})
	}
}

func (s *State) builtinStringStartsWith(args ...Value) {
	if len(args) != 2 {
		s.Break = fmt.Errorf("string_starts_with expects 2 arguments, got %d", len(args))
		return
	}
	
	str, okA := args[0].(string)
	prefix, okB := args[1].(string)
	if !okA || !okB {
		s.Break = fmt.Errorf("string_starts_with expects string arguments")
		return
	}
	
	if strings.HasPrefix(str, prefix) {
		s.SetValue(&Tagged{Tag: "True", Value: make(map[string]Value)})
	} else {
		s.SetValue(&Tagged{Tag: "False", Value: make(map[string]Value)})
	}
}

func (s *State) builtinStringLength(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("string_length expects 1 argument, got %d", len(args))
		return
	}
	
	a, ok := args[0].(string)
	if !ok {
		s.Break = fmt.Errorf("string_length expects string argument")
		return
	}
	
	s.SetValue(float64(len(a)))
}

func (s *State) builtinListPop(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("list_pop expects 1 argument, got %d", len(args))
		return
	}
	
	list, ok := args[0].([]Value)
	if !ok {
		s.Break = fmt.Errorf("list_pop expects list argument")
		return
	}
	
	if len(list) == 0 {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
	} else {
		head := list[0]
		tail := list[1:]
		result := make(map[string]Value)
		result["head"] = head
		result["tail"] = tail
		s.SetValue(&Tagged{Tag: "Ok", Value: result})
	}
}

func (s *State) builtinListFold(args ...Value) {
	if len(args) != 3 {
		s.Break = fmt.Errorf("list_fold expects 3 arguments, got %d", len(args))
		return
	}
	
	list, ok := args[0].([]Value)
	if !ok {
		s.Break = fmt.Errorf("list_fold expects list as first argument")
		return
	}
	
	state := args[1]
	fn := args[2]
	
	if len(list) == 0 {
		s.SetValue(state)
		return
	}
	
	// Recursive implementation: fold(tail, fn(head, state), fn)
	head := list[0]
	tail := list[1:]
	
	// Set up the continuation stack for the recursive call
	s.Push(CallCont{Arg: fn, Env: s.copyEnv()})
	s.Push(ApplyCont{Func: &Partial{
		Exp: Expression{"0": BUILTIN, "l": "list_fold"},
		Applied: []Value{tail},
		Impl: func(s *State, args ...Value) { s.builtinListFold(args...) },
	}, Env: s.copyEnv()})
	s.Push(CallCont{Arg: state, Env: s.copyEnv()})
	s.Push(CallCont{Arg: head, Env: s.copyEnv()})
	s.SetValue(fn)
}

// Helper function for value equality
func (s *State) valuesEqual(a, b Value) bool {
	// Handle Tagged values specially
	if taggedA, okA := a.(*Tagged); okA {
		if taggedB, okB := b.(*Tagged); okB {
			return taggedA.Tag == taggedB.Tag && s.valuesEqual(taggedA.Value, taggedB.Value)
		}
		return false
	}
	
	// Handle slices (lists)
	if sliceA, okA := a.([]Value); okA {
		if sliceB, okB := b.([]Value); okB {
			if len(sliceA) != len(sliceB) {
				return false
			}
			for i := range sliceA {
				if !s.valuesEqual(sliceA[i], sliceB[i]) {
					return false
				}
			}
			return true
		}
		return false
	}
	
	// Handle maps (records)
	if mapA, okA := a.(map[string]Value); okA {
		if mapB, okB := b.(map[string]Value); okB {
			if len(mapA) != len(mapB) {
				return false
			}
			for k, v := range mapA {
				if vB, exists := mapB[k]; !exists || !s.valuesEqual(v, vB) {
					return false
				}
			}
			return true
		}
		return false
	}
	
	// For other types, use direct comparison
	return a == b
}

// Binary-related builtin functions
func (s *State) builtinStringToBinary(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("string_to_binary expects 1 argument, got %d", len(args))
		return
	}
	
	str, ok := args[0].(string)
	if !ok {
		s.Break = fmt.Errorf("string_to_binary expects string argument")
		return
	}
	
	// Convert string to binary format expected by EYG
	bytes := []byte(str)
	encoded := base64.StdEncoding.EncodeToString(bytes)
	// Remove padding as expected by EYG format
	encoded = strings.TrimRight(encoded, "=")
	
	// Create the expected record structure: {"/": {"bytes": "base64data"}}
	innerRecord := make(map[string]Value)
	innerRecord["bytes"] = encoded
	
	outerRecord := make(map[string]Value)
	outerRecord["/"] = innerRecord
	
	s.SetValue(outerRecord)
}

func (s *State) builtinStringFromBinary(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("string_from_binary expects 1 argument, got %d", len(args))
		return
	}
	
	// Expect binary format: {"/": {"bytes": "base64data"}}
	outerRecord, ok := args[0].(map[string]Value)
	if !ok {
		// Try map[string]interface{} for test compatibility
		if outerInterface, ok2 := args[0].(map[string]interface{}); ok2 {
			// Convert to map[string]Value
			outerRecord = make(map[string]Value)
			for k, v := range outerInterface {
				outerRecord[k] = v
			}
		} else {
			s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
			return
		}
	}
	
	innerValue, exists := outerRecord["/"]
	if !exists {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
		return
	}
	
	innerRecord, ok := innerValue.(map[string]Value)
	if !ok {
		// Try map[string]interface{} for test compatibility
		if innerInterface, ok2 := innerValue.(map[string]interface{}); ok2 {
			innerRecord = make(map[string]Value)
			for k, v := range innerInterface {
				innerRecord[k] = v
			}
		} else {
			s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
			return
		}
	}
	
	bytesValue, exists := innerRecord["bytes"]
	if !exists {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
		return
	}
	
	encoded, ok := bytesValue.(string)
	if !ok {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
		return
	}
	
	// Add padding if needed for base64 decoding
	for len(encoded)%4 != 0 {
		encoded += "="
	}
	
	// Decode base64 to bytes
	bytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
		return
	}
	
	// Check if bytes form valid UTF-8
	result := string(bytes)
	if !utf8.ValidString(result) {
		s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
		return
	}
	
	s.SetValue(&Tagged{Tag: "Ok", Value: result})
}

func (s *State) builtinBinaryFromIntegers(args ...Value) {
	if len(args) != 1 {
		s.Break = fmt.Errorf("binary_from_integers expects 1 argument, got %d", len(args))
		return
	}
	
	list, ok := args[0].([]Value)
	if !ok {
		s.Break = fmt.Errorf("binary_from_integers expects list argument")
		return
	}
	
	// Convert list of integers to binary
	bytes := make([]byte, len(list))
	for i, v := range list {
		if n, ok := v.(float64); ok && n >= 0 && n <= 255 && n == float64(int(n)) {
			bytes[i] = byte(n)
		} else {
			s.SetValue(&Tagged{Tag: "Error", Value: make(map[string]Value)})
			return
		}
	}
	
	// Create the expected binary format
	encoded := base64.StdEncoding.EncodeToString(bytes)
	// Remove padding as expected by EYG format
	encoded = strings.TrimRight(encoded, "=")
	innerRecord := make(map[string]Value)
	innerRecord["bytes"] = encoded
	
	outerRecord := make(map[string]Value)
	outerRecord["/"] = innerRecord
	
	s.SetValue(outerRecord)
}

func (s *State) builtinBinaryFold(args ...Value) {
	if len(args) != 3 {
		s.Break = fmt.Errorf("binary_fold expects 3 arguments, got %d", len(args))
		return
	}
	
	// Extract binary data from the expected format
	outerRecord, ok := args[0].(map[string]Value)
	if !ok {
		// Try map[string]interface{} for test compatibility
		if outerInterface, ok2 := args[0].(map[string]interface{}); ok2 {
			outerRecord = make(map[string]Value)
			for k, v := range outerInterface {
				outerRecord[k] = v
			}
		} else {
			s.Break = fmt.Errorf("binary_fold expects binary as first argument")
			return
		}
	}
	
	innerValue, exists := outerRecord["/"]
	if !exists {
		s.Break = fmt.Errorf("binary_fold: invalid binary format")
		return
	}
	
	innerRecord, ok := innerValue.(map[string]Value)
	if !ok {
		// Try map[string]interface{} for test compatibility
		if innerInterface, ok2 := innerValue.(map[string]interface{}); ok2 {
			innerRecord = make(map[string]Value)
			for k, v := range innerInterface {
				innerRecord[k] = v
			}
		} else {
			s.Break = fmt.Errorf("binary_fold: invalid binary format")
			return
		}
	}
	
	bytesValue, exists := innerRecord["bytes"]
	if !exists {
		s.Break = fmt.Errorf("binary_fold: invalid binary format")
		return
	}
	
	encoded, ok := bytesValue.(string)
	if !ok {
		s.Break = fmt.Errorf("binary_fold: invalid binary format")
		return
	}
	
	// Add padding if needed for base64 decoding
	for len(encoded)%4 != 0 {
		encoded += "="
	}
	
	// Decode base64 to bytes
	bytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		s.Break = fmt.Errorf("binary_fold: invalid base64 data")
		return
	}
	
	state := args[1]
	fn := args[2]
	
	if len(bytes) == 0 {
		s.SetValue(state)
		return
	}
	
	// Convert bytes to Value array for processing
	binary := make([]Value, len(bytes))
	for i, b := range bytes {
		binary[i] = float64(b)
	}
	
	// Recursive implementation: fold(tail, fn(head, state), fn)
	head := binary[0]
	tail := binary[1:]
	
	// Create binary format for tail
	tailBytes := make([]byte, len(tail))
	for i, v := range tail {
		tailBytes[i] = byte(v.(float64))
	}
	tailEncoded := base64.StdEncoding.EncodeToString(tailBytes)
	// Remove padding as expected by EYG format
	tailEncoded = strings.TrimRight(tailEncoded, "=")
	tailInner := make(map[string]Value)
	tailInner["bytes"] = tailEncoded
	tailOuter := make(map[string]Value)
	tailOuter["/"] = tailInner
	
	// Set up the continuation stack for the recursive call
	s.Push(CallCont{Arg: fn, Env: s.copyEnv()})
	s.Push(ApplyCont{Func: &Partial{
		Exp: Expression{"0": BUILTIN, "l": "binary_fold"},
		Applied: []Value{tailOuter},
		Impl: func(s *State, args ...Value) { s.builtinBinaryFold(args...) },
	}, Env: s.copyEnv()})
	s.Push(CallCont{Arg: state, Env: s.copyEnv()})
	s.Push(CallCont{Arg: head, Env: s.copyEnv()})
	s.SetValue(fn)
}