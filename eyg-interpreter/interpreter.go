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
	default:
		return 1
	}
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
	// For now, return nil for all builtins
	// We'll implement these as needed
	return nil
}

// Eval evaluates an expression and returns the final state
func Eval(src Expression) *State {
	state := NewState(src)
	state.Loop()
	return state
}