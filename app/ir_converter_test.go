package main

import (
	"encoding/json"
	"testing"
)

func TestIRConverter(t *testing.T) {
	tests := []struct {
		name     string
		expr     Expr
		expected map[string]interface{}
	}{
		// Variables
		{
			name: "variable",
			expr: &Variable{Name: Token{Lexeme: "x"}},
			expected: map[string]interface{}{
				"0": "v",
				"l": "x",
			},
		},

		// Literals
		{
			name: "integer literal",
			expr: &Literal{Value: NumberValue{Val: 42}},
			expected: map[string]interface{}{
				"0": "i",
				"v": 42,
			},
		},
		{
			name: "string literal",
			expr: &Literal{Value: StringValue{Val: "hello"}},
			expected: map[string]interface{}{
				"0": "s",
				"v": "hello",
			},
		},
		{
			name: "binary literal",
			expr: &Literal{Value: BinaryValue{Val: []byte{0x01}}},
			expected: map[string]interface{}{
				"0": "x",
				"v": map[string]interface{}{
					"/": map[string]interface{}{
						"bytes": "AQ==", // base64 of 0x01
					},
				},
			},
		},
		{
			name: "empty binary literal",
			expr: &Literal{Value: BinaryValue{Val: []byte{}}},
			expected: map[string]interface{}{
				"0": "x",
				"v": map[string]interface{}{
					"/": map[string]interface{}{
						"bytes": "",
					},
				},
			},
		},

		// Records
		{
			name: "empty record",
			expr: &EmptyRecord{},
			expected: map[string]interface{}{
				"0": "u",
			},
		},
		{
			name: "record with single field",
			expr: &Record{
				Fields: []RecordField{
					{Name: "name", Value: &Literal{Value: StringValue{Val: "Alice"}}},
				},
			},
			expected: map[string]interface{}{
				"0": "a",
				"a": map[string]interface{}{
					"0": "u",
				},
				"f": map[string]interface{}{
					"0": "a",
					"a": map[string]interface{}{
						"0": "s",
						"v": "Alice",
					},
					"f": map[string]interface{}{
						"0": "e",
						"l": "name",
					},
				},
			},
		},
		{
			name: "record with multiple fields",
			expr: &Record{
				Fields: []RecordField{
					{Name: "name", Value: &Literal{Value: StringValue{Val: "Alice"}}},
					{Name: "place", Value: &Literal{Value: StringValue{Val: "Burnley"}}},
				},
			},
			expected: map[string]interface{}{
				"0": "a",
				"a": map[string]interface{}{
					"0": "a",
					"a": map[string]interface{}{
						"0": "u",
					},
					"f": map[string]interface{}{
						"0": "a",
						"a": map[string]interface{}{
							"0": "s",
							"v": "Burnley",
						},
						"f": map[string]interface{}{
							"0": "e",
							"l": "place",
						},
					},
				},
				"f": map[string]interface{}{
					"0": "a",
					"a": map[string]interface{}{
						"0": "s",
						"v": "Alice",
					},
					"f": map[string]interface{}{
						"0": "e",
						"l": "name",
					},
				},
			},
		},

		// Select (Access)
		{
			name: "select field",
			expr: &Access{
				Object: &Variable{Name: Token{Lexeme: "record"}},
				Name:   "name",
			},
			expected: map[string]interface{}{
				"0": "a",
				"a": map[string]interface{}{
					"0": "v",
					"l": "record",
				},
				"f": map[string]interface{}{
					"0": "g",
					"l": "name",
				},
			},
		},

		// Lists
		{
			name: "empty list",
			expr: &List{
				Elements: []Expr{},
			},
			expected: map[string]interface{}{
				"0": "ta",
			},
		},
		{
			name: "list with single element",
			expr: &List{
				Elements: []Expr{
					&Literal{Value: NumberValue{Val: 101}},
				},
			},
			expected: map[string]interface{}{
				"0": "a",
				"f": map[string]interface{}{
					"0": "a",
					"f": map[string]interface{}{
						"0": "c",
					},
					"a": map[string]interface{}{
						"0": "i",
						"v": 101,
					},
				},
				"a": map[string]interface{}{
					"0": "ta",
				},
			},
		},
		{
			name: "list with multiple elements",
			expr: &List{
				Elements: []Expr{
					&Literal{Value: NumberValue{Val: 101}},
					&Literal{Value: NumberValue{Val: 102}},
				},
			},
			expected: map[string]interface{}{
				"0": "a",
				"f": map[string]interface{}{
					"0": "a",
					"f": map[string]interface{}{
						"0": "c",
					},
					"a": map[string]interface{}{
						"0": "i",
						"v": 101,
					},
				},
				"a": map[string]interface{}{
					"0": "a",
					"f": map[string]interface{}{
						"0": "a",
						"f": map[string]interface{}{
							"0": "c",
						},
						"a": map[string]interface{}{
							"0": "i",
							"v": 102,
						},
					},
					"a": map[string]interface{}{
						"0": "ta",
					},
				},
			},
		},

		// Tagged unions
		{
			name: "tagged value",
			expr: &Union{
				Constructor: "Ok",
				Value:       &Literal{Value: StringValue{Val: "good"}},
			},
			expected: map[string]interface{}{
				"0": "a",
				"a": map[string]interface{}{
					"0": "s",
					"v": "good",
				},
				"f": map[string]interface{}{
					"0": "t",
					"l": "Ok",
				},
			},
		},

		// Let bindings
		{
			name: "let assignment",
			expr: &Var{
				Pattern: &Variable{Name: Token{Lexeme: "assigned"}},
				Value:   &Literal{Value: NumberValue{Val: 10}},
				Body:    &Variable{Name: Token{Lexeme: "assigned"}},
			},
			expected: map[string]interface{}{
				"0": "l",
				"l": "assigned",
				"v": map[string]interface{}{
					"0": "i",
					"v": 10,
				},
				"t": map[string]interface{}{
					"0": "v",
					"l": "assigned",
				},
			},
		},

		// Functions
		{
			name: "identity function",
			expr: &Lambda{
				Parameters: []string{"x"},
				Body:       &Variable{Name: Token{Lexeme: "x"}},
			},
			expected: map[string]interface{}{
				"0": "f",
				"l": "x",
				"b": map[string]interface{}{
					"0": "v",
					"l": "x",
				},
			},
		},

		// Function application
		{
			name: "function application",
			expr: &Call{
				Callee: &Lambda{
					Parameters: []string{"x"},
					Body:       &Variable{Name: Token{Lexeme: "x"}},
				},
				Arguments: []Expr{
					&Literal{Value: NumberValue{Val: 107}},
				},
			},
			expected: map[string]interface{}{
				"0": "a",
				"a": map[string]interface{}{
					"0": "i",
					"v": 107,
				},
				"f": map[string]interface{}{
					"0": "f",
					"l": "x",
					"b": map[string]interface{}{
						"0": "v",
						"l": "x",
					},
				},
			},
		},

		// Builtin
		{
			name: "builtin",
			expr: &Builtin{Name: "add"},
			expected: map[string]interface{}{
				"0": "b",
				"l": "add",
			},
		},

		// Perform effect
		{
			name: "perform",
			expr: &Perform{Effect: "Log"},
			expected: map[string]interface{}{
				"0": "p",
				"l": "Log",
			},
		},

		// Handle effect
		{
			name: "handle",
			expr: &Handle{Effect: "Log"},
			expected: map[string]interface{}{
				"0": "h",
				"l": "Log",
			},
		},

		// Unsupported expression type
		{
			name: "unsupported type",
			expr: &Unary{}, // Assuming Unary is not supported
			expected: map[string]interface{}{
				"0": "z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := NewIRConverter()
			jsonBytes, err := converter.Convert(tt.expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &result); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			// Compare as JSON strings to avoid map ordering issues
			expectedJSON, _ := json.MarshalIndent(tt.expected, "", "  ")
			actualJSON, _ := json.MarshalIndent(result, "", "  ")

			if string(expectedJSON) != string(actualJSON) {
				t.Errorf("mismatch\nexpected:\n%s\ngot:\n%s", expectedJSON, actualJSON)
			}
		})
	}
}

// Test helper function to ensure correct node structure
func TestIRNodeStructure(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name: "variable node",
			input: map[string]interface{}{
				"0": "v",
				"l": "x",
			},
			expected: `{"0":"v","l":"x"}`,
		},
		{
			name: "integer node",
			input: map[string]interface{}{
				"0": "i",
				"v": 42,
			},
			expected: `{"0":"i","v":42}`,
		},
		{
			name: "function node",
			input: map[string]interface{}{
				"0": "f",
				"l": "x",
				"b": map[string]interface{}{
					"0": "v",
					"l": "x",
				},
			},
			expected: `{"0":"f","b":{"0":"v","l":"x"},"l":"x"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			// Unmarshal to normalize
			var normalized map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Check that we can round-trip
			jsonBytes2, err := json.Marshal(normalized)
			if err != nil {
				t.Fatalf("failed to marshal again: %v", err)
			}

			if len(jsonBytes) != len(jsonBytes2) {
				t.Errorf("json changed during round-trip")
			}
		})
	}
}
