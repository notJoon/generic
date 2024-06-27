package generic

import (
    "go/ast"
    "testing"
)

func TestInferType(t *testing.T) {
    tests := []struct {
        name     string
        expr     ast.Expr
        env      TypeEnv
        wantType Type
        wantErr  error
    }{
        {
            name: "Infer type of identifier",
            expr: &ast.Ident{Name: "x"},
            env: TypeEnv{
                "x": &TypeConstant{Name: "int"},
            },
            wantType: &TypeConstant{Name: "int"},
            wantErr:  nil,
        },
        {
            name: "Infer type of unknown identifier",
            expr: &ast.Ident{Name: "y"},
            env:  TypeEnv{},
            wantType: nil,
            wantErr:  ErrUnknownIdent,
        },
        {
            name: "Infer type of function call",
            expr: &ast.CallExpr{
                Fun: &ast.Ident{Name: "f"},
                Args: []ast.Expr{
                    &ast.Ident{Name: "x"},
                },
            },
            env: TypeEnv{
                "f": &FunctionType{
                    ParamTypes: []Type{&TypeConstant{Name: "int"}},
                    ReturnType: &TypeConstant{Name: "string"},
                },
                "x": &TypeConstant{Name: "int"},
            },
            wantType: &TypeConstant{Name: "string"},
            wantErr:  nil,
        },
        {
            name: "Infer type of function call with type mismatch",
            expr: &ast.CallExpr{
                Fun: &ast.Ident{Name: "f"},
                Args: []ast.Expr{
                    &ast.Ident{Name: "x"},
                },
            },
            env: TypeEnv{
                "f": &FunctionType{
                    ParamTypes: []Type{&TypeConstant{Name: "int"}},
                    ReturnType: &TypeConstant{Name: "string"},
                },
                "x": &TypeConstant{Name: "string"},
            },
            wantType: nil,
            wantErr:  ErrTypeMismatch,
        },
        {
            name: "Infer type of non-function call",
            expr: &ast.CallExpr{
                Fun: &ast.Ident{Name: "x"},
                Args: []ast.Expr{
                    &ast.Ident{Name: "y"},
                },
            },
            env: TypeEnv{
                "x": &TypeConstant{Name: "int"},
                "y": &TypeConstant{Name: "int"},
            },
            wantType: nil,
            wantErr:  ErrNotAFunction,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gotType, err := InferType(tt.expr, tt.env)
            if err != tt.wantErr {
                t.Errorf("InferType() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !TypesEqual(gotType, tt.wantType) {
                t.Errorf("InferType() = %v, want %v", gotType, tt.wantType)
            }
        })
    }
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
        t2, ok := t2.(*FunctionType)
        if !ok || len(t1.ParamTypes) != len(t2.ParamTypes) {
            return false
        }
        for i := range t1.ParamTypes {
            if !TypesEqual(t1.ParamTypes[i], t2.ParamTypes[i]) {
                return false
            }
        }
        return TypesEqual(t1.ReturnType, t2.ReturnType)
    default:
        return false
    }
}