package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ./your_program.sh tokenize <filename>")
		os.Exit(1)
	}

	command := os.Args[1]

	if command != "tokenize" {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
	}

	filename := os.Args[2]
	tokenized, err := Tokenize(filename)
	for _, tok := range tokenized {
		_, err := fmt.Fprintf(os.Stdout, "%s ", tok.String())
		if err != nil {
			os.Exit(1)
		}
	}
	fmt.Println("")
	if err != nil {
		os.Exit(1)
	}

	//fileContents, err := os.ReadFile(filename)
	//if err != nil {
	//	fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
	//	os.Exit(1)
	//}
	//
	//if len(fileContents) > 0 {
	//
	//} else {
	//	fmt.Println("EOF  null") // Placeholder, replace this line when implementing the scanner
	//}
}
