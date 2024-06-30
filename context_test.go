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
		{
			name: "Infer from function argument",
			genericFunc: &GenericType{
				Name:       "Apply",
				TypeParams: []Type{&TypeVariable{Name: "T"}, &TypeVariable{Name: "U"}},
				Constraints: map[string]TypeConstraint{
					"T": {Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "string"}}},
					"U": {Types: []Type{&TypeConstant{Name: "bool"}, &TypeConstant{Name: "float64"}}},
				},
			},
			args: []ast.Expr{
				&ast.BasicLit{Kind: token.INT, Value: "42"},
				&ast.FuncLit{
					Type: &ast.FuncType{
						Params:  &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}}},
						Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "float64"}}}},
					},
				},
			},
			env: TypeEnv{},
			want: []Type{
				&TypeConstant{Name: "int"},
				&TypeConstant{Name: "float64"},
			},
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
