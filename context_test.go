package generic

import (
	"go/ast"
	"go/token"
	"reflect"
	"testing"
)

func TestInferTypeArguments(t *testing.T) {
	tests := []struct {
		name        string
		genericFunc *GenericType
		args        []ast.Expr
		env         TypeEnv
		want        []Type
		wantErr     bool
	}{
		{
			name: "Infer single type argument",
			genericFunc: &GenericType{
				Name:       "Identity",
				TypeParams: []Type{&TypeVariable{Name: "T"}},
				Constraints: map[string]TypeConstraint{
					"T": {Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "string"}}},
				},
			},
			args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "42"}},
			env:  TypeEnv{},
			want: []Type{&TypeConstant{Name: "int"}},
		},
		{
			name: "Infer multiple type arguments",
			genericFunc: &GenericType{
				Name:       "Pair",
				TypeParams: []Type{&TypeVariable{Name: "T"}, &TypeVariable{Name: "U"}},
				Constraints: map[string]TypeConstraint{
					"T": {Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "float64"}}},
					"U": {Types: []Type{&TypeConstant{Name: "string"}, &TypeConstant{Name: "bool"}}},
				},
			},
			args: []ast.Expr{
				&ast.BasicLit{Kind: token.INT, Value: "42"},
				&ast.BasicLit{Kind: token.STRING, Value: `"hello"`},
			},
			env: TypeEnv{},
			want: []Type{
				&TypeConstant{Name: "int"},
				&TypeConstant{Name: "string"},
			},
		},
		{
			name: "Fail to infer due to constraint mismatch",
			genericFunc: &GenericType{
				Name:       "Identity",
				TypeParams: []Type{&TypeVariable{Name: "T"}},
				Constraints: map[string]TypeConstraint{
					"T": {Types: []Type{&TypeConstant{Name: "int"}}},
				},
			},
			args:    []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"hello"`}},
			env:     TypeEnv{},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferTypeArguments(tt.genericFunc, tt.args, tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferTypeArguments(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InferTypeArguments(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestInferFunctionLiteralType(t *testing.T) {
	tests := []struct {
		name     string
		funcLit  *ast.FuncLit
		env      TypeEnv
		wantType Type
		wantErr  bool
	}{
		{
			name: "Function with no params and one return value",
			funcLit: &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
					Results: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.Ident{Name: "int"}},
						},
					},
				},
			},
			env: TypeEnv{
				"int": &TypeConstant{Name: "int"},
			},
			wantType: &FunctionType{
				ParamTypes: []Type{},
				ReturnType: &TypeConstant{Name: "int"},
			},
			wantErr: false,
		},
		{
			name: "Function with one param and one return value",
			funcLit: &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.Ident{Name: "string"}},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.Ident{Name: "bool"}},
						},
					},
				},
			},
			env: TypeEnv{
				"string": &TypeConstant{Name: "string"},
				"bool":   &TypeConstant{Name: "bool"},
			},
			wantType: &FunctionType{
				ParamTypes: []Type{&TypeConstant{Name: "string"}},
				ReturnType: &TypeConstant{Name: "bool"},
			},
			wantErr: false,
		},
		{
			name: "Function with multiple params and multiple return values",
			funcLit: &ast.FuncLit{
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.Ident{Name: "int"}},
							{Type: &ast.Ident{Name: "string"}},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.Ident{Name: "bool"}},
							{Type: &ast.Ident{Name: "error"}},
						},
					},
				},
			},
			env: TypeEnv{
				"int":    &TypeConstant{Name: "int"},
				"string": &TypeConstant{Name: "string"},
				"bool":   &TypeConstant{Name: "bool"},
				"error":  &InterfaceType{Name: "error"},
			},
			wantType: &FunctionType{
				ParamTypes: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "string"}},
				ReturnType: &TupleType{
					Types: []Type{&TypeConstant{Name: "bool"}, &InterfaceType{Name: "error"}},
				},
			},
			wantErr: false,
		},
		// {
		// 	name: "Function with unknown type",
		// 	funcLit: &ast.FuncLit{
		// 		Type: &ast.FuncType{
		// 			Params: &ast.FieldList{
		// 				List: []*ast.Field{
		// 					{Type: &ast.Ident{Name: "unknownType"}},
		// 				},
		// 			},
		// 			Results: &ast.FieldList{},
		// 		},
		// 	},
		// 	env:      TypeEnv{},
		// 	wantType: nil,
		// 	wantErr:  true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, err := inferFunctionLiteralType(tt.funcLit, tt.env)
			if (err != nil) != tt.wantErr {
				t.Errorf("inferFunctionLiteralType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !TypesEqual(gotType, tt.wantType) {
				t.Errorf("inferFunctionLiteralType() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}
