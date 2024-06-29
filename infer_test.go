package generic

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"
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
			wantErr:  fmt.Errorf("unknown identifier: y"),
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
			wantErr:  fmt.Errorf("expected 2 type parameters, got 3"),
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
			name:     "Infer type of int literal",
			expr:     &ast.BasicLit{Kind: token.INT, Value: "42"},
			env:      TypeEnv{},
			wantType: &TypeConstant{Name: "int"},
			wantErr:  nil,
		},
		{
			name:     "Infer type of float literal",
			expr:     &ast.BasicLit{Kind: token.FLOAT, Value: "3.14"},
			env:      TypeEnv{},
			wantType: &TypeConstant{Name: "float64"},
			wantErr:  nil,
		},
		{
			name:     "Infer type of string literal",
			expr:     &ast.BasicLit{Kind: token.STRING, Value: `"hello"`},
			env:      TypeEnv{},
			wantType: &TypeConstant{Name: "string"},
			wantErr:  nil,
		},
		{
			name:     "Infer type of rune literal",
			expr:     &ast.BasicLit{Kind: token.CHAR, Value: "'A'"},
			env:      TypeEnv{},
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
					Name:       "Container",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
				},
				"Pair": &GenericType{
					Name:       "Pair",
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
			name: "Infer type of pointer to basic type",
			expr: &ast.StarExpr{
				X: &ast.Ident{Name: "int"},
			},
			env: TypeEnv{
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &PointerType{Base: &TypeConstant{Name: "int"}},
			wantErr:  nil,
		},
		{
			name: "Infer type of pointer to struct",
			expr: &ast.StarExpr{
				X: &ast.Ident{Name: "MyStruct"},
			},
			env: TypeEnv{
				"MyStruct": &StructType{Name: "MyStruct"},
			},
			wantType: &PointerType{Base: &StructType{Name: "MyStruct"}},
			wantErr:  nil,
		},
		{
			name: "Infer type of pointer to generic type",
			expr: &ast.StarExpr{
				X: &ast.IndexExpr{
					X:     &ast.Ident{Name: "Vector"},
					Index: &ast.Ident{Name: "int"},
				},
			},
			env: TypeEnv{
				"Vector": &GenericType{
					Name:       "Vector",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
				},
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &PointerType{
				Base: &GenericType{
					Name:       "Vector",
					TypeParams: []Type{&TypeConstant{Name: "int"}},
				},
			},
			wantErr: nil,
		},
		{
			name: "Infer type of generic type with pointer parameter",
			expr: &ast.IndexExpr{
				X: &ast.Ident{Name: "Container"},
				Index: &ast.StarExpr{
					X: &ast.Ident{Name: "int"},
				},
			},
			env: TypeEnv{
				"Container": &GenericType{
					Name:       "Container",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
				},
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &GenericType{
				Name: "Container",
				TypeParams: []Type{
					&PointerType{Base: &TypeConstant{Name: "int"}},
				},
			},
			wantErr: nil,
		},
		{
			name: "Infer type of function with pointer parameter and return type",
			expr: &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.StarExpr{X: &ast.Ident{Name: "int"}}},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.StarExpr{X: &ast.Ident{Name: "string"}}},
						},
					},
				},
			},
			env: TypeEnv{
				"int":    &TypeConstant{Name: "int"},
				"string": &TypeConstant{Name: "string"},
			},
			wantType: &FunctionType{
				ParamTypes: []Type{&PointerType{Base: &TypeConstant{Name: "int"}}},
				ReturnType: &PointerType{Base: &TypeConstant{Name: "string"}},
			},
			wantErr: nil,
		},
		{
			name: "Infer type of generic function with pointer constraint",
			expr: &ast.IndexExpr{
				X:     &ast.Ident{Name: "GenericFunc"},
				Index: &ast.StarExpr{X: &ast.Ident{Name: "int"}},
			},
			env: TypeEnv{
				"GenericFunc": &GenericType{
					Name:       "GenericFunc",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
					Constraints: map[string]TypeConstraint{
						"T": {Types: []Type{
							&PointerType{Base: &TypeConstant{Name: "int"}},
							&PointerType{Base: &TypeConstant{Name: "float64"}},
						}},
					},
				},
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &GenericType{
				Name:       "GenericFunc",
				TypeParams: []Type{&PointerType{Base: &TypeConstant{Name: "int"}}},
			},
			wantErr: nil,
		},
		{
			name: "infer type of struct literal",
			expr: &ast.CompositeLit{
				Type: &ast.Ident{Name: "Person"},
				Elts: []ast.Expr{
					&ast.KeyValueExpr{
						Key: &ast.Ident{Name: "Name"},
						Value: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"Alice"`,
						},
					},
					&ast.KeyValueExpr{
						Key: &ast.Ident{Name: "Age"},
						Value: &ast.BasicLit{
							Kind:  token.INT,
							Value: "30",
						},
					},
				},
			},
			env: TypeEnv{
				"Person": &StructType{
					Name: "Person",
					Fields: map[string]Type{
						"Name": &TypeConstant{Name: "string"},
						"Age":  &TypeConstant{Name: "int"},
					},
				},
			},
			wantType: &StructType{
				Name: "Person",
				Fields: map[string]Type{
					"Name": &TypeConstant{Name: "string"},
					"Age":  &TypeConstant{Name: "int"},
				},
			},
		},
		{
			name: "infer type of array literal",
			expr: &ast.CompositeLit{
				Type: &ast.ArrayType{
					Len: &ast.BasicLit{Kind: token.INT, Value: "3"},
					Elt: &ast.Ident{Name: "int"},
				},
				Elts: []ast.Expr{
					&ast.BasicLit{Kind: token.INT, Value: "1"},
					&ast.BasicLit{Kind: token.INT, Value: "2"},
					&ast.BasicLit{Kind: token.INT, Value: "3"},
				},
			},
			env: TypeEnv{
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &ArrayType{
				Len:         3,
				ElementType: &TypeConstant{Name: "int"},
			},
			wantErr: nil,
		},
		{
			name: "Infer type of nested struct literal",
			expr: &ast.CompositeLit{
				Type: &ast.Ident{Name: "Employee"},
				Elts: []ast.Expr{
					&ast.KeyValueExpr{
						Key: &ast.Ident{Name: "Person"},
						Value: &ast.CompositeLit{
							Type: &ast.Ident{Name: "Person"},
							Elts: []ast.Expr{
								&ast.KeyValueExpr{
									Key:   &ast.Ident{Name: "Name"},
									Value: &ast.BasicLit{Kind: token.STRING, Value: `"John"`},
								},
								&ast.KeyValueExpr{
									Key:   &ast.Ident{Name: "Age"},
									Value: &ast.BasicLit{Kind: token.INT, Value: "30"},
								},
							},
						},
					},
					&ast.KeyValueExpr{
						Key:   &ast.Ident{Name: "Position"},
						Value: &ast.BasicLit{Kind: token.STRING, Value: `"Manager"`},
					},
				},
			},
			env: TypeEnv{
				"Employee": &StructType{
					Name: "Employee",
					Fields: map[string]Type{
						"Person": &StructType{
							Name: "Person",
							Fields: map[string]Type{
								"Name": &TypeConstant{Name: "string"},
								"Age":  &TypeConstant{Name: "int"},
							},
						},
						"Position": &TypeConstant{Name: "string"},
					},
				},
				"Person": &StructType{
					Name: "Person",
					Fields: map[string]Type{
						"Name": &TypeConstant{Name: "string"},
						"Age":  &TypeConstant{Name: "int"},
					},
				},
			},
			wantType: &StructType{
				Name: "Employee",
				Fields: map[string]Type{
					"Person": &StructType{
						Name: "Person",
						Fields: map[string]Type{
							"Name": &TypeConstant{Name: "string"},
							"Age":  &TypeConstant{Name: "int"},
						},
					},
					"Position": &TypeConstant{Name: "string"},
				},
			},
			wantErr: nil,
		},
		{
			name: "Infer type of generic type instantiation",
			expr: &ast.IndexExpr{
				X:     &ast.Ident{Name: "Vector"},
				Index: &ast.Ident{Name: "int"},
			},
			env: TypeEnv{
				"Vector": &GenericType{
					Name:       "Vector",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
					Fields: map[string]Type{
						"Data": &SliceType{ElementType: &TypeVariable{Name: "T"}},
					},
				},
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &GenericType{
				Name:       "Vector",
				TypeParams: []Type{&TypeConstant{Name: "int"}},
				Fields: map[string]Type{
					"Data": &SliceType{ElementType: &TypeConstant{Name: "int"}},
				},
			},
			wantErr: nil,
		},
		{
			name: "Infer type of generic type instantiation with constraint",
			expr: &ast.IndexExpr{
				X:     &ast.Ident{Name: "NumericVector"},
				Index: &ast.Ident{Name: "float64"},
			},
			env: TypeEnv{
				"NumericVector": &GenericType{
					Name:       "NumericVector",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
					Constraints: map[string]TypeConstraint{
						"T": {Types: []Type{
							&TypeConstant{Name: "int"},
							&TypeConstant{Name: "float32"},
							&TypeConstant{Name: "float64"},
						}},
					},
					Fields: map[string]Type{
						"Data": &SliceType{ElementType: &TypeVariable{Name: "T"}},
					},
				},
				"float64": &TypeConstant{Name: "float64"},
			},
			wantType: &GenericType{
				Name:       "NumericVector",
				TypeParams: []Type{&TypeConstant{Name: "float64"}},
				Fields: map[string]Type{
					"Data": &SliceType{ElementType: &TypeConstant{Name: "float64"}},
				},
			},
			wantErr: nil,
		},
		{
			name: "Infer type of generic type instantiation with unsatisfied constraint",
			expr: &ast.IndexExpr{
				X:     &ast.Ident{Name: "NumericVector"},
				Index: &ast.Ident{Name: "string"},
			},
			env: TypeEnv{
				"NumericVector": &GenericType{
					Name:       "NumericVector",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
					Constraints: map[string]TypeConstraint{
						"T": {Types: []Type{
							&TypeConstant{Name: "int"},
							&TypeConstant{Name: "float32"},
							&TypeConstant{Name: "float64"},
						}},
					},
				},
				"string": &TypeConstant{Name: "string"},
			},
			wantType: nil,
			wantErr:  fmt.Errorf("type argument TypeConst(string) does not satisfy constraint for T"),
		},
		{
			name: "Infer type of non-generic type as generic",
			expr: &ast.IndexExpr{
				X:     &ast.Ident{Name: "int"},
				Index: &ast.Ident{Name: "float64"},
			},
			env: TypeEnv{
				"int":     &TypeConstant{Name: "int"},
				"float64": &TypeConstant{Name: "float64"},
			},
			wantType: nil,
			wantErr:  ErrNotAGenericType,
		},
		{
			name: "Infer empty interface",
			expr: &ast.InterfaceType{Methods: &ast.FieldList{}},
			env:  TypeEnv{},
			wantType: &InterfaceType{
				Name:    "interface{}",
				Methods: MethodSet{},
				IsEmpty: true,
			},
			wantErr: nil,
		},
		{
			name: "Infer interface with methods",
			expr: &ast.InterfaceType{
				Methods: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{{Name: "Method1"}},
							Type: &ast.FuncType{
								Params:  &ast.FieldList{},
								Results: &ast.FieldList{},
							},
						},
					},
				},
			},
			env: TypeEnv{},
			wantType: &InterfaceType{
				Name: "",
				Methods: MethodSet{
					"Method1": Method{
						Name:      "Method1",
						Params:    []Type{},
						Results:   []Type{},
						IsPointer: false,
					},
				},
				IsEmpty: false,
			},
			wantErr: nil,
		},
		{
			name: "Infer interface with embedded interface",
			expr: &ast.InterfaceType{
				Methods: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: nil, // Embedded interface
							Type:  &ast.Ident{Name: "EmbeddedInterface"},
						},
						{
							Names: []*ast.Ident{{Name: "Method1"}},
							Type: &ast.FuncType{
								Params:  &ast.FieldList{},
								Results: &ast.FieldList{},
							},
						},
					},
				},
			},
			env: TypeEnv{
				"EmbeddedInterface": &InterfaceType{
					Name: "EmbeddedInterface",
					Methods: MethodSet{
						"EmbeddedMethod": Method{
							Name:      "EmbeddedMethod",
							Params:    []Type{},
							Results:   []Type{},
							IsPointer: false,
						},
					},
				},
			},
			wantType: &InterfaceType{
				Name: "",
				Methods: MethodSet{
					"Method1": Method{
						Name:      "Method1",
						Params:    []Type{},
						Results:   []Type{},
						IsPointer: false,
					},
				},
				Embedded: []Type{
					&InterfaceType{
						Name: "EmbeddedInterface",
						Methods: MethodSet{
							"EmbeddedMethod": Method{
								Name:      "EmbeddedMethod",
								Params:    []Type{},
								Results:   []Type{},
								IsPointer: false,
							},
						},
					},
				},
				IsEmpty: false,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, err := InferType(tt.expr, tt.env)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("InferType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("InferType() error diff:\n%s", diffStrings(err.Error(), tt.wantErr.Error()))
				return
			}
			if !TypesEqual(gotType, tt.wantType) {
				t.Errorf("InferType() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestInferTypeWithMultipleTypeParams(t *testing.T) {
	env := TypeEnv{
		"Pair": &GenericType{
			Name: "Pair",
			TypeParams: []Type{
				&TypeVariable{Name: "T"},
				&TypeVariable{Name: "U"},
			},
			Fields: map[string]Type{
				"First":  &TypeVariable{Name: "T"},
				"Second": &TypeVariable{Name: "U"},
			},
		},
		"int":    &TypeConstant{Name: "int"},
		"string": &TypeConstant{Name: "string"},
	}

	expr := &ast.IndexListExpr{
		X: &ast.Ident{Name: "Pair"},
		Indices: []ast.Expr{
			&ast.Ident{Name: "int"},
			&ast.Ident{Name: "string"},
		},
	}

	result, err := InferType(expr, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := &GenericType{
		Name: "Pair",
		TypeParams: []Type{
			&TypeConstant{Name: "int"},
			&TypeConstant{Name: "string"},
		},
		Fields: map[string]Type{
			"First":  &TypeConstant{Name: "int"},
			"Second": &TypeConstant{Name: "string"},
		},
	}

	if !TypesEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestCalculateStructMethodSet(t *testing.T) {
	tests := []struct {
		name     string
		s        *StructType
		isPtr    bool
		expected MethodSet
	}{
		{
			name: "Simple struct with no methods",
			s: &StructType{
				Name:    "SimpleStruct",
				Methods: MethodSet{},
				Fields:  map[string]Type{},
			},
			isPtr:    false,
			expected: MethodSet{},
		},
		{
			name: "Struct with value receiver methods",
			s: &StructType{
				Name: "ValueMethodStruct",
				Methods: MethodSet{
					"Method1": Method{Name: "Method1", IsPointer: false},
					"Method2": Method{Name: "Method2", IsPointer: false},
				},
				Fields: map[string]Type{},
			},
			isPtr: false,
			expected: MethodSet{
				"Method1": Method{Name: "Method1", IsPointer: false},
				"Method2": Method{Name: "Method2", IsPointer: false},
			},
		},
		{
			name: "Struct with pointer receiver methods",
			s: &StructType{
				Name: "PtrMethodStruct",
				Methods: MethodSet{
					"Method1": Method{Name: "Method1", IsPointer: true},
					"Method2": Method{Name: "Method2", IsPointer: true},
				},
				Fields: map[string]Type{},
			},
			isPtr:    false,
			expected: MethodSet{},
		},
		{
			name: "Struct with mixed receiver methods, accessed as pointer",
			s: &StructType{
				Name: "MixedMethodStruct",
				Methods: MethodSet{
					"Method1": Method{Name: "Method1", IsPointer: false},
					"Method2": Method{Name: "Method2", IsPointer: true},
				},
				Fields: map[string]Type{},
			},
			isPtr: true,
			expected: MethodSet{
				"Method1": Method{Name: "Method1", IsPointer: false},
				"Method2": Method{Name: "Method2", IsPointer: true},
			},
		},
		{
			name: "Struct with embedded struct",
			s: &StructType{
				Name: "OuterStruct",
				Methods: MethodSet{
					"OuterMethod": Method{Name: "OuterMethod", IsPointer: false},
				},
				Fields: map[string]Type{
					"Inner": &StructType{
						Name: "InnerStruct",
						Methods: MethodSet{
							"InnerMethod": Method{Name: "InnerMethod", IsPointer: false},
						},
						Fields: map[string]Type{},
					},
				},
			},
			isPtr: false,
			expected: MethodSet{
				"OuterMethod": Method{Name: "OuterMethod", IsPointer: false},
				"InnerMethod": Method{Name: "InnerMethod", IsPointer: false},
			},
		},
		{
			name: "Struct with method name collision in embedded struct",
			s: &StructType{
				Name: "OuterStruct",
				Methods: MethodSet{
					"Method": Method{Name: "Method", IsPointer: false},
				},
				Fields: map[string]Type{
					"Inner": &StructType{
						Name: "InnerStruct",
						Methods: MethodSet{
							"Method": Method{Name: "Method", IsPointer: false},
						},
						Fields: map[string]Type{},
					},
				},
			},
			isPtr: false,
			expected: MethodSet{
				"Method": Method{Name: "Method", IsPointer: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateStructMethodSet(tt.s, tt.isPtr)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("calculateStructMethodSet() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSubstituteTypeVar(t *testing.T) {
	tests := []struct {
		name        string
		t           Type
		tv          *TypeVariable
		replacement Type
		expected    Type
	}{
		{
			name:        "replace type variable",
			t:           &TypeVariable{Name: "T"},
			tv:          &TypeVariable{Name: "T"},
			replacement: &TypeConstant{Name: "int"},
			expected:    &TypeConstant{Name: "int"},
		},
		{
			name:        "do not replace different type variable",
			t:           &TypeVariable{Name: "T"},
			tv:          &TypeVariable{Name: "U"},
			replacement: &TypeConstant{Name: "int"},
			expected:    &TypeVariable{Name: "T"},
		},
		{
			name: "replace in generic type",
			t: &GenericType{
				Name:       "Vector",
				TypeParams: []Type{&TypeVariable{Name: "T"}},
			},
			tv:          &TypeVariable{Name: "T"},
			replacement: &TypeConstant{Name: "int"},
			expected: &GenericType{
				Name:       "Vector",
				TypeParams: []Type{&TypeConstant{Name: "int"}},
			},
		},
		{
			name: "replace in nested genetic type",
			t: &GenericType{
				Name: "Outer",
				TypeParams: []Type{
					&GenericType{
						Name:       "Inner",
						TypeParams: []Type{&TypeVariable{Name: "T"}},
					},
				},
			},
			tv:          &TypeVariable{Name: "T"},
			replacement: &TypeConstant{Name: "int"},
			expected: &GenericType{
				Name: "Outer",
				TypeParams: []Type{
					&GenericType{
						Name:       "Inner",
						TypeParams: []Type{&TypeConstant{Name: "int"}},
					},
				},
			},
		},
		{
			name:        "Do not replace in non-variable, non-generic type",
			t:           &TypeConstant{Name: "string"},
			tv:          &TypeVariable{Name: "T"},
			replacement: &TypeConstant{Name: "int"},
			expected:    &TypeConstant{Name: "string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteTypeVar(tt.t, tt.tv, tt.replacement)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("substituteTypeVar() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateMethodSet(t *testing.T) {
	tests := []struct {
		name     string
		t        Type
		expected MethodSet
	}{
		{
			name: "struct type",
			t: &StructType{
				Name: "MyStruct",
				Methods: MethodSet{
					"Method1": Method{Name: "Method1", IsPointer: false},
					"Method2": Method{Name: "Method2", IsPointer: true},
				},
			},
			expected: MethodSet{
				"Method1": Method{Name: "Method1", IsPointer: false},
			},
		},
		{
			name: "interface type",
			t: &InterfaceType{
				Name: "MyInterface",
				Methods: MethodSet{
					"Method1": Method{Name: "Method1"},
					"Method2": Method{Name: "Method2"},
				},
			},
			expected: MethodSet{
				"Method1": Method{Name: "Method1"},
				"Method2": Method{Name: "Method2"},
			},
		},
		{
			name: "pointer to struct type",
			t: &PointerType{
				Base: &StructType{
					Name: "MyStruct",
					Methods: MethodSet{
						"Method1": Method{Name: "Method1", IsPointer: false},
						"Method2": Method{Name: "Method2", IsPointer: true},
					},
				},
			},
			expected: MethodSet{
				"Method1": Method{Name: "Method1", IsPointer: false},
				"Method2": Method{Name: "Method2", IsPointer: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateMethodSet(tt.t)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("CalculateMethodSet() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func diffStrings(a, b string) string {
	var diff strings.Builder
	aLines := strings.Split(a, "\n")
	bLines := strings.Split(b, "\n")

	maxLines := len(aLines)
	if len(bLines) > maxLines {
		maxLines = len(bLines)
	}

	for i := 0; i < maxLines; i++ {
		var aLine, bLine string
		if i < len(aLines) {
			aLine = aLines[i]
		}
		if i < len(bLines) {
			bLine = bLines[i]
		}
		if aLine != bLine {
			diff.WriteString(fmt.Sprintf("- %s\n+ %s\n", aLine, bLine))
		}
	}
	return diff.String()
}

func TestInferTypeSpec(t *testing.T) {
	tests := []struct {
		name     string
		spec     *ast.TypeSpec
		env      TypeEnv
		wantType Type
		wantErr  error
	}{
		{
			name: "Simple type alias",
			spec: &ast.TypeSpec{
				Name:   &ast.Ident{Name: "MyInt"},
				Assign: token.Pos(1), // 1 is a valid non-zero position
				Type:   &ast.Ident{Name: "int"},
			},
			env: TypeEnv{"int": &TypeConstant{Name: "int"}},
			wantType: &TypeAlias{
				Name:      "MyInt",
				AliasedTo: &TypeConstant{Name: "int"},
			},
			wantErr: nil,
		},
		{
			name: "Type alias to a custom type",
			spec: &ast.TypeSpec{
				Name:   &ast.Ident{Name: "MyCustomType"},
				Assign: token.Pos(1),
				Type:   &ast.Ident{Name: "CustomType"},
			},
			env: TypeEnv{"CustomType": &StructType{Name: "CustomType"}},
			wantType: &TypeAlias{
				Name:      "MyCustomType",
				AliasedTo: &StructType{Name: "CustomType"},
			},
			wantErr: nil,
		},
		{
			name: "Type alias to a generic type",
			spec: &ast.TypeSpec{
				Name:   &ast.Ident{Name: "MyVector"},
				Assign: token.Pos(1),
				Type: &ast.IndexExpr{
					X:     &ast.Ident{Name: "Vector"},
					Index: &ast.Ident{Name: "int"},
				},
			},
			env: TypeEnv{
				"Vector": &GenericType{
					Name:       "Vector",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
				},
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &TypeAlias{
				Name: "MyVector",
				AliasedTo: &GenericType{
					Name:       "Vector",
					TypeParams: []Type{&TypeConstant{Name: "int"}},
				},
			},
			wantErr: nil,
		},
		{
			name: "Type alias to an unknown type",
			spec: &ast.TypeSpec{
				Name:   &ast.Ident{Name: "MyUnknown"},
				Assign: token.Pos(1),
				Type:   &ast.Ident{Name: "UnknownType"},
			},
			env:      TypeEnv{},
			wantType: nil,
			wantErr:  fmt.Errorf("unknown identifier: UnknownType"),
		},
		{
			name: "New interface type declaration",
			spec: &ast.TypeSpec{
				Name: &ast.Ident{Name: "MyInterface"},
				Type: &ast.InterfaceType{
					Methods: &ast.FieldList{
						List: []*ast.Field{
							{
								Names: []*ast.Ident{{Name: "Method1"}},
								Type: &ast.FuncType{
									Params:  &ast.FieldList{},
									Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}}},
								},
							},
						},
					},
				},
			},
			env: TypeEnv{"int": &TypeConstant{Name: "int"}},
			wantType: &InterfaceType{
				Name: "MyInterface",
				Methods: MethodSet{
					"Method1": Method{
						Name:    "Method1",
						Params:  []Type{},
						Results: []Type{&TypeConstant{Name: "int"}},
					},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inferTypeSpec(tt.spec, tt.env)
			if err != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("InferTypeSpec() error diff:\n%s", diffStrings(err.Error(), tt.wantErr.Error()))
				return
			}
			if !TypesEqual(got, tt.wantType) {
				t.Errorf("InferTypeSpec() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestInferResult(t *testing.T) {
	tests := []struct {
		name     string
		results  *ast.FieldList
		env      TypeEnv
		wantType Type
		wantErr  error
	}{
		{
			name:     "No return value",
			results:  &ast.FieldList{},
			env:      TypeEnv{},
			wantType: &TypeConstant{Name: "void"},
			wantErr:  nil,
		},
		{
			name: "Single return value",
			results: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "int"}},
				},
			},
			env:      TypeEnv{"int": &TypeConstant{Name: "int"}},
			wantType: &TypeConstant{Name: "int"},
			wantErr:  nil,
		},
		{
			name: "Multiple return values",
			results: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "int"}},
					{Type: &ast.Ident{Name: "string"}},
				},
			},
			env: TypeEnv{
				"int":    &TypeConstant{Name: "int"},
				"string": &TypeConstant{Name: "string"},
			},
			wantType: &TupleType{
				Types: []Type{
					&TypeConstant{Name: "int"},
					&TypeConstant{Name: "string"},
				},
			},
			wantErr: nil,
		},
		{
			name: "Named return values",
			results: &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "x"}}, Type: &ast.Ident{Name: "int"}},
					{Names: []*ast.Ident{{Name: "y"}}, Type: &ast.Ident{Name: "string"}},
				},
			},
			env: TypeEnv{
				"int":    &TypeConstant{Name: "int"},
				"string": &TypeConstant{Name: "string"},
			},
			wantType: &TupleType{
				Types: []Type{
					&TypeConstant{Name: "int"},
					&TypeConstant{Name: "string"},
				},
			},
			wantErr: nil,
		},
		{
			name: "Mixed named and unnamed return values",
			results: &ast.FieldList{
				List: []*ast.Field{
					{Names: []*ast.Ident{{Name: "x"}}, Type: &ast.Ident{Name: "int"}},
					{Type: &ast.Ident{Name: "string"}},
				},
			},
			env: TypeEnv{
				"int":    &TypeConstant{Name: "int"},
				"string": &TypeConstant{Name: "string"},
			},
			wantType: &TupleType{
				Types: []Type{
					&TypeConstant{Name: "int"},
					&TypeConstant{Name: "string"},
				},
			},
			wantErr: nil,
		},
		{
			name: "Unknown return type",
			results: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "UnknownType"}},
				},
			},
			env:      TypeEnv{},
			wantType: nil,
			wantErr:  fmt.Errorf("unknown identifier: UnknownType"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := inferResult(tt.results, tt.env)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("inferResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("inferResult() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !TypesEqual(got, tt.wantType) {
				t.Errorf("inferResult() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestInferVariadicFunction(t *testing.T) {
	expr := &ast.FuncType{
		Params: &ast.FieldList{
			List: []*ast.Field{
				{Names: []*ast.Ident{{Name: "x"}}, Type: &ast.Ident{Name: "int"}},
				{Names: []*ast.Ident{{Name: "y"}}, Type: &ast.Ellipsis{Elt: &ast.Ident{Name: "string"}}},
			},
		},
		Results: &ast.FieldList{
			List: []*ast.Field{{Type: &ast.Ident{Name: "bool"}}},
		},
	}

	env := TypeEnv{
		"int":    &TypeConstant{Name: "int"},
		"string": &TypeConstant{Name: "string"},
		"bool":   &TypeConstant{Name: "bool"},
	}

	expected := &FunctionType{
		ParamTypes: []Type{
			&TypeConstant{Name: "int"},
			&SliceType{ElementType: &TypeConstant{Name: "string"}},
		},
		ReturnType: &TypeConstant{Name: "bool"},
		IsVariadic: true,
	}

	result, err := InferType(expr, env)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !TypesEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestInferTypeEllipsis(t *testing.T) {
	tests := []struct {
		name     string
		expr     ast.Expr
		env      TypeEnv
		wantType Type
		wantErr  error
	}{
		{
			name: "Ellipsis with int type",
			expr: &ast.Ellipsis{Elt: &ast.Ident{Name: "int"}},
			env:  TypeEnv{"int": &TypeConstant{Name: "int"}},
			wantType: &SliceType{
				ElementType: &TypeConstant{Name: "int"},
			},
			wantErr: nil,
		},
		{
			name: "Empty ellipsis",
			expr: &ast.Ellipsis{Elt: nil},
			env:  TypeEnv{},
			wantType: &SliceType{
				ElementType: &InterfaceType{Name: "interface{}", IsEmpty: true},
			},
			wantErr: nil,
		},
		{
			name: "Ellipsis with custom type",
			expr: &ast.Ellipsis{Elt: &ast.Ident{Name: "CustomType"}},
			env:  TypeEnv{"CustomType": &StructType{Name: "CustomType"}},
			wantType: &SliceType{
				ElementType: &StructType{Name: "CustomType"},
			},
			wantErr: nil,
		},
		{
			name: "Ellipsis with generic type",
			expr: &ast.Ellipsis{Elt: &ast.IndexExpr{
				X:     &ast.Ident{Name: "Vector"},
				Index: &ast.Ident{Name: "int"},
			}},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferType(tt.expr, tt.env)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("InferType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("InferType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !TypesEqual(got, tt.wantType) {
				t.Errorf("InferType() = %v, want %v", got, tt.wantType)
			}
		})
	}
}
