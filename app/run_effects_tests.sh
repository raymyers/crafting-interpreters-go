#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
TOTAL=0

# Function to run a single test
run_test() {
    local name="$1"
    local input="$2"
    local expected="$3"
    local description="$4"
    
    echo -e "${BLUE}Running: $name${NC}"
    echo "Description: $description"
    
    # Create temp file
    echo "$input" > temp_effect_test.eyg
    
    # Run evaluator
    result=$(export PATH=$PATH:/usr/local/go/bin && go run main.go ast.go evaluator.go parser.go printer.go token.go tokenizer.go evaluate temp_effect_test.eyg 2>&1)
    
    # Clean up
    rm -f temp_effect_test.eyg
    
    # Compare results
    if [ "$result" = "$expected" ]; then
        echo -e "${GREEN}✅ PASS${NC}"
        ((PASSED++))
    else
        echo -e "${RED}❌ FAIL${NC}"
        echo "Expected: $expected"
        echo "Got:      $result"
    fi
    echo
    ((TOTAL++))
}

echo "Running Effect Handling Tests"
echo "============================="
echo

# Test 1: Basic perform
run_test "Basic perform" \
    'perform Alert("test")' \
    '{Alert [{test}] {0xc0000980d0 0xc0000a4088}}' \
    "Basic perform should return an EffectValue"

# Test 2: Simple handle
run_test "Simple handle" \
    'handle Alert(|value, resume| value, perform Alert("test"))' \
    "test" \
    "Simple handler that returns the effect value"

# Test 3: Effect in lambda
run_test "Effect in lambda" \
    'f = |_| perform Alert("test")
handle Alert(|value, resume| value, f({}))' \
    "test" \
    "Effects inside lambdas should bubble up to handlers"

# Test 4: Single effect in sequence
run_test "Single effect in sequence" \
    'run = |_| {
  _ = perform Alert("first")
  {}
}
handle Alert(|value, resume| value, run({}))' \
    "first" \
    "First effect in sequence should be caught"

# Test 4b: Debug single effect in sequence
run_test "Debug single effect in sequence" \
    'run = |_| {
  _ = perform Alert("first")
  {}
}
handle Alert(|value, resume| {
  _ = perform Log("Handler called with: " + value)
  value
}, run({}))' \
    "first" \
    "Debug: Handler should be called and return the value"

# Test 5: Handler fallback
run_test "Handler fallback" \
    'handle Alert(|value, resume| value, |_| "fallback")' \
    "fallback" \
    "Fallback lambda should be called when no effects occur"

echo "Results: $PASSED/$TOTAL tests passed"
if [ $PASSED -eq $TOTAL ]; then
    echo -e "${GREEN}🎉 All tests passed!${NC}"
else
    echo -e "${RED}Some tests failed${NC}"
fi