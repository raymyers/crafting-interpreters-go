package main

import (
	"bytes"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func evaluateToString(input string, output *bytes.Buffer) string {
	tokens, err := TokenizeString(input)
	if err != nil {
		return "Tokenization error: " + err.Error()
	}

	parser := NewParser(tokens)
	expr, err := parser.Parse()
	if err != nil {
		return "Parse error: " + err.Error()
	}

	evaluator := NewEvaluator(NewDefaultScope(output), output)
	result := evaluator.Evaluate(expr)
	if ev, isErrVal := result.(ErrorValue); isErrVal {
		return "Evaluation error: " + ev.Message
	}

	return formatValue(result)
}

type EvaluatorTestCase struct {
	Name           string `yaml:"name"`
	Input          string `yaml:"input"`
	Expected       string `yaml:"expected"`
	ExpectedOutput string `yaml:"expectedOutput"`
}

type EvaluatorTestSuite struct {
	Tests []EvaluatorTestCase `yaml:"evaluator_tests"`
}

func loadEvaluatorTests() ([]EvaluatorTestCase, error) {
	data, err := os.ReadFile("evaluator_tests.yaml")
	if err != nil {
		return nil, err
	}

	var suite EvaluatorTestSuite
	err = yaml.Unmarshal(data, &suite)
	if err != nil {
		return nil, err
	}

	return suite.Tests, nil
}

func TestEvaluatorCases(t *testing.T) {
	testCases, err := loadEvaluatorTests()
	if err != nil {
		t.Fatalf("Failed to load test cases: %v", err)
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			var output bytes.Buffer
			result := evaluateToString(tc.Input, &output)

			// Check the return value
			if result != tc.Expected {
				t.Errorf("Test %s failed: expected result %q, got %q", tc.Name, tc.Expected, result)
			}

			// Check the output if expectedOutput is specified
			if tc.ExpectedOutput != "" {
				actualOutput := output.String()
				if actualOutput != tc.ExpectedOutput {
					t.Errorf("Test %s failed: expected output %q, got %q", tc.Name, tc.ExpectedOutput, actualOutput)
				}
			}
		})
	}
}
