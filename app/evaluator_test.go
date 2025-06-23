package main

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func evaluateToString(input string) string {
	tokens, err := TokenizeString(input)
	if err != nil {
		return "Tokenization error: " + err.Error()
	}

	parser := NewParser(tokens)
	expr, err := parser.Parse()
	if err != nil {
		return "Parse error: " + err.Error()
	}

	evaluator := &Evaluator{}
	result, err := evaluator.Evaluate(expr)
	if err != nil {
		return "Evaluation error: " + err.Error()
	}

	return formatValue(result)
}

type EvaluatorTestCase struct {
	Name     string `yaml:"name"`
	Input    string `yaml:"input"`
	Expected string `yaml:"expected"`
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
			result := evaluateToString(tc.Input)
			if result != tc.Expected {
				t.Errorf("Test %s failed: expected %q, got %q", tc.Name, tc.Expected, result)
			}
		})
	}
}
