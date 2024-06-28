package generic

import "fmt"

// Type represents any type in the type system.
// It serves as the base interface for all types in the generic type system.
type Type interface {
	String() string
}

// TypeVariable represents a type variable with a name.
// In generic programming, type variables are placeholders for types
// that will be specified later, allowing for polymorphic code.
type TypeVariable struct {
	Name string
}

func (tv *TypeVariable) String() string {
	return fmt.Sprintf("TypeVar(%s)", tv.Name)
}

// TypeConstant represents a constant type with a name.
// These are concrete types like `int`, `string`, or user-defined types.
type TypeConstant struct {
	Name string
}

func (tc *TypeConstant) String() string {
	return fmt.Sprintf("TypeConst(%s)", tc.Name)
}

// FunctionType represents a function type with parameter types and return type.
// It describes the signature of a function in the type system.
type FunctionType struct {
	ParamTypes []Type
	ReturnType Type
}

func (ft *FunctionType) String() string {
	params := make([]string, len(ft.ParamTypes))
	for i, param := range ft.ParamTypes {
		params[i] = param.String()
	}
	return fmt.Sprintf("Func(%v) -> %s", params, ft.ReturnType.String())
}

type Interface struct {
	Name    string
	Methods MethodSet
}

func (it *Interface) String() string {
	return fmt.Sprintf("Interface(%s)", it.Name)
}

// InterfaceType represents an interface type with methods.
type InterfaceType struct {
	Name    string
	Methods MethodSet
}

func (it *InterfaceType) String() string {
	return fmt.Sprintf("InterfaceType(%s)", it.Name)
}

type Method struct {
	Name      string
	Params    []Type
	Results   []Type
	IsPointer bool
}

type MethodSet map[string]Method

type PointerType struct {
	Base Type
}

func (pt *PointerType) String() string {
	return fmt.Sprintf("*%s", pt.Base.String())
}

var _ Type = (*PointerType)(nil)

// StructType represents a struct type with fields and methods.
type StructType struct {
	Name    string
	Fields  map[string]Type
	Methods MethodSet
}

func (st *StructType) String() string {
	return fmt.Sprintf("Struct(%s)", st.Name)
}

// SliceType represents a slice type
type SliceType struct {
	ElementType Type
}

func (st *SliceType) String() string {
	return fmt.Sprintf("Slice(%s)", st.ElementType.String())
}

type ArrayType struct {
	ElementType Type
	Len         int
}

func (at *ArrayType) String() string {
	return fmt.Sprintf("[%d]%s", at.Len, at.ElementType.String())
}

// MapType represents a map type
type MapType struct {
	KeyType   Type
	ValueType Type
}

func (mt *MapType) String() string {
	return fmt.Sprintf("Map[%v]%v", mt.KeyType, mt.ValueType)
}

type TypeConstraint struct {
	Interfaces []Interface
	Types      []Type
}

func (tc *TypeConstraint) String() string {
	return fmt.Sprintf("Constraint(%v, %v)", tc.Interfaces, tc.Types)
}

// GenericType represents a generic type with type parameters.
type GenericType struct {
	Name        string
	TypeParams  []Type
	Constraints map[string]TypeConstraint
	Fields      map[string]Type
}

func (gt *GenericType) String() string {
	params := make([]string, len(gt.TypeParams))
	for i, param := range gt.TypeParams {
		params[i] = param.String()
	}
	return fmt.Sprintf("Generic(%s, %v)", gt.Name, params)
}

// TypeEnv store and manage type variables and their types.
// It acts as a symbol table for type inference, mapping type variable names
// to their inferred or declared types.
type TypeEnv map[string]Type
