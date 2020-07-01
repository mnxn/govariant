package main

import (
	"go/ast"
	"strings"
)

type variant struct {
	Name          string             // variant name declared in comment
	Package       string             // package of original variant declaration
	Imports       []string           // imports neccessary for types
	Constructors  []constructor      // map constructor names to types
	Methods       *ast.InterfaceType // underlying interface type, used for generation
	ExplicitCheck bool               // whether to generate explicit interface implementation checks
	Unpack        bool               // whether to generate unpack methods
	Visitor       bool               // whether to generate visitor struct and methods
}

type constructor struct {
	Name       string
	Parameters []parameter
}

type parameter struct {
	Name string
	Type ast.Expr
}

func (c constructor) Type() ast.Expr {
	if len(c.Parameters) == 0 {
		return &ast.StructType{Fields: &ast.FieldList{}} // empty struct{}
	} else if len(c.Parameters) == 1 && c.Parameters[0].Name == "" {
		return c.Parameters[0].Type
	}
	structType := &ast.StructType{Fields: &ast.FieldList{
		List: []*ast.Field{},
	}}
	for _, param := range c.Parameters {
		structType.Fields.List = append(
			structType.Fields.List,
			&ast.Field{
				Names: []*ast.Ident{{Name: param.Name}},
				Type:  param.Type,
			},
		)
	}
	return structType
}

func (c constructor) UnpackTypes(optionalParen bool) (string, error) {
	if len(c.Parameters) == 0 {
		if optionalParen {
			return "", nil
		}
		return "()", nil
	}

	var paramTypes []string
	for _, param := range c.Parameters {
		paramType, err := gofmt(param.Type)
		if err != nil {
			return "", err
		}
		paramTypes = append(paramTypes, paramType)
	}

	if optionalParen && len(c.Parameters) == 1 {
		return paramTypes[0] + "", nil
	}
	return "(" + strings.Join(paramTypes, ", ") + ") ", nil
}

func (c constructor) UnpackValues() (string, error) {
	if len(c.Parameters) == 0 {
		return "", nil
	} else if len(c.Parameters) == 1 {
		if c.Parameters[0].Name == "" {
			typeString, err := gofmt(c.Parameters[0].Type)
			if err != nil {
				return "", err
			}
			return "(" + typeString + ")(rcv)", nil
		}
		return "rcv." + c.Parameters[0].Name, nil
	}

	var values []string
	for _, param := range c.Parameters {
		values = append(values, "rcv."+param.Name)
	}
	return strings.Join(values, ", "), nil
}
