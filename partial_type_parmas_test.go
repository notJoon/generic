package generic

import (
	"go/ast"
	"testing"
)

func TestPartialTypeParameterInference(t *testing.T) {
	env := TypeEnv{
		"int":    &TypeConstant{Name: "int"},
		"string": &TypeConstant{Name: "string"},
		"MyFunc": &GenericType{
			Name: "MyFunc",
			TypeParams: []Type{
				&TypeVariable{Name: "T"},
				&TypeVariable{Name: "U"},
			},
			Constraints: map[string]TypeConstraint{
				"T": {Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "float64"}}},
				"U": {Types: []Type{&TypeConstant{Name: "string"}, &TypeConstant{Name: "bool"}}},
			},
		},
	}

	tests := []struct {
		name           string
		expr           ast.Expr
		expectedParams []Type
		expectError    bool
	}{
		{
			name: "Fully specified type parameters",
			expr: &ast.IndexListExpr{
				X: &ast.Ident{Name: "MyFunc"},
				Indices: []ast.Expr{
					&ast.Ident{Name: "int"},
					&ast.Ident{Name: "string"},
				},
			},
			expectedParams: []Type{
				&TypeConstant{Name: "int"},
				&TypeConstant{Name: "string"},
			},
			expectError: false,
		},
		{
			name: "Partially specified type parameters",
			expr: &ast.IndexListExpr{
				X: &ast.Ident{Name: "MyFunc"},
				Indices: []ast.Expr{
					&ast.Ident{Name: "int"},
				},
			},
			expectedParams: []Type{
				&TypeConstant{Name: "int"},
				&TypeVariable{Name: "U"},
			},
			expectError: false,
		},
		{
			name: "Infer all type parameters",
			expr: &ast.Ident{Name: "MyFunc"},
			expectedParams: []Type{
				&TypeVariable{Name: "T"},
				&TypeVariable{Name: "U"},
			},
			expectError: false,
		},
		{
			name: "Invalid type parameter",
			expr: &ast.IndexListExpr{
				X: &ast.Ident{Name: "MyFunc"},
				Indices: []ast.Expr{
					&ast.Ident{Name: "bool"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inferred, err := InferType(tt.expr, env, nil)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			genericType, ok := inferred.(*GenericType)
			if !ok {
				t.Fatalf("Expected GenericType, got %T", inferred)
			}

			if len(genericType.TypeParams) != len(tt.expectedParams) {
				t.Fatalf("Expected %d type parameters, got %d", len(tt.expectedParams), len(genericType.TypeParams))
			}

			for i, param := range genericType.TypeParams {
				if !TypesEqual(param, tt.expectedParams[i]) {
					t.Errorf("Type parameter %d: expected %v, got %v", i, tt.expectedParams[i], param)
				}
			}

			// Check if constraints are satisfied
			originalGeneric := env["MyFunc"].(*GenericType)
			for i, param := range genericType.TypeParams {
				constraint := originalGeneric.Constraints[originalGeneric.TypeParams[i].(*TypeVariable).Name]
				if !checkConstraint(param, constraint) {
					t.Errorf("Type parameter %d (%v) does not satisfy constraint %v", i, param, constraint)
				}
			}
		})
	}
}