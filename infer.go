package generic

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
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
func InferType(node interface{}, env TypeEnv, ctx *InferenceContext) (Type, error) {
	if ctx == nil {
		ctx = NewInferenceContext()
	}

	// [2024.06.24 @notJoon] Since the `ast.AssignStmt` and `ast.ReturnStmt` are dynamically typed,
	// we need to change the `InferType` function's parameter to `interface{}`.
	//
	// By applying this, we can handle the all types of the `ast.Expr` and `ast.Stmt`.
	switch expr := node.(type) {
	case *ast.Ident:
		if typ, ok := env[expr.Name]; ok {
			if alias, ok := typ.(*TypeAlias); ok {
				return alias.AliasedTo, nil
			}
			return typ, nil
		}
		return nil, fmt.Errorf("unknown identifier: %s", expr.Name)
	case *ast.AssignStmt:
		for i, rhs := range expr.Rhs {
			var expected Type
			if i < len(expr.Lhs) {
				expected, _ = InferType(expr.Lhs[i], env, ctx)
			}
			rhsCtx := NewInferenceContext(
				WithExpectedType(ctx.ExpectedType),
				WithAssignment(),
			)
			rhsType, err := InferType(rhs, env, rhsCtx)
			if err != nil {
				return nil, err
			}

			// check type compatibility
			if expected != nil {
				if err := Unify(expected, rhsType, env); err != nil {
					return nil, fmt.Errorf("assignment type mismatch for %s: %v", expr.Lhs[i], err)
				}
			}
		}
		return nil, nil // assignment statement does not have a type
	case *ast.ReturnStmt:
		if ctx.ExpectedType == nil {
			return nil, fmt.Errorf("return statement outside of function context")
		}
		funcType, ok := ctx.ExpectedType.(*FunctionType)
		if !ok {
			return nil, fmt.Errorf("expected function type in return context")
		}
		if len(expr.Results) != len(funcType.ReturnType.(*TupleType).Types) {
			return nil, fmt.Errorf("expected %d return values, got %d", len(funcType.ReturnType.(*TupleType).Types), len(expr.Results))
		}
		for i, result := range expr.Results {
			resultCtx := NewInferenceContext(
				WithExpectedType(funcType.ReturnType.(*TupleType).Types[i]),
				WithReturnValue(),
			)
			resultType, err := InferType(result, env, resultCtx)
			if err != nil {
				return nil, err
			}
			if err := Unify(funcType.ReturnType.(*TupleType).Types[i], resultType, env); err != nil {
				return nil, fmt.Errorf("return type mismatch for %dth result: %v", i, err)
			}
		}
		return nil, nil // return statement does not have a type
	case *ast.CallExpr:
		if selExpr, ok := expr.Fun.(*ast.SelectorExpr); ok {
			// might be a method call
			recvType, err := InferType(selExpr.X, env, ctx)
			if err != nil {
				return nil, err
			}

			mthdName := selExpr.Sel.Name

			// check if it's a generic method
			var (
				genericMethod GenericMethod
				found         bool
			)

			switch t := recvType.(type) {
			case *StructType:
				genericMethod, found = t.GenericMethods[mthdName]
			case *InterfaceType:
				genericMethod, found = t.GenericMethods[mthdName]
			}

			if found {
				// 1. it's generic method call
				if len(expr.Args) == 0 {
					return nil, fmt.Errorf("generic method call requires type arguments")
				}

				// 2. the first argument should be the type arguments
				typeArgs, ok := expr.Args[0].(*ast.CompositeLit)
				if !ok {
					return nil, fmt.Errorf("expected type argument for generic method call")
				}

				var typeArgTypes []Type
				for _, elt := range typeArgs.Elts {
					typeArg, err := InferType(elt, env, NewInferenceContext())
					if err != nil {
						return nil, err
					}
					typeArgTypes = append(typeArgTypes, typeArg)
				}

				var args []ast.Expr
				args = append(args, expr.Args[1:]...)

				return inferGenericMethod(genericMethod, typeArgTypes, args, env, ctx)
			}

			method, err := findMethod(recvType, mthdName)
			if err != nil {
				return nil, err
			}

			return inferMethodCall(method, expr.Args, env, ctx)
		}

		// regular function call
		funcTyp, err := InferType(expr.Fun, env, ctx)
		if err != nil {
			return nil, err
		}
		return inferFuncionCall(funcTyp, expr.Args, env, ctx)
	case *ast.IndexExpr:
		ctx := NewInferenceContext(WithExpectedType(ctx.ExpectedType))
		return inferGenericType(expr.X, []ast.Expr{expr.Index}, env, ctx)
	case *ast.IndexListExpr:
		ctx := NewInferenceContext(WithExpectedType(ctx.ExpectedType))
		return inferGenericType(expr.X, expr.Indices, env, ctx)
	case *ast.CompositeLit:
		switch typeExpr := expr.Type.(type) {
		case *ast.MapType:
			kt, err := InferType(typeExpr.Key, env, ctx)
			if err != nil {
				return nil, err
			}
			vt, err := InferType(typeExpr.Value, env, ctx)
			if err != nil {
				return nil, err
			}
			for _, elt := range expr.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					k, err := InferType(kv.Key, env, ctx)
					if err != nil {
						return nil, err
					}
					v, err := InferType(kv.Value, env, ctx)
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
		case *ast.ArrayType:
			// handle slice literal
			if typeExpr.Len == nil {
				// inference the element type
				etCtx := NewInferenceContext(WithExpectedType(ctx.ExpectedType))
				et, err := InferType(typeExpr.Elt, env, etCtx)
				if err != nil {
					return nil, err
				}
				if len(expr.Elts) == 0 {
					// empty slice literal, use the specified element type
					return &SliceType{ElementType: et}, nil
				}

				// check the types of the remaining elements and ensure they are consistent
				//
				// create a new context when checking the element types
				eltCtx := NewInferenceContext(WithExpectedType(et))
				for _, elt := range expr.Elts {
					eltType, err := InferType(elt, env, eltCtx)
					if err != nil {
						return nil, err
					}
					if err := Unify(et, eltType, env); err != nil {
						return nil, errors.New("inconsistent element types in slice literal")
					}
				}
				return &SliceType{ElementType: et}, nil
			}
			// handle array literal
			lenExpr, ok := typeExpr.Len.(*ast.BasicLit)
			if !ok || lenExpr.Kind != token.INT {
				return nil, errors.New("invalid array length expression")
			}
			length, err := strconv.Atoi(lenExpr.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid array length: %v", err)
			}

			etCtx := NewInferenceContext(WithExpectedType(ctx.ExpectedType))
			elemType, err := InferType(typeExpr.Elt, env, etCtx)
			if err != nil {
				return nil, err
			}

			// check element types of the array literal
			eltCtx := NewInferenceContext(WithExpectedType(elemType))
			for _, elt := range expr.Elts {
				et, err := InferType(elt, env, eltCtx)
				if err != nil {
					return nil, err
				}
				if !TypesEqual(elemType, et) {
					return nil, fmt.Errorf("element type mismatch: %v", err)
				}
			}
			return &ArrayType{ElementType: elemType, Len: length}, nil
		case *ast.Ident:
			structType, ok := env[typeExpr.Name].(*StructType)
			if !ok {
				return nil, fmt.Errorf("unknown struct type: %s", typeExpr.Name)
			}

			newStruct := &StructType{
				Name:   structType.Name,
				Fields: make(map[string]Type),
			}

			// handle each field
			for _, elt := range expr.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					return nil, fmt.Errorf("invalid struct literal")
				}

				fieldName := kv.Key.(*ast.Ident).Name
				fieldType, ok := structType.Fields[fieldName]
				if !ok {
					return nil, fmt.Errorf("unknown field: %s", fieldName)
				}

				// create a new context for the field
				fieldCtx := NewInferenceContext(WithExpectedType(fieldType))

				// nested struct
				if nestedCompLit, ok := kv.Value.(*ast.CompositeLit); ok {
					nestedType, err := InferType(nestedCompLit, env, fieldCtx)
					if err != nil {
						return nil, err
					}
					if !TypesEqual(fieldType, nestedType) {
						return nil, fmt.Errorf("type mismatch for field %s: %v. got %v", fieldName, fieldType, nestedType)
					}
					newStruct.Fields[fieldName] = nestedType
				} else {
					fieldValue, err := InferType(kv.Value, env, fieldCtx)
					if err != nil {
						return nil, err
					}
					if !TypesEqual(fieldType, fieldValue) {
						return nil, fmt.Errorf("type mismatch for field %s: %v. got %v", fieldName, fieldType, fieldValue)
					}
					newStruct.Fields[fieldName] = fieldValue
				}
			}
			return newStruct, nil
		// genetic type instantiation
		case *ast.IndexExpr:
			genericType, err := resolveTypeByName(typeExpr.X.(*ast.Ident).Name, env)
			if err != nil {
				return nil, err
			}
			gt, ok := genericType.(*GenericType)
			if !ok {
				return nil, fmt.Errorf("not a generic type: %v", genericType)
			}

			// infer the type argument
			taCtx := NewInferenceContext(WithExpectedType(ctx.ExpectedType))
			typeArg, err := InferType(typeExpr.Index, env, taCtx)
			if err != nil {
				return nil, err
			}

			// check if the type argument satisfies the constraint
			if constraint, ok := gt.Constraints[gt.TypeParams[0].(*TypeVariable).Name]; ok {
				if !checkConstraint(typeArg, constraint) {
					return nil, fmt.Errorf("type argument %v does not satisfy constraint %v", typeArg, constraint)
				}
			}

			instantiatedType := &GenericType{
				Name:       gt.Name,
				TypeParams: []Type{typeArg},
				Fields:     make(map[string]Type),
			}

			// type check the each struct fields
			for fname, ftype := range gt.Fields {
				instantiatedFieldType := substituteTypeParams(ftype, gt.TypeParams, []Type{typeArg}, NewTypeVisitor())
				instantiatedType.Fields[fname] = instantiatedFieldType
			}

			// create a new context for the struct literal
			structCtx := NewInferenceContext(WithExpectedType(instantiatedType))

			// check struct literal's field values
			for _, elt := range expr.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					fname := kv.Key.(*ast.Ident).Name
					fType, ok := instantiatedType.Fields[fname]
					if !ok {
						return nil, fmt.Errorf("unknown field %s in generic type %s", fname, gt.Name)
					}
					vt, err := InferType(kv.Value, env, structCtx)
					if err != nil {
						return nil, err
					}
					if err := Unify(fType, vt, env); err != nil {
						return nil, fmt.Errorf("type mismatch for field %s: %v. got %v", fname, fType, vt)
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
			if strings.Contains(expr.Value, ".") {
				return &TypeConstant{Name: "float64"}, nil
			}
			return &TypeConstant{Name: "float32"}, nil
		case token.STRING:
			return &TypeConstant{Name: "string"}, nil
		case token.CHAR:
			return &TypeConstant{Name: "rune"}, nil
		default:
			return nil, fmt.Errorf("unknown basic literal kind: %v", expr.Kind)
		}
	case *ast.StarExpr:
		btCtx := NewInferenceContext(WithExpectedType(ctx.ExpectedType))
		bt, err := InferType(expr.X, env, btCtx)
		if err != nil {
			return nil, err
		}
		return &PointerType{Base: bt}, nil
	case *ast.FuncType:
		paramCtx := NewInferenceContext(WithFunctionArg())
		ptypes, err := inferParams(expr.Params, env, paramCtx)
		if err != nil {
			return nil, err
		}

		retCtx := NewInferenceContext(WithReturnValue())
		retType, err := inferResult(expr.Results, env, retCtx)
		if err != nil {
			return nil, err
		}

		isVariadic := expr.Params.NumFields() > 0 && expr.Params.List[len(expr.Params.List)-1].Type.(*ast.Ellipsis) != nil
		funcType := &FunctionType{
			ParamTypes: ptypes,
			ReturnType: retType,
			IsVariadic: isVariadic,
		}

		// if ctx has expected type, check the function type compatibility
		if ctx != nil && ctx.ExpectedType != nil {
			if expected, ok := ctx.ExpectedType.(*FunctionType); ok {
				if err := checkFunctionCompatibility(funcType, expected); err != nil {
					return nil, fmt.Errorf("function type incompatible with expected type: %v", err)
				}
			}
		}

		return funcType, nil
	case *ast.FuncLit:
		funcCtx := NewInferenceContext()
		if ctx != nil && ctx.ExpectedType != nil {
			if ft, ok := ctx.ExpectedType.(*FunctionType); ok {
				funcCtx.ExpectedType = ft
			}
		}
		return inferFunctionType(expr.Type, env, funcCtx)
	case *ast.Ellipsis:
		var expectedElemType Type
		if ctx != nil && ctx.ExpectedType != nil {
			if sliceType, ok := ctx.ExpectedType.(*SliceType); ok {
				expectedElemType = sliceType.ElementType
			}
		}
		if expr.Elt == nil {
			return &SliceType{
				ElementType: &InterfaceType{Name: "interface{}", IsEmpty: true},
			}, nil
		}
		elemCtx := NewInferenceContext(WithExpectedType(expectedElemType))
		elemType, err := InferType(expr.Elt, env, elemCtx)
		if err != nil {
			return nil, err
		}
		return &SliceType{ElementType: elemType}, nil
	case *ast.InterfaceType:
		iface := &InterfaceType{Name: "", Methods: MethodSet{}, Embedded: []Type{}}
		for _, field := range expr.Methods.List {
			if len(field.Names) == 0 {
				embeddedCtx := NewInferenceContext()
				embeddedType, err := InferType(field.Type, env, embeddedCtx)
				if err != nil {
					return nil, err
				}
				iface.Embedded = append(iface.Embedded, embeddedType)
			} else {
				for _, name := range field.Names {
					mt, ok := field.Type.(*ast.FuncType)
					if !ok {
						return nil, fmt.Errorf("expected function type for method %s", name.Name)
					}

					paramCtx := NewInferenceContext(WithFunctionArg())
					params, err := inferParams(mt.Params, env, paramCtx)
					if err != nil {
						return nil, fmt.Errorf("error inferring parameters for method %s: %v", name.Name, err)
					}

					// infer the method results
					resultsCtx := NewInferenceContext(WithReturnValue())
					results, err := inferParams(mt.Results, env, resultsCtx)
					if err != nil {
						return nil, fmt.Errorf("error inferring results for method %s: %v", name.Name, err)
					}

					iface.Methods[name.Name] = Method{
						Name:    name.Name,
						Params:  params,
						Results: results,
					}
				}
			}
		}
		// if context contains expected type, need to check interface compatibility
		if ctx != nil && ctx.ExpectedType != nil {
			if expected, ok := ctx.ExpectedType.(*InterfaceType); ok {
				if err := checkInterfaceCompatibility(iface, expected); err != nil {
					return nil, fmt.Errorf("interface incompatible with expected type: %v", err)
				}
			}
		}
		return iface, nil
	default:
		return nil, fmt.Errorf("unsupported node type: %T", node)
	}
	return nil, fmt.Errorf("unknown expression: %T", node)
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
func inferGenericType(x ast.Expr, indices []ast.Expr, env TypeEnv, ctx *InferenceContext) (Type, error) {
	baseType, err := InferType(x, env, ctx)
	if err != nil {
		return nil, err
	}
	genericType, ok := baseType.(*GenericType)
	if !ok {
		return nil, ErrNotAGenericType
	}

	if len(indices) != len(genericType.TypeParams) {
		return nil, fmt.Errorf("expected %d type parameters, got %d", len(genericType.TypeParams), len(indices))
	}

	var typeParams []Type
	for i, index := range indices {
		paramCtx := NewInferenceContext(WithExpectedType(genericType.TypeParams[i]))
		paramType, err := InferType(index, env, paramCtx)
		if err != nil {
			return nil, err
		}
		if i < len(genericType.TypeParams) {
			paramName := genericType.TypeParams[i].(*TypeVariable).Name
			if cst, ok := genericType.Constraints[paramName]; ok {
				if !checkConstraint(paramType, cst) {
					return nil, fmt.Errorf("type argument %v does not satisfy constraint for %s", paramType, paramName)
				}
			}
		}
		typeParams = append(typeParams, paramType)
	}

	inferredType := &GenericType{
		Name:       genericType.Name,
		TypeParams: typeParams,
		Fields:     genericType.Fields, // create new map takes more time than just copying the reference. about 2x slower
	}

	visitor := NewTypeVisitor()
	// substitute type parameters in the fields
	for name, typ := range genericType.Fields {
		inferredType.Fields[name] = substituteTypeParams(typ, genericType.TypeParams, typeParams, visitor)
	}

	return inferredType, nil
}

