package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ./your_program.sh <command> <filename>")
		os.Exit(1)
	}

	command := os.Args[1]
	filename := os.Args[2]

	switch command {
	case "tokenize":
		handleTokenize(filename)
	case "parse":
		handleParse(filename)
	case "evaluate":
		handleEvaluate(filename)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func handleTokenize(filename string) {
	tokenized, tokenizeErr := TokenizeFile(filename)

	for _, tok := range tokenized {
		_, err := fmt.Fprintf(os.Stdout, "%s\n", tok.String())
		if err != nil {
			os.Exit(1)
		}
	}
	if tokenizeErr != nil {
		os.Exit(65)
	}
}

func handleParse(filename string) {
	// Tokenize the file first
	tokens, tokenizeErr := TokenizeFile(filename)
	if tokenizeErr != nil {
		fmt.Fprintf(os.Stderr, "Tokenization error: %v\n", tokenizeErr)
		os.Exit(65)
	}

	// Parse the tokens into an AST
	parser := NewParser(tokens)
	expr, parseErr := parser.Parse()
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", parseErr)
		os.Exit(65)
	}

	// Print the AST as S-expression
	printer := &AstPrinter{}
	result := printer.Print(expr)
	fmt.Println(result)
}

func handleEvaluate(filename string) {
	// Tokenize the file first
	tokens, tokenizeErr := TokenizeFile(filename)
	if tokenizeErr != nil {
		fmt.Fprintf(os.Stderr, "Tokenization error: %v\n", tokenizeErr)
		os.Exit(65)
	}

	// Parse the tokens into an AST
	parser := NewParser(tokens)
	expr, parseErr := parser.Parse()
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", parseErr)
		os.Exit(65)
	}

	// Evaluate the expression
	evaluator := &Evaluator{}
	result, evalErr := evaluator.Evaluate(expr)
	if evalErr != nil {
		fmt.Fprintf(os.Stderr, "Evaluation error: %v\n", evalErr)
		os.Exit(70)
	}

	// Print the result
	fmt.Println(formatValue(result))
}

func formatValue(value interface{}) string {
	if value == nil {
		return "nil"
	}
	
	switch v := value.(type) {
	case float64:
		// Format numbers to match expected output
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.1f", v)
		}
		return fmt.Sprintf("%g", v)
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", value)
	}
}
