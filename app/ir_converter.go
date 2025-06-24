package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// IRNode represents a node in the IR format
type IRNode struct {
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

	// Special handling for Statements to return each statement as a separate IR node
	if statements, ok := expr.(*Statements); ok {
		var allNodes []IRNode
		for _, stmt := range statements.Exprs {
			nodes := ic.convertExpr(stmt)
			allNodes = append(allNodes, nodes...)
		}
		return json.MarshalIndent(allNodes, "", "  ")
	}

	nodes := ic.convertExpr(expr)
	return json.MarshalIndent(nodes, "", "  ")
}

// convertExpr converts an expression to IR nodes
func (ic *IRConverter) convertExpr(expr Expr) []IRNode {
	switch e := expr.(type) {
	case *Variable:
		return []IRNode{ic.convertVariable(e)}
	case *Lambda:
		return []IRNode{ic.convertLambda(e)}
	case *Call:
		return []IRNode{ic.convertCall(e)}
	case *Let:
		return []IRNode{ic.convertLet(e)}
	case *VarStatement:
		return []IRNode{ic.convertVarStatement(e)}
	case *Literal:
		return ic.convertLiteral(e)
	case *EmptyRecord:
		return []IRNode{ic.convertEmptyRecord(e)}
	case *Record:
		return []IRNode{ic.convertRecord(e)}
	case *List:
		return []IRNode{ic.convertList(e)}
	case *Access:
		return []IRNode{ic.convertAccess(e)}
	case *Builtin:
		return []IRNode{ic.convertBuiltin(e)}
	case *Union:
		return []IRNode{ic.convertUnion(e)}
	case *Perform:
		return []IRNode{ic.convertPerform(e)}
	case *Handle:
		return []IRNode{ic.convertHandle(e)}
	case *Binary:
		return []IRNode{ic.convertBinary(e)}
	case *Grouping:
		return ic.convertGrouping(e)
	case *Statements:
		var nodes []IRNode
		for _, stmt := range e.Exprs {
			nodes = append(nodes, ic.convertExpr(stmt)...)
		}
		return nodes
	default:
		// Default fallback for unsupported types
		return []IRNode{{
			Name:   fmt.Sprintf("unsupported_%T", expr),
			Source: map[string]interface{}{"0": "z"}, // vacant
			Code:   fmt.Sprintf("unsupported(%T)", expr),
		}}
	}
}

