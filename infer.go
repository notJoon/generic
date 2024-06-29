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
func InferType(expr ast.Expr, env TypeEnv) (Type, error) {
	switch expr := expr.(type) {
	case *ast.Ident:
		if typ, ok := env[expr.Name]; ok {
			if alias, ok := typ.(*TypeAlias); ok {
				return alias.AliasedTo, nil
			}
			return typ, nil
		}
		return nil, fmt.Errorf("unknown identifier: %s", expr.Name)
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
		switch typeExpr := expr.Type.(type) {
		case *ast.MapType:
			kt, err := InferType(typeExpr.Key, env)
			if err != nil {
				return nil, err
			}
			vt, err := InferType(typeExpr.Value, env)
			if err != nil {
				return nil, err
			}
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
		case *ast.ArrayType:
			// handle slice literal
			if typeExpr.Len == nil {
				et, err := InferType(typeExpr.Elt, env)
				if err != nil {
					return nil, err
				}
				if len(expr.Elts) == 0 {
					// empty slice literal, use the specified element type
					return &SliceType{ElementType: et}, nil
				}

				// check the types of the remaining elements and ensure they are consistent
				for _, elt := range expr.Elts {
					eltType, err := InferType(elt, env)
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
			elemType, err := InferType(typeExpr.Elt, env)
			if err != nil {
				return nil, err
			}
			// check element types of the array literal
			for _, elt := range expr.Elts {
				et, err := InferType(elt, env)
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

				// nested struct
				if nestedCompLit, ok := kv.Value.(*ast.CompositeLit); ok {
					nestedType, err := InferType(nestedCompLit, env)
					if err != nil {
						return nil, err
					}
					if !TypesEqual(fieldType, nestedType) {
						return nil, fmt.Errorf("type mismatch for field %s", fieldName)
					}
					newStruct.Fields[fieldName] = nestedType
				} else {
					fieldValue, err := InferType(kv.Value, env)
					if err != nil {
						return nil, err
					}
					if !TypesEqual(fieldType, fieldValue) {
						return nil, fmt.Errorf("type mismatch for field %s", fieldName)
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
			typeArg, err := InferType(typeExpr.Index, env)
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
		bt, err := InferType(expr.X, env)
		if err != nil {
			return nil, err
		}
		return &PointerType{Base: bt}, nil
	case *ast.FuncType:
		ptypes, err := inferParams(expr.Params, env)
		if err != nil {
			return nil, err
		}
		retType, err := inferResult(expr.Results, env)
		if err != nil {
			return nil, err
		}
		isVariadic := expr.Params.NumFields() > 0 && expr.Params.List[len(expr.Params.List)-1].Type.(*ast.Ellipsis) != nil
		return &FunctionType{
			ParamTypes: ptypes,
			ReturnType: retType,
			IsVariadic: isVariadic,
		}, nil
	case *ast.FuncLit:
		return inferFunctionType(expr.Type, env)
	case *ast.Ellipsis:
		if expr.Elt == nil {
			return &SliceType{
				ElementType: &InterfaceType{Name: "interface{}", IsEmpty: true},
			}, nil
		}
		elemType, err := InferType(expr.Elt, env)
		if err != nil {
			return nil, err
		}
		return &SliceType{ElementType: elemType}, nil
	case *ast.InterfaceType:
		iface := &InterfaceType{Name: "", Methods: MethodSet{}, Embedded: []Type{}}
		for _, field := range expr.Methods.List {
			if len(field.Names) == 0 {
				embeddedType, err := InferType(field.Type, env)
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

					params, err := inferParams(mt.Params, env)
					if err != nil {
						return nil, fmt.Errorf("error inferring parameters for method %s: %v", name.Name, err)
					}

					results, err := inferParams(mt.Results, env)
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
		return iface, nil
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
			return nil, fmt.Errorf("type argument %v does not satisfy constraint %v", paramType, cst)
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

func inferFunctionType(ft *ast.FuncType, env TypeEnv) (Type, error) {
	var (
		paramTypes []Type
		returnType Type
	)

	if ft.Params != nil {
		for _, fld := range ft.Params.List {
			fldt, err := InferType(fld.Type, env)
			if err != nil {
				return nil, err
			}
			paramTypes = append(paramTypes, fldt)
		}
	}
	if ft.Results != nil {
		if len(ft.Results.List) == 1 {
			var err error
			returnType, err = InferType(ft.Results.List[0].Type, env)
			if err != nil {
				return nil, err
			}
		} else if len(ft.Results.List) > 1 {
			// handle multiple return values. like tuple type or anonymous struct type
			panic("multiple return values not supported yet")
		}
	}
	return &FunctionType{ParamTypes: paramTypes, ReturnType: returnType}, nil
}

func inferParams(fieldList *ast.FieldList, env TypeEnv) ([]Type, error) {
	if fieldList == nil {
		return nil, nil
	}

	var params []Type
	for _, field := range fieldList.List {
		fieldType, err := InferType(field.Type, env)
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

func inferResult(results *ast.FieldList, env TypeEnv) (Type, error) {
	if results == nil || len(results.List) == 0 {
		return &TypeConstant{Name: "void"}, nil
	}

	if len(results.List) == 1 && len(results.List[0].Names) == 0 {
		// single return value
		return InferType(results.List[0].Type, env)
	}

	// multiple return values. like tuple type or anonymous struct type
	var tt []Type
	for _, fld := range results.List {
		fldType, err := InferType(fld.Type, env)
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
	return &TupleType{Types: tt}, nil
}

func inferTypeSpec(spec *ast.TypeSpec, env TypeEnv) (Type, error) {
	// type alias
	if spec.Assign.IsValid() {
		aliased, err := InferType(spec.Type, env)
		if err != nil {
			return nil, err
		}
		alias := &TypeAlias{
			Name:      spec.Name.Name,
			AliasedTo: aliased,
		}
		env[spec.Name.Name] = alias // add the alias to the environment
		return alias, nil
	}

	// normal type declaration
	definedType, err := InferType(spec.Type, env)
	if err != nil {
		return nil, err
	}

	// create a new type with the name and add it to the environment
	switch t := definedType.(type) {
	case *StructType:
		newType := &StructType{
			Name:    spec.Name.Name,
			Fields:  t.Fields,
			Methods: t.Methods,
		}
		env[spec.Name.Name] = newType
		return newType, nil
	case *InterfaceType:
		newType := &InterfaceType{
			Name:     spec.Name.Name,
			Methods:  t.Methods,
			Embedded: t.Embedded,
		}
		env[spec.Name.Name] = newType
		return newType, nil
	default:
		env[spec.Name.Name] = definedType
		return definedType, nil
	}
}

func InferPackageTypes(file *ast.File, env TypeEnv) (TypeEnv, error) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			inferredType, err := inferTypeSpec(typeSpec, env)
			if err != nil {
				return nil, err
			}

			env[typeSpec.Name.Name] = inferredType
		}
	}

	return env, nil
}
