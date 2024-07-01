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
			ctx := NewInferenceContext()
			got, err := InferType(tt.expr, tt.env, ctx)
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

func TestImplInterface(t *testing.T) {
	tests := []struct {
		name           string
		t              Type
		iface          Interface
		expectedResult bool
	}{
		{
			name: "int implements Stringer",
			t:    &TypeConstant{Name: "int"},
			iface: Interface{
				Name: "Stringer",
				Methods: MethodSet{
					"String": Method{
						Name:    "String",
						Params:  []Type{},
						Results: []Type{&TypeConstant{Name: "string"}},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "string implements Comparable",
			t:    &TypeConstant{Name: "string"},
			iface: Interface{
				Name: "Comparable",
				Methods: MethodSet{
					"Compare": Method{
						Name:    "Compare",
						Params:  []Type{&TypeVariable{Name: "T"}},
						Results: []Type{&TypeConstant{Name: "int"}},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "float64 does not implement error",
			t:    &TypeConstant{Name: "float64"},
			iface: Interface{
				Name: "error",
				Methods: MethodSet{
					"Error": Method{
						Name:    "Error",
						Params:  []Type{},
						Results: []Type{&TypeConstant{Name: "string"}},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "custom struct implements custom interface",
			t: &StructType{
				Name: "MyStruct",
				Methods: MethodSet{
					"MyMethod": Method{
						Name:    "MyMethod",
						Params:  []Type{},
						Results: []Type{&TypeConstant{Name: "int"}},
					},
				},
			},
			iface: Interface{
				Name: "MyInterface",
				Methods: MethodSet{
					"MyMethod": Method{
						Name:    "MyMethod",
						Params:  []Type{},
						Results: []Type{&TypeConstant{Name: "int"}},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "custom struct does not implement interface with additional method",
			t: &StructType{
				Name: "MyStruct",
				Methods: MethodSet{
					"MyMethod": Method{
						Name:    "MyMethod",
						Params:  []Type{},
						Results: []Type{&TypeConstant{Name: "int"}},
					},
				},
			},
			iface: Interface{
				Name: "MyInterface",
				Methods: MethodSet{
					"MyMethod": Method{
						Name:    "MyMethod",
						Params:  []Type{},
						Results: []Type{&TypeConstant{Name: "int"}},
					},
					"AnotherMethod": Method{
						Name:    "AnotherMethod",
						Params:  []Type{},
						Results: []Type{&TypeConstant{Name: "string"}},
					},
				},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := implInterface(tt.t, tt.iface)
			if result != tt.expectedResult {
				t.Errorf("implInterface(%v, %v) = %v, want %v", tt.t, tt.iface, result, tt.expectedResult)
			}
		})
	}
}

func TestCheckPrimitiveTypeInterface(t *testing.T) {
	tests := []struct {
		name           string
		typeName       string
		iface          Interface
		expectedResult bool
	}{
		{
			name:           "int implements Stringer",
			typeName:       "int",
			iface:          Interface{Name: "Stringer"},
			expectedResult: true,
		},
		{
			name:           "string implements Comparable",
			typeName:       "string",
			iface:          Interface{Name: "Comparable"},
			expectedResult: true,
		},
		{
			name:           "float64 does not implement error",
			typeName:       "float64",
			iface:          Interface{Name: "error"},
			expectedResult: false,
		},
		{
			name:           "bool does not implement Stringer",
			typeName:       "bool",
			iface:          Interface{Name: "Stringer"},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkPrimitiveTypeInterface(tt.typeName, tt.iface)
			if result != tt.expectedResult {
				t.Errorf("checkPrimitiveTypeInterface(%q, %v) = %v, want %v", tt.typeName, tt.iface, result, tt.expectedResult)
			}
		})
	}
}
