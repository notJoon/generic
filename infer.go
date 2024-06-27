package generic

import (
	"errors"
	"go/ast"
)

var (
	ErrUnknownIdent    = errors.New("unknown identifier")
	ErrNotAFunction    = errors.New("not a function")
	ErrUnknownExpr     = errors.New("unknown expression")
	ErrNotAGenericType = errors.New("not a generic type")
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
	case *ast.IndexExpr: // for generic type instantiation
		baseType, err := InferType(expr.X, env)
		if err != nil {
			return nil, err
		}
		genericType, ok := baseType.(*GenericType)
		if !ok {
			return nil, ErrNotAGenericType
		}
		indexType, err := InferType(expr.Index, env)
		if err != nil {
			return nil, err
		}
		// just for simplicity, only using 1st type parameter
		return &GenericType{
			Name:       genericType.Name,
			TypeParams: []Type{indexType},
		}, nil
	default:
		return nil, ErrUnknownExpr
	}
}
