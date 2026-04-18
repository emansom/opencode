package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func edit(op Operation) (*EditResult, error) {
	// Normalize commonly confused parameters.
	// Gemma 4 sometimes sends "method" instead of "name" for create_method,
	// and "receiver" instead of "receiverType". Accept both.
	NormalizeOp(&op)

	src, err := os.ReadFile(op.File)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, op.File, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	var modified []byte
	switch op.Op {
	// Declaration creation
	case "create_function":
		modified, err = editCreateFunction(src, fset, file, op)
	case "create_method":
		modified, err = editCreateMethod(src, fset, file, op)
	case "create_struct":
		modified, err = editCreateStruct(src, fset, file, op)
	case "create_interface":
		modified, err = editCreateInterface(src, fset, file, op)
	case "create_type_alias":
		modified, err = editCreateTypeAlias(src, fset, file, op)
	case "create_named_type":
		modified, err = editCreateNamedType(src, fset, file, op)
	case "add_const":
		modified, err = editAddConst(src, fset, file, op)
	case "add_var":
		modified, err = editAddVar(src, fset, file, op)
	case "delete":
		modified, err = editDelete(src, fset, file, op)

	// Function/method signature
	case "add_parameter":
		modified, err = editAddParameter(src, fset, file, op)
	case "remove_parameter":
		modified, err = editRemoveParameter(src, fset, file, op)
	case "set_return_types":
		modified, err = editSetReturnTypes(src, fset, file, op)
	case "change_receiver":
		modified, err = editChangeReceiver(src, fset, file, op)
	case "set_function_name":
		modified, err = editSetFunctionName(src, fset, file, op)

	// Function/method body
	case "clear_body":
		modified, err = editClearBody(src, fset, file, op)
	case "remove_statement":
		modified, err = editRemoveStatement(src, fset, file, op)
	case "insert_call":
		modified, err = editInsertCall(src, fset, file, op)
	case "insert_method_call":
		modified, err = editInsertMethodCall(src, fset, file, op)
	case "insert_assign_call":
		modified, err = editInsertAssignCall(src, fset, file, op)
	case "insert_assign_value":
		modified, err = editInsertAssignValue(src, fset, file, op)
	case "insert_var_decl":
		modified, err = editInsertVarDecl(src, fset, file, op)
	case "insert_return":
		modified, err = editInsertReturn(src, fset, file, op)
	case "insert_if":
		modified, err = editInsertIf(src, fset, file, op)
	case "insert_if_init":
		modified, err = editInsertIfInit(src, fset, file, op)
	case "insert_for_range":
		modified, err = editInsertForRange(src, fset, file, op)
	case "insert_defer":
		modified, err = editInsertDefer(src, fset, file, op)
	case "insert_go":
		modified, err = editInsertGo(src, fset, file, op)
	case "insert_error_check":
		modified, err = editInsertErrorCheck(src, fset, file, op)
	case "push_arg":
		modified, err = editPushArg(src, fset, file, op)
	case "set_else":
		modified, err = editSetElse(src, fset, file, op)

	// Struct field editing
	case "add_struct_field":
		modified, err = editAddStructField(src, fset, file, op)
	case "remove_struct_field":
		modified, err = editRemoveStructField(src, fset, file, op)
	case "set_field_type":
		modified, err = editSetFieldType(src, fset, file, op)
	case "set_field_name":
		modified, err = editSetFieldName(src, fset, file, op)
	case "set_struct_tag":
		modified, err = editSetStructTag(src, fset, file, op)
	case "remove_struct_tag":
		modified, err = editRemoveStructTag(src, fset, file, op)

	// Interface editing
	case "add_interface_method":
		modified, err = editAddInterfaceMethod(src, fset, file, op)
	case "remove_interface_method":
		modified, err = editRemoveInterfaceMethod(src, fset, file, op)
	case "add_interface_embed":
		modified, err = editAddInterfaceEmbed(src, fset, file, op)

	// Import management
	case "add_import":
		modified, err = editAddImport(src, fset, file, op)
	case "remove_import":
		modified, err = editRemoveImport(src, fset, file, op)

	// Const/var editing
	case "set_const_value":
		modified, err = editSetConstValue(src, fset, file, op)
	case "set_var_value":
		modified, err = editSetVarValue(src, fset, file, op)
	case "set_const_type":
		modified, err = editSetConstType(src, fset, file, op)
	case "set_var_type":
		modified, err = editSetVarType(src, fset, file, op)
	case "remove_var":
		modified, err = editRemoveVar(src, fset, file, op)
	case "remove_const":
		modified, err = editRemoveConst(src, fset, file, op)

	// Package/file level
	case "set_package":
		modified, err = editSetPackage(src, fset, file, op)
	case "set_build_constraint":
		modified, err = editSetBuildConstraint(src, fset, file, op)
	case "add_generate_directive":
		modified, err = editAddGenerateDirective(src, fset, file, op)

	// Comments
	case "set_doc_comment":
		modified, err = editSetDocComment(src, fset, file, op)
	case "remove_doc_comment":
		modified, err = editRemoveDocComment(src, fset, file, op)
	case "add_line_comment":
		modified, err = editAddLineComment(src, fset, file, op)

	// Refactoring
	case "rename":
		modified, err = editRename(src, fset, file, op)
	case "extract_interface":
		modified, err = editExtractInterface(src, fset, file, op)

	// High-level structural replacement
	case "replace_body":
		modified, err = editReplaceBody(src, fset, file, op)
	case "replace_struct":
		modified, err = editReplaceStruct(src, fset, file, op)
	case "replace_interface":
		modified, err = editReplaceInterface(src, fset, file, op)
	case "replace_decl":
		modified, err = editReplaceDecl(src, fset, file, op)
	case "add_function_with_body":
		modified, err = editAddFunctionWithBody(src, fset, file, op)
	case "add_method_with_body":
		modified, err = editAddMethodWithBody(src, fset, file, op)
	case "replace_imports":
		modified, err = editReplaceImports(src, fset, file, op)
	case "replace_file":
		return editReplaceFile(src, op)
	case "insert_before_decl":
		modified, err = editInsertBeforeDecl(src, fset, file, op)
	case "insert_after_decl":
		modified, err = editInsertAfterDecl(src, fset, file, op)

	// Gopls-powered operations (cross-package, type-aware)
	case "gopls_rename":
		result, goplsErr := goplsRename(op)
		if goplsErr != nil {
			return nil, fmt.Errorf("gopls rename: %w", goplsErr)
		}
		return &EditResult{
			Success: result.Success,
			Diff:    result.Diff,
			Content: result.Content,
			Errors:  result.Errors,
		}, nil
	case "organize_imports":
		result, goplsErr := goplsOrganizeImports(op)
		if goplsErr != nil {
			return nil, fmt.Errorf("organize imports: %w", goplsErr)
		}
		return &EditResult{
			Success: result.Success,
			Diff:    result.Diff,
			Content: result.Content,
		}, nil

	default:
		return nil, fmt.Errorf("unknown operation: %q", op.Op)
	}

	if err != nil {
		return nil, err
	}

	// Format the result
	formatted, err := format.Source(modified)
	if err != nil {
		// Return unformatted with warning
		return &EditResult{
			Success: true,
			Content: string(modified),
			Diff:    computeDiff(string(src), string(modified)),
			Errors:  []string{fmt.Sprintf("gofmt warning: %v", err)},
		}, nil
	}

	return &EditResult{
		Success: true,
		Content: string(formatted),
		Diff:    computeDiff(string(src), string(formatted)),
	}, nil
}


