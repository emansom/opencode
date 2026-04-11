package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"unicode"
)

func inspect(filePath string) (*InspectResult, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	result := &InspectResult{
		Package: file.Name.Name,
		Imports: []ImportInfo{},
		Types:   []TypeInfo{},
		Functions: []FunctionInfo{},
		Consts:  []VarConstInfo{},
		Vars:    []VarConstInfo{},
	}

	// Build constraint
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "//go:build ") {
				result.BuildConstraint = strings.TrimPrefix(c.Text, "//go:build ")
			}
			if strings.HasPrefix(c.Text, "//go:generate ") {
				result.GenerateDirectives = append(result.GenerateDirectives, c.Text)
			}
		}
	}

	// Imports
	for _, imp := range file.Imports {
		info := ImportInfo{
			Path: strings.Trim(imp.Path.Value, `"`),
		}
		if imp.Name != nil {
			info.Alias = imp.Name.Name
		}
		result.Imports = append(result.Imports, info)
	}

	// Walk declarations in file order, tracking global order for occurrence counting.
	var declOrder []occRef

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			beforeTypes := len(result.Types)
			beforeConsts := len(result.Consts)
			beforeVars := len(result.Vars)
			inspectGenDecl(fset, d, result)
			for i := beforeTypes; i < len(result.Types); i++ {
				declOrder = append(declOrder, occRef{result.Types[i].Name, "type", i})
			}
			for i := beforeConsts; i < len(result.Consts); i++ {
				declOrder = append(declOrder, occRef{result.Consts[i].Name, "const", i})
			}
			for i := beforeVars; i < len(result.Vars); i++ {
				declOrder = append(declOrder, occRef{result.Vars[i].Name, "var", i})
			}
		case *ast.FuncDecl:
			fi := inspectFunc(fset, d)
			name := fi.Name
			if fi.Receiver != "" {
				if dot := strings.Index(fi.Name, "."); dot >= 0 {
					name = fi.Name[:dot]
				}
			}
			idx := len(result.Functions)
			result.Functions = append(result.Functions, fi)
			declOrder = append(declOrder, occRef{name, "func", idx})
		}
	}

	// Assign occurrence numbers for duplicate declarations.
	// Counted globally in file order (matching the resolver's counting).
	assignOccurrences(result, declOrder)

	return result, nil
}

// declRef is defined in the calling function (inspect). This type alias is for
// the assignOccurrences parameter — Go doesn't allow re-declaring the type here
// so we use a struct with the same shape.
type occRef struct {
	name string
	kind string
	idx  int
}

// assignOccurrences sets the Occurrence field on types, functions, consts, and
// vars when multiple declarations share the same name. The order slice is in
// file order (matching the resolver's single-pass counting).
func assignOccurrences(result *InspectResult, order []occRef) {
	// Count total occurrences per name
	totals := map[string]int{}
	for _, ref := range order {
		totals[ref.name]++
	}

	// Assign occurrence numbers only for names appearing more than once
	seen := map[string]int{}
	for _, ref := range order {
		if totals[ref.name] <= 1 {
			continue
		}
		seen[ref.name]++
		occ := seen[ref.name]
		switch ref.kind {
		case "type":
			result.Types[ref.idx].Occurrence = occ
		case "func":
			result.Functions[ref.idx].Occurrence = occ
		case "const":
			result.Consts[ref.idx].Occurrence = occ
		case "var":
			result.Vars[ref.idx].Occurrence = occ
		}
	}
}

func inspectGenDecl(fset *token.FileSet, decl *ast.GenDecl, result *InspectResult) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			result.Types = append(result.Types, inspectType(fset, decl, s))
		case *ast.ValueSpec:
			items := inspectValueSpec(fset, decl, s)
			if decl.Tok == token.CONST {
				result.Consts = append(result.Consts, items...)
			} else {
				result.Vars = append(result.Vars, items...)
			}
		}
	}
}

func inspectType(fset *token.FileSet, decl *ast.GenDecl, spec *ast.TypeSpec) TypeInfo {
	info := TypeInfo{
		Name:     spec.Name.Name,
		Exported: unicode.IsUpper(rune(spec.Name.Name[0])),
	}

	if decl.Doc != nil {
		info.Doc = strings.TrimSpace(decl.Doc.Text())
	} else if spec.Doc != nil {
		info.Doc = strings.TrimSpace(spec.Doc.Text())
	}

	if spec.Assign.IsValid() {
		info.Kind = "alias"
		info.Target = typeString(spec.Type)
		return info
	}

	switch t := spec.Type.(type) {
	case *ast.StructType:
		info.Kind = "struct"
		if t.Fields != nil {
			for _, f := range t.Fields.List {
				fi := FieldInfo{
					Type: typeString(f.Type),
				}
				if len(f.Names) > 0 {
					fi.Name = f.Names[0].Name
				}
				if f.Tag != nil {
					fi.Tag = strings.Trim(f.Tag.Value, "`")
				}
				if f.Doc != nil {
					fi.Doc = strings.TrimSpace(f.Doc.Text())
				}
				info.Fields = append(info.Fields, fi)
			}
		}
	case *ast.InterfaceType:
		info.Kind = "interface"
		if t.Methods != nil {
			for _, m := range t.Methods.List {
				if len(m.Names) > 0 {
					im := InterfaceMethod{
						Name: m.Names[0].Name,
					}
					im.Signature = m.Names[0].Name + funcTypeString(m.Type.(*ast.FuncType))
					info.Methods = append(info.Methods, im)
				} else {
					// Embedded interface
					info.Embeds = append(info.Embeds, typeString(m.Type))
				}
			}
		}
	default:
		info.Kind = "named"
		info.Target = typeString(spec.Type)
	}
	return info
}

