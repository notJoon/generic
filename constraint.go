package generic

// checkConstraint checks if a type t satisfies the given `TypeConstraint`.
//
// ## Process
//
// checkConstraint(t, constraint) =
//  1. For each interface i in constraint.Interfaces:
//     If not implementsInterface(t, i), return false
//  2. If constraint.Types is not empty:
//     For each allowedType in constraint.Types:
//     If TypesEqual(t, allowedType), return true
//     Return false
//  3. Return true
//
// λt.λconstraint.
//
//	(∀i ∈ constraint.Interfaces. implementsInterface(t, i)) ∧
//	(constraint.Types ≠ ∅ ⇒ ∃type ∈ constraint.Types. TypesEqual(t, type))
func checkConstraint(t Type, constraint TypeConstraint) bool {
	if ptr, ok := t.(*PointerType); ok {
		// if pointer type, check constraints against base type
		// return checkConstraint(ptr.Base, constraint)
		for _, allowedType := range constraint.Types {
			if ptrAllowed, ok := allowedType.(*PointerType); ok {
				if TypesEqual(ptr.Base, ptrAllowed.Base) {
					return true
				}
			}
		}
	}
	for _, iface := range constraint.Interfaces {
		if !implInterface(t, iface) {
			return false
		}
	}
	if len(constraint.Types) > 0 {
		for _, allowed := range constraint.Types {
			if TypesEqual(t, allowed) {
				return true
			}
		}
		return false
	}
	return true
}

// implInterface checks if a type t implements the given interface iface.
func implInterface(t Type, iface Interface) bool {
	switch concreteType := t.(type) {
	case *TypeConstant:
		// must handling for each primitive type for production level.
		return true
	case *GenericType:
		// generic type may be different due to the type parameters
		// but for simplicity, we assume it's the same
		return true
	case *FunctionType:
		// function type can't implement an interface
		return false
	case *InterfaceType:
		// check if the interface contains all methods of the type
		return interfaceContainsAll(concreteType, iface)
	case *StructType:
		// check each method of the interface is implemented by the struct
		return structImplsInterface(concreteType, iface)
	default:
		return false
	}
}

func interfaceContainsAll(t *InterfaceType, iface Interface) bool {
	for name := range iface.Methods {
		if _, ok := t.Methods[name]; !ok {
			return false
		}
	}
	return true
}

func structImplsInterface(t *StructType, iface Interface) bool {
	for name := range iface.Methods {
		if _, ok := t.Methods[name]; !ok {
			return false
		}
	}
	return true
}

// TypesEqual is a helper function to compare two Types
func TypesEqual(t1, t2 Type) bool {
	if t1 == nil || t2 == nil {
		return t1 == t2
	}
	switch t1 := t1.(type) {
	case *TypeConstant:
		t2, ok := t2.(*TypeConstant)
		return ok && t1.Name == t2.Name
	case *TypeVariable:
		t2, ok := t2.(*TypeVariable)
		return ok && t1.Name == t2.Name
	case *FunctionType:
		t2Func, ok := t2.(*FunctionType)
		if !ok {
			return false
		}
		if len(t1.ParamTypes) != len(t2Func.ParamTypes) {
			return false
		}
		if t1.IsVariadic != t2Func.IsVariadic {
			return false
		}
		for i := range t1.ParamTypes {
			if !TypesEqual(t1.ParamTypes[i], t2Func.ParamTypes[i]) {
				return false
			}
		}
		return TypesEqual(t1.ReturnType, t2Func.ReturnType)
	case *TupleType:
		t2Tuple, ok := t2.(*TupleType)
		if !ok || len(t1.Types) != len(t2Tuple.Types) {
			return false
		}
		for i := range t1.Types {
			if !TypesEqual(t1.Types[i], t2Tuple.Types[i]) {
				return false
			}
		}
		return true
	case *GenericType:
		t2, ok := t2.(*GenericType)
		if !ok || t1.Name != t2.Name || len(t1.TypeParams) != len(t2.TypeParams) {
			return false
		}
		for i := range t1.TypeParams {
			if !TypesEqual(t1.TypeParams[i], t2.TypeParams[i]) {
				return false
			}
		}
		return true
	case *SliceType:
		t2, ok := t2.(*SliceType)
		return ok && TypesEqual(t1.ElementType, t2.ElementType)
	case *ArrayType:
		t2, ok := t2.(*ArrayType)
		return ok && t1.Len == t2.Len && TypesEqual(t1.ElementType, t2.ElementType)
	case *InterfaceType:
		t2, ok := t2.(*InterfaceType)
		if !ok {
			return false
		}
		if t1.IsEmpty && t2.IsEmpty {
			return true
		}
		for name := range t1.Methods {
			if _, ok := t2.Methods[name]; !ok {
				return false
			}
		}
		return true
	case *StructType:
		t2, ok := t2.(*StructType)
		if !ok || t1.Name != t2.Name {
			return false
		}
		if len(t1.Fields) != len(t2.Fields) || len(t1.Methods) != len(t2.Methods) {
			return false
		}
		for name, fld1 := range t1.Fields {
			fld2, ok := t2.Fields[name]
			if !ok || !TypesEqual(fld1, fld2) {
				return false
			}
		}
		for name, m1 := range t1.Methods {
			m2, ok := t2.Methods[name]
			if !ok || !MethodsEqual(m1, m2) {
				return false
			}
		}
		return true
	case *MapType:
		t2, ok := t2.(*MapType)
		return ok && TypesEqual(t1.KeyType, t2.KeyType) && TypesEqual(t1.ValueType, t2.ValueType)
	case *PointerType:
		t2, ok := t2.(*PointerType)
		return ok && TypesEqual(t1.Base, t2.Base)
	case *TypeAlias:
		t2, ok := t2.(*TypeAlias)
		return ok && t1.Name == t2.Name && TypesEqual(t1.AliasedTo, t2.AliasedTo)
	default:
		return false
	}
}

// MethodsEqual compares two Method types for equality.
func MethodsEqual(m1, m2 Method) bool {
	if m1.Name != m2.Name || m1.IsPointer != m2.IsPointer {
		return false
	}
	if len(m1.Params) != len(m2.Params) || len(m1.Results) != len(m2.Results) {
		return false
	}
	for i := range m1.Params {
		if !TypesEqual(m1.Params[i], m2.Params[i]) {
			return false
		}
	}
	for i := range m1.Results {
		if !TypesEqual(m1.Results[i], m2.Results[i]) {
			return false
		}
	}
	return true
}