// formatNode formats an AST file back to source code.
func formatNode(fset *token.FileSet, file *ast.File) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// getStmtList gets the target statement list for body operations.
func getStmtList(file *ast.File, fset *token.FileSet, target string) (*[]ast.Stmt, ast.Node, error) {
	resolved, err := ResolveTarget(file, fset, target)
	if err != nil {
		return nil, nil, err
	}

	switch n := resolved.Node.(type) {
	case *ast.FuncDecl:
		if n.Body == nil {
			n.Body = &ast.BlockStmt{}
		}
		return &n.Body.List, n, nil
	case *ast.BlockStmt:
		return &n.List, resolved.Parent, nil
	}

	if resolved.StmtList != nil {
		return resolved.StmtList, resolved.Parent, nil
	}

	return nil, nil, fmt.Errorf("target %q is not a function or block", target)
}

// insertStmt inserts a statement at the given position in a statement list.
func insertStmt(list *[]ast.Stmt, stmt ast.Stmt, position string, index *int) {
	switch position {
	case "first":
		*list = append([]ast.Stmt{stmt}, *list...)
	case "at_index":
		if index != nil && *index >= 0 && *index <= len(*list) {
			*list = append(*list, nil)
			copy((*list)[*index+1:], (*list)[*index:])
			(*list)[*index] = stmt
		} else {
			*list = append(*list, stmt)
		}
	default: // "last" or empty
		*list = append(*list, stmt)
	}
}

// computeDiff creates a simple unified diff between two strings.
func computeDiff(old, new string) string {
	if old == new {
		return ""
	}
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")

	var diff strings.Builder
	diff.WriteString("--- a/file.go\n+++ b/file.go\n")

	// Simple line-by-line diff
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	i, j := 0, 0
	for i < len(oldLines) || j < len(newLines) {
		if i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j] {
			i++
			j++
			continue
		}
		// Find changed region
		diff.WriteString(fmt.Sprintf("@@ -%d +%d @@\n", i+1, j+1))
		// Output removed lines
		for i < len(oldLines) && (j >= len(newLines) || oldLines[i] != newLines[j]) {
			diff.WriteString("-" + oldLines[i] + "\n")
			i++
		}
		// Output added lines
		for j < len(newLines) && (i >= len(oldLines) || oldLines[i] != newLines[j]) {
			diff.WriteString("+" + newLines[j] + "\n")
			j++
		}
	}

	return diff.String()
}

// --- Declaration creation operations ---

func editCreateFunction(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	params, err := ParseFlatParams(op.Params)
	if err != nil {
		return nil, fmt.Errorf("parse params: %w", err)
	}
	returns, err := ParseReturnTypes(op.Returns)
	if err != nil {
		return nil, fmt.Errorf("parse returns: %w", err)
	}

	decl := &ast.FuncDecl{
		Name: ast.NewIdent(op.Name),
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: params},
			Results: returns,
		},
		Body: &ast.BlockStmt{},
	}

	file.Decls = appendDeclAtAnchor(file.Decls, decl, op.Anchor, op.Position)
	return formatNode(fset, file)
}

func editCreateMethod(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	params, err := ParseFlatParams(op.Params)
	if err != nil {
		return nil, fmt.Errorf("parse params: %w", err)
	}
	returns, err := ParseReturnTypes(op.Returns)
	if err != nil {
		return nil, fmt.Errorf("parse returns: %w", err)
	}

	recvVar := op.ReceiverVar
	if recvVar == "" {
		recvVar = strings.ToLower(op.ReceiverType[:1])
	}

	decl := &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{{
				Names: []*ast.Ident{ast.NewIdent(recvVar)},
				Type:  parseTypeExpr(op.ReceiverType),
			}},
		},
		Name: ast.NewIdent(op.Name),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: params},
			Results: returns,
		},
		Body: &ast.BlockStmt{},
	}

	file.Decls = appendDeclAtAnchor(file.Decls, decl, op.Anchor, op.Position)
	return formatNode(fset, file)
}

func editCreateStruct(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(op.Name),
				Type: &ast.StructType{
					Fields: &ast.FieldList{},
				},
			},
		},
	}
	file.Decls = appendDeclAtAnchor(file.Decls, decl, op.Anchor, op.Position)
	return formatNode(fset, file)
}

func editCreateInterface(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(op.Name),
				Type: &ast.InterfaceType{
					Methods: &ast.FieldList{},
				},
			},
		},
	}
	file.Decls = appendDeclAtAnchor(file.Decls, decl, op.Anchor, op.Position)
	return formatNode(fset, file)
}

func editCreateTypeAlias(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name:   ast.NewIdent(op.Name),
				Assign: 1, // non-zero means alias
				Type:   parseTypeExpr(op.TargetType),
			},
		},
	}
	file.Decls = appendDeclAtAnchor(file.Decls, decl, op.Anchor, op.Position)
	return formatNode(fset, file)
}

func editCreateNamedType(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(op.Name),
				Type: parseTypeExpr(op.UnderlyingType),
			},
		},
	}
	file.Decls = appendDeclAtAnchor(file.Decls, decl, op.Anchor, op.Position)
	return formatNode(fset, file)
}

