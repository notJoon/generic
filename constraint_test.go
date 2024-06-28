package generic

import (
	"go/ast"
	"testing"
)

var (
	// testing purpose
	Printable = Interface{
		Name: "Printable",
		Methods: MethodSet{
			"String": Method{
				Name:      "String",
				Params:    []Type{},
				Results:   []Type{&TypeConstant{Name: "string"}},
				IsPointer: false,
			},
		},
	}

	Comparable = Interface{
		Name: "Comparable",
		Methods: MethodSet{
			"Compare": Method{
				Name:      "Compare",
				Params:    []Type{&TypeVariable{Name: "T"}},
				Results:   []Type{&TypeConstant{Name: "int"}},
				IsPointer: false,
			},
		},
	}

	IntType    = &TypeConstant{Name: "int"}
	StringType = &TypeConstant{Name: "string"}
	FloatType  = &TypeConstant{Name: "float64"}
)

func TestInferTypeWithConstraints(t *testing.T) {
	tests := []struct {
		name     string
		expr     ast.Expr
		env      TypeEnv
		wantType Type
		wantErr  error
	}{
		{
			name: "generic type with satisfied constraints",
			expr: &ast.IndexExpr{
				X:     &ast.Ident{Name: "xs"},
				Index: &ast.Ident{Name: "i"},
			},
			env: TypeEnv{
				"xs": &GenericType{
					Name:       "xs",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
					Constraints: map[string]TypeConstraint{
						"T": {
							Interfaces: []Interface{Comparable},
						},
					},
				},
				"i": IntType,
			},
			wantType: &GenericType{
				Name:       "xs",
				TypeParams: []Type{IntType},
			},
		},
		{
			name: "generic type with multiple constraints",
			expr: &ast.IndexExpr{
				X:     &ast.Ident{Name: "Advanced"},
				Index: &ast.Ident{Name: "string"},
			},
			env: TypeEnv{
				"Advanced": &GenericType{
					Name:       "Advanced",
					TypeParams: []Type{&TypeVariable{Name: "T"}},
					Constraints: map[string]TypeConstraint{
						"T": {
							Interfaces: []Interface{Printable, Comparable},
							Types:      []Type{StringType, FloatType},
						},
					},
				},
				"string": StringType,
			},
			wantType: &GenericType{
				Name:       "Advanced",
				TypeParams: []Type{StringType},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferType(tt.expr, tt.env)
			if err != tt.wantErr {
				t.Errorf("InferType(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !TypesEqual(got, tt.wantType) {
				t.Errorf("InferType(%s) = %v, want %v", tt.name, got, tt.wantType)
			}
		})
	}
}
