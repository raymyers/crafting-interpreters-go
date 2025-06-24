package eyg

import (
	"encoding/json"
	"fmt"
	"log"
)

// Handler is the Go equivalent of your JS extrinsic[label] function:
// it takes the lifted value and returns a new Value (or an error).
type Handler func(Value) (Value, error)

// Extrinsic maps effect labels to their handlers.
type Extrinsic map[string]Handler

// Exec drives the interpreter until it either:
//   - terminates normally (no Break, no more continuations) → returns the final Value
//   - hits an Effect break               → invokes the corresponding handler and resumes
//   - hits any other Break              → returns an error
func Exec(src Expression, extrinsic Extrinsic) (Value, error) {
	fmt.Println(src)
	state := NewState(src)

	for {
		// Step one computation step
		state.Step()

		// If no break and stack is empty and we have a value, we’re done
		if state.Break == nil && state.IsValue && len(state.Stack) == 0 {
			return state.Control, nil
		}

		// If no break, keep going
		if state.Break == nil {
			continue
		}

		// We have a Break; see if it’s an Effect
		switch eff := state.Break.(type) {
		case *Effect:
			handler, ok := extrinsic[eff.Label]
			if !ok {
				return nil, fmt.Errorf("unhandled effect %q", eff.Label)
			}
			// clear the break before calling handler
			state.Break = nil

			// call the handler
			resumed, err := handler(eff.Lift)
			if err != nil {
				return nil, err
			}

			// feed the result back into the machine
			state.Resume(resumed)

		default:
			// some other error/break condition
			return nil, fmt.Errorf("execution stopped on unexpected break: %+v", eff)
		}
	}
}

// Run is analogous to your JS `run`:
// it calls Exec, converts the result to native Go, and prints it.
func Run(src Expression, extrinsic Extrinsic) error {
	result, err := Exec(src, extrinsic)
	if err != nil {
		return err
	}

	// convert to plain Go types (maps, slices, primitives)…
	plain := Native(result)

	// …and pretty-print as JSON
	out, err := json.MarshalIndent(plain, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

func RunExample(source Expression) {
	extrinsic := Extrinsic{
		"Log": func(val Value) (Value, error) {
			// val is the “lifted” argument
			msg, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("Log expected string, got %T", val)
			}
			fmt.Println("LOG:", msg)
			// return an empty record (i.e. no meaningful result)
			return make(map[string]Value), nil
		},
	}

	// source is your top‐level Expression
	if err := Run(source, extrinsic); err != nil {
		log.Fatal(err)
	}
}

// Native recursively walks your Value and turns any Lists or Records
// into []interface{} or map[string]interface{} so that JSON (or fmt)
// will print them sensibly.
func Native(v Value) interface{} {
	switch x := v.(type) {
	case []Value:
		arr := make([]interface{}, len(x))
		for i, e := range x {
			arr[i] = Native(e)
		}
		return arr

	case map[string]Value:
		m := make(map[string]interface{}, len(x))
		for k, e := range x {
			m[k] = Native(e)
		}
		return m

	case Tagged:
		// represent as a JSON object {"tag": "...", "value": ...}
		return map[string]interface{}{
			"tag":   x.Tag,
			"value": Native(x.Value),
		}

	case *Closure, *Partial:
		// you might want a custom repr, or just drop into fmt
		return fmt.Sprintf("%#v", x)

	default:
		// assume it's already a Go primitive (string, float64, bool…)
		return x
	}
}
