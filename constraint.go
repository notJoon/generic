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
	if _, ok := t.(*TypeVariable); ok {
		return true
	}
	// handle built-in constraints (e.g., "any", "comparable", etc.)
	if constraint.BuiltinConstraint != "" {
		return checkBuiltinConstraint(t, constraint.BuiltinConstraint)
	}

	// pointer type is a special case, we need to check the base type
	if ptr, ok := t.(*PointerType); ok {
		for _, allowedType := range constraint.Types {
			if ptrAllowed, ok := allowedType.(*PointerType); ok {
				if TypesEqual(ptr.Base, ptrAllowed.Base) {
					return true
				}
			}
		}
		return checkConstraint(ptr.Base, constraint)
	}

	// check if the type implements all the interfaces in the constraint
	for _, iface := range constraint.Interfaces {
		if !implInterface(t, iface) {
			return false
		}
	}

	// check if the type satisfies the type constraints
	if len(constraint.Types) > 0 {
		for _, allowedType := range constraint.Types {
			if constraint.IsUnderlying {
				if isUnderlyingType(t, allowedType) {
					return true
				}
			} else {
				if TypesEqual(t, allowedType) {
					return true
				}
			}
		}
		return false
	}

	// in here, we have no constraints or all constraints are satisfied
	// thus, the type satisfies the constraint
	return true
}

// implInterface checks if a type t implements the given interface iface.
func implInterface(t Type, iface Interface) bool {
	switch concreteType := t.(type) {
	case *TypeConstant:
		return checkPrimitiveTypeInterface(concreteType.Name, iface)
	case *GenericType:
		// generic type may be different due to the type parameters
		// but for simplicity, we assume it's the same
		return true
	case *FunctionType:
		// function type can't implement an interface
		return false
	case MethodHolder:
		return typeContainsAllMethods(concreteType, iface)
	default:
		return false
	}
}

func checkPrimitiveTypeInterface(tName string, iface Interface) bool {
	// define an interface to implement for each primitive type
	primitiveInterfaces := map[string][]string{
		TypeInt:     {"Stringer", "Printable", "Comparable"},
		TypeString:  {"Printable", "Comparable"},
		TypeFloat64: {"Stringer", "Printable", "Comparable"},
	}

	if interfaces, ok := primitiveInterfaces[tName]; ok {
		for _, i := range interfaces {
			if i == iface.Name {
				return true
			}
		}
	}
	return false
}

