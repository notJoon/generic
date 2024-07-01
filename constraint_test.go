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

func TestIsComparable(t *testing.T) {
	tests := []struct {
		name string
		t    Type
		want bool
	}{
		{"int is comparable", &TypeConstant{Name: "int"}, true},
		{"string is comparable", &TypeConstant{Name: "string"}, true},
		{"bool is comparable", &TypeConstant{Name: "bool"}, true},
		{"*int is comparable", &PointerType{Base: &TypeConstant{Name: "int"}}, true},
		{"[]int is not comparable", &SliceType{ElementType: &TypeConstant{Name: "int"}}, false},
		{"empty interface is comparable", &InterfaceType{IsEmpty: true}, true},
		{"struct with comparable fields is comparable", &StructType{
			Fields: map[string]Type{
				"a": &TypeConstant{Name: "int"},
				"b": &TypeConstant{Name: "string"},
			},
		}, true},
		{"struct with non-comparable fields is not comparable", &StructType{
			Fields: map[string]Type{
				"a": &TypeConstant{Name: "int"},
				"b": &SliceType{ElementType: &TypeConstant{Name: "int"}},
			},
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isComparable(tt.t); got != tt.want {
				t.Errorf("isComparable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsOrdered(t *testing.T) {
	tests := []struct {
		name string
		t    Type
		want bool
	}{
		{"int is ordered", &TypeConstant{Name: "int"}, true},
		{"float64 is ordered", &TypeConstant{Name: "float64"}, true},
		{"string is ordered", &TypeConstant{Name: "string"}, true},
		{"bool is not ordered", &TypeConstant{Name: "bool"}, false},
		{"complex128 is not ordered", &TypeConstant{Name: "complex128"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOrdered(tt.t); got != tt.want {
				t.Errorf("isOrdered() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsComplex(t *testing.T) {
	tests := []struct {
		name string
		t    Type
		want bool
	}{
		{"complex64 is complex", &TypeConstant{Name: "complex64"}, true},
		{"complex128 is complex", &TypeConstant{Name: "complex128"}, true},
		{"float64 is not complex", &TypeConstant{Name: "float64"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isComplex(tt.t); got != tt.want {
				t.Errorf("isComplex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFloat(t *testing.T) {
	tests := []struct {
		name string
		t    Type
		want bool
	}{
		{"float32 is float", &TypeConstant{Name: "float32"}, true},
		{"float64 is float", &TypeConstant{Name: "float64"}, true},
		{"int is not float", &TypeConstant{Name: "int"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFloat(tt.t); got != tt.want {
				t.Errorf("isFloat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInteger(t *testing.T) {
	tests := []struct {
		name string
		t    Type
		want bool
	}{
		{"int is integer", &TypeConstant{Name: "int"}, true},
		{"uint64 is integer", &TypeConstant{Name: "uint64"}, true},
		{"float64 is not integer", &TypeConstant{Name: "float64"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInteger(tt.t); got != tt.want {
				t.Errorf("isInteger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSigned(t *testing.T) {
	tests := []struct {
		name string
		t    Type
		want bool
	}{
		{"int is signed", &TypeConstant{Name: "int"}, true},
		{"int64 is signed", &TypeConstant{Name: "int64"}, true},
		{"uint is not signed", &TypeConstant{Name: "uint"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSigned(tt.t); got != tt.want {
				t.Errorf("isSigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnsigned(t *testing.T) {
	tests := []struct {
		name string
		t    Type
		want bool
	}{
		{"uint is unsigned", &TypeConstant{Name: "uint"}, true},
		{"uint64 is unsigned", &TypeConstant{Name: "uint64"}, true},
		{"int is not unsigned", &TypeConstant{Name: "int"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUnsigned(tt.t); got != tt.want {
				t.Errorf("isUnsigned() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckBuiltinConstraint(t *testing.T) {
	tests := []struct {
		name       string
		t          Type
		constraint string
		want       bool
	}{
		{"int satisfies any", &TypeConstant{Name: "int"}, "any", true},
		{"int satisfies comparable", &TypeConstant{Name: "int"}, "comparable", true},
		{"int satisfies ordered", &TypeConstant{Name: "int"}, "ordered", true},
		{"int satisfies integer", &TypeConstant{Name: "int"}, "integer", true},
		{"int satisfies signed", &TypeConstant{Name: "int"}, "signed", true},
		{"int does not satisfy unsigned", &TypeConstant{Name: "int"}, "unsigned", false},
		{"uint satisfies unsigned", &TypeConstant{Name: "uint"}, "unsigned", true},
		{"float64 satisfies float", &TypeConstant{Name: "float64"}, "float", true},
		{"complex128 satisfies complex", &TypeConstant{Name: "complex128"}, "complex", true},
		{"string satisfies ordered", &TypeConstant{Name: "string"}, "ordered", true},
		{"string does not satisfy numeric", &TypeConstant{Name: "string"}, "numeric", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkBuiltinConstraint(tt.t, tt.constraint); got != tt.want {
				t.Errorf("checkBuiltinConstraint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnderlyingType(t *testing.T) {
	tests := []struct {
		name           string
		t              Type
		underlyingType Type
		want           bool
	}{
		{
			name:           "int and int",
			t:              &TypeConstant{Name: "int"},
			underlyingType: &TypeConstant{Name: "int"},
			want:           true,
		},
		{
			name:           "MyInt and int",
			t:              &TypeAlias{Name: "MyInt", AliasedTo: &TypeConstant{Name: "int"}},
			underlyingType: &TypeConstant{Name: "int"},
			want:           true,
		},
		{
			name:           "int and float64",
			t:              &TypeConstant{Name: "int"},
			underlyingType: &TypeConstant{Name: "float64"},
			want:           false,
		},
		{
			name:           "MyString and string",
			t:              &TypeAlias{Name: "MyString", AliasedTo: &TypeConstant{Name: "string"}},
			underlyingType: &TypeConstant{Name: "string"},
			want:           true,
		},
		{
			name:           "[]MyInt and []int",
			t:              &SliceType{ElementType: &TypeAlias{Name: "MyInt", AliasedTo: &TypeConstant{Name: "int"}}},
			underlyingType: &SliceType{ElementType: &TypeConstant{Name: "int"}},
			want:           true,
		},
		{
			name:           "Map[string]MyInt and Map[string]int",
			t:              &MapType{KeyType: &TypeConstant{Name: "string"}, ValueType: &TypeAlias{Name: "MyInt", AliasedTo: &TypeConstant{Name: "int"}}},
			underlyingType: &MapType{KeyType: &TypeConstant{Name: "string"}, ValueType: &TypeConstant{Name: "int"}},
			want:           true,
		},
		{
			name: "Struct{X int, Y string} and Struct{X int, Y string}",
			t: &StructType{
				Fields: map[string]Type{
					"X": &TypeConstant{Name: "int"},
					"Y": &TypeConstant{Name: "string"},
				},
			},
			underlyingType: &StructType{
				Fields: map[string]Type{
					"X": &TypeConstant{Name: "int"},
					"Y": &TypeConstant{Name: "string"},
				},
			},
			want: true,
		},
		{
			name: "Struct{X MyInt, Y MyString} and Struct{X int, Y string}",
			t: &StructType{
				Fields: map[string]Type{
					"X": &TypeAlias{Name: "MyInt", AliasedTo: &TypeConstant{Name: "int"}},
					"Y": &TypeAlias{Name: "MyString", AliasedTo: &TypeConstant{Name: "string"}},
				},
			},
			underlyingType: &StructType{
				Fields: map[string]Type{
					"X": &TypeConstant{Name: "int"},
					"Y": &TypeConstant{Name: "string"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUnderlyingType(tt.t, tt.underlyingType); got != tt.want {
				t.Errorf("isUnderlyingType() = %v, want %v", got, tt.want)
			}
		})
	}
}
