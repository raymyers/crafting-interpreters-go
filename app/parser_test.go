package main

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func parseToString(input string) string {
	tokens, err := TokenizeString(input)
	if err != nil {
		return "Tokenization error: " + err.Error()
	}

	parser := NewParser(tokens)
	expr, err := parser.Parse()
	if err != nil {
		return "Parse error: " + err.Error()
	}

	printer := &AstPrinter{}
	return printer.Print(expr)
}

type ParserTestCase struct {
	Name     string `yaml:"name"`
	Input    string `yaml:"input"`
	Expected string `yaml:"expected"`
}

type ParserTestSuite struct {
	Tests []ParserTestCase `yaml:"parser_tests"`
}

func loadParserTests() ([]ParserTestCase, error) {
	data, err := os.ReadFile("parser_tests.yaml")
	if err != nil {
		return nil, err
	}

	var suite ParserTestSuite
	err = yaml.Unmarshal(data, &suite)
	if err != nil {
		return nil, err
	}

	return suite.Tests, nil
}

func TestParserCases(t *testing.T) {
	testCases, err := loadParserTests()
	if err != nil {
		t.Fatalf("Failed to load test cases: %v", err)
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			result := parseToString(tc.Input)
			if result != tc.Expected {
				t.Errorf("Test %s failed:\nExpected: %s\nGot: %s", tc.Name, tc.Expected, result)
			}
		})
	}
}
