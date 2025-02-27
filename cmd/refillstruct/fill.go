// Copyright (c) 2023 shiba6v All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Some codes are copied from fillstruct from David R. Jenni.
// Here is the original copyright.
// Copyright (c) 2018 David R. Jenni. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func debugPrintf(format string, a ...interface{}) {
	f, err := os.OpenFile("/tmp/log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte(fmt.Sprintf(format, a...))); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func debugAstPrint(x interface{}) {
	f, err := os.OpenFile("/tmp/log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	ast.Fprint(f, token.NewFileSet(), x, ast.NotNilFilter)
}

// litInfo contains the information about
// a literal to fill with zero values.
type litInfo struct {
	typ       types.Type   // the base type of the literal
	name      *types.Named // name of the type or nil, e.g. for an anonymous struct type
	hideType  bool         // flag to hide the element type inside an array, slice or map literal
	isPointer bool         // true if the literal is of a pointer type
}

type filler struct {
	pkg         *packages.Package
	pos         token.Pos
	lines       int
	existing    map[string]*ast.KeyValueExpr
	otherElts   map[string]*ast.KeyValueExpr
	first       bool
	importNames map[string]string // import path -> import name
}

func refillValue(pkg *packages.Package, importNames map[string]string, lit *ast.CompositeLit, info litInfo, otherElts map[string]*ast.KeyValueExpr) (ast.Expr, int) {
	f := filler{
		pkg:         pkg,
		pos:         1,
		first:       true,
		existing:    make(map[string]*ast.KeyValueExpr),
		otherElts:   otherElts,
		importNames: importNames,
	}
	for _, e := range lit.Elts {
		kv := e.(*ast.KeyValueExpr)
		f.existing[kv.Key.(*ast.Ident).Name] = kv
	}
	return f.zero(info, make([]types.Type, 0, 8)), f.lines
}

func (f *filler) zero(info litInfo, visited []types.Type) ast.Expr {
	switch t := info.typ.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Bool:
			return &ast.Ident{Name: "false", NamePos: f.pos}
		case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
			return &ast.BasicLit{Value: "0", ValuePos: f.pos}
		case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
			return &ast.BasicLit{Value: "0", ValuePos: f.pos}
		case types.Uintptr:
			return &ast.BasicLit{Value: "uintptr(0)", ValuePos: f.pos}
		case types.UnsafePointer:
			return &ast.BasicLit{Value: "unsafe.Pointer(uintptr(0))", ValuePos: f.pos}
		case types.Float32, types.Float64:
			return &ast.BasicLit{Value: "0.0", ValuePos: f.pos}
		case types.Complex64, types.Complex128:
			return &ast.BasicLit{Value: "(0 + 0i)", ValuePos: f.pos}
		case types.String:
			return &ast.BasicLit{Value: `""`, ValuePos: f.pos}
		default:
			// Cannot create an expression for an invalid type.
			return nil
		}
	case *types.Chan:
		valTypeName, ok := typeString(f.pkg.Types, f.importNames, t.Elem())
		if !ok {
			return nil
		}

		var dir ast.ChanDir
		switch t.Dir() {
		case types.SendRecv:
			dir = ast.SEND | ast.RECV
		case types.SendOnly:
			dir = ast.SEND
		case types.RecvOnly:
			dir = ast.RECV
		}

		return &ast.CallExpr{
			Fun: &ast.Ident{
				NamePos: f.pos,
				Name:    "make",
			},
			Lparen: f.pos,
			Args: []ast.Expr{
				&ast.ChanType{
					Dir:   dir,
					Value: ast.NewIdent(valTypeName),
				},
			},
			Rparen: f.pos,
		}
	case *types.Interface:
		return &ast.Ident{Name: "nil", NamePos: f.pos}
	case *types.Map:
		keyTypeName, ok := typeString(f.pkg.Types, f.importNames, t.Key())
		if !ok {
			return nil
		}
		valTypeName, ok := typeString(f.pkg.Types, f.importNames, t.Elem())
		if !ok {
			return nil
		}
		lit := &ast.CompositeLit{
			Lbrace: f.pos,
			Type: &ast.MapType{
				Map:   f.pos,
				Key:   ast.NewIdent(keyTypeName),
				Value: ast.NewIdent(valTypeName),
			},
		}
		f.pos++
		lit.Elts = []ast.Expr{
			&ast.KeyValueExpr{
				Key:   f.zero(litInfo{typ: t.Key(), name: info.name, hideType: true}, visited),
				Colon: f.pos,
				Value: f.zero(litInfo{typ: t.Elem(), name: info.name, hideType: true}, visited),
			},
		}
		f.pos++
		lit.Rbrace = f.pos
		f.lines += 2
		return lit
	case *types.Signature:
		params := make([]*ast.Field, t.Params().Len())
		for i := 0; i < t.Params().Len(); i++ {
			typeName, ok := typeString(f.pkg.Types, f.importNames, t.Params().At(i).Type())
			if !ok {
				return nil
			}
			params[i] = &ast.Field{
				Type: ast.NewIdent(typeName),
			}
		}
		results := make([]*ast.Field, t.Results().Len())
		for i := 0; i < t.Results().Len(); i++ {
			typeName, ok := typeString(f.pkg.Types, f.importNames, t.Results().At(i).Type())
			if !ok {
				return nil
			}
			results[i] = &ast.Field{
				Type: ast.NewIdent(typeName),
			}
		}
		return &ast.FuncLit{
			Type: &ast.FuncType{
				Func:    f.pos,
				Params:  &ast.FieldList{List: params},
				Results: &ast.FieldList{List: results},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{X: ast.NewIdent(`panic("not implemented")`)},
				},
			},
		}
	case *types.Slice:
		return f.fillSequence(info, visited, t, nil)

	case *types.Array:
		return f.fillSequence(info, visited, t, &ast.BasicLit{Value: strconv.FormatInt(t.Len(), 10)})

	case *types.Named:
		if _, ok := t.Underlying().(*types.Struct); ok {
			info.name = t
		}
		info.typ = t.Underlying()
		return f.zero(info, visited)

	case *types.Pointer:
		if _, ok := t.Elem().Underlying().(*types.Struct); ok {
			info.typ = t.Elem()
			info.isPointer = true
			return f.zero(info, visited)
		}
		return &ast.Ident{Name: "nil", NamePos: f.pos}

	case *types.Struct:
		newlit := &ast.CompositeLit{Lbrace: f.pos}
		if !info.hideType && info.name != nil {
			typeName, ok := typeString(f.pkg.Types, f.importNames, info.name)
			if !ok {
				return nil
			}
			newlit.Type = ast.NewIdent(typeName)
			if info.isPointer {
				newlit.Type.(*ast.Ident).Name = "&" + newlit.Type.(*ast.Ident).Name
			}
		} else if !info.hideType && info.name == nil {
			typeName, ok := typeString(f.pkg.Types, f.importNames, t)
			if !ok {
				return nil
			}
			newlit.Type = ast.NewIdent(typeName)
		}

		for _, typ := range visited {
			if t == typ {
				return newlit
			}
		}
		visited = append(visited, t)

		first := f.first
		f.first = false
		lines := 0
		imported := isImported(f.pkg.Types, info.name)

		for i := 0; i < t.NumFields(); i++ {
			field := t.Field(i)
			// debugPrintf("%#v\n", field.Name())
			// don't fill the field if it a gRPC system field
			if strings.HasPrefix(field.Name(), "XXX_") {
				continue
			}
			if kv, ok := f.existing[field.Name()]; first && ok {
				f.pos++
				lines++
				f.fixExprPos(kv)
				newlit.Elts = append(newlit.Elts, kv)
			} else if kv, ok := f.otherElts[strings.ToLower(field.Name())]; first && ok {
				// refill value from other elements.
				f.pos++
				lines++
				// use the same key.
				k := &ast.Ident{Name: field.Name(), NamePos: f.pos}
				kv.Key = k
				kv.Colon = f.pos
				newlit.Elts = append(newlit.Elts, kv)
			} else if !ok && !imported || field.Exported() {
				f.pos++
				k := &ast.Ident{Name: field.Name(), NamePos: f.pos}
				if v := f.zero(litInfo{typ: field.Type(), name: nil}, visited); v != nil {
					lines++
					newlit.Elts = append(newlit.Elts, &ast.KeyValueExpr{
						Key:   k,
						Value: v,
					})
				} else {
					f.pos--
				}
			}
		}
		if lines > 0 {
			f.lines += lines + 2
			f.pos++
		}
		newlit.Rbrace = f.pos
		return newlit

	default:
		panic(fmt.Sprintf("unexpected type %T", t))
	}
}

