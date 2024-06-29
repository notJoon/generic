package generic

import (
	"testing"
)

// Test case for unification of two identical type variables
func TestUnifyIdenticalTypeVariables(t *testing.T) {
	env := TypeEnv{}
	tv1 := &TypeVariable{Name: "T"}
	tv2 := &TypeVariable{Name: "T"}

	err := Unify(tv1, tv2, env)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// Test case for unification of a type variable with a type constant
func TestUnifyTypeVariableWithTypeConstant(t *testing.T) {
	env := TypeEnv{}
	tv := &TypeVariable{Name: "T"}
	tc := &TypeConstant{Name: "int"}

	err := Unify(tv, tc, env)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if env["T"] != tc {
		t.Fatalf("Expected T to be unified with int, got %v", env["T"])
	}
}

// Test case for unification of two different type constants
func TestUnifyDifferentTypeConstants(t *testing.T) {
	env := TypeEnv{}
	tc1 := &TypeConstant{Name: "int"}
	tc2 := &TypeConstant{Name: "string"}

	err := Unify(tc1, tc2, env)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

// Test case for unification of function types
func TestUnifyFunctionTypes(t *testing.T) {
	env := TypeEnv{}
	ft1 := &FunctionType{
		ParamTypes: []Type{&TypeConstant{Name: "int"}},
		ReturnType: &TypeConstant{Name: "int"},
	}
	ft2 := &FunctionType{
		ParamTypes: []Type{&TypeConstant{Name: "int"}},
		ReturnType: &TypeConstant{Name: "int"},
	}

	err := Unify(ft1, ft2, env)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// Test case for unification of function types with different arities
func TestUnifyFunctionTypesDifferentArity(t *testing.T) {
	env := TypeEnv{}
	ft1 := &FunctionType{
		ParamTypes: []Type{&TypeConstant{Name: "int"}},
		ReturnType: &TypeConstant{Name: "int"},
	}
	ft2 := &FunctionType{
		ParamTypes: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "string"}},
		ReturnType: &TypeConstant{Name: "int"},
	}

	err := Unify(ft1, ft2, env)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

// TestCircularReference tests if the unification detects circular references
func TestCircularReference(t *testing.T) {
	env := TypeEnv{}
	tv := &TypeVariable{Name: "T"}
	ft := &FunctionType{
		ParamTypes: []Type{tv},
		ReturnType: tv,
	}

	err := Unify(tv, ft, env)
	if err != ErrCircularReference {
		t.Fatalf("Expected ErrCircularReference, got %v", err)
	}
}

// TestNestedCircularReference tests if the unification detects nested circular references
func TestNestedCircularReference(t *testing.T) {
	env := TypeEnv{}
	tv1 := &TypeVariable{Name: "T1"}
	tv2 := &TypeVariable{Name: "T2"}
	ft1 := &FunctionType{
		ParamTypes: []Type{tv2},
		ReturnType: &TypeConstant{Name: "int"},
	}
	ft2 := &FunctionType{
		ParamTypes: []Type{tv1},
		ReturnType: &TypeConstant{Name: "string"},
	}

	err := Unify(tv1, ft1, env)
	if err != nil {
		t.Fatalf("Expected no error in first unification, got %v", err)
	}

	err = Unify(tv2, ft2, env)
	if err != ErrCircularReference {
		t.Fatalf("Expected ErrCircularReference in second unification, got %v", err)
	}
}

func TestUnifyInterface(t *testing.T) {
	tests := []struct {
		name    string
		t1      Type
		t2      Type
		wantErr error
	}{
		{
			name: "Unify identical interfaces",
			t1: &InterfaceType{
				Methods: MethodSet{
					"Method1": Method{Name: "Method1", Params: []Type{}, Results: []Type{}},
				},
			},
			t2: &InterfaceType{
				Methods: MethodSet{
					"Method1": Method{Name: "Method1", Params: []Type{}, Results: []Type{}},
				},
			},
			wantErr: nil,
		},
		{
			name: "Unify interfaces with different method names",
			t1: &InterfaceType{
				Methods: MethodSet{
					"Method1": Method{Name: "Method1", Params: []Type{}, Results: []Type{}},
				},
			},
			t2: &InterfaceType{
				Methods: MethodSet{
					"Method2": Method{Name: "Method2", Params: []Type{}, Results: []Type{}},
				},
			},
			wantErr: ErrTypeMismatch,
		},
		{
			name: "Unify interface with empty interface",
			t1: &InterfaceType{
				Methods: MethodSet{
					"Method1": Method{Name: "Method1", Params: []Type{}, Results: []Type{}},
				},
			},
			t2: &InterfaceType{
				IsEmpty: true,
			},
			wantErr: nil,
		},
		{
			name: "Unify interfaces with embedded interfaces",
			t1: &InterfaceType{
				Methods: MethodSet{
					"Method1": Method{Name: "Method1", Params: []Type{}, Results: []Type{}},
				},
				Embedded: []Type{
					&InterfaceType{
						Methods: MethodSet{
							"EmbeddedMethod": Method{Name: "EmbeddedMethod", Params: []Type{}, Results: []Type{}},
						},
					},
				},
			},
			t2: &InterfaceType{
				Methods: MethodSet{
					"Method1":        Method{Name: "Method1", Params: []Type{}, Results: []Type{}},
					"EmbeddedMethod": Method{Name: "EmbeddedMethod", Params: []Type{}, Results: []Type{}},
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := TypeEnv{}
			err := Unify(tt.t1, tt.t2, env)
			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("Unify() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErr != nil && err.Error() != tt.wantErr.Error() {
				t.Errorf("Unify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnifyTupleType(t *testing.T) {
	tests := []struct {
		name    string
		t1      Type
		t2      Type
		wantErr error
	}{
		{
			name:    "Identical tuples",
			t1:      &TupleType{Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "string"}}},
			t2:      &TupleType{Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "string"}}},
			wantErr: nil,
		},
		{
			name:    "Different tuple lengths",
			t1:      &TupleType{Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "string"}}},
			t2:      &TupleType{Types: []Type{&TypeConstant{Name: "int"}}},
			wantErr: ErrArityMismatch,
		},
		{
			name:    "Mismatched tuple types",
			t1:      &TupleType{Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "string"}}},
			t2:      &TupleType{Types: []Type{&TypeConstant{Name: "int"}, &TypeConstant{Name: "int"}}},
			wantErr: ErrTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := TypeEnv{}
			err := Unify(tt.t1, tt.t2, env)
			if err != tt.wantErr {
				t.Errorf("Unify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnifyGenericType(t *testing.T) {
	tests := []struct {
		name    string
		t1      Type
		t2      Type
		wantErr error
	}{
		{
			name: "Identical generic types",
			t1: &GenericType{
				Name:       "List",
				TypeParams: []Type{&TypeVariable{Name: "T"}},
			},
			t2: &GenericType{
				Name:       "List",
				TypeParams: []Type{&TypeVariable{Name: "T"}},
			},
			wantErr: nil,
		},
		{
			name: "Different generic type names",
			t1: &GenericType{
				Name:       "List",
				TypeParams: []Type{&TypeVariable{Name: "T"}},
			},
			t2: &GenericType{
				Name:       "Vector",
				TypeParams: []Type{&TypeVariable{Name: "T"}},
			},
			wantErr: ErrTypeMismatch,
		},
		{
			name: "Different number of type parameters",
			t1: &GenericType{
				Name:       "Pair",
				TypeParams: []Type{&TypeVariable{Name: "T"}, &TypeVariable{Name: "U"}},
			},
			t2: &GenericType{
				Name:       "Pair",
				TypeParams: []Type{&TypeVariable{Name: "T"}},
			},
			wantErr: ErrTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := TypeEnv{}
			err := Unify(tt.t1, tt.t2, env)
			if err != tt.wantErr {
				t.Errorf("Unify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnifyMapType(t *testing.T) {
	tests := []struct {
		name    string
		t1      Type
		t2      Type
		wantErr error
	}{
		{
			name:    "Identical map types",
			t1:      &MapType{KeyType: &TypeConstant{Name: "string"}, ValueType: &TypeConstant{Name: "int"}},
			t2:      &MapType{KeyType: &TypeConstant{Name: "string"}, ValueType: &TypeConstant{Name: "int"}},
			wantErr: nil,
		},
		{
			name:    "Different key types",
			t1:      &MapType{KeyType: &TypeConstant{Name: "string"}, ValueType: &TypeConstant{Name: "int"}},
			t2:      &MapType{KeyType: &TypeConstant{Name: "int"}, ValueType: &TypeConstant{Name: "int"}},
			wantErr: ErrTypeMismatch,
		},
		{
			name:    "Different value types",
			t1:      &MapType{KeyType: &TypeConstant{Name: "string"}, ValueType: &TypeConstant{Name: "int"}},
			t2:      &MapType{KeyType: &TypeConstant{Name: "string"}, ValueType: &TypeConstant{Name: "string"}},
			wantErr: ErrTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := TypeEnv{}
			err := Unify(tt.t1, tt.t2, env)
			if err != tt.wantErr {
				t.Errorf("Unify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUnifyPointerType(t *testing.T) {
	tests := []struct {
		name    string
		t1      Type
		t2      Type
		wantErr error
	}{
		{
			name:    "Identical pointer types",
			t1:      &PointerType{Base: &TypeConstant{Name: "int"}},
			t2:      &PointerType{Base: &TypeConstant{Name: "int"}},
			wantErr: nil,
		},
		{
			name:    "Different base types",
			t1:      &PointerType{Base: &TypeConstant{Name: "int"}},
			t2:      &PointerType{Base: &TypeConstant{Name: "string"}},
			wantErr: ErrTypeMismatch,
		},
		{
			name:    "Pointer type with non-pointer type",
			t1:      &PointerType{Base: &TypeConstant{Name: "int"}},
			t2:      &TypeConstant{Name: "int"},
			wantErr: ErrTypeMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := TypeEnv{}
			err := Unify(tt.t1, tt.t2, env)
			if err != tt.wantErr {
				t.Errorf("Unify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
