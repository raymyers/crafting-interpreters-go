package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
)

func TestIRConverter(t *testing.T) {
	// Load the IR fixture
	fixtureBytes, err := ioutil.ReadFile("ir-fixture.json")
	if err != nil {
		t.Fatalf("Failed to read IR fixture: %v", err)
	}

	var fixtureNodes []IRNode
	err = json.Unmarshal(fixtureBytes, &fixtureNodes)
	if err != nil {
		t.Fatalf("Failed to parse IR fixture: %v", err)
	}

	// Test cases for different AST nodes
	testCases := []struct {
		name     string
		input    string
		expected IRNode
	}{
		{
			name:     "Variable",
			input:    "foo",
			expected: fixtureNodes[0], // variable
		},
		{
			name:     "Lambda",
			input:    "|x| { x }",
			expected: fixtureNodes[1], // function
		},
		{
			name:     "Call",
			input:    "(|x| { x })(\"foo\")",
			expected: fixtureNodes[2], // apply
		},
		{
			name:     "Let",
			input:    "x = \"hi\"; x",
			expected: fixtureNodes[3], // let
		},
		{
			name:     "Integer",
			input:    "5",
			expected: fixtureNodes[5], // integer
		},
		{
			name:     "String",
			input:    "\"hello\"",
			expected: fixtureNodes[6], // string
		},
		{
			name:     "Empty List",
			input:    "[]",
			expected: fixtureNodes[7], // empty list
		},
		{
			name:     "Empty Record",
			input:    "{}",
			expected: fixtureNodes[10], // empty record
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the input
			tokens, err := TokenizeString(tc.input)
			if err != nil {
				t.Fatalf("Failed to tokenize input: %v", err)
			}

			parser := NewParser(tokens)
			expr, err := parser.Parse()
			if err != nil {
				t.Fatalf("Failed to parse input: %v", err)
			}

			// Convert to IR
			converter := NewIRConverter()
			irJson, err := converter.Convert(expr)
			if err != nil {
				t.Fatalf("Failed to convert to IR: %v", err)
			}

			// Parse the IR JSON
			var irNodes []IRNode
			err = json.Unmarshal(irJson, &irNodes)
			if err != nil {
				t.Fatalf("Failed to parse IR JSON: %v", err)
			}

			// Check if we have at least one node
			if len(irNodes) == 0 {
				t.Fatalf("No IR nodes generated")
			}

			// Compare the first node with the expected node
			// Note: This is a simplified comparison that only checks the name
			// A more thorough test would compare the entire structure
			if irNodes[0].Name != tc.expected.Name {
				t.Errorf("Expected node name %q, got %q", tc.expected.Name, irNodes[0].Name)
			}
		})
	}
}

// TestIRCommandWithFixture tests the IR command with examples from the fixture
func TestIRCommandWithFixture(t *testing.T) {
	// Create a temporary file for testing
	tmpfile, err := ioutil.TempFile("", "test-*.lox")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Test with a variable expression
	_, err = tmpfile.WriteString("foo")
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the IR command
	handleIR(tmpfile.Name())

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	outputBytes, _ := ioutil.ReadAll(r)
	output := string(outputBytes)

	// Check if the output contains expected IR structure
	if len(output) == 0 {
		t.Errorf("Expected non-empty output")
	}

	// Parse the output as JSON to verify it's valid
	var irNodes []IRNode
	err = json.Unmarshal(outputBytes, &irNodes)
	if err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Check if we have at least one node
	if len(irNodes) == 0 {
		t.Fatalf("No IR nodes in output")
	}

	// Check if the first node has the expected name
	if irNodes[0].Name != "variable" {
		t.Errorf("Expected node name 'variable', got %q", irNodes[0].Name)
	}
}