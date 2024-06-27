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
	for _, iface := range constraint.Interfaces {
		if !impl(t, iface) {
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

// impl checks if a type t implements the given interface iface.
func impl(t Type, iface Interface) bool {
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
	for name, method := range iface.Methods {
		if tm, ok := t.Methods[name]; !ok || !TypesEqual(tm, method) {
			return false
		}
	}
	return true
}

func structImplsInterface(t *StructType, iface Interface) bool {
	for name, method := range iface.Methods {
		if tm, ok := t.Methods[name]; !ok || !TypesEqual(tm, method) {
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
		t2, ok := t2.(*FunctionType)
		if !ok || len(t1.ParamTypes) != len(t2.ParamTypes) {
			return false
		}
		for i := range t1.ParamTypes {
			if !TypesEqual(t1.ParamTypes[i], t2.ParamTypes[i]) {
				return false
			}
		}
		return TypesEqual(t1.ReturnType, t2.ReturnType)
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
	default:
		return false
	}
}
