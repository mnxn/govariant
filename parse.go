package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// variantsFromFile returns all the variants from a file
func variantsFromFile(path string) (variants []variant, err error) {
	// use the official go parser
	fset := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// keep track of variant names so there are no duplicates in one file
	variantNames := map[string]struct{}{}

	// loop over all declarations in parsed file
	for _, decl := range parsedFile.Decls {
		// skip non-generic declarations or those with zero comments
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Doc == nil {
			continue
		}
		// split last comment by whitespace
		lastComment := genDecl.Doc.List[len(genDecl.Doc.List)-1]
		fields := strings.Fields(lastComment.Text)
		// skip declarations without @variant annotation
		if len(fields) >= 2 && fields[1] != "@variant" {
			continue
		}
		// must have variant name
		if len(fields) < 3 {
			return nil, parseError("missing variant name", lastComment.Pos(), fset)
		}
		// must be a valid go identifier
		parsedName, err := parser.ParseExpr(fields[2])
		nameIdent, ok := parsedName.(*ast.Ident)
		if err != nil || !ok {
			return nil, parseError("variant name is not a valid go identifier", lastComment.Pos(), fset)
		}
		variantName := nameIdent.Name
		// find all imports from @import annotations
		var imports []string
		for _, comment := range genDecl.Doc.List {
			commentFields := strings.Fields(comment.Text)
			// must have package name in import
			if len(commentFields) == 2 && commentFields[1] == "@import" {
				return nil, parseError("missing package name in @import", comment.Pos(), fset)
			}
			// package name must be a valid go string
			if len(commentFields) >= 2 && commentFields[1] == "@import" {
				parsedPackageName, err := parser.ParseExpr(commentFields[2])
				if err != nil {
					return nil, parseError("@import package name failed to parse as string", comment.Pos(), fset)
				}
				packageNameLit, ok := parsedPackageName.(*ast.BasicLit)
				if !ok || packageNameLit.Kind != token.STRING {
					return nil, parseError("@import package name failed to parse as string", comment.Pos(), fset)
				}
				imports = append(imports, packageNameLit.Value)
			}
		}
		// must be a single declaration
		if !(genDecl.Lparen == 0 && genDecl.Rparen == 0) {
			return nil, parseError("type declaration must not be surrounded by parentheses", genDecl.Pos(), fset)
		}
		// generic declaration must be a type declaration
		typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
		if !ok {
			return nil, parseError("variant must be a defined with a type declaration", genDecl.Pos(), fset)
		}
		// type must be named "_"
		if typeSpec.Name.Name != "_" {
			return nil, parseError("type declaration must be named _", typeSpec.Pos(), fset)
		}
		// type must be defined as an interface
		interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
		if !ok {
			return nil, parseError("type must be defined as an interface", typeSpec.Pos(), fset)
		}

		// make sure there isn't a variant already defined in the same file
		if _, ok := variantNames[variantName]; ok {
			return nil, parseError("cannot have duplicate variant names in file ("+variantName+")", typeSpec.Pos(), fset)
		}

		// transform interface to variant
		variantResult, err := variantFromInterface(interfaceType, variantName, parsedFile.Name.Name, imports, fset)
		if err != nil {
			return nil, err
		}
		variantNames[variantName] = struct{}{}

		variants = append(variants, variantResult)
	}
	return
}

