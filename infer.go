package generic

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
)

var (
	ErrUnknownIdent           = errors.New("unknown identifier")
	ErrNotAFunction           = errors.New("not a function")
	ErrUnknownExpr            = errors.New("unknown expression")
	ErrNotAGenericType        = errors.New("not a generic type")
	ErrTypeParamsNotMatch     = errors.New("type parameters do not match")
	ErrConstraintNotSatisfied = errors.New("type does not satisfy constraint")
)

// InferType infers the type of an AST expression in the given type environment.
//
// ## Process
//
// InferType(expr, env) =
//  1. case expr of
//     Ident i → if i.Name ∈ dom(env) then env[i.Name] else error
//     CallExpr c →
//     a. funcType = InferType(c.Fun, env)
//     b. if not isFunctionType(funcType) then error
//     c. if len(funcType.ParamTypes) ≠ len(c.Args) then error
//     d. for i = 0 to len(c.Args) - 1 do
//     argType = InferType(c.Args[i], env)
//     Unify(funcType.ParamTypes[i], argType, env)
//     e. return funcType.ReturnType
//     _ → error
//
// λexpr.λenv. case expr of
//
//	Ident i → if i.Name ∈ dom(env) then env(i.Name) else error
//	CallExpr c → let funcType = InferType(c.Fun, env) in
//	             if not isFunctionType(funcType) then error
//	             else if length(funcType.ParamTypes) ≠ length(c.Args) then error
//	             else let _ = map (λ(param, arg). Unify(param, InferType(arg, env), env))
//	                             (zip funcType.ParamTypes c.Args)
//	                  in funcType.ReturnType
//	_ → error
func InferType(expr ast.Expr, env TypeEnv) (Type, error) {
	switch expr := expr.(type) {
	case *ast.Ident:
		if typ, ok := env[expr.Name]; ok {
			return typ, nil
		}
		return nil, ErrUnknownIdent
	case *ast.CallExpr:
		funcTyp, err := InferType(expr.Fun, env)
		if err != nil {
			return nil, err
		}
		funcTypeCast, ok := funcTyp.(*FunctionType)
		if !ok {
			return nil, ErrNotAFunction
		}
		if len(funcTypeCast.ParamTypes) != len(expr.Args) {
			return nil, ErrArityMismatch
		}
		for i, arg := range expr.Args {
			argType, err := InferType(arg, env)
			if err != nil {
				return nil, err
			}
			if err := Unify(funcTypeCast.ParamTypes[i], argType, env); err != nil {
				return nil, err
			}
		}
		return funcTypeCast.ReturnType, nil
	case *ast.IndexExpr:
		return inferGenericType(expr.X, []ast.Expr{expr.Index}, env)
	case *ast.IndexListExpr:
		return inferGenericType(expr.X, expr.Indices, env)
	case *ast.CompositeLit:
		if mapType, ok := expr.Type.(*ast.MapType); ok {
			kt, err := InferType(mapType.Key, env)
			if err != nil {
				return nil, err
			}
			vt, err := InferType(mapType.Value, env)
			if err != nil {
				return nil, err
			}

			// check the element types of the map literal
			for _, elt := range expr.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					k, err := InferType(kv.Key, env)
					if err != nil {
						return nil, err
					}
					v, err := InferType(kv.Value, env)
					if err != nil {
						return nil, err
					}
					if err := Unify(kt, k, env); err != nil {
						return nil, fmt.Errorf("map key type mismatch: %v", err)
					}
					if err := Unify(vt, v, env); err != nil {
						return nil, fmt.Errorf("map value type mismatch: %v", err)
					}
				}
			}
			return &MapType{KeyType: kt, ValueType: vt}, nil
		}
		if arrayType, ok := expr.Type.(*ast.ArrayType); ok && arrayType.Len == nil {
			// infer the element type
			elementType, err := InferType(arrayType.Elt, env)
			if err != nil {
				return nil, err
			}

			if len(expr.Elts) == 0 {
				// empty slice literal, use the specified element type
				return &SliceType{ElementType: elementType}, nil
			}

			// check the types of the remaining elements and ensure they are consistent
			for _, elt := range expr.Elts {
				eltType, err := InferType(elt, env)
				if err != nil {
					return nil, err
				}
				if err := Unify(elementType, eltType, env); err != nil {
					return nil, errors.New("inconsistent element types in slice literal")
				}
			}
			return &SliceType{ElementType: elementType}, nil
		}
		// struct and generic types
		if ident, ok := expr.Type.(*ast.Ident); ok {
			structType, err := resolveTypeByName(ident.Name, env)
			if err != nil {
				return nil, err
			}

			switch st := structType.(type) {
			case *StructType:
				// normal struct type
				for _, elt := range expr.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						fieldName := kv.Key.(*ast.Ident).Name
						fieldType, ok := st.Fields[fieldName]
						if !ok {
							return nil, fmt.Errorf("unknown field: %s in struct %s", fieldName, st.Name)
						}
						vt, err := InferType(kv.Value, env)
						if err != nil {
							return nil, err
						}
						if err := Unify(fieldType, vt, env); err != nil {
							return nil, fmt.Errorf("type mismatch for field %s: %v", fieldName, err)
						}
					}
				}
				return st, nil
			case *GenericType:
				// generic struct type
				return st, nil
			}
		}
		// handle generic type instantiation
		if indexExpr, ok := expr.Type.(*ast.IndexExpr); ok {
			genericType, err := resolveTypeByName(indexExpr.X.(*ast.Ident).Name, env)
			if err != nil {
				return nil, err
			}
			gt, ok := genericType.(*GenericType)
			if !ok {
				return nil, fmt.Errorf("not a generic type: %v", genericType)
			}

			typeArg, err := InferType(indexExpr.Index, env)
			if err != nil {
				return nil, err
			}

			// check if the type argument satisfies the constraint
			if constraint, ok := gt.Constraints[gt.TypeParams[0].(*TypeVariable).Name]; ok {
				if !checkConstraint(typeArg, constraint) {
					return nil, fmt.Errorf("type argument does not satisfy constraint")
				}
			}

			instantiatedType := &GenericType{
				Name:       gt.Name,
				TypeParams: []Type{typeArg},
				Fields: make(map[string]Type),
			}

			// type check the each struct fields
			for fname, ftype := range gt.Fields {
				instantiatedFieldType := substituteTypeVar(ftype, gt.TypeParams[0].(*TypeVariable), typeArg)
				instantiatedType.Fields[fname] = instantiatedFieldType
			}

			// check struct literal's field values
			for _, elt := range expr.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					fname := kv.Key.(*ast.Ident).Name
					fType, ok := instantiatedType.Fields[fname]
					if !ok {
						return nil, fmt.Errorf("unknown field %s in generic type %s", fname, gt.Name)
					}
					vt, err := InferType(kv.Value, env)
					if err != nil {
						return nil, err
					}
					if err := Unify(fType, vt, env); err != nil {
						return nil, fmt.Errorf("type mismatch for field %s: %v", fname, err)
					}
				}
			}
			return instantiatedType, nil
		}
	case *ast.BasicLit:
		switch expr.Kind {
		case token.INT:
			return &TypeConstant{Name: "int"}, nil
		case token.FLOAT:
			return &TypeConstant{Name: "float64"}, nil
		case token.STRING:
			return &TypeConstant{Name: "string"}, nil
		case token.CHAR:
			return &TypeConstant{Name: "rune"}, nil
		default:
			return nil, fmt.Errorf("unknown basic literal kind: %v", expr.Kind)
		}
	}
	return nil, fmt.Errorf("unknown expression: %T", expr)
}

