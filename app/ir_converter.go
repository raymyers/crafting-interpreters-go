package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// IRNode represents a node in the IR format
type IRNode map[string]interface{}
type IRTestCase struct {
	Name   string                 `json:"name"`
	Source map[string]interface{} `json:"source"`
	Code   string                 `json:"code"`
}

// IRConverter converts AST to IR format
type IRConverter struct{}

// NewIRConverter creates a new IR converter
func NewIRConverter() *IRConverter {
	return &IRConverter{}
}

// Convert converts an AST expression to IR format
func (ic *IRConverter) Convert(expr Expr) ([]byte, error) {
	if expr == nil {
		return nil, fmt.Errorf("cannot convert nil expression")
	}

	node := ic.convertExpr(expr)
	return json.MarshalIndent(node, "", "  ")
}

// convertExpr converts an expression to IR nodes
func (ic *IRConverter) convertExpr(expr Expr) IRNode {
	switch e := expr.(type) {
	case *Variable:
		return ic.convertVariable(e)
	case *Lambda:
		return ic.convertLambda(e)
	case *Call:
		return ic.convertCall(e)
	case *Var:
		return ic.convertLet(e)
	case *LetStatement:
		return ic.convertVarStatement(e)
	case *Literal:
		return ic.convertLiteral(e)
	case *EmptyRecord:
		return ic.convertEmptyRecord(e)
	case *Record:
		return ic.convertRecord(e)
	case *List:
		return ic.convertList(e)
	case *Access:
		return ic.convertAccess(e)
	case *Builtin:
		return ic.convertBuiltin(e)
	case *Union:
		return ic.convertUnion(e)
	case *Perform:
		return ic.convertPerform(e)
	case *Handle:
		return ic.convertHandle(e)
	case *Binary:
		return ic.convertBinary(e)
	case *Grouping:
		return ic.convertGrouping(e)
	default:
		// Default fallback for unsupported types
		return map[string]interface{}{"0": "z"}
	}
}

// convertVariable converts a Variable expression to IR
func (ic *IRConverter) convertVariable(expr *Variable) IRNode {
	return map[string]interface{}{
		"0": "v",
		"l": expr.Name.Lexeme,
	}
}

// convertLambda converts a Lambda expression to IR
func (ic *IRConverter) convertLambda(expr *Lambda) IRNode {
	// For simplicity, we'll handle only single parameter lambdas in this example
	var paramName string
	if len(expr.Parameters) > 0 {
		paramName = expr.Parameters[0]
	} else {
		paramName = ""
	}

	bodyNode := ic.convertExpr(expr.Body)

	return map[string]interface{}{
		"0": "f",
		"l": paramName,
		"b": bodyNode,
	}
}

// convertCall converts a Call expression to IR
func (ic *IRConverter) convertCall(expr *Call) IRNode {
	calleeNode := map[string]interface{}{"0": "z"} // vacant

	// For simplicity, we'll handle only single argument calls in this example
	var argSource map[string]interface{}

	return map[string]interface{}{
		"0": "a",
		"f": calleeNode,
		"a": argSource,
	}
}

// convertLet converts a Var expression to IR
func (ic *IRConverter) convertLet(expr *Var) IRNode {
	var patternName string
	if variable, ok := expr.Pattern.(*Variable); ok {
		patternName = variable.Name.Lexeme
	} else {
		patternName = "x" // Default name for non-variable patterns
	}

	valueNode := ic.convertExpr(expr.Value)

	bodyNode := ic.convertExpr(expr.Body)

	return map[string]interface{}{
		"0": "l",
		"l": patternName,
		"v": valueNode,
		"t": bodyNode,
	}
}

// convertLiteral converts a Literal expression to IR
func (ic *IRConverter) convertLiteral(expr *Literal) IRNode {
	switch v := expr.Value.(type) {
	case StringValue:
		return map[string]interface{}{
			"0": "s",
			"v": v.Val,
		}
	case NumberValue:
		return map[string]interface{}{
			"0": "i",
			"v": int(v.Val), // Convert to int for simplicity
		}
	case BoolValue:
		// Represent booleans as tagged unions in IR
		if v.Val {
			return map[string]interface{}{
				"0": "t",
				"l": "true",
			}
		} else {
			return map[string]interface{}{
				"0": "t",
				"l": "false",
			}

		}
	case NilValue:
		return map[string]interface{}{
			"0": "z",
		}
	default:
		return map[string]interface{}{
			"0": "z",
		}
	}
}

