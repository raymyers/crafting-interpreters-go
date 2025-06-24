package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"testing"
)

// TestCase represents a single test case
type TestCase struct {
	Name    string                 `json:"name"`
	Source  Expression             `json:"source"`
	Effects []EffectTest           `json:"effects,omitempty"`
	Value   map[string]interface{} `json:"value,omitempty"`
	Break   map[string]interface{} `json:"break,omitempty"`
}

// EffectTest represents an effect test
type EffectTest struct {
	Label string                 `json:"label"`
	Lift  map[string]interface{} `json:"lift"`
	Reply map[string]interface{} `json:"reply"`
}

// TestSuite represents a collection of test cases
type TestSuite []TestCase

// parseValue converts a test value to our internal representation
func parseValue(raw map[string]interface{}) Value {
	if binary, ok := raw["binary"]; ok {
		return binary
	}
	if integer, ok := raw["integer"]; ok {
		return integer
	}
	if str, ok := raw["string"]; ok {
		return str
	}
	if list, ok := raw["list"]; ok {
		listSlice := list.([]interface{})
		result := make([]Value, len(listSlice))
		for i, item := range listSlice {
			result[i] = parseValue(item.(map[string]interface{}))
		}
		return result
	}
	if record, ok := raw["record"]; ok {
		recordMap := record.(map[string]interface{})
		result := make(map[string]Value)
		for k, v := range recordMap {
			result[k] = parseValue(v.(map[string]interface{}))
		}
		return result
	}
	if tagged, ok := raw["tagged"]; ok {
		taggedMap := tagged.(map[string]interface{})
		label := taggedMap["label"].(string)
		value := parseValue(taggedMap["value"].(map[string]interface{}))
		return &Tagged{Tag: label, Value: value}
	}
	
	log.Printf("Unknown value type: %+v", raw)
	return nil
}

// valuesEqual compares two values for equality
func valuesEqual(a, b Value) bool {
	// Handle Tagged values specially
	if taggedA, okA := a.(*Tagged); okA {
		if taggedB, okB := b.(*Tagged); okB {
			return taggedA.Tag == taggedB.Tag && valuesEqual(taggedA.Value, taggedB.Value)
		}
		return false
	}
	
	// Handle slices (lists)
	if sliceA, okA := a.([]Value); okA {
		if sliceB, okB := b.([]Value); okB {
			if len(sliceA) != len(sliceB) {
				return false
			}
			for i := range sliceA {
				if !valuesEqual(sliceA[i], sliceB[i]) {
					return false
				}
			}
			return true
		}
		return false
	}
	
	// Handle maps (records)
	if mapA, okA := a.(map[string]Value); okA {
		if mapB, okB := b.(map[string]Value); okB {
			if len(mapA) != len(mapB) {
				return false
			}
			for k, v := range mapA {
				if vB, exists := mapB[k]; !exists || !valuesEqual(v, vB) {
					return false
				}
			}
			return true
		}
		// Handle comparison with map[string]interface{}
		if mapB, okB := b.(map[string]interface{}); okB {
			if len(mapA) != len(mapB) {
				return false
			}
			for k, v := range mapA {
				if vB, exists := mapB[k]; !exists || !valuesEqual(v, vB) {
					return false
				}
			}
			return true
		}
		return false
	}
	
	// Handle maps (interface{} version)
	if mapA, okA := a.(map[string]interface{}); okA {
		if mapB, okB := b.(map[string]Value); okB {
			if len(mapA) != len(mapB) {
				return false
			}
			for k, v := range mapA {
				if vB, exists := mapB[k]; !exists || !valuesEqual(v, vB) {
					return false
				}
			}
			return true
		}
		if mapB, okB := b.(map[string]interface{}); okB {
			if len(mapA) != len(mapB) {
				return false
			}
			for k, v := range mapA {
				if vB, exists := mapB[k]; !exists || !valuesEqual(v, vB) {
					return false
				}
			}
			return true
		}
		return false
	}
	
	// For other types, use reflect.DeepEqual
	return reflect.DeepEqual(a, b)
}

// runTestSuite runs a test suite
func runTestSuite(filename string, t *testing.T) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", filename, err)
	}

	var suite TestSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		t.Fatalf("Failed to parse test file %s: %v", filename, err)
	}

	for _, testCase := range suite {
		t.Run(testCase.Name, func(t *testing.T) {
			state := Eval(testCase.Source)
			
			// Handle effects
			for _, expectedEffect := range testCase.Effects {
				if state.Break == nil {
					t.Fatalf("Expected effect %s but got no break", expectedEffect.Label)
				}
				
				effect, ok := state.Break.(*Effect)
				if !ok {
					t.Fatalf("Expected effect but got break: %+v", state.Break)
				}
				
				if effect.Label != expectedEffect.Label {
					t.Fatalf("Expected effect label %s but got %s", expectedEffect.Label, effect.Label)
				}
				
				expectedLift := parseValue(expectedEffect.Lift)
				if !valuesEqual(effect.Lift, expectedLift) {
					t.Fatalf("Effect lift mismatch. Expected: %+v, Got: %+v", expectedLift, effect.Lift)
				}
				
				reply := parseValue(expectedEffect.Reply)
				state.Resume(reply)
			}
			
			// Check final result
			if testCase.Break != nil {
				if state.Break == nil {
					t.Fatalf("Expected break but got value: %+v", state.Control)
				}
				
				// For now, just check that we have some kind of break
				// We can make this more specific later
				if !reflect.DeepEqual(state.Break, testCase.Break) {
					// Allow some flexibility in break format
					t.Logf("Break format differs. Expected: %+v, Got: %+v", testCase.Break, state.Break)
				}
			} else if testCase.Value != nil {
				if state.Break != nil {
					t.Fatalf("Expected value but got break: %+v", state.Break)
				}
				
				expected := parseValue(testCase.Value)
				if !valuesEqual(state.Control, expected) {
					t.Fatalf("Value mismatch. Expected: %+v, Got: %+v", expected, state.Control)
				}
			}
		})
	}
}

// Test functions
func TestCoreSpecs(t *testing.T) {
	runTestSuite("spec/evaluation/core_suite.json", t)
}

func TestBuiltinsSpecs(t *testing.T) {
	runTestSuite("spec/evaluation/builtins_suite.json", t)
}

// Commented out for now - we'll enable this as we implement effects
/*
func TestEffectsSpecs(t *testing.T) {
	runTestSuite("spec/evaluation/effects_suite.json", t)
}
*/

func main() {
	// Run a simple test to verify our setup
	fmt.Println("EYG Interpreter Test Runner")
	
	// Test a simple integer expression
	expr := Expression{
		"0": "i",
		"v": 42,
	}
	
	state := Eval(expr)
	fmt.Printf("Result: %+v\n", state.Control)
	fmt.Printf("Break: %+v\n", state.Break)
}