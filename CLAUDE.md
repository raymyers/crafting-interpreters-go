# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go implementation of the Lox interpreter following the "Crafting Interpreters" book by Robert Nystrom. It's part of the CodeCrafters "Build your own Interpreter" challenge. The interpreter implements a tree-walk interpreter that can tokenize, parse, and evaluate Lox expressions.

## Common Commands

### Building and Running
- `make build` - Build the interpreter binary to `build/interpreter`
- `make run ARGS="tokenize example.lox"` - Run the interpreter with arguments
- `./your_program.sh tokenize filename.lox` - Tokenize a Lox file
- `./your_program.sh parse filename.lox` - Parse a Lox file and print AST
- `./your_program.sh evaluate filename.lox` - Evaluate a Lox expression

### Testing
- `make test` - Run all tests with verbose output
- `make test-coverage` - Run tests with coverage report
- `go test ./app -v -run TestEvaluatorCases/GroupedString` - Run a specific test case

**Important**: Test cases are defined in YAML files:
- `tokenizer_tests.yaml` - Tests for lexical analysis
- `parser_tests.yaml` - Tests for parsing and AST generation
- `evaluator_tests.yaml` - Tests for expression evaluation and runtime behavior

When adding new features, always add corresponding test cases to the appropriate YAML file rather than creating manual test files. The test framework automatically reads these YAML files and runs the test cases.

### Code Quality
- `make fmt` - Format Go code
- `make vet` - Run Go vet for static analysis
- `make check` - Run fmt, vet, and test together
- `make lint` - Run golangci-lint (requires golangci-lint installation)

### Development
- `make deps` - Download and tidy Go dependencies
- `make clean` - Remove build artifacts

## Architecture Overview

The interpreter follows a traditional pipeline architecture:

### Core Components

1. **Tokenizer** (`tokenizer.go`): Lexical analysis that converts source code into tokens
   - Handles Lox operators, keywords, literals, and comments
   - Supports string literals with embedded newlines
   - Manages line numbers for error reporting

2. **Parser** (`parser.go`): Recursive descent parser that builds an AST
   - Implements expression grammar with proper precedence
   - Handles binary operations, unary operations, grouping, and literals
   - Uses visitor pattern for AST traversal

3. **AST** (`ast.go`): Abstract Syntax Tree definitions
   - Four main expression types: Binary, Unary, Literal, Grouping
   - Implements visitor pattern with ExprVisitor interface

4. **Evaluator** (`evaluator.go`): Tree-walk interpreter that evaluates expressions
   - Implements ExprVisitor to evaluate AST nodes
   - Handles type coercion and runtime errors
   - Currently supports literals and grouping expressions

5. **Printer** (`printer.go`): AST printer that outputs S-expressions
   - Formats AST as Lisp-style S-expressions for debugging

### Data Flow

1. Source code → Tokenizer → Tokens
2. Tokens → Parser → AST
3. AST → Evaluator → Result value
4. AST → Printer → S-expression (for parse command)

### Testing Strategy

The project uses approval testing with the `go-approval-tests` library:
- Test cases are defined in `*_test.go` files
- Expected outputs are stored in `*.approved.txt` files
- Test failures generate `*.received.txt` files for comparison
- Tests cover tokenization, parsing, and evaluation separately

### Current Implementation Status

- ✅ Tokenization: Complete with all Lox tokens
- ✅ Parsing: Complete expression parsing with precedence
- ✅ Evaluation: Literals and grouping expressions implemented
- ❌ Evaluation: Binary and unary operations need implementation
- ❌ Statements: Not yet implemented
- ❌ Variables and scoping: Not yet implemented

### Key Files

- `main.go`: CLI entry point handling tokenize/parse/evaluate commands
- `token.go`: Token type definitions and string representations
- `tokenizer.go`: Lexical analyzer implementation
- `parser.go`: Recursive descent parser
- `ast.go`: AST node definitions with visitor pattern
- `evaluator.go`: Expression evaluator (partially implemented)
- `printer.go`: AST to S-expression printer

### Error Handling

- Tokenization errors: Printed to stderr with line numbers
- Parse errors: Return error with descriptive messages
- Evaluation errors: Return error for runtime issues
- Exit codes: 65 for syntax errors, 70 for runtime errors