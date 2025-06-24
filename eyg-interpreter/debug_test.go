package main

import (
	"fmt"
	"testing"
)

func TestDebugHandlerOrder(t *testing.T) {
	// Simplified version of the failing test
	// let _ = perform Push(1) in
	// let _ = perform Push(2) in 
	// []
	expr := Expression{
		"0": "l",
		"l": "_", 
		"t": Expression{
			"0": "l",
			"l": "_",
			"t": Expression{
				"0": "ta", // empty list
			},
			"v": Expression{
				"0": "a",
				"a": Expression{"0": "i", "v": float64(2)},
				"f": Expression{"0": "p", "l": "Push"},
			},
		},
		"v": Expression{
			"0": "a", 
			"a": Expression{"0": "i", "v": float64(1)},
			"f": Expression{"0": "p", "l": "Push"},
		},
	}
	
	// Handler that builds a list: (value, kont) -> cons(value, kont({}))
	handler := Expression{
		"0": "f",
		"l": "value",
		"b": Expression{
			"0": "f", 
			"l": "kont",
			"b": Expression{
				"0": "a",
				"a": Expression{
					"0": "a",
					"a": Expression{"0": "u"}, // empty record
					"f": Expression{"0": "v", "l": "kont"},
				},
				"f": Expression{
					"0": "a",
					"a": Expression{"0": "v", "l": "value"},
					"f": Expression{"0": "c"}, // cons
				},
			},
		},
	}
	
	// handle Push handler expr
	handleExpr := Expression{
		"0": "a",
		"a": Expression{
			"0": "f",
			"l": "_", 
			"b": expr,
		},
		"f": Expression{
			"0": "a",
			"a": handler,
			"f": Expression{"0": "h", "l": "Push"},
		},
	}
	
	state := Eval(handleExpr)
	
	if state.Break != nil {
		t.Fatalf("Got break: %+v", state.Break)
	}
	
	result, ok := state.Control.([]Value)
	if !ok {
		t.Fatalf("Expected list, got %T: %+v", state.Control, state.Control)
	}
	
	if len(result) != 2 {
		t.Fatalf("Expected list of length 2, got %d", len(result))
	}
	
	fmt.Printf("Result: %+v\n", result)
	
	// Check order
	if v1, ok := result[0].(float64); !ok || v1 != 1 {
		t.Errorf("Expected first element to be 1, got %+v", result[0])
	}
	if v2, ok := result[1].(float64); !ok || v2 != 2 {
		t.Errorf("Expected second element to be 2, got %+v", result[1])
	}
}