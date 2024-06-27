package generic

import (
	"go/ast"
	"go/token"
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
		{
			name: "infer type of empty slice literal",
			expr: &ast.CompositeLit{
				Type: &ast.ArrayType{Elt: &ast.Ident{Name: "int"}},
			},
			env: TypeEnv{
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &SliceType{ElementType: &TypeConstant{Name: "int"}},
			wantErr:  nil,
		},
		{
            name: "Infer type of slice of generic type",
            expr: &ast.CompositeLit{
                Type: &ast.ArrayType{
                    Elt: &ast.IndexExpr{
                        X:     &ast.Ident{Name: "Vector"},
                        Index: &ast.Ident{Name: "int"},
                    },
                },
            },
            env: TypeEnv{
                "Vector": &GenericType{
                    Name:       "Vector",
                    TypeParams: []Type{&TypeVariable{Name: "T"}},
                },
                "int": &TypeConstant{Name: "int"},
            },
            wantType: &SliceType{
                ElementType: &GenericType{
                    Name:       "Vector",
                    TypeParams: []Type{&TypeConstant{Name: "int"}},
                },
            },
            wantErr: nil,
        },
		{
            name: "Infer type of non-empty slice literal",
            expr: &ast.CompositeLit{
                Type: &ast.ArrayType{Elt: &ast.Ident{Name: "int"}},
                Elts: []ast.Expr{
                    &ast.BasicLit{Kind: token.INT, Value: "1"},
                    &ast.BasicLit{Kind: token.INT, Value: "2"},
                    &ast.BasicLit{Kind: token.INT, Value: "3"},
                },
            },
            env: TypeEnv{
                "int": &TypeConstant{Name: "int"},
            },
            wantType: &SliceType{ElementType: &TypeConstant{Name: "int"}},
            wantErr:  nil,
        },
		{
            name: "Infer type of int literal",
            expr: &ast.BasicLit{Kind: token.INT, Value: "42"},
            env:  TypeEnv{},
            wantType: &TypeConstant{Name: "int"},
            wantErr:  nil,
        },
		{
            name: "Infer type of float literal",
            expr: &ast.BasicLit{Kind: token.FLOAT, Value: "3.14"},
            env:  TypeEnv{},
            wantType: &TypeConstant{Name: "float64"},
            wantErr:  nil,
        },
		{
            name: "Infer type of string literal",
            expr: &ast.BasicLit{Kind: token.STRING, Value: `"hello"`},
            env:  TypeEnv{},
            wantType: &TypeConstant{Name: "string"},
            wantErr:  nil,
        },
		{
            name: "Infer type of rune literal",
            expr: &ast.BasicLit{Kind: token.CHAR, Value: "'A'"},
            env:  TypeEnv{},
            wantType: &TypeConstant{Name: "rune"},
            wantErr:  nil,
        },
		{
            name: "Infer type of slice with different basic literals",
            expr: &ast.CompositeLit{
                Type: &ast.ArrayType{Elt: &ast.Ident{Name: "interface{}"}},
                Elts: []ast.Expr{
                    &ast.BasicLit{Kind: token.INT, Value: "1"},
                    &ast.BasicLit{Kind: token.FLOAT, Value: "2.5"},
                    &ast.BasicLit{Kind: token.STRING, Value: `"three"`},
                },
            },
            env: TypeEnv{
                "interface{}": &InterfaceType{Name: "interface{}"},
            },
            wantType: &SliceType{ElementType: &InterfaceType{Name: "interface{}"}},
            wantErr:  nil,
        },
		{
            name: "Infer type of empty map literal",
            expr: &ast.CompositeLit{
                Type: &ast.MapType{
                    Key:   &ast.Ident{Name: "string"},
                    Value: &ast.Ident{Name: "int"},
                },
            },
            env: TypeEnv{
                "string": &TypeConstant{Name: "string"},
                "int":    &TypeConstant{Name: "int"},
            },
            wantType: &MapType{
                KeyType:   &TypeConstant{Name: "string"},
                ValueType: &TypeConstant{Name: "int"},
            },
            wantErr: nil,
        },
		{
            name: "Infer type of non-empty map literal",
            expr: &ast.CompositeLit{
                Type: &ast.MapType{
                    Key:   &ast.Ident{Name: "string"},
                    Value: &ast.Ident{Name: "int"},
                },
                Elts: []ast.Expr{
                    &ast.KeyValueExpr{
                        Key:   &ast.BasicLit{Kind: token.STRING, Value: `"one"`},
                        Value: &ast.BasicLit{Kind: token.INT, Value: "1"},
                    },
                    &ast.KeyValueExpr{
                        Key:   &ast.BasicLit{Kind: token.STRING, Value: `"two"`},
                        Value: &ast.BasicLit{Kind: token.INT, Value: "2"},
                    },
                },
            },
            env: TypeEnv{
                "string": &TypeConstant{Name: "string"},
                "int":    &TypeConstant{Name: "int"},
            },
            wantType: &MapType{
                KeyType:   &TypeConstant{Name: "string"},
                ValueType: &TypeConstant{Name: "int"},
            },
            wantErr: nil,
        },
		{
            name: "Infer type of generic function call",
            expr: &ast.CallExpr{
                Fun: &ast.IndexExpr{
                    X:     &ast.Ident{Name: "GenericFunc"},
                    Index: &ast.Ident{Name: "int"},
                },
                Args: []ast.Expr{
                    &ast.BasicLit{Kind: token.INT, Value: "42"},
                },
            },
            env: TypeEnv{
                "GenericFunc": &GenericType{
                    Name: "GenericFunc",
                    TypeParams: []Type{&TypeVariable{Name: "T"}},
                    Constraints: map[string]TypeConstraint{
                        "T": {Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "float64"}}},
                    },
                },
                "int": &TypeConstant{Name: "int"},
            },
            wantType: &TypeConstant{Name: "int"},
            wantErr:  nil,
        },
		{
            name: "Infer type of generic struct instantiation",
            expr: &ast.CompositeLit{
                Type: &ast.IndexExpr{
                    X:     &ast.Ident{Name: "GenericPair"},
                    Index: &ast.Ident{Name: "string"},
                },
                Elts: []ast.Expr{
                    &ast.KeyValueExpr{
                        Key:   &ast.Ident{Name: "First"},
                        Value: &ast.BasicLit{Kind: token.STRING, Value: `"hello"`},
                    },
                    &ast.KeyValueExpr{
                        Key:   &ast.Ident{Name: "Second"},
                        Value: &ast.BasicLit{Kind: token.STRING, Value: `"world"`},
                    },
                },
            },
            env: TypeEnv{
                "GenericPair": &GenericType{
                    Name: "GenericPair",
                    TypeParams: []Type{&TypeVariable{Name: "T"}},
                },
                "string": &TypeConstant{Name: "string"},
            },
            wantType: &GenericType{
                Name: "GenericPair",
                TypeParams: []Type{&TypeConstant{Name: "string"}},
            },
            wantErr: nil,
        },
        {
            name: "Infer type of nested generic types",
            expr: &ast.IndexExpr{
                X: &ast.Ident{Name: "Container"},
                Index: &ast.IndexExpr{
                    X:     &ast.Ident{Name: "Pair"},
                    Index: &ast.Ident{Name: "int"},
                },
            },
            env: TypeEnv{
                "Container": &GenericType{
                    Name: "Container",
                    TypeParams: []Type{&TypeVariable{Name: "T"}},
                },
                "Pair": &GenericType{
                    Name: "Pair",
                    TypeParams: []Type{&TypeVariable{Name: "U"}},
                },
                "int": &TypeConstant{Name: "int"},
            },
            wantType: &GenericType{
                Name: "Container",
                TypeParams: []Type{
                    &GenericType{
                        Name:       "Pair",
                        TypeParams: []Type{&TypeConstant{Name: "int"}},
                    },
                },
            },
            wantErr: nil,
        },
        {
            name: "Infer type with type constraint violation",
            expr: &ast.IndexExpr{
                X:     &ast.Ident{Name: "NumericContainer"},
                Index: &ast.Ident{Name: "string"},
            },
            env: TypeEnv{
                "NumericContainer": &GenericType{
                    Name: "NumericContainer",
                    TypeParams: []Type{&TypeVariable{Name: "T"}},
                    Constraints: map[string]TypeConstraint{
                        "T": {Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "float64"}}},
                    },
                },
                "string": &TypeConstant{Name: "string"},
            },
            wantType: nil,
            wantErr:  ErrConstraintNotSatisfied,
        },
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
