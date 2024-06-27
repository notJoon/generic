package generic

import (
	"errors"
	"go/ast"
)

var (
	ErrUnknownIdent       = errors.New("unknown identifier")
	ErrNotAFunction       = errors.New("not a function")
	ErrUnknownExpr        = errors.New("unknown expression")
	ErrNotAGenericType    = errors.New("not a generic type")
	ErrTypeParamsNotMatch = errors.New("type parameters do not match")
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
	default:
		return nil, ErrUnknownExpr
	}
}

// inferGenericType infers the type of a generic expression with its type parameters.
// It handles both single type parameter (IndexExpr) and multiple type parameters (IndexListExpr).
//
// ## Process
//
// inferGenericType(x, indices, env) =
//  1. baseType = InferType(x, env)
//  2. if not isGenericType(baseType) then error
//  3. typeParams = []
//  4. for each index in indices:
//     paramType = InferType(index, env)
//     append paramType to typeParams
//  5. if len(typeParams) ≠ len(baseType.TypeParameters) then error
//  6. return new GenericType with baseType.Name and typeParams
//
// λx.λindices.λenv.
//
//	let baseType = InferType(x, env) in
//	if not isGenericType(baseType) then error else
//	let typeParams = map (λindex. InferType(index, env)) indices in
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

	var typeParams []Type
	for _, index := range indices {
		paramType, err := InferType(index, env)
		if err != nil {
			return nil, err
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
