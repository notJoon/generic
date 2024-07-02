package generic

import (
	"go/ast"
	"testing"
)

func BenchmarkInferTypeSimple(b *testing.B) {
	env := TypeEnv{
		"x": &TypeConstant{Name: "int"},
	}
	expr := &ast.Ident{Name: "x"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = InferType(expr, env, nil)
	}
}

func BenchmarkInferTypeFunction(b *testing.B) {
	env := TypeEnv{
		"f": &FunctionType{
			ParamTypes: []Type{&TypeConstant{Name: "int"}},
			ReturnType: &TypeConstant{Name: "string"},
		},
		"x": &TypeConstant{Name: "int"},
	}
	expr := &ast.CallExpr{
		Fun:  &ast.Ident{Name: "f"},
		Args: []ast.Expr{&ast.Ident{Name: "x"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = InferType(expr, env, nil)
	}
}

func BenchmarkInferTypeGeneric(b *testing.B) {
	env := TypeEnv{
		"Vector": &GenericType{
			Name:       "Vector",
			TypeParams: []Type{&TypeVariable{Name: "T"}},
		},
		"int": &TypeConstant{Name: "int"},
	}
	expr := &ast.IndexExpr{
		X:     &ast.Ident{Name: "Vector"},
		Index: &ast.Ident{Name: "int"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = InferType(expr, env, nil)
	}
}

func BenchmarkInferTypeComplex(b *testing.B) {
	env := TypeEnv{
		"Map": &GenericType{
			Name: "Map",
			TypeParams: []Type{
				&TypeVariable{Name: "K"},
				&TypeVariable{Name: "V"},
			},
		},
		"Vector": &GenericType{
			Name:       "Vector",
			TypeParams: []Type{&TypeVariable{Name: "T"}},
		},
		"string": &TypeConstant{Name: "string"},
		"int":    &TypeConstant{Name: "int"},
	}
	expr := &ast.IndexListExpr{
		X: &ast.Ident{Name: "Map"},
		Indices: []ast.Expr{
			&ast.Ident{Name: "string"},
			&ast.IndexExpr{
				X:     &ast.Ident{Name: "Vector"},
				Index: &ast.Ident{Name: "int"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = InferType(expr, env, nil)
	}
}
