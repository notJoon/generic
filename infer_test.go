package generic

import (
	"go/ast"
	"testing"
)

func TestInferType(t *testing.T) {
	tests := []struct {
		name     string
		expr     ast.Expr
		env      TypeEnv
		wantType Type
		wantErr  error
	}{
		{
			name: "Infer type of identifier",
			expr: &ast.Ident{Name: "x"},
			env: TypeEnv{
				"x": &TypeConstant{Name: "int"},
			},
			wantType: &TypeConstant{Name: "int"},
			wantErr:  nil,
		},
		{
			name:     "Infer type of unknown identifier",
			expr:     &ast.Ident{Name: "y"},
			env:      TypeEnv{},
			wantType: nil,
			wantErr:  ErrUnknownIdent,
		},
		{
			name: "Infer type of function call",
			expr: &ast.CallExpr{
				Fun: &ast.Ident{Name: "f"},
				Args: []ast.Expr{
					&ast.Ident{Name: "x"},
				},
			},
			env: TypeEnv{
				"f": &FunctionType{
					ParamTypes: []Type{&TypeConstant{Name: "int"}},
					ReturnType: &TypeConstant{Name: "string"},
				},
				"x": &TypeConstant{Name: "int"},
			},
			wantType: &TypeConstant{Name: "string"},
			wantErr:  nil,
		},
		{
			name: "Infer type of function call with type mismatch",
			expr: &ast.CallExpr{
				Fun: &ast.Ident{Name: "f"},
				Args: []ast.Expr{
					&ast.Ident{Name: "x"},
				},
			},
			env: TypeEnv{
				"f": &FunctionType{
					ParamTypes: []Type{&TypeConstant{Name: "int"}},
					ReturnType: &TypeConstant{Name: "string"},
				},
				"x": &TypeConstant{Name: "string"},
			},
			wantType: nil,
			wantErr:  ErrTypeMismatch,
		},
		{
			name: "Infer type of non-function call",
			expr: &ast.CallExpr{
				Fun: &ast.Ident{Name: "x"},
				Args: []ast.Expr{
					&ast.Ident{Name: "y"},
				},
			},
			env: TypeEnv{
				"x": &TypeConstant{Name: "int"},
				"y": &TypeConstant{Name: "int"},
			},
			wantType: nil,
			wantErr:  ErrNotAFunction,
		},
		{
			name: "Infer type of non-generic type as generic",
			expr: &ast.IndexExpr{
				X:     &ast.Ident{Name: "int"},
				Index: &ast.Ident{Name: "float"},
			},
			env: TypeEnv{
				"int":   &TypeConstant{Name: "int"},
				"float": &TypeConstant{Name: "float"},
			},
			wantType: nil,
			wantErr:  ErrNotAGenericType,
		},
		{
			name: "Infer type of generic type with single parameter",
			expr: &ast.IndexExpr{
				X:     &ast.Ident{Name: "Vector"},
				Index: &ast.Ident{Name: "int"},
			},
			env: TypeEnv{
				"Vector": &GenericType{
					Name:       "Vector",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
				},
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &GenericType{
				Name:       "Vector",
				TypeParams: []Type{&TypeConstant{Name: "int"}},
			},
			wantErr: nil,
		},
		{
			name: "Infer type of generic type with multiple parameters",
			expr: &ast.IndexListExpr{
				X: &ast.Ident{Name: "Map"},
				Indices: []ast.Expr{
					&ast.Ident{Name: "string"},
					&ast.Ident{Name: "int"},
				},
			},
			env: TypeEnv{
				"Map": &GenericType{
					Name:       "Map",
					TypeParams: []Type{&TypeVariable{Name: "K"}, &TypeVariable{Name: "V"}},
				},
				"string": &TypeConstant{Name: "string"},
				"int":    &TypeConstant{Name: "int"},
			},
			wantType: &GenericType{
				Name:       "Map",
				TypeParams: []Type{&TypeConstant{Name: "string"}, &TypeConstant{Name: "int"}},
			},
			wantErr: nil,
		},
		{
			name: "Infer type of generic type with mismatched parameter count",
			expr: &ast.IndexListExpr{
				X: &ast.Ident{Name: "Pair"},
				Indices: []ast.Expr{
					&ast.Ident{Name: "int"},
					&ast.Ident{Name: "string"},
					&ast.Ident{Name: "bool"},
				},
			},
			env: TypeEnv{
				"Pair": &GenericType{
					Name:       "Pair",
					TypeParams: []Type{&TypeVariable{Name: "T1"}, &TypeVariable{Name: "T2"}},
				},
				"int":    &TypeConstant{Name: "int"},
				"string": &TypeConstant{Name: "string"},
				"bool":   &TypeConstant{Name: "bool"},
			},
			wantType: nil,
			wantErr:  ErrTypeParamsNotMatch,
		},
		{
			name: "Infer type of nested generic type",
			expr: &ast.IndexListExpr{
				X: &ast.Ident{Name: "Outer"},
				Indices: []ast.Expr{
					&ast.Ident{Name: "int"},
					&ast.IndexExpr{
						X:     &ast.Ident{Name: "Inner"},
						Index: &ast.Ident{Name: "string"},
					},
				},
			},
			env: TypeEnv{
				"Outer": &GenericType{
					Name:       "Outer",
					TypeParams: []Type{&TypeVariable{Name: "T"}, &TypeVariable{Name: "U"}},
				},
				"Inner": &GenericType{
					Name:       "Inner",
					TypeParams: []Type{&TypeVariable{Name: "V"}},
				},
				"int":    &TypeConstant{Name: "int"},
				"string": &TypeConstant{Name: "string"},
			},
			wantType: &GenericType{
				Name: "Outer",
				TypeParams: []Type{
					&TypeConstant{Name: "int"},
					&GenericType{
						Name:       "Inner",
						TypeParams: []Type{&TypeConstant{Name: "string"}},
					},
				},
			},
			wantErr: nil,
		},
		// {
		//     name: "Infer type of slice literal",
		//     expr: &ast.CompositeLit{
		//         Type: &ast.ArrayType{
		//             Elt: &ast.Ident{Name: "int"},
		//         },
		//     },
		//     env: TypeEnv{
		//         "int": &TypeConstant{Name: "int"},
		//     },
		//     wantType: &SliceType{
		//         ElementType: &TypeConstant{Name: "int"},
		//     },
		//     wantErr: nil,
		// },
		// {
		//     name: "Infer type of map literal",
		//     expr: &ast.CompositeLit{
		//         Type: &ast.MapType{
		//             Key:   &ast.Ident{Name: "string"},
		//             Value: &ast.Ident{Name: "int"},
		//         },
		//     },
		//     env: TypeEnv{
		//         "string": &TypeConstant{Name: "string"},
		//         "int":    &TypeConstant{Name: "int"},
		//     },
		//     wantType: &MapType{
		//         KeyType:   &TypeConstant{Name: "string"},
		//         ValueType: &TypeConstant{Name: "int"},
		//     },
		//     wantErr: nil,
		// },
		// {
		//     name: "Infer type of slice indexing",
		//     expr: &ast.IndexExpr{
		//         X:     &ast.Ident{Name: "s"},
		//         Index: &ast.Ident{Name: "i"},
		//     },
		//     env: TypeEnv{
		//         "s": &SliceType{
		//             ElementType: &TypeConstant{Name: "float64"},
		//         },
		//         "i": &TypeConstant{Name: "int"},
		//     },
		//     wantType: &TypeConstant{Name: "float64"},
		//     wantErr:  nil,
		// },
		// {
		//     name: "Infer type of map indexing",
		//     expr: &ast.IndexExpr{
		//         X:     &ast.Ident{Name: "m"},
		//         Index: &ast.Ident{Name: "k"},
		//     },
		//     env: TypeEnv{
		//         "m": &MapType{
		//             KeyType:   &TypeConstant{Name: "string"},
		//             ValueType: &TypeConstant{Name: "bool"},
		//         },
		//         "k": &TypeConstant{Name: "string"},
		//     },
		//     wantType: &TypeConstant{Name: "bool"},
		//     wantErr:  nil,
		// },
		// {
		//     name: "Infer type of slice with incorrect index type",
		//     expr: &ast.IndexExpr{
		//         X:     &ast.Ident{Name: "s"},
		//         Index: &ast.Ident{Name: "k"},
		//     },
		//     env: TypeEnv{
		//         "s": &SliceType{
		//             ElementType: &TypeConstant{Name: "int"},
		//         },
		//         "k": &TypeConstant{Name: "string"},
		//     },
		//     wantType: nil,
		//     wantErr:  ErrTypeMismatch,
		// },
		// {
		//     name: "Infer type of map with incorrect key type",
		//     expr: &ast.IndexExpr{
		//         X:     &ast.Ident{Name: "m"},
		//         Index: &ast.Ident{Name: "i"},
		//     },
		//     env: TypeEnv{
		//         "m": &MapType{
		//             KeyType:   &TypeConstant{Name: "string"},
		//             ValueType: &TypeConstant{Name: "int"},
		//         },
		//         "i": &TypeConstant{Name: "int"},
		//     },
		//     wantType: nil,
		//     wantErr:  ErrTypeMismatch,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, err := InferType(tt.expr, tt.env)
			if err != tt.wantErr {
				t.Errorf("InferType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !TypesEqual(gotType, tt.wantType) {
				t.Errorf("InferType() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}
