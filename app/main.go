package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ./your_program.sh <command> [filename]")
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  tokenize <filename>  - Tokenize a file")
		fmt.Fprintln(os.Stderr, "  parse <filename>     - Parse a file to AST")
		fmt.Fprintln(os.Stderr, "  ir <filename>        - Convert a file to IR JSON")
		fmt.Fprintln(os.Stderr, "  ir --in              - Convert stdin input to IR JSON")
		fmt.Fprintln(os.Stderr, "  evaluate <filename>  - Evaluate a file and print result")
		fmt.Fprintln(os.Stderr, "  run <filename>       - Run a file without printing result")
		fmt.Fprintln(os.Stderr, "  repl                 - Start interactive REPL")
		fmt.Fprintln(os.Stderr, "  suite [filter]       - Run test suite with optional filter")
		os.Exit(1)
	}

	command := os.Args[1]

	// Check if command is repl
	if command == "repl" {
		handleRepl()
		return
	}

	// Special case for "ir" command with "--in" option
	if command == "ir" && len(os.Args) >= 3 && os.Args[2] == "--in" {
		handleIR("--in")
		return
	}

	// For other commands, require a filename
	if len(os.Args) < 3 && command != "suite" {
		fmt.Fprintln(os.Stderr, "Usage: ./your_program.sh <command> <filename>")
		fmt.Fprintln(os.Stderr, "       ./your_program.sh ir --in  # Read from stdin")
		fmt.Fprintln(os.Stderr, "       ./your_program.sh suite [filter]  # Run test suite")
		os.Exit(1)
	}

	filename := os.Args[2]

	switch command {
	case "tokenize":
		handleTokenize(filename)
	case "parse":
		handleParse(filename)
	case "ir":
		handleIR(filename)
	case "evaluate":
		handleEvaluate(filename, true)
	case "run":
		handleEvaluate(filename, false)
	case "suite":
		// For suite command, the filename is optional and used as a filter
		filter := ""
		if len(os.Args) >= 3 {
			filter = os.Args[2]
		}
		if err := RunSuite(filter); err != nil {
			fmt.Fprintf(os.Stderr, "Error running test suite: %v\n", err)
			os.Exit(1)
		}
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
	evaluator := NewEvaluator(NewDefaultScope(os.Stdout), os.Stdout)
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
		return "{}"
	case NumberValue:
		return fmt.Sprintf("%g", v.Val)
	case StringValue:
		return v.Val
	case BoolValue:
		if v.Val {
			return "true"
		}
		return "false"
	case UnionValue:
		if _, isNil := v.Value.(NilValue); isNil {
			return fmt.Sprintf("%s({})", v.Constructor)
		}
		return fmt.Sprintf("%s(%s)", v.Constructor, formatValue(v.Value))
	case RecordValue:
		if len(v.Fields) == 0 {
			return "{}"
		}
		// Sort keys for consistent output, with "return" first
		keys := make([]string, 0, len(v.Fields))
		for key := range v.Fields {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			// "return" comes first
			if keys[i] == "return" {
				return true
			}
			if keys[j] == "return" {
				return false
			}
			// Otherwise, alphabetical order
			return keys[i] < keys[j]
		})

		result := "{"
		first := true
		for _, key := range keys {
			if !first {
				result += ", "
			}
			fieldValue := v.Fields[key]
			// Quote strings in record context
			if sv, ok := fieldValue.(StringValue); ok {
				result += fmt.Sprintf("%s: \"%s\"", key, sv.Val)
			} else {
				result += fmt.Sprintf("%s: %s", key, formatValue(fieldValue))
			}
			first = false
		}
		result += "}"
		return result
	case ListValue:
		if len(v.Elements) == 0 {
			return "[]"
		}
		result := "["
		for i, element := range v.Elements {
			if i > 0 {
				result += ", "
			}
			// Quote strings in list context
			if sv, ok := element.(StringValue); ok {
				result += fmt.Sprintf("\"%s\"", sv.Val)
			} else {
				result += formatValue(element)
			}
		}
		result += "]"
		return result
	case LambdaValue:
		return "<lambda>"
	case FunValue:
		return fmt.Sprintf("<fn %s>", v.Val.Name)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func handleIR(filename string) {
	var tokens []Token
	var tokenizeErr error

	// Check if we should read from stdin
	if filename == "--in" {
		// Read from stdin
		var input string
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input += scanner.Text() + "\n"
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(65)
		}
		
		// Tokenize the input string
		tokens, tokenizeErr = TokenizeString(input)
	} else {
		// Tokenize the file
		tokens, tokenizeErr = TokenizeFile(filename)
	}

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

	// Convert the AST to IR format
	converter := NewIRConverter()
	irJson, err := converter.Convert(expr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "IR conversion error: %v\n", err)
		os.Exit(65)
	}

	// Print the IR JSON
	fmt.Println(string(irJson))
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