func editAddConst(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	val, err := buildValueExpr(op)
	if err != nil {
		return nil, err
	}

	spec := &ast.ValueSpec{
		Names: []*ast.Ident{ast.NewIdent(op.Name)},
	}
	if op.Type != "" {
		spec.Type = parseTypeExpr(op.Type)
	}
	if val != nil {
		spec.Values = []ast.Expr{val}
	}

	decl := &ast.GenDecl{
		Tok:   token.CONST,
		Specs: []ast.Spec{spec},
	}
	file.Decls = append(file.Decls, decl)
	return formatNode(fset, file)
}

func editAddVar(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	val, err := buildValueExpr(op)
	if err != nil {
		return nil, err
	}

	spec := &ast.ValueSpec{
		Names: []*ast.Ident{ast.NewIdent(op.Name)},
	}
	if op.Type != "" {
		spec.Type = parseTypeExpr(op.Type)
	}
	if val != nil {
		spec.Values = []ast.Expr{val}
	}

	decl := &ast.GenDecl{
		Tok:   token.VAR,
		Specs: []ast.Spec{spec},
	}
	file.Decls = append(file.Decls, decl)
	return formatNode(fset, file)
}

func editDelete(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	switch n := resolved.Node.(type) {
	case *ast.FuncDecl:
		var newDecls []ast.Decl
		for _, d := range file.Decls {
			if d != n {
				newDecls = append(newDecls, d)
			}
		}
		file.Decls = newDecls
	case *ast.TypeSpec:
		return deleteSpec(fset, file, resolved)
	case *ast.ValueSpec:
		return deleteSpec(fset, file, resolved)
	default:
		return nil, fmt.Errorf("cannot delete %T", n)
	}

	return formatNode(fset, file)
}

func deleteSpec(fset *token.FileSet, file *ast.File, resolved *ResolvedTarget) ([]byte, error) {
	genDecl, ok := resolved.Parent.(*ast.GenDecl)
	if !ok {
		return nil, fmt.Errorf("parent is not *ast.GenDecl")
	}

	if len(genDecl.Specs) == 1 {
		// Remove entire GenDecl
		var newDecls []ast.Decl
		for _, d := range file.Decls {
			if d != genDecl {
				newDecls = append(newDecls, d)
			}
		}
		file.Decls = newDecls
	} else {
		// Remove just this spec
		var newSpecs []ast.Spec
		for _, s := range genDecl.Specs {
			if s != resolved.Node {
				newSpecs = append(newSpecs, s)
			}
		}
		genDecl.Specs = newSpecs
	}

	return formatNode(fset, file)
}

// --- Signature operations ---

func editAddParameter(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	fd, ok := resolved.Node.(*ast.FuncDecl)
	if !ok {
		return nil, fmt.Errorf("target %q is not a function", op.Target)
	}

	newField := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(op.Name)},
		Type:  parseTypeExpr(op.Type),
	}

	if fd.Type.Params == nil {
		fd.Type.Params = &ast.FieldList{}
	}

	switch op.Position {
	case "first":
		fd.Type.Params.List = append([]*ast.Field{newField}, fd.Type.Params.List...)
	case "after":
		inserted := false
		for i, p := range fd.Type.Params.List {
			if len(p.Names) > 0 && p.Names[0].Name == op.Anchor {
				newList := make([]*ast.Field, 0, len(fd.Type.Params.List)+1)
				newList = append(newList, fd.Type.Params.List[:i+1]...)
				newList = append(newList, newField)
				newList = append(newList, fd.Type.Params.List[i+1:]...)
				fd.Type.Params.List = newList
				inserted = true
				break
			}
		}
		if !inserted {
			fd.Type.Params.List = append(fd.Type.Params.List, newField)
		}
	default: // "last" or empty
		fd.Type.Params.List = append(fd.Type.Params.List, newField)
	}

	return formatNode(fset, file)
}

func editRemoveParameter(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	// Target format: "FuncName.ParamName" or "Type.Method.ParamName"
	parts := strings.Split(op.Target, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("remove_parameter target must be dotted: %q", op.Target)
	}
	paramName := parts[len(parts)-1]
	funcTarget := strings.Join(parts[:len(parts)-1], ".")

	resolved, err := ResolveTarget(file, fset, funcTarget)
	if err != nil {
		return nil, err
	}
	fd, ok := resolved.Node.(*ast.FuncDecl)
	if !ok {
		return nil, fmt.Errorf("target %q is not a function", funcTarget)
	}

	if fd.Type.Params == nil {
		return nil, fmt.Errorf("function has no parameters")
	}

	var newList []*ast.Field
	found := false
	for _, p := range fd.Type.Params.List {
		if len(p.Names) > 0 && p.Names[0].Name == paramName {
			found = true
			continue
		}
		newList = append(newList, p)
	}
	if !found {
		return nil, fmt.Errorf("parameter %q not found", paramName)
	}
	fd.Type.Params.List = newList

	return formatNode(fset, file)
}

func editSetReturnTypes(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	fd, ok := resolved.Node.(*ast.FuncDecl)
	if !ok {
		return nil, fmt.Errorf("target %q is not a function", op.Target)
	}

	returns, err := ParseReturnTypes(op.Returns)
	if err != nil {
		return nil, err
	}
	fd.Type.Results = returns

	return formatNode(fset, file)
}

func editChangeReceiver(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	fd, ok := resolved.Node.(*ast.FuncDecl)
	if !ok {
		return nil, fmt.Errorf("target %q is not a function", op.Target)
	}

	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return nil, fmt.Errorf("function %q has no receiver", op.Target)
	}

	fd.Recv.List[0].Type = parseTypeExpr(op.ReceiverType)
	if op.ReceiverVar != "" {
		fd.Recv.List[0].Names = []*ast.Ident{ast.NewIdent(op.ReceiverVar)}
	}

	return formatNode(fset, file)
}

func editSetFunctionName(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	fd, ok := resolved.Node.(*ast.FuncDecl)
	if !ok {
		return nil, fmt.Errorf("target %q is not a function", op.Target)
	}

	fd.Name = ast.NewIdent(op.Name)
	return formatNode(fset, file)
}

// --- Body operations ---

func editClearBody(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	fd, ok := resolved.Node.(*ast.FuncDecl)
	if !ok {
		return nil, fmt.Errorf("target %q is not a function", op.Target)
	}
	fd.Body.List = nil
	return formatNode(fset, file)
}

