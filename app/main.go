package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/chzyer/readline"
)

// Args holds the command-line arguments
type Args struct {
	// Commands
	Tokenize *TokenizeCmd `arg:"subcommand:tokenize" help:"Tokenize a file or code string"`
	Parse    *ParseCmd    `arg:"subcommand:parse" help:"Parse a file or code string to AST"`
	IR       *IRCmd       `arg:"subcommand:ir" help:"Convert a file or code string to IR JSON"`
	Evaluate *EvaluateCmd `arg:"subcommand:evaluate" help:"Evaluate a file or code string and print result"`
	Run      *RunCmd      `arg:"subcommand:run" help:"Run a file or code string without printing result"`
	Repl     *ReplCmd     `arg:"subcommand:repl" help:"Start interactive REPL"`
	Suite    *SuiteCmd    `arg:"subcommand:suite" help:"Run test suite with optional filter"`
}

// TokenizeCmd represents the tokenize command
type TokenizeCmd struct {
	File string `arg:"positional" help:"File to tokenize"`
	Code string `arg:"-c,--code" help:"Code string to tokenize"`
}

// ParseCmd represents the parse command
type ParseCmd struct {
	File string `arg:"positional" help:"File to parse"`
	Code string `arg:"-c,--code" help:"Code string to parse"`
}

// IRCmd represents the ir command
type IRCmd struct {
	File  string `arg:"positional" help:"File to convert to IR"`
	Code  string `arg:"-c,--code" help:"Code string to convert to IR"`
	StdIn bool   `arg:"--in" help:"Read from stdin"`
}

// EvaluateCmd represents the evaluate command
type EvaluateCmd struct {
	File string `arg:"positional" help:"File to evaluate"`
	Code string `arg:"-c,--code" help:"Code string to evaluate"`
}

// RunCmd represents the run command
type RunCmd struct {
	File string `arg:"positional" help:"File to run"`
	Code string `arg:"-c,--code" help:"Code string to run"`
}

// ReplCmd represents the repl command
type ReplCmd struct{}

// SuiteCmd represents the suite command
type SuiteCmd struct {
	Filter string `arg:"positional" help:"Optional filter for test suite"`
}

func main() {
	var args Args
	p := arg.MustParse(&args)

	// Check which subcommand was invoked
	switch {
	case args.Tokenize != nil:
		handleTokenizeCmd(args.Tokenize)
	case args.Parse != nil:
		handleParseCmd(args.Parse)
	case args.IR != nil:
		handleIRCmd(args.IR)
	case args.Evaluate != nil:
		handleEvaluateCmd(args.Evaluate, true)
	case args.Run != nil:
		handleRunCmd(args.Run)
	case args.Repl != nil:
		handleRepl()
	case args.Suite != nil:
		handleSuiteCmd(args.Suite)
	default:
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}
}

func handleTokenizeCmd(cmd *TokenizeCmd) {
	// Validate that exactly one input source is provided
	if (cmd.File == "" && cmd.Code == "") || (cmd.File != "" && cmd.Code != "") {
		fmt.Fprintln(os.Stderr, "Error: Specify either a file or use -c/--code, but not both")
		os.Exit(1)
	}

	var tokens []Token
	var tokenizeErr error

	if cmd.Code != "" {
		tokens, tokenizeErr = TokenizeString(cmd.Code)
	} else {
		tokens, tokenizeErr = TokenizeFile(cmd.File)
	}

	for _, tok := range tokens {
		_, err := fmt.Fprintf(os.Stdout, "%s\n", tok.String())
		if err != nil {
			os.Exit(1)
		}
	}
	if tokenizeErr != nil {
		os.Exit(65)
	}
}

func handleParseCmd(cmd *ParseCmd) {
	// Validate that exactly one input source is provided
	if (cmd.File == "" && cmd.Code == "") || (cmd.File != "" && cmd.Code != "") {
		fmt.Fprintln(os.Stderr, "Error: Specify either a file or use -c/--code, but not both")
		os.Exit(1)
	}

	var tokens []Token
	var tokenizeErr error

	if cmd.Code != "" {
		tokens, tokenizeErr = TokenizeString(cmd.Code)
	} else {
		tokens, tokenizeErr = TokenizeFile(cmd.File)
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

	// Print the AST as S-expression
	printer := &AstPrinter{}
	result := printer.Print(expr)
	fmt.Println(result)
}

func handleIRCmd(cmd *IRCmd) {
	// Validate that exactly one input source is provided
	inputCount := 0
	if cmd.File != "" {
		inputCount++
	}
	if cmd.Code != "" {
		inputCount++
	}
	if cmd.StdIn {
		inputCount++
	}

	if inputCount != 1 {
		fmt.Fprintln(os.Stderr, "Error: Specify exactly one of: file, -c/--code, or --in")
		os.Exit(1)
	}

	var tokens []Token
	var tokenizeErr error

	if cmd.StdIn {
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
		tokens, tokenizeErr = TokenizeString(input)
	} else if cmd.Code != "" {
		tokens, tokenizeErr = TokenizeString(cmd.Code)
	} else {
		tokens, tokenizeErr = TokenizeFile(cmd.File)
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

func handleEvaluateCmd(cmd *EvaluateCmd, printResult bool) {
	// Validate that exactly one input source is provided
	if (cmd.File == "" && cmd.Code == "") || (cmd.File != "" && cmd.Code != "") {
		fmt.Fprintln(os.Stderr, "Error: Specify either a file or use -c/--code, but not both")
		os.Exit(1)
	}

	var tokens []Token
	var tokenizeErr error

	if cmd.Code != "" {
		tokens, tokenizeErr = TokenizeString(cmd.Code)
	} else {
		tokens, tokenizeErr = TokenizeFile(cmd.File)
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

func handleRunCmd(cmd *RunCmd) {
	evaluateCmd := &EvaluateCmd{
		File: cmd.File,
		Code: cmd.Code,
	}
	handleEvaluateCmd(evaluateCmd, false)
}

func handleSuiteCmd(cmd *SuiteCmd) {
	if err := RunSuite(cmd.Filter); err != nil {
		fmt.Fprintf(os.Stderr, "Error running test suite: %v\n", err)
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