// convertVariable converts a Variable expression to IR
func (ic *IRConverter) convertVariable(expr *Variable) IRNode {
	return IRNode{
		Name: "variable",
		Source: map[string]interface{}{
			"0": "v",
			"l": expr.Name.Lexeme,
		},
		Code: expr.Name.Lexeme,
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

	bodyNodes := ic.convertExpr(expr.Body)
	var bodySource map[string]interface{}
	var bodyCode string

	if len(bodyNodes) > 0 {
		bodySource = bodyNodes[0].Source
		bodyCode = bodyNodes[0].Code
	} else {
		bodySource = map[string]interface{}{"0": "z"} // vacant
		bodyCode = "vacant()"
	}

	return IRNode{
		Name: "function",
		Source: map[string]interface{}{
			"0": "f",
			"l": paramName,
			"b": bodySource,
		},
		Code: fmt.Sprintf("|%s| { %s }", paramName, bodyCode),
	}
}

// convertCall converts a Call expression to IR
func (ic *IRConverter) convertCall(expr *Call) IRNode {
	calleeNodes := ic.convertExpr(expr.Callee)
	var calleeSource map[string]interface{}
	var calleeCode string

	if len(calleeNodes) > 0 {
		calleeSource = calleeNodes[0].Source
		calleeCode = calleeNodes[0].Code
	} else {
		calleeSource = map[string]interface{}{"0": "z"} // vacant
		calleeCode = "vacant()"
	}

	// For simplicity, we'll handle only single argument calls in this example
	var argSource map[string]interface{}
	var argCode string

	if len(expr.Arguments) > 0 {
		argNodes := ic.convertExpr(expr.Arguments[0])
		if len(argNodes) > 0 {
			argSource = argNodes[0].Source
			argCode = argNodes[0].Code
		} else {
			argSource = map[string]interface{}{"0": "z"} // vacant
			argCode = "vacant()"
		}
	} else {
		argSource = map[string]interface{}{"0": "z"} // vacant
		argCode = "vacant()"
	}

	return IRNode{
		Name: "apply",
		Source: map[string]interface{}{
			"0": "a",
			"f": calleeSource,
			"a": argSource,
		},
		Code: fmt.Sprintf("(%s)(%s)", calleeCode, argCode),
	}
}

// convertLet converts a Let expression to IR
func (ic *IRConverter) convertLet(expr *Let) IRNode {
	var patternName string
	if variable, ok := expr.Pattern.(*Variable); ok {
		patternName = variable.Name.Lexeme
	} else {
		patternName = "x" // Default name for non-variable patterns
	}

	valueNodes := ic.convertExpr(expr.Value)
	var valueSource map[string]interface{}
	var valueCode string

	if len(valueNodes) > 0 {
		valueSource = valueNodes[0].Source
		valueCode = valueNodes[0].Code
	} else {
		valueSource = map[string]interface{}{"0": "z"} // vacant
		valueCode = "vacant()"
	}

	bodyNodes := ic.convertExpr(expr.Body)
	var bodySource map[string]interface{}
	var bodyCode string

	if len(bodyNodes) > 0 {
		bodySource = bodyNodes[0].Source
		bodyCode = bodyNodes[0].Code
	} else {
		bodySource = map[string]interface{}{"0": "z"} // vacant
		bodyCode = "vacant()"
	}

	return IRNode{
		Name: "let",
		Source: map[string]interface{}{
			"0": "l",
			"l": patternName,
			"v": valueSource,
			"t": bodySource,
		},
		Code: fmt.Sprintf("%s = %s\n%s", patternName, valueCode, bodyCode),
	}
}

// convertLiteral converts a Literal expression to IR
func (ic *IRConverter) convertLiteral(expr *Literal) []IRNode {
	switch v := expr.Value.(type) {
	case StringValue:
		return []IRNode{{
			Name: "string",
			Source: map[string]interface{}{
				"0": "s",
				"v": v.Val,
			},
			Code: fmt.Sprintf("\"%s\"", v.Val),
		}}
	case NumberValue:
		return []IRNode{{
			Name: "integer",
			Source: map[string]interface{}{
				"0": "i",
				"v": int(v.Val), // Convert to int for simplicity
			},
			Code: fmt.Sprintf("%d", int(v.Val)),
		}}
	case BoolValue:
		// Represent booleans as tagged unions in IR
		if v.Val {
			return []IRNode{{
				Name: "tag",
				Source: map[string]interface{}{
					"0": "t",
					"l": "true",
				},
				Code: "tag(\"true\")",
			}}
		} else {
			return []IRNode{{
				Name: "tag",
				Source: map[string]interface{}{
					"0": "t",
					"l": "false",
				},
				Code: "tag(\"false\")",
			}}
		}
	case NilValue:
		return []IRNode{{
			Name: "vacant",
			Source: map[string]interface{}{
				"0": "z",
			},
			Code: "vacant()",
		}}
	default:
		return []IRNode{{
			Name: "vacant",
			Source: map[string]interface{}{
				"0": "z",
			},
			Code: "vacant()",
		}}
	}
}

// convertEmptyRecord converts an EmptyRecord expression to IR
func (ic *IRConverter) convertEmptyRecord(expr *EmptyRecord) IRNode {
	return IRNode{
		Name: "empty record",
		Source: map[string]interface{}{
			"0": "u",
		},
		Code: "{}",
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
		valueNodes := ic.convertExpr(field.Value)
		var valueSource map[string]interface{}
		
		if len(valueNodes) > 0 {
			valueSource = valueNodes[0].Source
		} else {
			valueSource = map[string]interface{}{"0": "z"} // vacant
		}
		
		// Extend the record with the field
		recordNode = IRNode{
			Name: "extend record",
			Source: map[string]interface{}{
				"0": "e",
				"l": field.Name,
				"v": valueSource,
			},
			Code: fmt.Sprintf("extend(\"%s\")", field.Name),
		}
	}
	
	return recordNode
}

// convertList converts a List expression to IR
func (ic *IRConverter) convertList(expr *List) IRNode {
	if len(expr.Elements) == 0 {
		return IRNode{
			Name: "empty list",
			Source: map[string]interface{}{
				"0": "ta",
			},
			Code: "[]",
		}
	}
	
	// For simplicity, we'll just handle the first element
	elemNodes := ic.convertExpr(expr.Elements[0])
	var elemSource map[string]interface{}
	var elemCode string
	
	if len(elemNodes) > 0 {
		elemSource = elemNodes[0].Source
		elemCode = elemNodes[0].Code
	} else {
		elemSource = map[string]interface{}{"0": "z"} // vacant
		elemCode = "vacant()"
	}
	
	return IRNode{
		Name: "list cons",
		Source: map[string]interface{}{
			"0": "c",
			"h": elemSource,
			"t": map[string]interface{}{"0": "ta"}, // empty tail
		},
		Code: fmt.Sprintf("cons(%s, [])", elemCode),
	}
}

// convertAccess converts an Access expression to IR
func (ic *IRConverter) convertAccess(expr *Access) IRNode {
	objectNodes := ic.convertExpr(expr.Object)
	var objectSource map[string]interface{}
	var objectCode string
	
	if len(objectNodes) > 0 {
		objectSource = objectNodes[0].Source
		objectCode = objectNodes[0].Code
	} else {
		objectSource = map[string]interface{}{"0": "z"} // vacant
		objectCode = "vacant()"
	}
	
	return IRNode{
		Name: "select field",
		Source: map[string]interface{}{
			"0": "g",
			"l": expr.Name,
			"r": objectSource,
		},
		Code: fmt.Sprintf("%s.%s", objectCode, expr.Name),
	}
}

// convertBuiltin converts a Builtin expression to IR
func (ic *IRConverter) convertBuiltin(expr *Builtin) IRNode {
	return IRNode{
		Name: "add integer builtin",
		Source: map[string]interface{}{
			"0": "b",
			"l": expr.Name,
		},
		Code: fmt.Sprintf("builtin(\"%s\")", expr.Name),
	}
}

// convertUnion converts a Union expression to IR
func (ic *IRConverter) convertUnion(expr *Union) IRNode {
	valueNodes := ic.convertExpr(expr.Value)
	var valueSource map[string]interface{}
	var valueCode string
	
	if len(valueNodes) > 0 {
		valueSource = valueNodes[0].Source
		valueCode = valueNodes[0].Code
	} else {
		valueSource = map[string]interface{}{"0": "z"} // vacant
		valueCode = "vacant()"
	}
	
	return IRNode{
		Name: "tag",
		Source: map[string]interface{}{
			"0": "t",
			"l": expr.Constructor,
			"v": valueSource,
		},
		Code: fmt.Sprintf("tag(\"%s\", %s)", expr.Constructor, valueCode),
	}
}

// convertPerform converts a Perform expression to IR
func (ic *IRConverter) convertPerform(expr *Perform) IRNode {
	return IRNode{
		Name: "perform effect",
		Source: map[string]interface{}{
			"0": "p",
			"l": expr.Effect,
		},
		Code: fmt.Sprintf("perform(\"%s\")", expr.Effect),
	}
}

// convertHandle converts a Handle expression to IR
func (ic *IRConverter) convertHandle(expr *Handle) IRNode {
	return IRNode{
		Name: "handle effect",
		Source: map[string]interface{}{
			"0": "h",
			"l": expr.Effect,
		},
		Code: fmt.Sprintf("handle(\"%s\")", expr.Effect),
	}
}

// convertBinary converts a Binary expression to IR
// This is a simplified implementation that only handles binary data
func (ic *IRConverter) convertBinary(expr *Binary) IRNode {
	// For simplicity, we'll just create a binary node with a sample value
	sampleBytes := []byte{0x01}
	encoded := base64.StdEncoding.EncodeToString(sampleBytes)
	
	return IRNode{
		Name: "binary",
		Source: map[string]interface{}{
			"0": "x",
			"v": map[string]interface{}{
				"/": map[string]interface{}{
					"bytes": encoded,
				},
			},
		},
		Code: "binary(0x01)",
	}
}

// convertGrouping converts a Grouping expression to IR
func (ic *IRConverter) convertGrouping(expr *Grouping) []IRNode {
	// A grouping just passes through to its inner expression
	return ic.convertExpr(expr.Expression)
}

// convertVarStatement converts a VarStatement expression to IR
func (ic *IRConverter) convertVarStatement(expr *VarStatement) IRNode {
	valueNodes := ic.convertExpr(expr.Expression)
	var valueSource map[string]interface{}
	var valueCode string
	
	if len(valueNodes) > 0 {
		valueSource = valueNodes[0].Source
		valueCode = valueNodes[0].Code
	} else {
		valueSource = map[string]interface{}{"0": "z"} // vacant
		valueCode = "vacant()"
	}
	
	return IRNode{
		Name: "let",
		Source: map[string]interface{}{
			"0": "l",
			"l": expr.name,
			"v": valueSource,
			"t": map[string]interface{}{
				"0": "v",
				"l": expr.name,
			},
		},
		Code: fmt.Sprintf("%s = %s\n%s", expr.name, valueCode, expr.name),
	}
}