func editRemoveStatement(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	idx := 0
	if op.Index != nil {
		idx = *op.Index
	}
	if idx < 0 || idx >= len(*stmts) {
		return nil, fmt.Errorf("index %d out of range (have %d statements)", idx, len(*stmts))
	}
	*stmts = append((*stmts)[:idx], (*stmts)[idx+1:]...)
	return formatNode(fset, file)
}

func editInsertCall(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	args, err := ParseFlatValues(op.Args)
	if err != nil {
		return nil, err
	}

	stmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  parseFuncExpr(op.Func),
			Args: args,
		},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertMethodCall(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	args, err := ParseFlatValues(op.Args)
	if err != nil {
		return nil, err
	}

	stmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(op.Receiver),
				Sel: ast.NewIdent(op.Method),
			},
			Args: args,
		},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertAssignCall(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	args, err := ParseFlatValues(op.Args)
	if err != nil {
		return nil, err
	}

	varNames := splitCSV(op.Vars)
	lhs := make([]ast.Expr, len(varNames))
	for i, v := range varNames {
		lhs[i] = ast.NewIdent(v)
	}

	tok := token.ASSIGN
	if op.Short != nil && *op.Short {
		tok = token.DEFINE
	}

	var fun ast.Expr
	if op.Receiver != "" && op.Method != "" {
		fun = &ast.SelectorExpr{
			X:   parseFuncExpr(op.Receiver),
			Sel: ast.NewIdent(op.Method),
		}
	} else {
		fun = parseFuncExpr(op.Func)
	}

	stmt := &ast.AssignStmt{
		Lhs: lhs,
		Tok: tok,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun:  fun,
				Args: args,
			},
		},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertAssignValue(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	val, err := ParseFlatValue(op.ValueSpec)
	if err != nil {
		return nil, err
	}

	stmt := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent(op.VarName)},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{val},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertVarDecl(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	stmt := &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{ast.NewIdent(op.VarName)},
					Type:  parseTypeExpr(op.VarType),
				},
			},
		},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertReturn(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	vals, err := ParseFlatValues(op.Values)
	if err != nil {
		return nil, err
	}

	stmt := &ast.ReturnStmt{Results: vals}
	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertIf(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	left, err := ParseFlatValue(op.CondLeft)
	if err != nil {
		return nil, fmt.Errorf("parse condLeft: %w", err)
	}
	right, err := ParseFlatValue(op.CondRight)
	if err != nil {
		return nil, fmt.Errorf("parse condRight: %w", err)
	}

	opTok, err := parseOperator(op.Operator)
	if err != nil {
		return nil, err
	}

	stmt := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  left,
			Op: opTok,
			Y:  right,
		},
		Body: &ast.BlockStmt{},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertIfInit(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	left, err := ParseFlatValue(op.CondLeft)
	if err != nil {
		return nil, fmt.Errorf("parse condLeft: %w", err)
	}
	right, err := ParseFlatValue(op.CondRight)
	if err != nil {
		return nil, fmt.Errorf("parse condRight: %w", err)
	}
	opTok, err := parseOperator(op.Operator)
	if err != nil {
		return nil, err
	}

	initArgs, err := ParseFlatValues(op.InitArgs)
	if err != nil {
		return nil, fmt.Errorf("parse initArgs: %w", err)
	}

	initVars := splitCSV(op.InitVars)
	lhs := make([]ast.Expr, len(initVars))
	for i, v := range initVars {
		lhs[i] = ast.NewIdent(v)
	}

	stmt := &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: lhs,
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun:  parseFuncExpr(op.InitFunc),
					Args: initArgs,
				},
			},
		},
		Cond: &ast.BinaryExpr{
			X:  left,
			Op: opTok,
			Y:  right,
		},
		Body: &ast.BlockStmt{},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertForRange(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	iterExpr, err := ParseFlatValue(op.Iterable)
	if err != nil {
		return nil, fmt.Errorf("parse iterable: %w", err)
	}

	var keyExpr, valExpr ast.Expr
	if op.Key != "" {
		keyExpr = ast.NewIdent(op.Key)
	} else {
		keyExpr = ast.NewIdent("_")
	}
	if op.Value != "" {
		valExpr = ast.NewIdent(op.Value)
	}

	stmt := &ast.RangeStmt{
		Key:   keyExpr,
		Value: valExpr,
		Tok:   token.DEFINE,
		X:     iterExpr,
		Body:  &ast.BlockStmt{},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertDefer(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	args, err := ParseFlatValues(op.Args)
	if err != nil {
		return nil, err
	}

	stmt := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun:  parseFuncExpr(op.Func),
			Args: args,
		},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertGo(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	args, err := ParseFlatValues(op.Args)
	if err != nil {
		return nil, err
	}

	stmt := &ast.GoStmt{
		Call: &ast.CallExpr{
			Fun:  parseFuncExpr(op.Func),
			Args: args,
		},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editInsertErrorCheck(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	retVals, err := ParseFlatValues(op.ReturnValues)
	if err != nil {
		return nil, err
	}

	stmt := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  ast.NewIdent(op.ErrVar),
			Op: token.NEQ,
			Y:  ast.NewIdent("nil"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{Results: retVals},
			},
		},
	}

	insertStmt(stmts, stmt, op.Position, op.Index)
	return formatNode(fset, file)
}

func editPushArg(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	if op.StmtIndex == nil {
		return nil, fmt.Errorf("stmtIndex is required for push_arg")
	}
	idx := *op.StmtIndex
	if idx < 0 || idx >= len(*stmts) {
		return nil, fmt.Errorf("stmtIndex %d out of range", idx)
	}

	argExpr, err := ParseFlatValue(op.Arg)
	if err != nil {
		return nil, err
	}

	call := findCall((*stmts)[idx], op.CallIndex)
	if call == nil {
		return nil, fmt.Errorf("no call expression found at statement index %d", idx)
	}

	call.Args = append(call.Args, argExpr)
	return formatNode(fset, file)
}

func editSetElse(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	stmts, _, err := getStmtList(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	if op.StmtIndex == nil {
		return nil, fmt.Errorf("stmtIndex is required for set_else")
	}
	idx := *op.StmtIndex
	if idx < 0 || idx >= len(*stmts) {
		return nil, fmt.Errorf("stmtIndex %d out of range", idx)
	}

	ifStmt, ok := (*stmts)[idx].(*ast.IfStmt)
	if !ok {
		return nil, fmt.Errorf("statement at index %d is not an if statement", idx)
	}

	ifStmt.Else = &ast.BlockStmt{}
	return formatNode(fset, file)
}

// --- Struct field operations ---

func editAddStructField(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}

	var structType *ast.StructType
	switch n := resolved.Node.(type) {
	case *ast.TypeSpec:
		st, ok := n.Type.(*ast.StructType)
		if !ok {
			return nil, fmt.Errorf("target %q is not a struct", op.Target)
		}
		structType = st
	default:
		return nil, fmt.Errorf("target %q is not a type", op.Target)
	}

	newField := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(op.FieldName)},
		Type:  parseTypeExpr(op.FieldType),
	}
	if op.Tag != "" {
		newField.Tag = &ast.BasicLit{Kind: token.STRING, Value: "`" + op.Tag + "`"}
	}

	if structType.Fields == nil {
		structType.Fields = &ast.FieldList{}
	}

	switch op.Position {
	case "first":
		structType.Fields.List = append([]*ast.Field{newField}, structType.Fields.List...)
	case "after":
		inserted := false
		for i, f := range structType.Fields.List {
			if len(f.Names) > 0 && f.Names[0].Name == op.Anchor {
				newList := make([]*ast.Field, 0, len(structType.Fields.List)+1)
				newList = append(newList, structType.Fields.List[:i+1]...)
				newList = append(newList, newField)
				newList = append(newList, structType.Fields.List[i+1:]...)
				structType.Fields.List = newList
				inserted = true
				break
			}
		}
		if !inserted {
			structType.Fields.List = append(structType.Fields.List, newField)
		}
	default: // "last" or empty
		structType.Fields.List = append(structType.Fields.List, newField)
	}

	return formatNode(fset, file)
}

