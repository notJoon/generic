package generic

import "errors"

var (
	ErrTypeMismatch  = errors.New("type mismatch")
	ErrArityMismatch = errors.New("number of parameters do not match")
	ErrUnknownType    = errors.New("unknown type")
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
//   1. t1' = resolve(t1, env)
//   2. t2' = resolve(t2, env)
//   3. case (t1', t2') of
//      (TypeVariable v, _) → unifyVar(v, t2', env)
//      (TypeConstant c1, TypeConstant c2) → if c1.Name = c2.Name then ok else error
//      (FunctionType f1, FunctionType f2) → 
//        if length(f1.ParamTypes) ≠ length(f2.ParamTypes) then error
//        else forall i. Unify(f1.ParamTypes[i], f2.ParamTypes[i], env) ∧ 
//             Unify(f1.ReturnType, f2.ReturnType, env)
//      (_, _) → error
func Unify(t1, t2 Type, env TypeEnv) error {
	// resolve any type variables to their current bindings
	t1 = resolve(t1, env)
	t2 = resolve(t2, env)
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
		t2, ok := t2.(*FunctionType)
		if !ok {
			return ErrTypeMismatch
		}
		if len(t1.ParamTypes) != len(t2.ParamTypes) {
			return ErrArityMismatch
		}
		for i := range t1.ParamTypes {
			if err := Unify(t1.ParamTypes[i], t2.ParamTypes[i], env); err != nil {
				return err
			}
		}
		return Unify(t1.ReturnType, t2.ReturnType, env)
	}
	return ErrUnknownType
}

// unifyVar unifies a type variable with another type, updating the type environment env.
// This is a helper function for the unify operation, specifically handling the case
// where one of the types a `TypeVariable`.
//
// ## Process
//
// unifyVar(v, t, env) =
//   1. t' = resolve(t, env)
//   2. if v = t' then ok
//   3. else if occurs(v, t', env) then error
//   4. else env[v.Name] = t'
// λv.λt.λenv. let t' = resolve(t, env) in
//   if v = t' then ok
//   else if occurs(v, t', env) then error
//   else env[v.Name] ← t'
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
//   1. t' = resolve(t, env)
//   2. case t' of
//      TypeVariable v' → v = v'
//      FunctionType f → exists p in f.ParamTypes. occurs(v, p, env) ∨ occurs(v, f.ReturnType, env)
//      _ → false
// λv.λt.λenv. let t' = resolve(t, env) in
//   case t' of
//     TypeVariable v' → v = v'
//     FunctionType f → ∃p ∈ f.ParamTypes. occurs(v, p, env) ∨ occurs(v, f.ReturnType, env)
//     _ → false
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
//   1. if t is TypeVariable and t.Name in env then resolve(env[t.Name], env)
//   2. else t
// λt.λenv. case t of
//   TypeVariable v → if v.Name ∈ dom(env) then resolve(env(v.Name), env) else t
//   _ → t
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