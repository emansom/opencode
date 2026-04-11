package main

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestParseFlatValueIdent(t *testing.T) {
	expr, err := ParseFlatValue("ident:err")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id, ok := expr.(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident, got %T", expr)
	}
	if id.Name != "err" {
		t.Errorf("expected name err, got %s", id.Name)
	}
}

func TestParseFlatValueNil(t *testing.T) {
	expr, err := ParseFlatValue("nil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id, ok := expr.(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident, got %T", expr)
	}
	if id.Name != "nil" {
		t.Errorf("expected name nil, got %s", id.Name)
	}
}

func TestParseFlatValueTrue(t *testing.T) {
	expr, err := ParseFlatValue("true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id, ok := expr.(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident, got %T", expr)
	}
	if id.Name != "true" {
		t.Errorf("expected name true, got %s", id.Name)
	}
}

func TestParseFlatValueFalse(t *testing.T) {
	expr, err := ParseFlatValue("false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id, ok := expr.(*ast.Ident)
	if !ok {
		t.Fatalf("expected *ast.Ident, got %T", expr)
	}
	if id.Name != "false" {
		t.Errorf("expected name false, got %s", id.Name)
	}
}

func TestParseFlatValueInt(t *testing.T) {
	expr, err := ParseFlatValue("int:42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		t.Fatalf("expected *ast.BasicLit, got %T", expr)
	}
	if lit.Kind != token.INT {
		t.Errorf("expected token.INT, got %v", lit.Kind)
	}
	if lit.Value != "42" {
		t.Errorf("expected value 42, got %s", lit.Value)
	}
}

func TestParseFlatValueFloat(t *testing.T) {
	expr, err := ParseFlatValue("float:3.14")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		t.Fatalf("expected *ast.BasicLit, got %T", expr)
	}
	if lit.Kind != token.FLOAT {
		t.Errorf("expected token.FLOAT, got %v", lit.Kind)
	}
	if lit.Value != "3.14" {
		t.Errorf("expected value 3.14, got %s", lit.Value)
	}
}

func TestParseFlatValueString(t *testing.T) {
	expr, err := ParseFlatValue("string:hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		t.Fatalf("expected *ast.BasicLit, got %T", expr)
	}
	if lit.Kind != token.STRING {
		t.Errorf("expected token.STRING, got %v", lit.Kind)
	}
	if lit.Value != `"hello"` {
		t.Errorf("expected value %q, got %s", `"hello"`, lit.Value)
	}
}

func TestParseFlatValueStringWithColon(t *testing.T) {
	expr, err := ParseFlatValue("string:host:port")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		t.Fatalf("expected *ast.BasicLit, got %T", expr)
	}
	if lit.Value != `"host:port"` {
		t.Errorf("expected value %q, got %s", `"host:port"`, lit.Value)
	}
}

func TestParseFlatValueSelector(t *testing.T) {
	expr, err := ParseFlatValue("selector:s.Port")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		t.Fatalf("expected *ast.SelectorExpr, got %T", expr)
	}
	x, ok := sel.X.(*ast.Ident)
	if !ok {
		t.Fatalf("expected X to be *ast.Ident, got %T", sel.X)
	}
	if x.Name != "s" {
		t.Errorf("expected X.Name s, got %s", x.Name)
	}
	if sel.Sel.Name != "Port" {
		t.Errorf("expected Sel.Name Port, got %s", sel.Sel.Name)
	}
}

func TestParseFlatValueAddr(t *testing.T) {
	expr, err := ParseFlatValue("addr:cfg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok {
		t.Fatalf("expected *ast.UnaryExpr, got %T", expr)
	}
	if unary.Op != token.AND {
		t.Errorf("expected token.AND, got %v", unary.Op)
	}
	id, ok := unary.X.(*ast.Ident)
	if !ok {
		t.Fatalf("expected X to be *ast.Ident, got %T", unary.X)
	}
	if id.Name != "cfg" {
		t.Errorf("expected name cfg, got %s", id.Name)
	}
}

func TestParseFlatValueCall(t *testing.T) {
	expr, err := ParseFlatValue("call:fmt.Errorf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected *ast.CallExpr, got %T", expr)
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		t.Fatalf("expected Fun to be *ast.SelectorExpr, got %T", call.Fun)
	}
	if sel.Sel.Name != "Errorf" {
		t.Errorf("expected Errorf, got %s", sel.Sel.Name)
	}
	if len(call.Args) != 0 {
		t.Errorf("expected 0 args, got %d", len(call.Args))
	}
}

func TestParseFlatValueCallSimple(t *testing.T) {
	expr, err := ParseFlatValue("call:close")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected *ast.CallExpr, got %T", expr)
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		t.Fatalf("expected Fun to be *ast.Ident, got %T", call.Fun)
	}
	if id.Name != "close" {
		t.Errorf("expected close, got %s", id.Name)
	}
}

func TestParseFlatValues(t *testing.T) {
	exprs, err := ParseFlatValues("nil,ident:err")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exprs) != 2 {
		t.Fatalf("expected 2 exprs, got %d", len(exprs))
	}
	if exprs[0].(*ast.Ident).Name != "nil" {
		t.Errorf("expected nil, got %s", exprs[0].(*ast.Ident).Name)
	}
	if exprs[1].(*ast.Ident).Name != "err" {
		t.Errorf("expected err, got %s", exprs[1].(*ast.Ident).Name)
	}
}

func TestParseFlatValuesEmpty(t *testing.T) {
	exprs, err := ParseFlatValues("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exprs != nil {
		t.Errorf("expected nil, got %v", exprs)
	}
}

func TestParseFlatParams(t *testing.T) {
	fields, err := ParseFlatParams("ctx:context.Context,id:string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0].Names[0].Name != "ctx" {
		t.Errorf("expected name ctx, got %s", fields[0].Names[0].Name)
	}
	if fields[1].Names[0].Name != "id" {
		t.Errorf("expected name id, got %s", fields[1].Names[0].Name)
	}
}

func TestParseFlatParamsEmpty(t *testing.T) {
	fields, err := ParseFlatParams("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields != nil {
		t.Errorf("expected nil, got %v", fields)
	}
}

func TestParseReturnTypes(t *testing.T) {
	fl, err := ParseReturnTypes("*Config,error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fl.List) != 2 {
		t.Fatalf("expected 2 return types, got %d", len(fl.List))
	}
}

func TestParseReturnTypesEmpty(t *testing.T) {
	fl, err := ParseReturnTypes("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fl != nil {
		t.Errorf("expected nil, got %v", fl)
	}
}

func TestParseFlatValueErrors(t *testing.T) {
	_, err := ParseFlatValue("")
	if err == nil {
		t.Error("expected error for empty value")
	}

	_, err = ParseFlatValue("unknown:foo")
	if err == nil {
		t.Error("expected error for unknown kind")
	}

	_, err = ParseFlatValue("ident:")
	if err == nil {
		t.Error("expected error for empty ident")
	}

	_, err = ParseFlatValue("selector:nopoint")
	if err == nil {
		t.Error("expected error for selector without dot")
	}
}