func editRemoveStructField(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	// Target: "Struct.Field"
	parts := strings.SplitN(op.Target, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("remove_struct_field target must be Struct.Field: %q", op.Target)
	}

	resolved, err := ResolveTarget(file, fset, parts[0])
	if err != nil {
		return nil, err
	}
	ts, ok := resolved.Node.(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("target %q is not a type", parts[0])
	}
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("target %q is not a struct", parts[0])
	}

	var newFields []*ast.Field
	found := false
	for _, f := range st.Fields.List {
		if len(f.Names) > 0 && f.Names[0].Name == parts[1] {
			found = true
			continue
		}
		newFields = append(newFields, f)
	}
	if !found {
		return nil, fmt.Errorf("field %q not found in struct %q", parts[1], parts[0])
	}
	st.Fields.List = newFields

	return formatNode(fset, file)
}

func editSetFieldType(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	f, ok := resolved.Node.(*ast.Field)
	if !ok {
		return nil, fmt.Errorf("target %q is not a field", op.Target)
	}
	f.Type = parseTypeExpr(op.FieldType)
	return formatNode(fset, file)
}

func editSetFieldName(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	f, ok := resolved.Node.(*ast.Field)
	if !ok {
		return nil, fmt.Errorf("target %q is not a field", op.Target)
	}
	if len(f.Names) == 0 {
		return nil, fmt.Errorf("field has no name (embedded)")
	}
	f.Names[0] = ast.NewIdent(op.FieldName)
	return formatNode(fset, file)
}

func editSetStructTag(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	f, ok := resolved.Node.(*ast.Field)
	if !ok {
		return nil, fmt.Errorf("target %q is not a field", op.Target)
	}

	raw := ""
	if f.Tag != nil {
		raw = strings.Trim(f.Tag.Value, "`")
	}
	raw = SetTagKey(raw, op.TagKey, op.TagValue)
	f.Tag = &ast.BasicLit{Kind: token.STRING, Value: "`" + raw + "`"}

	return formatNode(fset, file)
}

func editRemoveStructTag(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	f, ok := resolved.Node.(*ast.Field)
	if !ok {
		return nil, fmt.Errorf("target %q is not a field", op.Target)
	}

	if f.Tag == nil {
		return formatNode(fset, file)
	}

	raw := strings.Trim(f.Tag.Value, "`")
	raw = RemoveTagKey(raw, op.TagKey)
	if raw == "" {
		f.Tag = nil
	} else {
		f.Tag = &ast.BasicLit{Kind: token.STRING, Value: "`" + raw + "`"}
	}

	return formatNode(fset, file)
}

// --- Interface operations ---

func editAddInterfaceMethod(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	ts, ok := resolved.Node.(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("target %q is not a type", op.Target)
	}
	iface, ok := ts.Type.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("target %q is not an interface", op.Target)
	}

	params, err := ParseFlatParams(op.Params)
	if err != nil {
		return nil, err
	}
	returns, err := ParseReturnTypes(op.Returns)
	if err != nil {
		return nil, err
	}

	method := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(op.MethodName)},
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: params},
			Results: returns,
		},
	}

	if iface.Methods == nil {
		iface.Methods = &ast.FieldList{}
	}
	iface.Methods.List = append(iface.Methods.List, method)

	return formatNode(fset, file)
}

func editRemoveInterfaceMethod(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	parts := strings.SplitN(op.Target, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("target must be Interface.Method: %q", op.Target)
	}

	resolved, err := ResolveTarget(file, fset, parts[0])
	if err != nil {
		return nil, err
	}
	ts, ok := resolved.Node.(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("target %q is not a type", parts[0])
	}
	iface, ok := ts.Type.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("target %q is not an interface", parts[0])
	}

	var newMethods []*ast.Field
	found := false
	for _, m := range iface.Methods.List {
		if len(m.Names) > 0 && m.Names[0].Name == parts[1] {
			found = true
			continue
		}
		newMethods = append(newMethods, m)
	}
	if !found {
		return nil, fmt.Errorf("method %q not found in interface %q", parts[1], parts[0])
	}
	iface.Methods.List = newMethods

	return formatNode(fset, file)
}

func editAddInterfaceEmbed(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	ts, ok := resolved.Node.(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("target %q is not a type", op.Target)
	}
	iface, ok := ts.Type.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("target %q is not an interface", op.Target)
	}

	embed := &ast.Field{
		Type: parseTypeExpr(op.EmbedType),
	}

	if iface.Methods == nil {
		iface.Methods = &ast.FieldList{}
	}
	iface.Methods.List = append(iface.Methods.List, embed)

	return formatNode(fset, file)
}