// inferGenericType infers the type of a generic expression with its type parameters,
// including nested generic types.
//
// ## Process
//
// inferGenericType(x, indices, env) =
//  1. baseType = InferType(x, env)
//  2. if not isGenericType(baseType) then error
//  3. typeParams = []
//  4. for each index in indices:
//     paramType = InferType(index, env)
//     if isGenericType(paramType) then
//     paramType = inferGenericType(paramType.Name, paramType.TypeParameters, env)
//     append paramType to typeParams
//  5. if len(typeParams) ≠ len(baseType.TypeParameters) then error
//  6. return new GenericType with baseType.Name and typeParams
//
// λx.λindices.λenv.
//
//	let baseType = InferType(x, env) in
//	if not isGenericType(baseType) then error else
//	let typeParams = map (λindex.
//	  let paramType = InferType(index, env) in
//	  if isGenericType(paramType) then
//	    inferGenericType(paramType.Name, paramType.TypeParameters, env)
//	  else paramType
//	) indices in
//	if length(typeParams) ≠ length(baseType.TypeParameters) then error else
//	GenericType { Name: baseType.Name, TypeParameters: typeParams }
func inferGenericType(x ast.Expr, indices []ast.Expr, env TypeEnv) (Type, error) {
	baseType, err := InferType(x, env)
	if err != nil {
		return nil, err
	}
	genericType, ok := baseType.(*GenericType)
	if !ok {
		return nil, ErrNotAGenericType
	}

	if len(indices) != len(genericType.TypeParams) {
		return nil, ErrTypeParamsNotMatch
	}

	var typeParams []Type
	for i, index := range indices {
		paramType, err := InferType(index, env)
		if err != nil {
			return nil, err
		}
		cst, ok := genericType.Constraints[genericType.TypeParams[i].(*TypeVariable).Name]
		if ok && !checkConstraint(paramType, cst) {
			return nil, ErrConstraintNotSatisfied
		}
		typeParams = append(typeParams, paramType)
	}

	if len(typeParams) != len(genericType.TypeParams) {
		return nil, ErrTypeParamsNotMatch
	}

	return &GenericType{
		Name:       genericType.Name,
		TypeParams: typeParams,
	}, nil
}

func substituteTypeVar(t Type, tv *TypeVariable, replacement Type) Type {
	switch t := t.(type) {
	case *TypeVariable:
		if t.Name == tv.Name {
			return replacement
		}
	case *GenericType:
		newParams := make([]Type, len(t.TypeParams))
		for i, param := range t.TypeParams {
			newParams[i] = substituteTypeVar(param, tv, replacement)
		}
		return &GenericType{
			Name:       t.Name,
			TypeParams: newParams,
		}
	}
	return t
}