func inferGenericMethod(method GenericMethod, typeArgs []Type, args []ast.Expr, env TypeEnv, ctx *InferenceContext) (Type, error) {
	if len(typeArgs) != len(method.TypeParams) {
		return nil, fmt.Errorf("expected %d type arguments, got %d", len(method.TypeParams), len(typeArgs))
	}

	// Create a new environment with type parameters bound to concrete types
	newEnv := make(TypeEnv)
	for k, v := range env {
		newEnv[k] = v
	}
	for i, param := range method.TypeParams {
		newEnv[param.(*TypeVariable).Name] = typeArgs[i]
	}

	// Substitute type parameters in the method signature
	substitutedMethod := substituteTypeParams(method.Method, method.TypeParams, typeArgs, NewTypeVisitor()).(Method)

	// Check argument types
	if len(args) != len(substitutedMethod.Params) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(substitutedMethod.Params), len(args))
	}
	for i, arg := range args {
		argContext := NewInferenceContext(
			WithExpectedType(substitutedMethod.Params[i]),
			WithFunctionArg(),
		)
		argType, err := InferType(arg, env, argContext)
		if err != nil {
			return nil, err
		}
		if err := Unify(substitutedMethod.Params[i], argType, newEnv); err != nil {
			return nil, fmt.Errorf("argument type mismatch for arg: %v", err)
		}
	}

	// Substitute type parameters in the result type
	if len(substitutedMethod.Results) == 0 {
		return &TypeConstant{Name: "void"}, nil
	}
	resultType := substituteTypeParams(substitutedMethod.Results[0], method.TypeParams, typeArgs, NewTypeVisitor())

	if ctx != nil && ctx.ExpectedType != nil {
		if err := Unify(resultType, ctx.ExpectedType, newEnv); err != nil {
			return nil, fmt.Errorf("return type mismatch: %v", err)
		}
	}

	return resultType, nil
}