// --- Import operations ---

func editAddImport(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	if op.Alias != "" {
		astutil.AddNamedImport(fset, file, op.Alias, op.Path)
	} else {
		astutil.AddImport(fset, file, op.Path)
	}
	return formatNode(fset, file)
}

func editRemoveImport(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	astutil.DeleteImport(fset, file, op.Path)
	ast.SortImports(fset, file)
	return formatNode(fset, file)
}

// --- Const/var editing ---

func editSetConstValue(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	return setValueSpecValue(file, fset, op, token.CONST)
}

func editSetVarValue(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	return setValueSpecValue(file, fset, op, token.VAR)
}

func setValueSpecValue(file *ast.File, fset *token.FileSet, op Operation, tok token.Token) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	vs, ok := resolved.Node.(*ast.ValueSpec)
	if !ok {
		return nil, fmt.Errorf("target %q is not a value spec", op.Target)
	}

	val, err := buildValueExpr(op)
	if err != nil {
		return nil, err
	}
	if val != nil {
		vs.Values = []ast.Expr{val}
	}

	return formatNode(fset, file)
}

func editSetConstType(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	return setValueSpecType(file, fset, op)
}

func editSetVarType(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	return setValueSpecType(file, fset, op)
}

func setValueSpecType(file *ast.File, fset *token.FileSet, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	vs, ok := resolved.Node.(*ast.ValueSpec)
	if !ok {
		return nil, fmt.Errorf("target %q is not a value spec", op.Target)
	}
	vs.Type = parseTypeExpr(op.Type)
	return formatNode(fset, file)
}

func editRemoveVar(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	return editDelete(src, fset, file, op)
}

func editRemoveConst(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	return editDelete(src, fset, file, op)
}

// --- Package/file level ---

func editSetPackage(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	file.Name = ast.NewIdent(op.Name)
	return formatNode(fset, file)
}

func editSetBuildConstraint(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	formatted, err := formatNode(fset, file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(formatted), "\n")
	var result []string

	// Remove existing build constraint
	for _, line := range lines {
		if !strings.HasPrefix(line, "//go:build ") {
			result = append(result, line)
		}
	}

	// Add new constraint before package declaration
	var final []string
	added := false
	for _, line := range result {
		if !added && strings.HasPrefix(line, "package ") {
			final = append(final, "//go:build "+op.Constraint)
			final = append(final, "")
			added = true
		}
		final = append(final, line)
	}

	return []byte(strings.Join(final, "\n")), nil
}

func editAddGenerateDirective(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	formatted, err := formatNode(fset, file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(formatted), "\n")
	directive := "//go:generate " + op.Directive

	// Add before package declaration
	var result []string
	added := false
	for _, line := range lines {
		if !added && strings.HasPrefix(line, "package ") {
			result = append(result, directive)
			result = append(result, "")
			added = true
		}
		result = append(result, line)
	}

	return []byte(strings.Join(result, "\n")), nil
}

// --- Comment operations ---

func editSetDocComment(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	formatted, err := formatNode(fset, file)
	if err != nil {
		return nil, err
	}

	// Re-parse to get fresh positions
	fset2 := token.NewFileSet()
	file2, err := parser.ParseFile(fset2, "", formatted, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	resolved, err := ResolveTarget(file2, fset2, op.Target)
	if err != nil {
		return nil, err
	}

	var pos token.Pos
	switch n := resolved.Node.(type) {
	case *ast.FuncDecl:
		pos = n.Pos()
	case *ast.TypeSpec:
		if gd, ok := resolved.Parent.(*ast.GenDecl); ok && len(gd.Specs) == 1 {
			pos = gd.Pos()
		} else {
			pos = n.Pos()
		}
	case *ast.ValueSpec:
		if gd, ok := resolved.Parent.(*ast.GenDecl); ok && len(gd.Specs) == 1 {
			pos = gd.Pos()
		} else {
			pos = n.Pos()
		}
	default:
		return nil, fmt.Errorf("cannot set doc comment on %T", n)
	}

	lines := strings.Split(string(formatted), "\n")
	targetLine := fset2.Position(pos).Line - 1 // 0-indexed

	// Build comment lines
	commentLines := []string{}
	for _, line := range strings.Split(op.Text, "\n") {
		commentLines = append(commentLines, "// "+line)
	}

	// Remove existing doc comment (contiguous // lines immediately before target)
	startRemove := targetLine
	for startRemove > 0 && strings.HasPrefix(strings.TrimSpace(lines[startRemove-1]), "//") {
		startRemove--
	}

	// Build result
	var result []string
	result = append(result, lines[:startRemove]...)
	result = append(result, commentLines...)
	result = append(result, lines[targetLine:]...)

	return []byte(strings.Join(result, "\n")), nil
}

func editRemoveDocComment(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	formatted, err := formatNode(fset, file)
	if err != nil {
		return nil, err
	}

	fset2 := token.NewFileSet()
	file2, err := parser.ParseFile(fset2, "", formatted, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	resolved, err := ResolveTarget(file2, fset2, op.Target)
	if err != nil {
		return nil, err
	}

	var pos token.Pos
	switch n := resolved.Node.(type) {
	case *ast.FuncDecl:
		pos = n.Pos()
	case *ast.TypeSpec:
		if gd, ok := resolved.Parent.(*ast.GenDecl); ok && len(gd.Specs) == 1 {
			pos = gd.Pos()
		} else {
			pos = n.Pos()
		}
	default:
		return nil, fmt.Errorf("cannot remove doc comment from %T", n)
	}

	lines := strings.Split(string(formatted), "\n")
	targetLine := fset2.Position(pos).Line - 1

	startRemove := targetLine
	for startRemove > 0 && strings.HasPrefix(strings.TrimSpace(lines[startRemove-1]), "//") {
		startRemove--
	}

	if startRemove == targetLine {
		return formatted, nil // no comment to remove
	}

	var result []string
	result = append(result, lines[:startRemove]...)
	result = append(result, lines[targetLine:]...)

	return []byte(strings.Join(result, "\n")), nil
}

func editAddLineComment(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	formatted, err := formatNode(fset, file)
	if err != nil {
		return nil, err
	}

	fset2 := token.NewFileSet()
	file2, err := parser.ParseFile(fset2, "", formatted, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	resolved, err := ResolveTarget(file2, fset2, op.Target)
	if err != nil {
		return nil, err
	}

	var pos token.Pos
	switch n := resolved.Node.(type) {
	case *ast.FuncDecl:
		pos = n.Pos()
	case *ast.TypeSpec:
		pos = n.Pos()
	case *ast.Field:
		pos = n.Pos()
	case *ast.ValueSpec:
		pos = n.Pos()
	default:
		pos = resolved.Node.Pos()
	}

	lines := strings.Split(string(formatted), "\n")
	targetLine := fset2.Position(pos).Line - 1

	if targetLine >= 0 && targetLine < len(lines) {
		lines[targetLine] = lines[targetLine] + " // " + op.Text
	}

	return []byte(strings.Join(lines, "\n")), nil
}

// --- Refactoring operations ---

func editRename(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	oldName := op.Target
	newName := op.NewName

	// Walk all identifiers and rename matches
	ast.Inspect(file, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == oldName {
			id.Name = newName
		}
		return true
	})

	return formatNode(fset, file)
}

func editExtractInterface(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	typeName := op.Target
	ifaceName := op.InterfaceName

	var methods []*ast.Field
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			continue
		}
		recvType := baseReceiverType(fd.Recv.List[0].Type)
		if recvType == typeName && ast.IsExported(fd.Name.Name) {
			m := &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(fd.Name.Name)},
				Type:  fd.Type,
			}
			methods = append(methods, m)
		}
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no exported methods found for type %q", typeName)
	}

	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: ast.NewIdent(ifaceName),
				Type: &ast.InterfaceType{
					Methods: &ast.FieldList{List: methods},
				},
			},
		},
	}

	file.Decls = append(file.Decls, decl)
	return formatNode(fset, file)
}

