package generic

import (
	"fmt"
	"go/ast"
)

type InferenceContext struct {
	ExpectedType  Type
	IsAssignment  bool
	IsFunctionArg bool
	IsReturnValue bool
}

func NewInferenceContext(options ...func(*InferenceContext)) *InferenceContext {
	ctx := &InferenceContext{}
	for _, opt := range options {
		opt(ctx)
	}
	return ctx
}

func WithExpectedType(t Type) func(*InferenceContext) {
	return func(ctx *InferenceContext) {
		ctx.ExpectedType = t
	}
}

func WithAssignment() func(*InferenceContext) {
	return func(ctx *InferenceContext) {
		ctx.IsAssignment = true
	}
}

func WithFunctionArg() func(*InferenceContext) {
	return func(ctx *InferenceContext) {
		ctx.IsFunctionArg = true
	}
}

func WithReturnValue() func(*InferenceContext) {
	return func(ctx *InferenceContext) {
		ctx.IsReturnValue = true
	}
}

// InferType infers the type of an expression.
func checkInterfaceCompatibility(iface, expected *InterfaceType) error {
	for name, method := range expected.Methods {
		ifaceMethod, ok := iface.Methods[name]
		if !ok {
			return fmt.Errorf("missing method %s", name)
		}
		if !MethodsEqual(method, ifaceMethod) {
			return fmt.Errorf("method %s has incompatible signature", name)
		}
	}
	return nil
}

// checkFunctionCompatibility checks if two function types are compatible.
func checkFunctionCompatibility(func1, func2 *FunctionType) error {
	if len(func1.ParamTypes) != len(func2.ParamTypes) {
		return fmt.Errorf("expected %d parameters, got %d", len(func2.ParamTypes), len(func1.ParamTypes))
	}

	for i, param1 := range func1.ParamTypes {
		if !TypesEqual(param1, func2.ParamTypes[i]) {
			return fmt.Errorf("parameter type mismatch at position %d", i)
		}
	}
	if !TypesEqual(func1.ReturnType, func2.ReturnType) {
		return fmt.Errorf("return type mismatch")
	}
	if func1.IsVariadic != func2.IsVariadic {
		return fmt.Errorf("variadic parameter mismatch")
	}
	return nil
}

// checkTupleCompatibility checks if two tuple types are compatible.
func checkTupleCompatibility(tuple1, tuple2 *TupleType) error {
	if len(tuple1.Types) != len(tuple2.Types) {
		return fmt.Errorf("tuple length mismatch. expected %d elements, got %d", len(tuple2.Types), len(tuple1.Types))
	}
	for i, type1 := range tuple1.Types {
		if !TypesEqual(type1, tuple2.Types[i]) {
			return fmt.Errorf("element type mismatch at position %d", i)
		}
	}
	return nil
}

// InferTypeArguments infers the type arguments for a generic function based on the provided arguments.
func InferTypeArguments(genericFunc *GenericType, args []ast.Expr, env TypeEnv) ([]Type, error) {
	inferred := make([]Type, len(genericFunc.TypeParams))
	for i, param := range genericFunc.TypeParams {
		constraint := genericFunc.Constraints[param.(*TypeVariable).Name]
		for _, arg := range args {
			var (
				argType Type
				err error
			)

			// special handling for function literals
			if funcLit, ok := arg.(*ast.FuncLit); ok {
				// TODO: fix this function to handle function literals properly
				argType, err = inferFunctionLiteralType(funcLit, env)
			} else {
				argType, err = InferType(arg, env, nil)
			}

			if err != nil {
				return nil, err
			}
			if checkConstraint(argType, constraint) {
				inferred[i] = argType
				break
			}
		}
		if inferred[i] == nil {
			return nil, fmt.Errorf("could not infer type for parameter %s", param)
		}
	}
	return inferred, nil
}

// FIXME
func inferFunctionLiteralType(funcLit *ast.FuncLit, env TypeEnv) (Type, error) {
	ctx := NewInferenceContext(WithFunctionArg())

	// infer the parameter types
	pTypes, err := inferParams(funcLit.Type.Results, env, ctx)
	if err != nil {
		return nil, fmt.Errorf("error inferring parameter types: %v", err)
	}

	resultCtx := NewInferenceContext(WithReturnValue())
	resultType, err := inferResult(funcLit.Type.Results, env, resultCtx)
	if err != nil {
		return nil, fmt.Errorf("error inferring return type: %v", err)
	}

	if typeConst, ok := resultType.(*TypeConstant); ok && typeConst.Name == "void" {
		resultType = nil
	}

	return &FunctionType{
		ParamTypes:  pTypes,
		ReturnType:  resultType,
	}, nil
}
