package main

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func tokensToString(tokenized []Token) string {
	var result string
	for _, tok := range tokenized {
		result += tok.String() + "\n"
	}
	return result
}

type TokenizerTestCase struct {
	Name     string `yaml:"name"`
	Input    string `yaml:"input"`
	Expected string `yaml:"expected"`
}

type TokenizerTestSuite struct {
	Tests []TokenizerTestCase `yaml:"tokenizer_tests"`
}

func loadTokenizerTests() ([]TokenizerTestCase, error) {
	data, err := os.ReadFile("tokenizer_tests.yaml")
	if err != nil {
		return nil, err
	}

	var suite TokenizerTestSuite
	err = yaml.Unmarshal(data, &suite)
	if err != nil {
		return nil, err
	}

	return suite.Tests, nil
}

func TestCases(t *testing.T) {
	testCases, err := loadTokenizerTests()
	if err != nil {
		t.Fatalf("Failed to load test cases: %v", err)
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			tokens, err := TokenizeString(tc.Input)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			result := strings.TrimRight(tokensToString(tokens), "\n")
			expected := strings.TrimRight(tc.Expected, "\n")
			if result != expected {
				t.Errorf("Test %s failed:\nExpected:\n%s\nGot:\n%s", tc.Name, expected, result)
			}
		})
	}
}
