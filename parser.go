package generic

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// Since the language intended to introduce generic type has syntax identical to `Go`,
// for convenience we use `go/parser` to parse the source code, then create an AST with it.
// Also, use that AST for type inference.

// Parser parses the source code and returns the AST.
func Parser(src string) (*ast.File, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, err
	}
	return node, nil
}
