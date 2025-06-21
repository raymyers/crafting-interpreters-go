package main

import (
	approvals "github.com/approvals/go-approval-tests"
	"testing"
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

type ParserTestCaseParameters struct {
	name  string
	value string
}

var ParserParameterizedTestcases = []ParserTestCaseParameters{
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

func TestParserCases(t *testing.T) {
	for _, tc := range ParserParameterizedTestcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := parseToString(tc.value)
			approvals.VerifyString(t, result)
		})
	}
}
