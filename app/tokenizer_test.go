package main

import (
	approvals "github.com/approvals/go-approval-tests"
	"testing"
)

func tokensToString(tokenized []Token) string {
	var result string
	for _, tok := range tokenized {
		result += tok.String() + "\n"
	}
	return result
}

type TestCaseParameters struct {
	name  string
	value string
}

var ParameterizedTestcases = []TestCaseParameters{
	{name: "Parens", value: "(())"},
	{name: "Braces", value: "{{}}"},
	{name: "Ops", value: "({*.,+-;})"},
	{name: "Compare", value: "(!=) == (<=) >= (<) >"},
	{name: "SlashComment", value: "(/)//()"},
}

func TestCases(t *testing.T) {
	for _, tc := range ParameterizedTestcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tokens, err := TokenizeString(tc.value)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			approvals.VerifyString(t, tokensToString(tokens))
		})
	}
}