// substituteTypeParams substitutes type parameters in a type with concrete types.
// It uses a TypeVisitor to detect and handle circular references in the type structure.
func substituteTypeParams(t Type, from, to []Type, visitor *TypeVisitor) Type {
	// circular reference check
	if visitor.Visit(t) {
		return t
	}
	switch t := t.(type) {
	case *TypeVariable:
		for i, param := range from {
			if TypesEqual(t, param) {
				return to[i]
			}
		}
	case *GenericType:
		newParams := make([]Type, len(t.TypeParams))
		for i, param := range t.TypeParams {
			newParams[i] = substituteTypeParams(param, from, to, visitor)
		}
		newFld := make(map[string]Type)
		for name, typ := range t.Fields {
			newFld[name] = substituteTypeParams(typ, from, to, visitor)
		}
		return &GenericType{
			Name:       t.Name,
			TypeParams: newParams,
			Fields:     t.Fields,
		}
	case *SliceType:
		return &SliceType{
			ElementType: substituteTypeParams(t.ElementType, from, to, visitor),
		}
	case *MapType:
		return &MapType{
			KeyType:   substituteTypeParams(t.KeyType, from, to, visitor),
			ValueType: substituteTypeParams(t.ValueType, from, to, visitor),
		}
	case *FunctionType:
		newParams := make([]Type, len(t.ParamTypes))
		for i, param := range t.ParamTypes {
			newParams[i] = substituteTypeParams(param, from, to, visitor)
		}
		newReturn := substituteTypeParams(t.ReturnType, from, to, visitor)
		return &FunctionType{
			ParamTypes: newParams,
			ReturnType: newReturn,
			IsVariadic: t.IsVariadic,
		}
	}
	return t
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

func CalculateMethodSet(t Type) MethodSet {
	switch t := t.(type) {
	case *StructType:
		return calculateStructMethodSet(t, false)
	case *InterfaceType:
		return t.Methods
	case *PointerType:
		if st, ok := t.Base.(*StructType); ok {
			return calculateStructMethodSet(st, true)
		}
	default:
		return MethodSet{}
	}
	return MethodSet{}
}

func calculateStructMethodSet(s *StructType, isPtr bool) MethodSet {
	ms := make(MethodSet)

	// direct methods of the struct
	for name, method := range s.Methods {
		if isPtr || !method.IsPointer {
			ms[name] = method
		}
	}

	// methods from embedded fields
	for _, fld := range s.Fields {
		if embeddedType, ok := fld.(*StructType); ok {
			embeddedMethods := calculateStructMethodSet(embeddedType, false)
			for name, method := range embeddedMethods {
				if _, exists := ms[name]; !exists {
					ms[name] = method
				}
			}
		}
	}
	return ms
}

func inferFunctionType(ft *ast.FuncType, env TypeEnv, ctx *InferenceContext) (Type, error) {
	var (
		paramTypes []Type
		returnType Type
	)

	paramCtx := NewInferenceContext(WithFunctionArg())
	if ft.Params != nil {
		for i, fld := range ft.Params.List {
			var expectedParamType Type
			if ctx != nil && ctx.ExpectedType != nil {
				if expectedFt, ok := ctx.ExpectedType.(*FunctionType); ok && i < len(expectedFt.ParamTypes) {
					expectedParamType = expectedFt.ParamTypes[i]
				}
			}
			paramCtx.ExpectedType = expectedParamType
			fldt, err := InferType(fld.Type, env, paramCtx)
			if err != nil {
				return nil, err
			}
			paramTypes = append(paramTypes, fldt)
		}
	}
	returnCtx := NewInferenceContext(WithReturnValue())
	if ft.Results != nil {
		if len(ft.Results.List) == 1 {
			if ctx != nil && ctx.ExpectedType != nil {
				if expectedFt, ok := ctx.ExpectedType.(*FunctionType); ok {
					returnCtx.ExpectedType = expectedFt.ReturnType
				}
			}
			var err error
			returnType, err = InferType(ft.Results.List[0].Type, env, returnCtx)
			if err != nil {
				return nil, err
			}
		} else if len(ft.Results.List) > 1 {
			tupleTypes := make([]Type, len(ft.Results.List))
			for i, result := range ft.Results.List {
				resultType, err := InferType(result.Type, env, returnCtx)
				if err != nil {
					return nil, err
				}
				tupleTypes[i] = resultType
			}
			returnType = &TupleType{Types: tupleTypes}
		}
	}
	funcType := &FunctionType{
		ParamTypes: paramTypes,
		ReturnType: returnType,
	}

	// if context has expected type, check the function type compatibility
	if ctx != nil && ctx.ExpectedType != nil {
		if expectedFunc, ok := ctx.ExpectedType.(*FunctionType); ok {
			if err := checkFunctionCompatibility(funcType, expectedFunc); err != nil {
				return nil, fmt.Errorf("function type incompatible with expected type: %v", err)
			}
		}
	}

	return funcType, nil
}

func inferParams(fieldList *ast.FieldList, env TypeEnv, ctx *InferenceContext) ([]Type, error) {
	if fieldList == nil {
		return nil, nil
	}

	var params []Type
	for _, field := range fieldList.List {
		fieldType, err := InferType(field.Type, env, ctx)
		if err != nil {
			return nil, err
		}
		// multiple names in a field. like (a, b int)
		if len(field.Names) == 0 {
			params = append(params, fieldType)
		} else {
			for range field.Names {
				params = append(params, fieldType)
			}
		}
	}
	return params, nil
}

func inferResult(results *ast.FieldList, env TypeEnv, ctx *InferenceContext) (Type, error) {
	if results == nil || len(results.List) == 0 {
		return &TypeConstant{Name: "void"}, nil
	}

	if len(results.List) == 1 && len(results.List[0].Names) == 0 {
		// single return value
		return InferType(results.List[0].Type, env, ctx)
	}

	// multiple return values. like tuple type or anonymous struct type
	var tt []Type
	for _, fld := range results.List {
		fldType, err := InferType(fld.Type, env, ctx)
		if err != nil {
			if strings.HasPrefix(err.Error(), "unknown identifier") {
				return nil, fmt.Errorf("unknown type: %s", strings.TrimPrefix(err.Error(), "unknown identifier: "))
			}
			return nil, err
		}
		if len(fld.Names) == 0 {
			tt = append(tt, fldType)
		} else {
			for range fld.Names {
				tt = append(tt, fldType)
			}
		}
	}
	tupleType := &TupleType{Types: tt}

	// if context has expected type, check the tuple type compatibility
	if ctx != nil && ctx.ExpectedType != nil {
		if expected, ok := ctx.ExpectedType.(*TupleType); ok {
			if err := checkTupleCompatibility(tupleType, expected); err != nil {
				return nil, fmt.Errorf("tuple type incompatible with expected type: %v", err)
			}
		}
	}

	return tupleType, nil
}

func findMethod(recvType Type, methodName string) (Method, error) {
	switch t := recvType.(type) {
	case *StructType:
		if method, ok := t.Methods[methodName]; ok {
			return method, nil
		}
	case *InterfaceType:
		if method, ok := t.Methods[methodName]; ok {
			return method, nil
		}
	}
	return Method{}, fmt.Errorf("method %s not found in type %v", methodName, recvType)
}

func inferMethodCall(method Method, args []ast.Expr, env TypeEnv, ctx *InferenceContext) (Type, error) {
	if len(args) != len(method.Params) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(method.Params), len(args))
	}
	for i, arg := range args {
		argContext := NewInferenceContext(
			WithExpectedType(method.Params[i]),
			WithFunctionArg(),
		)
		argType, err := InferType(arg, env, argContext)
		if err != nil {
			return nil, err
		}
		if err := Unify(method.Params[i], argType, env); err != nil {
			return nil, fmt.Errorf("argument type mismatch for arg %d: %v", i, err)
		}
	}
	if len(method.Results) == 0 {
		return &TypeConstant{Name: "void"}, nil
	}
	return method.Results[0], nil
}

