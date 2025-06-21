package main

import (
	"testing"
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

type EvaluatorTestCaseParameters struct {
	name     string
	value    string
	expected string
}

var EvaluatorParameterizedTestcases = []EvaluatorTestCaseParameters{
	{name: "Number", value: "42", expected: "42"},
	{name: "String", value: `"hello"`, expected: "hello"},
	{name: "Boolean", value: "true", expected: "true"},
	{name: "Nil", value: "nil", expected: "nil"},
	{name: "Addition", value: "2 + 3", expected: "5"},
	{name: "Subtraction", value: "5 - 2", expected: "3"},
	{name: "Multiplication", value: "4 * 6", expected: "24"},
	{name: "Division", value: "8 / 2", expected: "4"},
	{name: "LessThan", value: "3 < 5", expected: "true"},
	{name: "LessThanOrEqual", value: "3 <= 5", expected: "true"},
	{name: "GreaterThan", value: "5 > 3", expected: "true"},
	{name: "GreaterThanOrEqual", value: "5 >= 3", expected: "true"},
	{name: "Equality", value: "1 == 1", expected: "nil"},
	{name: "Inequality", value: "1 != 2", expected: "nil"},
	{name: "UnaryMinus", value: "-42", expected: "-42"},
	{name: "UnaryMinusFloat", value: "-73", expected: "-73"},
	{name: "UnaryBang", value: "!true", expected: "false"},
	{name: "UnaryBangFloat", value: "!10.40", expected: "false"},
	{name: "UnaryBangGrouped", value: "!((false))", expected: "true"},
	{name: "Grouping", value: "(2 + 3)", expected: "5"},
	{name: "ComplexExpression", value: "2 + 3 * 4", expected: "14"},
	{name: "GroupedExpression", value: "(2 + 3) * 4", expected: "20"},
	{name: "NestedGrouping", value: "((1 + 2) * 3)", expected: "9"},
	{name: "MixedTypes", value: `"hello" == "world"`, expected: "nil"},
	{name: "FloatNumbers", value: "3.14 + 2.71", expected: "5.85"},
	{name: "GroupedString", value: `( "hello" )`, expected: "hello"},
	{name: "StringConcat", value: `"hel" + "lo"`, expected: "hello"},
}

func TestEvaluatorCases(t *testing.T) {
	for _, tc := range EvaluatorParameterizedTestcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := evaluateToString(tc.value)
			if result != tc.expected {
				t.Errorf("Test %s failed: expected %q, got %q", tc.name, tc.expected, result)
			}
		})
	}
}
