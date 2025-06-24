#!/bin/bash

# Test script to verify IR conversion against fixture examples

# Set up colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to run a test case
run_test() {
  local name=$1
  local input=$2
  local expected_name=$3
  
  echo -e "\nTesting ${name}..."
  
  # Run the IR command with the input
  result=$(echo "$input" | ./your_program.sh ir --in)
  
  # Check if the result contains the expected name
  if echo "$result" | grep -q "\"name\": \"$expected_name\""; then
    echo -e "${GREEN}PASS${NC}: IR output contains expected name '$expected_name'"
  else
    echo -e "${RED}FAIL${NC}: IR output does not contain expected name '$expected_name'"
    echo "Output: $result"
    exit 1
  fi
}

# Test cases
run_test "Variable" "foo" "variable"
run_test "Function" "|x| { x }" "function"
run_test "Apply" "(|x| { x })(\"foo\")" "apply"
run_test "Let" "x = \"hi\"; x" "let"
run_test "Integer" "5" "integer"
run_test "String" "\"hello\"" "string"
run_test "Empty List" "[]" "empty list"
run_test "Empty Record" "{}" "empty record"

echo -e "\n${GREEN}All tests passed!${NC}"