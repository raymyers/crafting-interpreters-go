package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// TestCase represents a single test case from the YAML file
type TestCase struct {
	Name           string `yaml:"name"`
	Input          string `yaml:"input"`
	Expected       string `yaml:"expected"`
	ExpectedOutput string `yaml:"expectedOutput,omitempty"`
}

// TestSuite represents the entire test suite from the YAML file
type TestSuite struct {
	Tests []TestCase `yaml:"evaluator_tests"`
}

// RunSuite runs all tests in the evaluator_tests.yaml file
func RunSuite(filter string) error {
	// Read the YAML file
	yamlFile, err := os.ReadFile("app/evaluator_tests.yaml")
	if err != nil {
		return fmt.Errorf("error reading YAML file: %v", err)
	}

	// Parse the YAML file
	var testSuite TestSuite
	err = yaml.Unmarshal(yamlFile, &testSuite)
	if err != nil {
		return fmt.Errorf("error parsing YAML file: %v", err)
	}

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "eyg-tests")
	if err != nil {
		return fmt.Errorf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Run each test
	for _, test := range testSuite.Tests {
		// Skip tests that don't match the filter
		if filter != "" && !strings.Contains(test.Name, filter) {
			continue
		}

		fmt.Printf("========================================================\n")
		fmt.Printf("Running test: %s\n", test.Name)
		fmt.Printf("========================================================\n")

		// Create a temporary file for the test input
		tempFile := filepath.Join(tempDir, fmt.Sprintf("%s.eyg", test.Name))
		err := os.WriteFile(tempFile, []byte(test.Input), 0644)
		if err != nil {
			fmt.Printf("Error creating temporary file: %v\n", err)
			continue
		}

		// Print the code
		fmt.Printf("CODE:\n")
		fmt.Printf("----------------------------------------\n")
		fmt.Printf("%s\n", test.Input)
		fmt.Printf("----------------------------------------\n")

		// Parse the code to get the AST
		tokens, tokenizeErr := TokenizeFile(tempFile)
		if tokenizeErr != nil {
			fmt.Printf("Tokenization error: %v\n", tokenizeErr)
			continue
		}

		parser := NewParser(tokens)
		expr, parseErr := parser.Parse()
		if parseErr != nil {
			fmt.Printf("Parse error: %v\n", parseErr)
			continue
		}

		// Print the AST
		fmt.Printf("AST:\n")
		fmt.Printf("----------------------------------------\n")
		printer := &AstPrinter{}
		astResult := printer.Print(expr)
		fmt.Println(astResult)
		fmt.Printf("----------------------------------------\n")

		// Convert to IR
		converter := NewIRConverter()
		irJson, irErr := converter.Convert(expr)
		if irErr != nil {
			fmt.Printf("IR conversion error: %v\n", irErr)
			continue
		}

		// Print the IR
		fmt.Printf("IR:\n")
		fmt.Printf("----------------------------------------\n")
		fmt.Println(string(irJson))
		fmt.Printf("----------------------------------------\n")

		// Print the expected result
		fmt.Printf("EXPECTED:\n")
		fmt.Printf("----------------------------------------\n")
		fmt.Printf("%s\n", test.Expected)
		if test.ExpectedOutput != "" {
			fmt.Printf("Expected Output: %s\n", test.ExpectedOutput)
		}
		fmt.Printf("----------------------------------------\n")

		// Save IR to a file
		irFile := filepath.Join(tempDir, fmt.Sprintf("%s.ir.json", test.Name))
		err = os.WriteFile(irFile, irJson, 0644)
		if err != nil {
			fmt.Printf("Error writing IR file: %v\n", err)
			continue
		}

		// Run the interpreter
		fmt.Printf("INTERPRETER RESULT:\n")
		fmt.Printf("----------------------------------------\n")
		
		// Parse the IR JSON into an Expression
		var irExpressions []map[string]interface{}
		err = json.Unmarshal(irJson, &irExpressions)
		if err != nil {
			fmt.Printf("Error parsing IR JSON: %v\n", err)
		} else if len(irExpressions) > 0 {
			// Use the first expression as the entry point
			expr := irExpressions[0]
			
			// Import the interpreter package
			// Note: This is a workaround since we can't directly import from eyg-interpreter
			// In a real implementation, we would refactor the interpreter to be importable
			fmt.Printf("IR saved to: %s\n", irFile)
			fmt.Printf("Note: Direct interpreter integration not available.\n")
			fmt.Printf("IR Expression: %v\n", expr)
		} else {
			fmt.Printf("No expressions found in IR JSON\n")
		}
		fmt.Printf("----------------------------------------\n")

		fmt.Printf("\n")
	}

	return nil
}