// --- Helpers ---

func buildValueExpr(op Operation) (ast.Expr, error) {
	if op.ValueFunc != "" {
		args, err := ParseFlatValues(op.ValueArgs)
		if err != nil {
			return nil, fmt.Errorf("parse valueArgs: %w", err)
		}
		return &ast.CallExpr{
			Fun:  parseFuncExpr(op.ValueFunc),
			Args: args,
		}, nil
	}
	if op.ValueSpec != "" {
		return ParseFlatValue(op.ValueSpec)
	}
	return nil, nil
}

func parseOperator(s string) (token.Token, error) {
	switch s {
	case "==":
		return token.EQL, nil
	case "!=":
		return token.NEQ, nil
	case "<":
		return token.LSS, nil
	case ">":
		return token.GTR, nil
	case "<=":
		return token.LEQ, nil
	case ">=":
		return token.GEQ, nil
	default:
		return 0, fmt.Errorf("unknown operator: %q", s)
	}
}

func findCall(stmt ast.Stmt, callIndex *int) *ast.CallExpr {
	idx := 0
	if callIndex != nil {
		idx = *callIndex
	}

	var calls []*ast.CallExpr
	ast.Inspect(stmt, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			calls = append(calls, c)
		}
		return true
	})

	if idx >= 0 && idx < len(calls) {
		return calls[idx]
	}
	return nil
}

// --- High-level structural replacement operations ---

// parseSyntheticBody parses a function body from source text.
// The body is the content between { and }, e.g. "return 42\n".
func parseSyntheticBody(body string) (*ast.BlockStmt, error) {
	synthetic := fmt.Sprintf("package _\nfunc _() {\n%s\n}", body)
	synFset := token.NewFileSet()
	synFile, err := parser.ParseFile(synFset, "", synthetic, 0)
	if err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}
	return synFile.Decls[0].(*ast.FuncDecl).Body, nil
}

// parseSyntheticDecl parses a complete top-level declaration from source text.
// source must be a valid Go declaration (func, type, var, const).
func parseSyntheticDecl(source string) (ast.Decl, error) {
	synthetic := fmt.Sprintf("package _\n%s", source)
	synFset := token.NewFileSet()
	synFile, err := parser.ParseFile(synFset, "", synthetic, 0)
	if err != nil {
		return nil, fmt.Errorf("parse declaration: %w", err)
	}
	if len(synFile.Decls) == 0 {
		return nil, fmt.Errorf("no declarations found in source")
	}
	return synFile.Decls[0], nil
}

// editReplaceBody replaces the body of a function or method.
// op.Target is the dotted path (e.g. "handleRequest" or "Server.Start").
// op.Body is the new body content between { and }.
func editReplaceBody(_ []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	fd, ok := resolved.Node.(*ast.FuncDecl)
	if !ok {
		return nil, fmt.Errorf("target %q is not a function or method", op.Target)
	}
	newBody, err := parseSyntheticBody(op.Body)
	if err != nil {
		return nil, err
	}
	fd.Body = newBody
	return formatNode(fset, file)
}

// editReplaceStruct replaces all fields of a struct type.
// op.Target is the struct name. op.Body is the new field list content.
func editReplaceStruct(_ []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	ts, ok := resolved.Node.(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("target %q is not a type", op.Target)
	}
	st, ok := ts.Type.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("target %q is not a struct", op.Target)
	}
	synthetic := fmt.Sprintf("package _\ntype _ struct {\n%s\n}", op.Body)
	synFset := token.NewFileSet()
	synFile, err := parser.ParseFile(synFset, "", synthetic, 0)
	if err != nil {
		return nil, fmt.Errorf("parse struct body: %w", err)
	}
	synTs := synFile.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	st.Fields = synTs.Type.(*ast.StructType).Fields
	return formatNode(fset, file)
}

// editReplaceInterface replaces all methods of an interface type.
// op.Target is the interface name. op.Body is the new method list content.
func editReplaceInterface(_ []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	resolved, err := ResolveTarget(file, fset, op.Target)
	if err != nil {
		return nil, err
	}
	ts, ok := resolved.Node.(*ast.TypeSpec)
	if !ok {
		return nil, fmt.Errorf("target %q is not a type", op.Target)
	}
	it, ok := ts.Type.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("target %q is not an interface", op.Target)
	}
	synthetic := fmt.Sprintf("package _\ntype _ interface {\n%s\n}", op.Body)
	synFset := token.NewFileSet()
	synFile, err := parser.ParseFile(synFset, "", synthetic, 0)
	if err != nil {
		return nil, fmt.Errorf("parse interface body: %w", err)
	}
	synTs := synFile.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	it.Methods = synTs.Type.(*ast.InterfaceType).Methods
	return formatNode(fset, file)
}

