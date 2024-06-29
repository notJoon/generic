package generic

import (
	"errors"
	"fmt"
)

var (
	ErrTypeMismatch      = errors.New("type mismatch")
	ErrArityMismatch     = errors.New("number of parameters do not match")
	ErrUnknownType       = errors.New("unknown type")
	ErrCircularReference = errors.New("circular reference detected")
)

// Unify attempts to unify two types t1 and t2, updating the type environment env.
// Unification is a key operation in type inference, where it tries to make two types
// equivalent by finding a substitution that makes them equal.
//
// It returns an error if the types cannot be unified.
//
// ## Process
//
// Unify(t1, t2, env) =
//  1. t1' = resolve(t1, env)
//  2. t2' = resolve(t2, env)
//  3. case (t1', t2') of
//     (TypeVariable v, _) → unifyVar(v, t2', env)
//     (TypeConstant c1, TypeConstant c2) → if c1.Name = c2.Name then ok else error
//     (FunctionType f1, FunctionType f2) →
//     if length(f1.ParamTypes) ≠ length(f2.ParamTypes) then error
//     else forall i. Unify(f1.ParamTypes[i], f2.ParamTypes[i], env) ∧
//     Unify(f1.ReturnType, f2.ReturnType, env)
//     (_, _) → error
func Unify(t1, t2 Type, env TypeEnv) error {
	// resolve any type variables to their current bindings
	t1 = resolve(t1, env)
	t2 = resolve(t2, env)

	if isInterfaceAny(t1) || isInterfaceAny(t2) {
		return nil
	}

	switch t1 := t1.(type) {
	case *TypeVariable:
		return unifyVar(t1, t2, env)
	case *TypeConstant:
		t2, ok := t2.(*TypeConstant)
		if !ok || t1.Name != t2.Name {
			return ErrTypeMismatch
		}
		return nil
	case *FunctionType:
		t2Func, ok := t2.(*FunctionType)
		if !ok {
			return ErrTypeMismatch
		}
		if t1.IsVariadic != t2Func.IsVariadic {
			return ErrTypeMismatch
		}
		if len(t1.ParamTypes) != len(t2Func.ParamTypes) {
			return ErrArityMismatch
		}
		for i := range t1.ParamTypes {
			if err := Unify(t1.ParamTypes[i], t2Func.ParamTypes[i], env); err != nil {
				return err
			}
		}
		return Unify(t1.ReturnType, t2Func.ReturnType, env)
	case *TupleType:
		t2Tuple, ok := t2.(*TupleType)
		if !ok {
			return ErrTypeMismatch
		}
		if len(t1.Types) != len(t2Tuple.Types) {
			return ErrArityMismatch
		}
		for i := range t1.Types {
			if err := Unify(t1.Types[i], t2Tuple.Types[i], env); err != nil {
				return err
			}
		}
		return nil
	case *SliceType:
		if t2Slice, ok := t2.(*SliceType); ok {
			return Unify(t1.ElementType, t2Slice.ElementType, env)
		}
	case *GenericType:
		t2Generic, ok := t2.(*GenericType)
		if !ok || t1.Name != t2Generic.Name || len(t1.TypeParams) != len(t2Generic.TypeParams) {
			return ErrTypeMismatch
		}
		for i := range t1.TypeParams {
			if err := Unify(t1.TypeParams[i], t2Generic.TypeParams[i], env); err != nil {
				return err
			}
		}
		return nil
	case *InterfaceType:
		if t1.IsEmpty || t2.(*InterfaceType).IsEmpty {
			return nil
		}
		t2Interface, ok := t2.(*InterfaceType)
		if !ok || t1.Name != t2Interface.Name {
			return ErrTypeMismatch
		}
		for name, method1 := range t1.Methods {
			method2, ok := t2Interface.Methods[name]
			if !ok {
				return ErrTypeMismatch
			}
			// unify method signatures
			if err := unifyMethod(method1, method2, env); err != nil {
				return err
			}
		}
		for _, embedded := range t1.Embedded {
			if err := Unify(embedded, t2, env); err != nil {
				return err
			}
		}
		return nil
	case *MapType:
		t2Map, ok := t2.(*MapType)
		if !ok {
			return ErrTypeMismatch
		}
		if err := Unify(t1.KeyType, t2Map.KeyType, env); err != nil {
			return err
		}
		return Unify(t1.ValueType, t2Map.ValueType, env)
	case *PointerType:
		t2Ptr, ok := t2.(*PointerType)
		if !ok {
			return ErrTypeMismatch
		}
		return Unify(t1.Base, t2Ptr.Base, env)
	}
	return ErrUnknownType
}

