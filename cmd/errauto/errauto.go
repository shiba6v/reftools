package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"text/template"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

type Input struct {
	F   *ast.File
	Pkg *packages.Package
	Pos token.Pos
}

type Result struct {
	Start int
	End   int
	N     ast.Node
	Lines int
}

func getEnclosingFunc(f *ast.File, pos token.Pos) (*ast.FuncDecl, error) {
	path, _ := astutil.PathEnclosingInterval(f, pos, pos)
	for _, n := range path {
		if decl, ok := n.(*ast.FuncDecl); ok {
			return decl, nil
		}
	}
	return nil, Wrap(fmt.Errorf("enclosing Func is null"))
}

// エラーハンドリング直前の関数
func getPreviousFuncName(enclosingDecl *ast.FuncDecl, pos token.Pos) string {
	previousFuncName := "func"
	for _, n := range enclosingDecl.Body.List {
		if n.Pos() > pos {
			break
		}
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			continue
		}
		if len(assign.Rhs) == 0 {
			continue
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		switch t := call.Fun.(type) {
		case *ast.Ident:
			// t.Name // string
			previousFuncName = t.Name
		case *ast.SelectorExpr:
			// t.X // somepkg
			// t.Sel // PkgOutput2
			previousFuncName = fmt.Sprintf("%s.%s", t.X, t.Sel)
		}
		// debugAstPrint(n)
		// debugPrintf("%v\n", n.Pos())
	}
	return previousFuncName
}

func ErrAuto(in Input) (Result, error) {
	enclosingDecl, err := getEnclosingFunc(in.F, in.Pos)
	if err != nil {
		return Result{}, Wrap(err)
	}
	previousFuncName := getPreviousFuncName(enclosingDecl, in.Pos)
	pos := in.Pos
	// debugAstPrint(enclosingDecl.Type.Results.List)
	returnExprs := make([]ast.Expr, 0)
	// 関数の返り値の最後(err)以外を順に見ていく
	rList := enclosingDecl.Type.Results.List
	for _, f := range rList[:len(rList)-1] {
		var ct ast.Expr
		switch t := f.Type.(type) {
		case *ast.Ident:
			// t.Name // string
			ct = t
		case *ast.SelectorExpr:
			// t.X // somepkg
			// t.Sel // PkgOutput2
			ct = t
		default:
			// debugPrintf("default %v", t)
		}
		u := in.Pkg.TypesInfo.Types[f.Type].Type.Underlying()

		switch t := u.(type) {
		case *types.Struct:
			returnExprs = append(returnExprs, &ast.CompositeLit{
				Type:       ct,
				Lbrace:     0,
				Elts:       []ast.Expr{},
				Rbrace:     0,
				Incomplete: false,
			})
		case *types.Basic:
			var ex ast.Expr
			switch t.Kind() {
			case types.Bool:
				ex = &ast.Ident{Name: "false", NamePos: pos}
			case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
				ex = &ast.BasicLit{Value: "0", ValuePos: pos}
			case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
				ex = &ast.BasicLit{Value: "0", ValuePos: pos}
			case types.Uintptr:
				ex = &ast.BasicLit{Value: "uintptr(0)", ValuePos: pos}
			case types.UnsafePointer:
				ex = &ast.BasicLit{Value: "unsafe.Pointer(uintptr(0))", ValuePos: pos}
			case types.Float32, types.Float64:
				ex = &ast.BasicLit{Value: "0.0", ValuePos: pos}
			case types.Complex64, types.Complex128:
				ex = &ast.BasicLit{Value: "(0 + 0i)", ValuePos: pos}
			case types.String:
				ex = &ast.BasicLit{Value: `""`, ValuePos: pos}
			default:
			}
			if ex != nil {
				returnExprs = append(returnExprs, ex)
			}
		case *types.Interface:
			returnExprs = append(returnExprs, &ast.Ident{
				Name: "nil",
			})
		case *types.Pointer:
			returnExprs = append(returnExprs, &ast.Ident{
				Name: "nil",
			})
		}
	}
	errMsgTmplParams := struct {
		Func1 string
		Func2 string
	}{
		Func1: enclosingDecl.Name.Name,
		Func2: previousFuncName,
	}
	errMsgTmpl, err := template.New("errMsg").Parse(`"{{.Func1}}: {{.Func2}} failed, %w"`)
	if err != nil {
		return Result{}, Wrap(err)
	}
	var buf bytes.Buffer
	err = errMsgTmpl.Execute(&buf, errMsgTmplParams)
	if err != nil {
		return Result{}, Wrap(err)
	}
	errMsg := buf.String()

	returnExprs = append(returnExprs, &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.Ident{
				Name: "fmt",
			},
			Sel: &ast.Ident{
				NamePos: 0,
				Name:    "Errorf",
			},
		},
		Lparen: 0,
		Args: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: errMsg,
			},
			&ast.Ident{
				Name: "err",
			},
		},
		Ellipsis: 0,
		Rparen:   0,
	})
	returnStmt := &ast.ReturnStmt{
		Return:  in.Pos,
		Results: returnExprs,
	}
	ifStmt := &ast.IfStmt{
		If:   0,
		Init: nil, // TODO: errが1個の場合、ここに元のstmtを入れる
		Cond: &ast.BinaryExpr{
			X: &ast.Ident{
				Name: "err",
			},
			OpPos: 0,
			Op:    token.NEQ,
			Y: &ast.Ident{
				Name: "nil",
			},
		},
		Body: &ast.BlockStmt{
			Lbrace: 0,
			List: []ast.Stmt{
				returnStmt,
			},
			Rbrace: 0,
		},
		Else: nil,
	}
	return Result{
		Start: in.Pkg.Fset.Position(in.Pos).Offset,
		End:   in.Pkg.Fset.Position(in.Pos).Offset,
		N:     ifStmt,
		Lines: 1,
	}, nil
}