// editReplaceDecl replaces an entire declaration (function, method, or type) with new source.
// op.Target is the dotted path. op.Source is the complete new declaration source.
func editReplaceDecl(_ []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	newDecl, err := parseSyntheticDecl(op.Source)
	if err != nil {
		return nil, err
	}
	target := op.Target
	for i, decl := range file.Decls {
		name := declName(decl)
		if name == target {
			file.Decls[i] = newDecl
			return formatNode(fset, file)
		}
	}
	return nil, fmt.Errorf("target %q not found", target)
}

// editAddFunctionWithBody creates a new function with a complete body.
// op.Name is the function name. op.Params, op.Returns define the signature. op.Body is the body content.
func editAddFunctionWithBody(_ []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	params, err := ParseFlatParams(op.Params)
	if err != nil {
		return nil, fmt.Errorf("parse params: %w", err)
	}
	returns, err := ParseReturnTypes(op.Returns)
	if err != nil {
		return nil, fmt.Errorf("parse returns: %w", err)
	}
	body, err := parseSyntheticBody(op.Body)
	if err != nil {
		return nil, err
	}
	decl := &ast.FuncDecl{
		Name: ast.NewIdent(op.Name),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: params},
			Results: returns,
		},
		Body: body,
	}
	file.Decls = appendDeclAtAnchor(file.Decls, decl, op.Anchor, op.Position)
	return formatNode(fset, file)
}

// editAddMethodWithBody creates a new method with a complete body.
func editAddMethodWithBody(_ []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	params, err := ParseFlatParams(op.Params)
	if err != nil {
		return nil, fmt.Errorf("parse params: %w", err)
	}
	returns, err := ParseReturnTypes(op.Returns)
	if err != nil {
		return nil, fmt.Errorf("parse returns: %w", err)
	}
	body, err := parseSyntheticBody(op.Body)
	if err != nil {
		return nil, err
	}
	recvVar := op.ReceiverVar
	if recvVar == "" {
		recvVar = strings.ToLower(op.ReceiverType[:1])
	}
	decl := &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{{
				Names: []*ast.Ident{ast.NewIdent(recvVar)},
				Type:  parseTypeExpr(op.ReceiverType),
			}},
		},
		Name: ast.NewIdent(op.Name),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: params},
			Results: returns,
		},
		Body: body,
	}
	file.Decls = appendDeclAtAnchor(file.Decls, decl, op.Anchor, op.Position)
	return formatNode(fset, file)
}

// editReplaceImports replaces the entire import block with new imports.
// op.Imports is a newline-separated list of import specs, e.g.:
//
//	"net/http"
//	alias "pkg/path"
func editReplaceImports(src []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	// Remove all existing imports
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		astutil.DeleteNamedImport(fset, file, alias, path)
	}

	// Parse and add new imports
	lines := strings.Split(strings.TrimSpace(op.Imports), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse: optional alias then quoted path
		var alias, path string
		if strings.HasPrefix(line, `"`) {
			path = strings.Trim(line, `"`)
		} else {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				alias = parts[0]
				path = strings.Trim(parts[1], `"`)
			} else {
				path = strings.Trim(parts[0], `"`)
			}
		}
		if path != "" {
			astutil.AddNamedImport(fset, file, alias, path)
		}
	}

	return formatNode(fset, file)
}

// editReplaceFile replaces the entire file content with new source.
// op.Source must be a complete valid Go source file including package declaration.
func editReplaceFile(_ []byte, op Operation) (*EditResult, error) {
	synFset := token.NewFileSet()
	_, err := parser.ParseFile(synFset, op.File, op.Source, 0)
	if err != nil {
		return nil, fmt.Errorf("parse new source: %w", err)
	}
	formatted, err := format.Source([]byte(op.Source))
	if err != nil {
		return nil, fmt.Errorf("format new source: %w", err)
	}
	old, readErr := os.ReadFile(op.File)
	if readErr != nil {
		old = []byte{}
	}
	if err := os.WriteFile(op.File, formatted, 0o644); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	return &EditResult{
		Success: true,
		Content: string(formatted),
		Diff:    computeDiff(string(old), string(formatted)),
	}, nil
}

// editInsertBeforeDecl inserts a source declaration immediately before the named declaration.
// op.Target is the target declaration name. op.Source is the Go source to insert.
func editInsertBeforeDecl(_ []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	newDecl, err := parseSyntheticDecl(op.Source)
	if err != nil {
		return nil, err
	}
	file.Decls = appendDeclAtAnchor(file.Decls, newDecl, op.Target, "before")
	return formatNode(fset, file)
}

// editInsertAfterDecl inserts a source declaration immediately after the named declaration.
// op.Target is the target declaration name. op.Source is the Go source to insert.
func editInsertAfterDecl(_ []byte, fset *token.FileSet, file *ast.File, op Operation) ([]byte, error) {
	newDecl, err := parseSyntheticDecl(op.Source)
	if err != nil {
		return nil, err
	}
	file.Decls = appendDeclAtAnchor(file.Decls, newDecl, op.Target, "after")
	return formatNode(fset, file)
}

func appendDeclAtAnchor(decls []ast.Decl, newDecl ast.Decl, anchor, position string) []ast.Decl {
	if anchor == "" {
		return append(decls, newDecl)
	}

	for i, d := range decls {
		name := declName(d)
		if name == anchor {
			switch position {
			case "before":
				result := make([]ast.Decl, 0, len(decls)+1)
				result = append(result, decls[:i]...)
				result = append(result, newDecl)
				result = append(result, decls[i:]...)
				return result
			case "after":
				result := make([]ast.Decl, 0, len(decls)+1)
				result = append(result, decls[:i+1]...)
				result = append(result, newDecl)
				result = append(result, decls[i+1:]...)
				return result
			}
		}
	}

	return append(decls, newDecl)
}

func declName(d ast.Decl) string {
	switch n := d.(type) {
	case *ast.FuncDecl:
		if n.Recv != nil {
			return baseReceiverType(n.Recv.List[0].Type) + "." + n.Name.Name
		}
		return n.Name.Name
	case *ast.GenDecl:
		if len(n.Specs) > 0 {
			switch s := n.Specs[0].(type) {
			case *ast.TypeSpec:
				return s.Name.Name
			case *ast.ValueSpec:
				if len(s.Names) > 0 {
					return s.Names[0].Name
				}
			}
		}
	}
	return ""
}

// Keep astutil import used
var _ = astutil.AddImport
