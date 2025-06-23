package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type EffectTest struct {
	Name        string `yaml:"name"`
	Input       string `yaml:"input"`
	Expected    string `yaml:"expected"`
	Description string `yaml:"description"`
}

type EffectTestSuite struct {
	Tests []EffectTest `yaml:"tests"`
}

func main() {
	// Read the test file
	data, err := os.ReadFile("evaluator_effects_tests.yaml")
	if err != nil {
		fmt.Printf("Error reading test file: %v\n", err)
		return
	}

	// Parse the YAML
	var suite EffectTestSuite
	err = yaml.Unmarshal(data, &suite)
	if err != nil {
		fmt.Printf("Error parsing YAML: %v\n", err)
		return
	}

	// Run each test
	passed := 0
	total := len(suite.Tests)

	for _, test := range suite.Tests {
		fmt.Printf("Running: %s\n", test.Name)
		fmt.Printf("Description: %s\n", test.Description)
		
		// Create a temporary file with the test input
		tempFile := "temp_effect_test.eyg"
		err := os.WriteFile(tempFile, []byte(test.Input), 0644)
		if err != nil {
			fmt.Printf("❌ FAIL: Could not create temp file: %v\n\n", err)
			continue
		}

		// Run the evaluator
		result, err := runEvaluator(tempFile)
		if err != nil {
			fmt.Printf("❌ FAIL: Evaluator error: %v\n\n", err)
			continue
		}

		// Clean up temp file
		os.Remove(tempFile)

		// Compare result
		result = strings.TrimSpace(result)
		expected := strings.TrimSpace(test.Expected)
		
		if result == expected {
			fmt.Printf("✅ PASS\n\n")
			passed++
		} else {
			fmt.Printf("❌ FAIL\n")
			fmt.Printf("Expected: %s\n", expected)
			fmt.Printf("Got:      %s\n\n", result)
		}
	}

	fmt.Printf("Results: %d/%d tests passed\n", passed, total)
	if passed == total {
		fmt.Println("🎉 All tests passed!")
	}
}

func runEvaluator(filename string) (string, error) {
	// This would normally run the evaluator, but for now we'll use a placeholder
	// In a real implementation, this would execute the Go evaluator
	return "placeholder", nil
}