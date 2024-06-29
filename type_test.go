package generic

import (
	"fmt"
	"testing"
)

func TestStringMethods(t *testing.T) {
	tv := &TypeVariable{Name: "T"}
	tc := &TypeConstant{Name: "int"}
	ft := &FunctionType{
		ParamTypes: []Type{&TypeVariable{Name: "A"}, &TypeConstant{Name: "string"}},
		ReturnType: &TypeConstant{Name: "bool"},
		IsVariadic: true,
	}
	tt := &TupleType{
		Types: []Type{&TypeConstant{Name: "int"}, &TypeVariable{Name: "T"}},
	}
	it := &Interface{Name: "Reader"}
	itt := &InterfaceType{
		Name:    "io.Reader",
		Methods: MethodSet{"Read": {Name: "Read", Params: []Type{}, Results: []Type{&TypeConstant{Name: "int"}}}},
		IsEmpty: false,
	}
	emtIface := InterfaceType{IsEmpty: true}
	pt := &PointerType{Base: &TypeConstant{Name: "int"}}
	st := &StructType{Name: "Person"}
	slt := &SliceType{ElementType: &TypeConstant{Name: "string"}}
	at := &ArrayType{ElementType: &TypeVariable{Name: "T"}, Len: 10}
	mt := &MapType{KeyType: &TypeConstant{Name: "string"}, ValueType: &TypeConstant{Name: "int"}}
	tcst := &TypeConstraint{
		Interfaces: []Interface{{Name: "Stringer"}},
		Types:      []Type{&TypeConstant{Name: "int"}},
	}
	gt := &GenericType{
		Name:       "Stack",
		TypeParams: []Type{&TypeVariable{Name: "T"}},
		Constraints: map[string]TypeConstraint{
			"Stack": {Interfaces: []Interface{{Name: "Container"}}, Types: []Type{&TypeConstant{Name: "int"}}},
		},
		Fields: map[string]Type{"items": &SliceType{ElementType: &TypeVariable{Name: "T"}}},
	}
	ta := &TypeAlias{Name: "RuneReader", AliasedTo: &InterfaceType{Name: "io.RuneReader"}}

	tests := []struct {
		typ      Type
		expected string
	}{
		{tv, "TypeVar(T)"},
		{tc, "TypeConst(int)"},
		{ft, "func(TypeVar(A), TypeConst(string)...) TypeConst(bool)"},
		{tt, "(TypeConst(int), TypeVar(T))"},
		{it, "Interface(Reader)"},
		{itt, "InterfaceType(io.Reader)"},
		{&emtIface, "interface{}"},
		{pt, "*TypeConst(int)"},
		{st, "Struct(Person)"},
		{slt, "Slice(TypeConst(string))"},
		{at, "Arr[10]TypeVar(T)"},
		{mt, "Map[TypeConst(string)]TypeConst(int)"},
		{tcst, "Constraint([Stringer], [TypeConst(int)])"},
		{gt, "Generic(Stack, [TypeVar(T)])"},
		{ta, "TypeAlias(RuneReader = InterfaceType(io.RuneReader))"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.typ), func(t *testing.T) {
			result := tt.typ.String()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