func typeContainsAllMethods(t MethodHolder, iface Interface) bool {
	for name := range iface.Methods {
		if _, ok := t.GetMethods()[name]; !ok {
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

// checkBuiltinConstraint checks if a the given type satisfies the specified built-in constraint.
func checkBuiltinConstraint(t Type, constraint string) bool {
	switch constraint {
	case ConstraintAny:
		return true
	case ConstraintComparable:
		return isComparable(t)
	case ConstraintOrdered:
		return isOrdered(t)
	case ConstraintComplex:
		return isComplex(t)
	case ConstraintFloat:
		return isFloat(t)
	case ConstraintInteger:
		return isInteger(t)
	case ConstraintSigned:
		return isSigned(t)
	case ConstraintUnsigned:
		return isUnsigned(t)
	default:
		return false
	}
}

// isComparable determines if the given type is comparable.
func isComparable(t Type) bool {
	switch t := t.(type) {
	case *TypeConstant:
		// premitive types are comparable
		return t.Name == TypeBool || isNumeric(t) || t.Name == TypeString
	case *PointerType:
		return true // all pointer types are comparable
	case *InterfaceType:
		return true // all interface types are comparable
	case *StructType:
		// every field of the struct should be comparable
		for _, field := range t.Fields {
			if !isComparable(field) {
				return false
			}
		}
		return true
	case *ArrayType:
		// array is comparable if element type is comparable and length is the same
		return isComparable(t.ElementType)
	default:
		return false
	}
}

var (
	signedIntegers = map[string]bool{
		TypeInt: true, TypeInt8: true, TypeInt16: true, TypeInt32: true, TypeInt64: true,
	}
	unsignedIntegers = map[string]bool{
		TypeUint: true, TypeUint8: true, TypeUint16: true, TypeUint32: true, TypeUint64: true, TypeUintptr: true,
	}
	floats = map[string]bool{
		TypeFloat32: true, TypeFloat64: true,
	}
	complexes = map[string]bool{
		TypeComplex64: true, TypeComplex128: true,
	}
	orderedTypes = map[string]bool{
		TypeString: true,
	}
)

// isOrdered checks if the given type is ordered (can be used with comparison operators)
// (e.g., <, <=, >, >=)
func isOrdered(t Type) bool {
	if tc, ok := t.(*TypeConstant); ok {
		return signedIntegers[tc.Name] ||
			unsignedIntegers[tc.Name] ||
			floats[tc.Name] ||
			orderedTypes[tc.Name]
	}
	return false
}

// isComplex checks if the given type is a complex number type.
func isComplex(t Type) bool {
	if tc, ok := t.(*TypeConstant); ok {
		return complexes[tc.Name]
	}
	return false
}

// isFloat checks if the given type is a floating-point number type.
func isFloat(t Type) bool {
	if tc, ok := t.(*TypeConstant); ok {
		return floats[tc.Name]
	}
	return false
}

// isInteger checks if the given type is an integer type.
func isInteger(t Type) bool {
	if tc, ok := t.(*TypeConstant); ok {
		return signedIntegers[tc.Name] || unsignedIntegers[tc.Name]
	}
	return false
}

// isSigned checks if the given type is a signed integer type.
func isSigned(t Type) bool {
	if tc, ok := t.(*TypeConstant); ok {
		return signedIntegers[tc.Name]
	}
	return false
}

// isUnsigned checks if the given type is an unsigned integer type.
func isUnsigned(t Type) bool {
	if tc, ok := t.(*TypeConstant); ok {
		return unsignedIntegers[tc.Name]
	}
	return false
}

// isNumeric checks if the given type is a numeric type.
func isNumeric(t Type) bool {
	return isInteger(t) || isFloat(t) || isComplex(t)
}

// isUnderlyingType checks if the given type has the specified underlying type.
func isUnderlyingType(t Type, underlyingType Type) bool {
	for {
		if alias, ok := t.(*TypeAlias); ok {
			t = alias.AliasedTo
		} else {
			break
		}
	}

	switch concrete := t.(type) {
	case *TypeConstant:
		underlyingConst, ok := underlyingType.(*TypeConstant)
		return ok && concrete.Name == underlyingConst.Name

	case *SliceType:
		underlyingSlice, ok := underlyingType.(*SliceType)
		return ok && isUnderlyingType(concrete.ElementType, underlyingSlice.ElementType)

	case *MapType:
		underlyingMap, ok := underlyingType.(*MapType)
		return ok &&
			isUnderlyingType(concrete.KeyType, underlyingMap.KeyType) &&
			isUnderlyingType(concrete.ValueType, underlyingMap.ValueType)

	case *StructType:
		underlyingStruct, ok := underlyingType.(*StructType)
		if !ok || len(concrete.Fields) != len(underlyingStruct.Fields) {
			return false
		}
		for name, field := range concrete.Fields {
			underlyingField, ok := underlyingStruct.Fields[name]
			if !ok || !isUnderlyingType(field, underlyingField) {
				return false
			}
		}
		return true

	case *FunctionType:
		underlyingFunc, ok := underlyingType.(*FunctionType)
		if !ok || len(concrete.ParamTypes) != len(underlyingFunc.ParamTypes) {
			return false
		}
		for i, param := range concrete.ParamTypes {
			if !isUnderlyingType(param, underlyingFunc.ParamTypes[i]) {
				return false
			}
		}
		return isUnderlyingType(concrete.ReturnType, underlyingFunc.ReturnType)

	default:
		return false
	}
}