// convertEmptyRecord converts an EmptyRecord expression to IR
func (ic *IRConverter) convertEmptyRecord(expr *EmptyRecord) IRNode {
	return map[string]interface{}{
		"0": "u",
	}
}

// convertRecord converts a Record expression to IR
// This is a simplified implementation
func (ic *IRConverter) convertRecord(expr *Record) IRNode {
	// Start with an empty record
	recordNode := ic.convertEmptyRecord(nil)

	// For simplicity, we'll just use the first field if available
	if len(expr.Fields) > 0 {
		field := expr.Fields[0]
		valueNode := ic.convertExpr(field.Value)

		// Extend the record with the field
		recordNode = map[string]interface{}{
			"0": "e",
			"l": field.Name,
			"v": valueNode,
		}
	}

	return recordNode
}

// convertList converts a List expression to IR
func (ic *IRConverter) convertList(expr *List) IRNode {
	if len(expr.Elements) == 0 {
		return map[string]interface{}{
			"0": "ta",
		}
	}

	// For simplicity, we'll just handle the first element
	elemNode := ic.convertExpr(expr.Elements[0])

	return map[string]interface{}{
		"0": "c",
		"h": elemNode,
		"t": map[string]interface{}{"0": "ta"}, // empty tail
	}
}

// convertAccess converts an Access expression to IR
func (ic *IRConverter) convertAccess(expr *Access) IRNode {
	objectNode := ic.convertExpr(expr.Object)

	return map[string]interface{}{
		"0": "g",
		"l": expr.Name,
		"r": objectNode,
	}
}

// convertBuiltin converts a Builtin expression to IR
func (ic *IRConverter) convertBuiltin(expr *Builtin) IRNode {
	return map[string]interface{}{
		"0": "b",
		"l": expr.Name,
	}
}

// convertUnion converts a Union expression to IR
func (ic *IRConverter) convertUnion(expr *Union) IRNode {
	valueNode := ic.convertExpr(expr.Value)

	return map[string]interface{}{
		"0": "t",
		"l": expr.Constructor,
		"v": valueNode,
	}
}

// convertPerform converts a Perform expression to IR
func (ic *IRConverter) convertPerform(expr *Perform) IRNode {
	return map[string]interface{}{
		"0": "p",
		"l": expr.Effect,
	}
}

// convertHandle converts a Handle expression to IR
func (ic *IRConverter) convertHandle(expr *Handle) IRNode {
	return map[string]interface{}{
		"0": "h",
		"l": expr.Effect,
	}
}

// convertBinary converts a Binary expression to IR
// This is a simplified implementation that only handles binary data
func (ic *IRConverter) convertBinary(expr *Binary) IRNode {
	// For simplicity, we'll just create a binary node with a sample value
	sampleBytes := []byte{0x01}
	encoded := base64.StdEncoding.EncodeToString(sampleBytes)

	return map[string]interface{}{
		"0": "x",
		"v": map[string]interface{}{
			"/": map[string]interface{}{
				"bytes": encoded,
			},
		},
	}
}

// convertGrouping converts a Grouping expression to IR
func (ic *IRConverter) convertGrouping(expr *Grouping) IRNode {
	// A grouping just passes through to its inner expression
	return ic.convertExpr(expr.Expression)
}

// convertVarStatement converts a LetStatement expression to IR
func (ic *IRConverter) convertVarStatement(expr *LetStatement) IRNode {
	valueNode := ic.convertExpr(expr.Expression)

	return map[string]interface{}{
		"0": "l",
		"l": expr.name,
		"v": valueNode,
		"t": map[string]interface{}{
			"0": "v",
			"l": expr.name,
		},
	}
}
