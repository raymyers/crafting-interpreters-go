package main

import (
	approvals "github.com/approvals/go-approval-tests"
	"fmt"
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

	return fmt.Sprintf("%v", result)
}

type EvaluatorTestCaseParameters struct {
	name  string
	value string
}

var EvaluatorParameterizedTestcases = []EvaluatorTestCaseParameters{
	{name: "Number", value: "42"},
	{name: "String", value: `"hello"`},
	{name: "Boolean", value: "true"},
	{name: "Nil", value: "nil"},
	{name: "Addition", value: "2 + 3"},
	{name: "Subtraction", value: "5 - 2"},
	{name: "Multiplication", value: "4 * 6"},
	{name: "Division", value: "8 / 2"},
	{name: "Comparison", value: "3 < 5"},
	{name: "Equality", value: "1 == 1"},
	{name: "Inequality", value: "1 != 2"},
	{name: "UnaryMinus", value: "-42"},
	{name: "UnaryBang", value: "!true"},
	{name: "Grouping", value: "(2 + 3)"},
	{name: "ComplexExpression", value: "2 + 3 * 4"},
	{name: "GroupedExpression", value: "(2 + 3) * 4"},
	{name: "NestedGrouping", value: "((1 + 2) * 3)"},
	{name: "MixedTypes", value: `"hello" == "world"`},
	{name: "FloatNumbers", value: "3.14 + 2.71"},
}

func TestEvaluatorCases(t *testing.T) {
	for _, tc := range EvaluatorParameterizedTestcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := evaluateToString(tc.value)
			approvals.VerifyString(t, result)
		})
	}
}