// sequence is a interface that abstracts
// between *types.Slice and *types.Array
type sequence interface {
	Elem() types.Type
}

func (f *filler) fillSequence(info litInfo, visited []types.Type, t sequence, length ast.Expr) ast.Expr {
	lit := &ast.CompositeLit{Lbrace: f.pos}
	if !info.hideType {
		typeName, ok := typeString(f.pkg.Types, f.importNames, t.Elem())
		if !ok {
			return nil
		}
		lit.Type = &ast.ArrayType{
			Lbrack: f.pos,
			Len:    length,
			Elt:    ast.NewIdent(typeName),
		}
	}
	if arr, isArray := t.(*types.Array); isArray {
		lit.Elts = make([]ast.Expr, 0, arr.Len())
		for i := int64(0); i < arr.Len(); i++ {
			f.pos++
			elemInfo := litInfo{typ: t.Elem().Underlying(), hideType: true}
			elemInfo.name, _ = t.Elem().(*types.Named)
			if v := f.zero(elemInfo, visited); v != nil {
				lit.Elts = append(lit.Elts, v)
			}
		}
		f.lines += len(lit.Elts)
	}
	f.lines += 2
	f.pos++
	lit.Rbrace = f.pos
	return lit
}

func (f *filler) fixExprPos(expr ast.Expr) {
	switch expr := expr.(type) {
	case nil:
		// ignore
	case *ast.BasicLit:
		expr.ValuePos = f.pos
	case *ast.BinaryExpr:
		f.fixExprPos(expr.X)
		expr.OpPos = f.pos
		f.fixExprPos(expr.Y)
	case *ast.CallExpr:
		f.fixExprPos(expr.Fun)
		expr.Lparen = f.pos
		for _, arg := range expr.Args {
			f.fixExprPos(arg)
		}
		expr.Rparen = f.pos
	case *ast.CompositeLit:
		f.fixExprPos(expr.Type)
		expr.Lbrace = f.pos
		for _, e := range expr.Elts {
			f.pos++
			f.fixExprPos(e)
		}
		if l := len(expr.Elts); l > 0 {
			f.lines += l + 2
		}
		f.pos++
		expr.Rbrace = f.pos
	case *ast.Ellipsis:
		expr.Ellipsis = f.pos
	case *ast.FuncLit:
		expr.Type.Func = f.pos
	case *ast.Ident:
		expr.NamePos = f.pos
	case *ast.IndexExpr:
		f.fixExprPos(expr.X)
		expr.Lbrack = f.pos
		f.fixExprPos(expr.Index)
		expr.Rbrack = f.pos
	case *ast.KeyValueExpr:
		f.fixExprPos(expr.Key)
		f.fixExprPos(expr.Value)
	case *ast.ParenExpr:
		expr.Lparen = f.pos
	case *ast.SelectorExpr:
		f.fixExprPos(expr.X)
		expr.Sel.NamePos = f.pos
	case *ast.SliceExpr:
		f.fixExprPos(expr.X)
		expr.Lbrack = f.pos
		f.fixExprPos(expr.Low)
		f.fixExprPos(expr.High)
		f.fixExprPos(expr.Max)
		expr.Rbrack = f.pos
	case *ast.StarExpr:
		expr.Star = f.pos
		f.fixExprPos(expr.X)
	case *ast.UnaryExpr:
		expr.OpPos = f.pos
		f.fixExprPos(expr.X)
	}
}

func isImported(pkg *types.Package, n *types.Named) bool {
	return n != nil && pkg != n.Obj().Pkg()
}