// unifyMethod unifies two method signatures, ensuring that they have the same name,
// pointer type, and matching parameter and result types.
func unifyMethod(m1, m2 Method, env TypeEnv) error {
	if m1.Name != m2.Name || m1.IsPointer != m2.IsPointer {
		return ErrTypeMismatch
	}
	if len(m1.Params) != len(m2.Params) || len(m1.Results) != len(m2.Results) {
		return ErrArityMismatch
	}

	for i := range m1.Params {
		if err := Unify(m1.Params[i], m2.Params[i], env); err != nil {
			return err
		}
	}

	return nil
}

// unifyVar unifies a type variable with another type, updating the type environment env.
// This is a helper function for the unify operation, specifically handling the case
// where one of the types a `TypeVariable`.
//
// ## Process
//
// unifyVar(v, t, env) =
//  1. t' = resolve(t, env)
//  2. if v = t' then ok
//  3. else if occurs(v, t', env) then error
//  4. else env[v.Name] = t'
//
// λv.λt.λenv. let t' = resolve(t, env) in
//
//	if v = t' then ok
//	else if occurs(v, t', env) then error
//	else env[v.Name] ← t'
func unifyVar(v *TypeVariable, t Type, env TypeEnv) error {
	t = resolve(t, env)
	if v == t {
		return nil
	}
	if resolved, ok := env[v.Name]; ok {
		return Unify(resolved, t, env)
	}
	if tv, ok := t.(*TypeVariable); ok {
		if resolved, ok := env[tv.Name]; ok {
			return unifyVar(v, resolved, env)
		}
	}
	if occurs(v, t, env) {
		return ErrCircularReference
	}
	env[v.Name] = t
	return nil
}

// occurs checks if the type variable v occurs in the type t.
// This is used to prevent circular references in type unification.
// By adding this, we can detect and prevent following circular reference:
//
// ## Process
//
// occurs(v, t, env) =
//  1. t' = resolve(t, env)
//  2. case t' of
//     TypeVariable v' → v = v'
//     FunctionType f → exists p in f.ParamTypes. occurs(v, p, env) ∨ occurs(v, f.ReturnType, env)
//     _ → false
//
// λv.λt.λenv. let t' = resolve(t, env) in
//
//	case t' of
//	  TypeVariable v' → v = v'
//	  FunctionType f → ∃p ∈ f.ParamTypes. occurs(v, p, env) ∨ occurs(v, f.ReturnType, env)
//	  _ → false
func occurs(v *TypeVariable, t Type, env TypeEnv) bool {
	t = resolve(t, env)
	switch t := t.(type) {
	case *TypeVariable:
		if v == t {
			return true
		}
		if resolved, ok := env[t.Name]; ok {
			return occurs(v, resolved, env)
		}
		return false
	case *FunctionType:
		for _, paramType := range t.ParamTypes {
			if occurs(v, paramType, env) {
				return true
			}
		}
		return occurs(v, t.ReturnType, env)
	default:
		return false
	}
}

// resolve fully resolves a type by following type variable bindings in the environment.
//
// ## Process
//
// resolve(t, env) =
//  1. if t is TypeVariable and t.Name in env then resolve(env[t.Name], env)
//  2. else t
//
// λt.λenv. case t of
//
//	TypeVariable v → if v.Name ∈ dom(env) then resolve(env(v.Name), env) else t
//	_ → t
func resolve(t Type, env TypeEnv) Type {
	for {
		if tv, ok := t.(*TypeVariable); ok {
			if resolved, exists := env[tv.Name]; exists {
				t = resolved
			} else {
				return t
			}
		} else {
			return t
		}
	}
}

func resolveTypeByName(name string, env TypeEnv) (Type, error) {
	if t, ok := env[name]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("unknown type: %s", name)
}

func isInterfaceAny(t Type) bool {
	if it, ok := t.(*InterfaceType); ok {
		return it.Name == "interface{}"
	}
	return false
}