func inferFuncionCall(funcTyp Type, args []ast.Expr, env TypeEnv, ctx *InferenceContext) (Type, error) {
	ft, ok := funcTyp.(*FunctionType)
	if !ok {
		return nil, ErrNotAFunction
	}
	if len(args) != len(ft.ParamTypes) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(ft.ParamTypes), len(args))
	}
	for i, arg := range args {
		argContext := NewInferenceContext(
			WithExpectedType(ft.ParamTypes[i]),
			WithFunctionArg(),
		)
		argType, err := InferType(arg, env, argContext)
		if err != nil {
			return nil, err
		}
		if err := Unify(ft.ParamTypes[i], argType, env); err != nil {
			return nil, fmt.Errorf("argument type mismatch for arg %d: %v", i, err)
		}
	}
	return ft.ReturnType, nil
}

// func inferTypeSpec(spec *ast.TypeSpec, env TypeEnv) (Type, error) {
// 	// type alias
// 	if spec.Assign.IsValid() {
// 		aliased, err := InferType(spec.Type, env)
// 		if err != nil {
// 			return nil, err
// 		}
// 		alias := &TypeAlias{
// 			Name:      spec.Name.Name,
// 			AliasedTo: aliased,
// 		}
// 		env[spec.Name.Name] = alias // add the alias to the environment
// 		return alias, nil
// 	}

// 	// normal type declaration
// 	definedType, err := InferType(spec.Type, env)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// create a new type with the name and add it to the environment
// 	switch t := definedType.(type) {
// 	case *StructType:
// 		newType := &StructType{
// 			Name:    spec.Name.Name,
// 			Fields:  t.Fields,
// 			Methods: t.Methods,
// 		}
// 		env[spec.Name.Name] = newType
// 		return newType, nil
// 	case *InterfaceType:
// 		newType := &InterfaceType{
// 			Name:     spec.Name.Name,
// 			Methods:  t.Methods,
// 			Embedded: t.Embedded,
// 		}
// 		env[spec.Name.Name] = newType
// 		return newType, nil
// 	default:
// 		env[spec.Name.Name] = definedType
// 		return definedType, nil
// 	}
// }
