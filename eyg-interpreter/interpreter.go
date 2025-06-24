package main

import (
	"fmt"
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
	// Stub implementation
	s.Break = fmt.Errorf("int_parse not implemented")
}

func (s *State) builtinIntToString(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("int_to_string not implemented")
}

func (s *State) builtinStringAppend(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("string_append not implemented")
}

func (s *State) builtinStringSplit(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("string_split not implemented")
}

func (s *State) builtinStringSplitOnce(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("string_split_once not implemented")
}

func (s *State) builtinStringReplace(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("string_replace not implemented")
}

func (s *State) builtinStringUppercase(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("string_uppercase not implemented")
}

func (s *State) builtinStringLowercase(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("string_lowercase not implemented")
}

func (s *State) builtinStringEndsWith(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("string_ends_with not implemented")
}

func (s *State) builtinStringStartsWith(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("string_starts_with not implemented")
}

func (s *State) builtinStringLength(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("string_length not implemented")
}

func (s *State) builtinListPop(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("list_pop not implemented")
}

func (s *State) builtinListFold(args ...Value) {
	// Stub implementation
	s.Break = fmt.Errorf("list_fold not implemented")
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