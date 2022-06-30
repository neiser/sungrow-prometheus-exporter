package util

import (
	"errors"
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/ast"
	"github.com/antonmedv/expr/parser"
	"github.com/antonmedv/expr/vm"
)

type EnvEntry struct {
	identifier string
	value      interface{}
}

func (e EnvEntry) GetKey() string {
	return e.identifier
}

func Env(identifier string, value interface{}) *EnvEntry {
	return &EnvEntry{identifier, value}
}

func (e EnvEntry) And(es []*EnvEntry) []*EnvEntry {
	return append(es, &e)
}

func BuildEnv(envs ...*EnvEntry) map[string]interface{} {
	return MapFromNamedSlice(func(entry *EnvEntry) interface{} {
		return entry.value
	}, envs...)
}

func Compile(input string) (*vm.Program, error) {
	return expr.Compile(input)
}

func InvertAndCompile(input string) (*vm.Program, error) {
	tree, err := parser.Parse(input)
	if err != nil {
		return nil, err
	}
	inverse, err := findInverse(&tree.Node)
	if err != nil {
		return nil, err
	}
	return expr.Compile(nodeToString(inverse))
}

func nodeToString(node *ast.Node) string {
	switch n := (*node).(type) {
	case *ast.IdentifierNode:
		return n.Value
	case *ast.BinaryNode:
		return fmt.Sprintf("%s%s%s",
			encloseInParenthesis(&n.Left),
			n.Operator,
			encloseInParenthesis(&n.Right),
		)
	case *ast.FloatNode:
		return fmt.Sprintf("%v", n.Value)
	case *ast.IntegerNode:
		return fmt.Sprintf("%v", n.Value)
	default:
		return ""
	}
}

func encloseInParenthesis(node *ast.Node) string {
	switch (*node).(type) {
	case *ast.BinaryNode:
		return fmt.Sprintf("(%s)", nodeToString(node))
	default:
		return nodeToString(node)
	}
}

func findInverse(node *ast.Node) (*ast.Node, error) {
	inverse := &ast.IdentifierNode{}
	v := inverterVisitor{
		inverse:         inverse,
		identifierValue: &inverse.Value,
	}
	ast.Walk(node, &v)
	if v.err != nil {
		return nil, v.err
	}
	return &v.inverse, nil
}

type inverterVisitor struct {
	inverse         ast.Node
	identifierValue *string
	err             error
}

func (v *inverterVisitor) Enter(node *ast.Node) {
	switch n := (*node).(type) {
	case *ast.BinaryNode:
		v.invertBinaryNode(n)
	case *ast.IdentifierNode:
		if len(*v.identifierValue) > 0 {
			v.err = errors.New("found more than one identifier node")
		} else {
			*(v.identifierValue) = n.Value
		}
	default:
		if v.err != nil { // keep first error while walking
			v.err = fmt.Errorf("cannot find inverse of node %s", ast.Dump(*node))
		}
	}
}

func (v *inverterVisitor) Exit(*ast.Node) {

}

func (v *inverterVisitor) invertBinaryNode(node *ast.BinaryNode) {
	type numericNodeFunc func(numericNode ast.Node) *ast.BinaryNode
	withNumericNode := func(onLeft numericNodeFunc, onRight numericNodeFunc) {
		if isNumericNode(node.Left) {
			v.inverse = onLeft(node.Left)
		} else if isNumericNode(node.Right) { // x + 5
			v.inverse = onRight(node.Right)
		} else {
			v.err = fmt.Errorf("either left or right must be numeric in %s", ast.Dump(node))
		}
	}
	withAnyNumericNode := func(onAny numericNodeFunc) {
		withNumericNode(onAny, onAny)
	}

	switch node.Operator {
	case "+":
		withAnyNumericNode(func(numericNode ast.Node) *ast.BinaryNode {
			return &ast.BinaryNode{
				Operator: "-",
				Left:     v.inverse,
				Right:    numericNode,
			}
		})
	case "*":
		withAnyNumericNode(func(numericNode ast.Node) *ast.BinaryNode {
			return &ast.BinaryNode{
				Operator: "/",
				Left:     v.inverse,
				Right:    numericNode,
			}
		})
	case "-":
		withNumericNode(
			func(numericNode ast.Node) *ast.BinaryNode {
				return &ast.BinaryNode{
					Operator: "-",
					Left:     node.Left,
					Right:    v.inverse,
				}
			},
			func(numericNode ast.Node) *ast.BinaryNode {
				return &ast.BinaryNode{
					Operator: "+",
					Left:     v.inverse,
					Right:    numericNode,
				}
			},
		)
	case "/":
		withNumericNode(
			func(numericNode ast.Node) *ast.BinaryNode {
				return &ast.BinaryNode{
					Operator: "/",
					Left:     node.Left,
					Right:    v.inverse,
				}
			},
			func(numericNode ast.Node) *ast.BinaryNode {
				return &ast.BinaryNode{
					Operator: "*",
					Left:     v.inverse,
					Right:    numericNode,
				}
			},
		)
	default:
		v.err = fmt.Errorf("cannot find inverse of operator node %s", node.Operator)
	}
}

func isNumericNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.IntegerNode:
		return true
	case *ast.FloatNode:
		return true
	default:
		return false
	}
}
