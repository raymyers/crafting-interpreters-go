package main

import (
	"fmt"
	"os"
	"strings"
	
	"github.com/chzyer/readline"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ./your_program.sh <command> [filename]")
		os.Exit(1)
	}

	command := os.Args[1]
	
	// Check if command is repl
	if command == "repl" {
		handleRepl()
		return
	}
	
	// For other commands, require a filename
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ./your_program.sh <command> <filename>")
		os.Exit(1)
	}
	
	filename := os.Args[2]

	switch command {
	case "tokenize":
		handleTokenize(filename)
	case "parse":
		handleParse(filename)
	case "evaluate":
		handleEvaluate(filename, true)
	case "run":
		handleEvaluate(filename, false)
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

func handleEvaluate(filename string, printResult bool) {
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
	evaluator := NewEvaluator(NewScope(nil), os.Stdout)
	result := evaluator.Evaluate(expr)
	switch result.(type) {
	case ErrorValue:
		errorText := fmt.Errorf("[Line %d]\nError: %s", result.(ErrorValue).Line, result.(ErrorValue).Message)
		fmt.Fprintf(os.Stderr, "%v\n", errorText)
		os.Exit(70)
	default:
		if printResult {
			fmt.Println(formatValue(result))
		}
	}

}

func formatValue(value Value) string {
	switch v := value.(type) {
	case NilValue:
		return "nil"
	case NumberValue:
		return fmt.Sprintf("%g", v.Val)
	case StringValue:
		return v.Val
	case BoolValue:
		if v.Val {
			return "true"
		}
		return "false"
	case FunValue:
		return fmt.Sprintf("<fn %s>", v.Val.Name)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func handleRepl() {
	// Create readline instance for better line editing
	rl, err := readline.New("> ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing readline: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	// Create a persistent scope that will be reused across REPL commands
	scope := NewScope(nil)
	
	fmt.Println("Welcome to Lox REPL! Type 'exit' to quit.")
	
	for {
		// Read line from user
		line, err := rl.Readline()
		if err != nil { // io.EOF or other error
			break
		}
		
		// Handle exit command
		line = strings.TrimSpace(line)
		if line == "exit" || line == "quit" {
			break
		}
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Tokenize the input
		tokens, tokenizeErr := TokenizeString(line)
		
		// Print tokenization errors but continue
		if tokenizeErr != nil {
			fmt.Fprintf(os.Stderr, "Tokenization error: %v\n", tokenizeErr)
			continue
		}
		
		// Parse the tokens
		parser := NewParser(tokens)
		expr, parseErr := parser.Parse()
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Parse error: %v\n", parseErr)
			continue
		}
		
		// Evaluate the expression with the persistent scope
		evaluator := NewEvaluator(scope, os.Stdout)
		result := evaluator.Evaluate(expr)
		
		// Handle evaluation errors
		if errVal, isError := result.(ErrorValue); isError {
			fmt.Fprintf(os.Stderr, "Runtime error: %s\n", errVal.Message)
			continue
		}
		
		// Print the result only if it's not nil (statements return nil)
		if _, isNil := result.(NilValue); !isNil {
			fmt.Println(formatValue(result))
		}
	}
	
	fmt.Println("Goodbye!")
}