func inspectFunc(fset *token.FileSet, decl *ast.FuncDecl) FunctionInfo {
	info := FunctionInfo{
		Name:    decl.Name.Name,
		Params:  []ParamInfo{},
		Returns: []string{},
		Body:    []BodyStmt{},
	}

	if decl.Doc != nil {
		info.Doc = strings.TrimSpace(decl.Doc.Text())
	}

	// Receiver
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		recv := decl.Recv.List[0]
		info.Receiver = typeString(recv.Type)
		if len(recv.Names) > 0 {
			info.ReceiverVar = recv.Names[0].Name
		}
		info.Name = baseReceiverType(recv.Type) + "." + decl.Name.Name
	}

	// Signature
	info.Signature = buildSignature(decl)

	// Params
	if decl.Type.Params != nil {
		for _, p := range decl.Type.Params.List {
			t := typeString(p.Type)
			if len(p.Names) > 0 {
				for _, n := range p.Names {
					info.Params = append(info.Params, ParamInfo{Name: n.Name, Type: t})
				}
			} else {
				info.Params = append(info.Params, ParamInfo{Type: t})
			}
		}
	}

	// Returns
	if decl.Type.Results != nil {
		for _, r := range decl.Type.Results.List {
			info.Returns = append(info.Returns, typeString(r.Type))
		}
	}

	// Body statements
	if decl.Body != nil {
		for i, stmt := range decl.Body.List {
			info.Body = append(info.Body, BodyStmt{
				Index:     i,
				Statement: stmtString(fset, stmt),
			})
		}
	}

	return info
}

func inspectValueSpec(fset *token.FileSet, decl *ast.GenDecl, spec *ast.ValueSpec) []VarConstInfo {
	var items []VarConstInfo
	for i, name := range spec.Names {
		info := VarConstInfo{
			Name: name.Name,
		}
		if spec.Type != nil {
			info.Type = typeString(spec.Type)
		}
		if i < len(spec.Values) {
			info.Value = exprString(fset, spec.Values[i])
		}
		// Block name: if this is inside a grouped decl (parens), use empty string
		// We don't track block names for now
		items = append(items, info)
	}
	return items
}

func buildSignature(decl *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func ")
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		recv := decl.Recv.List[0]
		b.WriteString("(")
		if len(recv.Names) > 0 {
			b.WriteString(recv.Names[0].Name)
			b.WriteString(" ")
		}
		b.WriteString(typeString(recv.Type))
		b.WriteString(") ")
	}
	b.WriteString(decl.Name.Name)
	b.WriteString(funcTypeString(decl.Type))
	return b.String()
}

func funcTypeString(ft *ast.FuncType) string {
	var b strings.Builder
	b.WriteString("(")
	if ft.Params != nil {
		for i, p := range ft.Params.List {
			if i > 0 {
				b.WriteString(", ")
			}
			if len(p.Names) > 0 {
				for j, n := range p.Names {
					if j > 0 {
						b.WriteString(", ")
					}
					b.WriteString(n.Name)
				}
				b.WriteString(" ")
			}
			b.WriteString(typeString(p.Type))
		}
	}
	b.WriteString(")")
	if ft.Results != nil && len(ft.Results.List) > 0 {
		b.WriteString(" ")
		if len(ft.Results.List) == 1 && len(ft.Results.List[0].Names) == 0 {
			b.WriteString(typeString(ft.Results.List[0].Type))
		} else {
			b.WriteString("(")
			for i, r := range ft.Results.List {
				if i > 0 {
					b.WriteString(", ")
				}
				if len(r.Names) > 0 {
					for j, n := range r.Names {
						if j > 0 {
							b.WriteString(", ")
						}
						b.WriteString(n.Name)
					}
					b.WriteString(" ")
				}
				b.WriteString(typeString(r.Type))
			}
			b.WriteString(")")
		}
	}
	return b.String()
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeString(t.Elt)
		}
		return "[" + exprStringBasic(t.Len) + "]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *ast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "interface{}"
		}
		return "interface{...}"
	case *ast.FuncType:
		return "func" + funcTypeString(t)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + typeString(t.Value)
		case ast.RECV:
			return "<-chan " + typeString(t.Value)
		default:
			return "chan " + typeString(t.Value)
		}
	case *ast.Ellipsis:
		return "..." + typeString(t.Elt)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func exprStringBasic(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.Ident:
		return e.Name
	default:
		return "?"
	}
}

func baseReceiverType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return baseReceiverType(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return baseReceiverType(t.X)
	default:
		return typeString(expr)
	}
}

func exprString(fset *token.FileSet, expr ast.Expr) string {
	var b strings.Builder
	printer.Fprint(&b, fset, expr)
	return b.String()
}

func stmtString(fset *token.FileSet, stmt ast.Stmt) string {
	var b strings.Builder
	printer.Fprint(&b, fset, stmt)
	return b.String()
}