// variantFromInterface makes a variant given an interface declaration
func variantFromInterface(
	interfaceType *ast.InterfaceType,
	variantName string,
	packageName string,
	imports []string,
	fset *token.FileSet,
) (variant, error) {
	result := variant{
		Name:          variantName,
		Package:       packageName,
		Imports:       imports,
		Constructors:  []constructor{},
		Methods:       interfaceType,
		ExplicitCheck: !*noCheck,
		Unpack:        !*noUnpack,
		Visitor:       !*noVisitor,
	}
	// deleting in loop body, so iterate backwards
	for i := interfaceType.Methods.NumFields() - 1; i >= 0; i-- {
		field := interfaceType.Methods.List[i]
		// filter out constructors
		isConstructor, err := isConstructor(field, fset)
		if err != nil {
			return variant{}, err
		}
		if isConstructor {
			// delete constructors from interface method list
			interfaceType.Methods.List = append(
				interfaceType.Methods.List[:i],
				interfaceType.Methods.List[i+1:]...)
			// add named constructor to variant constructors
			result.Constructors = append(result.Constructors, constructor{
				Name:       field.Names[0].Name,
				Parameters: parametersFromFuncType(field.Type.(*ast.FuncType)),
			})
		}
		field.Comment, field.Doc = nil, nil
	}
	// must have at least one constructor
	if len(result.Constructors) == 0 {
		return variant{}, parseError("must have at least one constructor", interfaceType.Pos(), fset)
	}
	// reverse constructor list to match actual order
	for i, j := 0, len(result.Constructors)-1; i < j; i, j = i+1, j-1 {
		result.Constructors[i], result.Constructors[j] =
			result.Constructors[j], result.Constructors[i]
	}
	// add visit method to interface if novisitor flag is false
	if !*noVisitor {
		appendMethod(&interfaceType.Methods.List, "Visit",
			[]*ast.Field{{
				Type: &ast.Ident{Name: variantName + "Visitor"},
			}}, nil)
	}
	// append method to act as a tag for variant cases
	appendMethod(&interfaceType.Methods.List, "is"+strings.Title(variantName), nil, nil)

	return result, nil
}

// isConstructor checks whether an interface method can be used as a constructor
func isConstructor(field *ast.Field, fset *token.FileSet) (bool, error) {
	// check for methods which are annotated with @method
	if field.Comment != nil && field.Comment.List != nil {
		commentFields := strings.Fields(field.Comment.List[0].Text)
		if len(commentFields) >= 2 && commentFields[1] == "@method" {
			if _, ok := field.Type.(*ast.FuncType); ok {
				return false, nil
			}
			return false, parseError("cannot annotate interface with @method", field.Comment.List[0].Pos(), fset)
		}
	}

	funcType, ok := field.Type.(*ast.FuncType)
	if !ok {
		return false, nil
	}

	// must be void function
	if funcType.Results != nil {
		return false, parseError("cannot have a return type on a constructor", funcType.Results.Pos(), fset)
	}

	for _, param := range funcType.Params.List {
		// no variadic parameters in constructor
		if _, ok := param.Type.(*ast.Ellipsis); ok {
			return false, parseError("cannot have variadic constructor", param.Pos(), fset)
		}
		if len(param.Names) == 0 {
			// cannot have unnamed interface types
			if _, ok := param.Type.(*ast.InterfaceType); ok {
				return false, parseError("cannot have an unnamed interface type in constructor", param.Pos(), fset)
			}
			// cannot have unnamed pointer types
			if _, ok := param.Type.(*ast.StarExpr); ok {
				return false, parseError("cannot have an unnamed pointer type in constructor", param.Pos(), fset)
			}
		}
		if funcType.Params.NumFields() > 1 {
			// constructor may only have a single unnamed parameter
			if len(param.Names) == 0 {
				return false, parseError("cannot have multiple unnamed parameters in a constructor", param.Pos(), fset)
			}
			// if there is more than one parameter to the constructor, they must be named
			if len(param.Names) > 1 {
				return false, parseError("cannot use a name list as parameters in a constructor", param.Pos(), fset)
			}
		}
	}
	return true, nil
}

// parametersFromFuncType extracts the parameters from a function type
func parametersFromFuncType(funcType *ast.FuncType) (result []parameter) {
	for _, param := range funcType.Params.List {
		if len(param.Names) == 0 {
			return []parameter{{Name: "", Type: param.Type}}
		}
		result = append(result, parameter{Name: param.Names[0].Name, Type: param.Type})
	}
	return
}

// appendMethod appends a method to a field list given the method's parameters and results
func appendMethod(list *[]*ast.Field, name string, params []*ast.Field, results []*ast.Field) {
	*list = append(*list, &ast.Field{
		Names: []*ast.Ident{{
			Name: name,
		}},
		Type: &ast.FuncType{
			Func:    token.NoPos,
			Params:  &ast.FieldList{List: params},
			Results: &ast.FieldList{List: results},
		},
	},
	)
}

func parseError(message string, position token.Pos, fset *token.FileSet) error {
	return fmt.Errorf(
		"%s: govariant: "+message,
		fset.Position(position))
}
