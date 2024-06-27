package generic

// Type represents any type in the type system.
// It serves as the base interface for all types in the generic type system.
type Type interface{}

// TypeVariable represents a type variable with a name.
// In generic programming, type variables are placeholders for types
// that will be specified later, allowing for polymorphic code.
type TypeVariable struct {
	Name string
}

// TypeConstant represents a constant type with a name.
// These are concrete types like `int`, `string`, or user-defined types.
type TypeConstant struct {
	Name string
}

// FunctionType represents a function type with parameter types and return type.
// It describes the signature of a function in the type system.
type FunctionType struct {
	ParamTypes []Type
	ReturnType Type
}

type Interface struct {
	Name    string
	Methods map[string]Type
}

type InterfaceType struct {
	Name    string
	Methods map[string]Type
}

type StructType struct {
	Name    string
	Fields  map[string]Type
	Methods map[string]Type
}

// SliceType represents a slice type
type SliceType struct {
	ElementType Type
}

type TypeConstraint struct {
	Interfaces []Interface
	Types      []Type
}

// GenericType represents a generic type with type parameters.
type GenericType struct {
	Name        string
	TypeParams  []Type
	Constraints map[string]TypeConstraint
}

// TypeEnv store and manage type variables and their types.
// It acts as a symbol table for type inference, mapping type variable names
// to their inferred or declared types.
type TypeEnv map[string]Type
