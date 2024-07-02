package generic

import (
	"fmt"
	"go/ast"
)

// inferPartialTypeParams infers type parameters for a generic type,
// handling partial specification.
func inferPartialTypeParams(gt *GenericType, indices []ast.Expr, env TypeEnv, ctx *InferenceContext) ([]interface{}, error) {
	inferParams := make([]interface{}, len(gt.TypeParams))
	for i := range gt.TypeParams {
		inferParams[i] = gt.TypeParams[i] // start with the original type parameter
	}

	for i, index := range indices {
		if i >= len(gt.TypeParams) {
			return nil, fmt.Errorf("too many type parameters specified for %s", gt.Name)
		}

		pType, err := InferType(index, env, ctx)
		if err != nil {
			return nil, err
		}

		constraint, ok := gt.Constraints[gt.TypeParams[i].(*TypeVariable).Name]
		if !ok {
			return nil, fmt.Errorf("no constraint for type parameter %s", gt.TypeParams[i].(*TypeVariable).Name)
		}
		if !checkConstraint(pType, constraint) {
			return nil, fmt.Errorf("type argument %v does not satisfy constraint for %v", pType, constraint)
		}

		inferParams[i] = pType
	}

	return inferParams, nil
}
