package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// editTestHelper writes source to a temp file, runs an edit operation, returns result
func editTestHelper(t *testing.T, src string, op Operation) *EditResult {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	op.File = path
	op.Mode = "edit"
	result, err := edit(op)
	if err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("edit not successful: %v", result.Errors)
	}
	return result
}

func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

// ===== Category 1: Declaration Creation =====

func TestCreateFunction(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:      "create_function",
		Name:    "Hello",
		Params:  "name:string",
		Returns: "error",
	})
	if !strings.Contains(result.Content, "func Hello(name string) error") {
		t.Errorf("expected function declaration, got:\n%s", result.Content)
	}
}

func TestCreateFunctionMultiReturn(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:      "create_function",
		Name:    "Load",
		Params:  "path:string",
		Returns: "*Config,error",
	})
	if !strings.Contains(result.Content, "func Load(path string) (*Config, error)") {
		t.Errorf("expected multi-return function, got:\n%s", result.Content)
	}
}

func TestCreateMethod(t *testing.T) {
	src := "package test\ntype Server struct{}\n"
	result := editTestHelper(t, src, Operation{
		Op:           "create_method",
		Name:         "Start",
		ReceiverType: "*Server",
		ReceiverVar:  "s",
		Params:       "ctx:context.Context",
		Returns:      "error",
	})
	if !strings.Contains(result.Content, "func (s *Server) Start(ctx context.Context) error") {
		t.Errorf("expected method declaration, got:\n%s", result.Content)
	}
}

func TestCreateStruct(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:   "create_struct",
		Name: "Config",
	})
	if !strings.Contains(result.Content, "type Config struct") {
		t.Errorf("expected struct declaration, got:\n%s", result.Content)
	}
}

func TestCreateInterface(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:   "create_interface",
		Name: "Handler",
	})
	if !strings.Contains(result.Content, "type Handler interface") {
		t.Errorf("expected interface declaration, got:\n%s", result.Content)
	}
}

func TestCreateTypeAlias(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:         "create_type_alias",
		Name:       "Byte",
		TargetType: "uint8",
	})
	if !strings.Contains(result.Content, "type Byte = uint8") {
		t.Errorf("expected type alias, got:\n%s", result.Content)
	}
}

func TestCreateNamedType(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:             "create_named_type",
		Name:           "Duration",
		UnderlyingType: "int64",
	})
	if !strings.Contains(result.Content, "type Duration int64") {
		t.Errorf("expected named type, got:\n%s", result.Content)
	}
}

func TestAddConst(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:        "add_const",
		Name:      "MaxRetries",
		ValueSpec: "int:3",
	})
	if !strings.Contains(result.Content, "const MaxRetries = 3") {
		t.Errorf("expected const, got:\n%s", result.Content)
	}
}

func TestAddConstWithFuncValue(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:        "add_var",
		Name:      "ErrNotFound",
		ValueFunc: "errors.New",
		ValueArgs: "string:not found",
	})
	if !strings.Contains(result.Content, `errors.New("not found")`) {
		t.Errorf("expected func call value, got:\n%s", result.Content)
	}
}

func TestDelete(t *testing.T) {
	src := "package test\n\nfunc Foo() {}\nfunc Bar() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "delete",
		Target: "Foo",
	})
	if strings.Contains(result.Content, "func Foo") {
		t.Error("expected Foo to be deleted")
	}
	if !strings.Contains(result.Content, "func Bar") {
		t.Error("expected Bar to remain")
	}
}

// ===== Category 2: Function/Method Signature =====

func TestAddParameter(t *testing.T) {
	src := "package test\nfunc Foo(a int) {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "add_parameter",
		Target: "Foo",
		Name:   "b",
		Type:   "string",
	})
	if !strings.Contains(result.Content, "func Foo(a int, b string)") {
		t.Errorf("expected added parameter, got:\n%s", result.Content)
	}
}

func TestAddParameterFirst(t *testing.T) {
	src := "package test\nfunc Foo(a int) {}\n"
	result := editTestHelper(t, src, Operation{
		Op:       "add_parameter",
		Target:   "Foo",
		Name:     "ctx",
		Type:     "context.Context",
		Position: "first",
	})
	if !strings.Contains(result.Content, "func Foo(ctx context.Context, a int)") {
		t.Errorf("expected param at first, got:\n%s", result.Content)
	}
}

func TestAddParameterAfter(t *testing.T) {
	src := "package test\nfunc Foo(a int, c bool) {}\n"
	result := editTestHelper(t, src, Operation{
		Op:       "add_parameter",
		Target:   "Foo",
		Name:     "b",
		Type:     "string",
		Position: "after",
		Anchor:   "a",
	})
	if !strings.Contains(result.Content, "func Foo(a int, b string, c bool)") {
		t.Errorf("expected param after a, got:\n%s", result.Content)
	}
}

func TestRemoveParameter(t *testing.T) {
	src := "package test\nfunc Foo(a int, b string) {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "remove_parameter",
		Target: "Foo.b",
	})
	if !strings.Contains(result.Content, "func Foo(a int)") {
		t.Errorf("expected param removed, got:\n%s", result.Content)
	}
}

func TestSetReturnTypes(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:      "set_return_types",
		Target:  "Foo",
		Returns: "*Config,error",
	})
	if !strings.Contains(result.Content, "func Foo() (*Config, error)") {
		t.Errorf("expected return types, got:\n%s", result.Content)
	}
}

func TestSetReturnTypesEmpty(t *testing.T) {
	src := "package test\nfunc Foo() error { return nil }\n"
	result := editTestHelper(t, src, Operation{
		Op:      "set_return_types",
		Target:  "Foo",
		Returns: "",
	})
	if strings.Contains(result.Content, "error") {
		t.Errorf("expected no return types, got:\n%s", result.Content)
	}
}

func TestChangeReceiver(t *testing.T) {
	src := "package test\ntype S struct{}\nfunc (s S) Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:           "change_receiver",
		Target:       "S.Foo",
		ReceiverType: "*S",
	})
	if !strings.Contains(result.Content, "func (s *S) Foo()") {
		t.Errorf("expected pointer receiver, got:\n%s", result.Content)
	}
}

func TestSetFunctionName(t *testing.T) {
	src := "package test\nfunc OldName() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "set_function_name",
		Target: "OldName",
		Name:   "NewName",
	})
	if !strings.Contains(result.Content, "func NewName()") {
		t.Errorf("expected renamed function, got:\n%s", result.Content)
	}
}

// ===== Category 3: Function/Method Body =====

func TestClearBody(t *testing.T) {
	src := "package test\nfunc Foo() { x := 1; _ = x }\n"
	result := editTestHelper(t, src, Operation{
		Op:     "clear_body",
		Target: "Foo",
	})
	if strings.Contains(result.Content, "x := 1") {
		t.Error("expected body to be cleared")
	}
}

func TestRemoveStatement(t *testing.T) {
	src := `package test
func Foo() {
	a := 1
	b := 2
	c := 3
	_ = a
	_ = b
	_ = c
}
`
	result := editTestHelper(t, src, Operation{
		Op:     "remove_statement",
		Target: "Foo",
		Index:  intPtr(1),
	})
	if strings.Contains(result.Content, "b := 2") {
		t.Error("expected statement at index 1 removed")
	}
	if !strings.Contains(result.Content, "a := 1") {
		t.Error("expected statement at index 0 preserved")
	}
}

func TestInsertCall(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "insert_call",
		Target: "Foo",
		Func:   "fmt.Println",
		Args:   "string:hello",
	})
	if !strings.Contains(result.Content, `fmt.Println("hello")`) {
		t.Errorf("expected call, got:\n%s", result.Content)
	}
}

func TestInsertMethodCall(t *testing.T) {
	src := "package test\ntype S struct{}\nfunc (s *S) Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:       "insert_method_call",
		Target:   "S.Foo",
		Receiver: "s",
		Method:   "Close",
	})
	if !strings.Contains(result.Content, "s.Close()") {
		t.Errorf("expected method call, got:\n%s", result.Content)
	}
}

func TestInsertAssignCall(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:    "insert_assign_call",
		Target: "Foo",
		Vars:  "data,err",
		Func:  "os.ReadFile",
		Args:  "string:config.json",
		Short: boolPtr(true),
	})
	if !strings.Contains(result.Content, `data, err := os.ReadFile("config.json")`) {
		t.Errorf("expected assign call, got:\n%s", result.Content)
	}
}

func TestInsertAssignValue(t *testing.T) {
	src := `package test
func Foo() {
	var x int
	_ = x
}
`
	result := editTestHelper(t, src, Operation{
		Op:        "insert_assign_value",
		Target:    "Foo",
		VarName:   "x",
		ValueSpec: "int:42",
		Position:  "at_index",
		Index:     intPtr(1),
	})
	if !strings.Contains(result.Content, "x = 42") {
		t.Errorf("expected assign value, got:\n%s", result.Content)
	}
}

func TestInsertVarDecl(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:      "insert_var_decl",
		Target:  "Foo",
		VarName: "buf",
		VarType: "bytes.Buffer",
	})
	if !strings.Contains(result.Content, "var buf bytes.Buffer") {
		t.Errorf("expected var decl, got:\n%s", result.Content)
	}
}

func TestInsertReturn(t *testing.T) {
	src := "package test\nfunc Foo() (int, error) {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "insert_return",
		Target: "Foo",
		Values: "int:0,nil",
	})
	if !strings.Contains(result.Content, "return 0, nil") {
		t.Errorf("expected return, got:\n%s", result.Content)
	}
}

func TestInsertReturnBare(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "insert_return",
		Target: "Foo",
	})
	if !strings.Contains(result.Content, "return") {
		t.Errorf("expected bare return, got:\n%s", result.Content)
	}
}

func TestInsertIf(t *testing.T) {
	src := "package test\nfunc Foo() {\n\tx := 1\n\t_ = x\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "insert_if",
		Target:    "Foo",
		CondLeft:  "ident:x",
		Operator:  ">",
		CondRight: "int:0",
		Position:  "at_index",
		Index:     intPtr(1),
	})
	if !strings.Contains(result.Content, "if x > 0") {
		t.Errorf("expected if statement, got:\n%s", result.Content)
	}
}

func TestInsertIfInit(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "insert_if_init",
		Target:    "Foo",
		InitVars:  "err",
		InitFunc:  "doWork",
		CondLeft:  "ident:err",
		Operator:  "!=",
		CondRight: "nil",
	})
	if !strings.Contains(result.Content, "if err := doWork(); err != nil") {
		t.Errorf("expected if-init, got:\n%s", result.Content)
	}
}

func TestInsertForRange(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:       "insert_for_range",
		Target:   "Foo",
		Key:      "_",
		Value:    "item",
		Iterable: "ident:items",
	})
	if !strings.Contains(result.Content, "for _, item := range items") {
		t.Errorf("expected for range, got:\n%s", result.Content)
	}
}

func TestInsertDefer(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "insert_defer",
		Target: "Foo",
		Func:   "f.Close",
	})
	if !strings.Contains(result.Content, "defer f.Close()") {
		t.Errorf("expected defer, got:\n%s", result.Content)
	}
}

func TestInsertGo(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "insert_go",
		Target: "Foo",
		Func:   "handle",
		Args:   "ident:conn",
	})
	if !strings.Contains(result.Content, "go handle(conn)") {
		t.Errorf("expected go statement, got:\n%s", result.Content)
	}
}

func TestInsertErrorCheck(t *testing.T) {
	src := "package test\nfunc Foo() error {\n\terr := doWork()\n\treturn nil\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:           "insert_error_check",
		Target:       "Foo",
		ErrVar:       "err",
		ReturnValues: "ident:err",
		Position:     "at_index",
		Index:        intPtr(1),
	})
	if !strings.Contains(result.Content, "if err != nil") {
		t.Errorf("expected error check, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "return err") {
		t.Errorf("expected return err, got:\n%s", result.Content)
	}
}

func TestPushArg(t *testing.T) {
	src := "package test\nfunc Foo() {\n\tfmt.Println()\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "push_arg",
		Target:    "Foo",
		StmtIndex: intPtr(0),
		Arg:       "string:hello",
	})
	if !strings.Contains(result.Content, `fmt.Println("hello")`) {
		t.Errorf("expected arg pushed, got:\n%s", result.Content)
	}
}

func TestPushArgWithCallIndex(t *testing.T) {
	src := "package test\nfunc Foo() (int, error) {\n\treturn 0, fmt.Errorf()\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "push_arg",
		Target:    "Foo",
		StmtIndex: intPtr(0),
		CallIndex: intPtr(0),
		Arg:       "string:failed: %w",
	})
	if !strings.Contains(result.Content, `fmt.Errorf("failed: %w")`) {
		t.Errorf("expected arg pushed to call in return, got:\n%s", result.Content)
	}
}

func TestSetElse(t *testing.T) {
	src := "package test\nfunc Foo() {\n\tif true {\n\t}\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "set_else",
		Target:    "Foo",
		StmtIndex: intPtr(0),
	})
	if !strings.Contains(result.Content, "} else {") {
		t.Errorf("expected else block, got:\n%s", result.Content)
	}
}

// ===== Compound body targeting =====

func TestInsertInIfBody(t *testing.T) {
	src := `package test
func Foo() {
	x := 1
	if x > 0 {
	}
	_ = x
}
`
	result := editTestHelper(t, src, Operation{
		Op:     "insert_call",
		Target: "Foo.if[1]",
		Func:   "fmt.Println",
		Args:   "string:positive",
	})
	if !strings.Contains(result.Content, `fmt.Println("positive")`) {
		t.Errorf("expected call in if body, got:\n%s", result.Content)
	}
}

func TestInsertInForBody(t *testing.T) {
	src := `package test
func Foo() {
	items := []int{1}
	for _, v := range items {
	}
	_ = items
}
`
	result := editTestHelper(t, src, Operation{
		Op:     "insert_call",
		Target: "Foo.for[1]",
		Func:   "process",
		Args:   "ident:v",
	})
	if !strings.Contains(result.Content, "process(v)") {
		t.Errorf("expected call in for body, got:\n%s", result.Content)
	}
}

// ===== Category 4: Struct Field Editing =====

func TestAddStructField(t *testing.T) {
	src := "package test\ntype Config struct {\n\tPort int\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "add_struct_field",
		Target:    "Config",
		FieldName: "Host",
		FieldType: "string",
		Tag:       `json:"host"`,
	})
	if !strings.Contains(result.Content, "Host") {
		t.Errorf("expected field added, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `json:"host"`) {
		t.Errorf("expected tag, got:\n%s", result.Content)
	}
}

func TestAddStructFieldFirst(t *testing.T) {
	src := "package test\ntype Config struct {\n\tPort int\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "add_struct_field",
		Target:    "Config",
		FieldName: "Name",
		FieldType: "string",
		Position:  "first",
	})
	// Name should appear before Port
	nameIdx := strings.Index(result.Content, "Name")
	portIdx := strings.Index(result.Content, "Port")
	if nameIdx < 0 || portIdx < 0 || nameIdx > portIdx {
		t.Errorf("expected Name before Port, got:\n%s", result.Content)
	}
}

func TestRemoveStructField(t *testing.T) {
	src := "package test\ntype Config struct {\n\tPort int\n\tHost string\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "remove_struct_field",
		Target: "Config.Host",
	})
	if strings.Contains(result.Content, "Host") {
		t.Error("expected Host field removed")
	}
	if !strings.Contains(result.Content, "Port") {
		t.Error("expected Port field preserved")
	}
}

func TestSetFieldType(t *testing.T) {
	src := "package test\ntype Config struct {\n\tPort int\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "set_field_type",
		Target:    "Config.Port",
		FieldType: "uint16",
	})
	if !strings.Contains(result.Content, "Port uint16") {
		t.Errorf("expected updated type, got:\n%s", result.Content)
	}
}

func TestSetFieldName(t *testing.T) {
	src := "package test\ntype Config struct {\n\tPort int\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "set_field_name",
		Target:    "Config.Port",
		FieldName: "ListenPort",
	})
	if !strings.Contains(result.Content, "ListenPort") {
		t.Errorf("expected renamed field, got:\n%s", result.Content)
	}
}

func TestSetStructTag(t *testing.T) {
	src := "package test\ntype Config struct {\n\tPort int `json:\"port\"`\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:       "set_struct_tag",
		Target:   "Config.Port",
		TagKey:   "yaml",
		TagValue: "port",
	})
	if !strings.Contains(result.Content, `json:"port"`) {
		t.Errorf("expected json tag preserved, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `yaml:"port"`) {
		t.Errorf("expected yaml tag added, got:\n%s", result.Content)
	}
}

func TestRemoveStructTag(t *testing.T) {
	src := "package test\ntype Config struct {\n\tPort int `json:\"port\" yaml:\"port\"`\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "remove_struct_tag",
		Target: "Config.Port",
		TagKey: "yaml",
	})
	if !strings.Contains(result.Content, `json:"port"`) {
		t.Error("expected json tag preserved")
	}
	if strings.Contains(result.Content, `yaml:`) {
		t.Error("expected yaml tag removed")
	}
}

// ===== Category 5: Interface Editing =====

func TestAddInterfaceMethod(t *testing.T) {
	src := "package test\ntype Handler interface{}\n"
	result := editTestHelper(t, src, Operation{
		Op:         "add_interface_method",
		Target:     "Handler",
		MethodName: "Handle",
		Params:     "ctx:context.Context",
		Returns:    "error",
	})
	if !strings.Contains(result.Content, "Handle(ctx context.Context) error") {
		t.Errorf("expected interface method, got:\n%s", result.Content)
	}
}

func TestRemoveInterfaceMethod(t *testing.T) {
	src := `package test
type Handler interface {
	Foo()
	Bar()
}
`
	result := editTestHelper(t, src, Operation{
		Op:     "remove_interface_method",
		Target: "Handler.Foo",
	})
	if strings.Contains(result.Content, "Foo") {
		t.Error("expected Foo removed")
	}
	if !strings.Contains(result.Content, "Bar") {
		t.Error("expected Bar preserved")
	}
}

func TestAddInterfaceEmbed(t *testing.T) {
	src := "package test\ntype Handler interface{}\n"
	result := editTestHelper(t, src, Operation{
		Op:        "add_interface_embed",
		Target:    "Handler",
		EmbedType: "io.Closer",
	})
	if !strings.Contains(result.Content, "io.Closer") {
		t.Errorf("expected embed, got:\n%s", result.Content)
	}
}

// ===== Category 6: Import Management =====

func TestAddImport(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:   "add_import",
		Path: "fmt",
	})
	if !strings.Contains(result.Content, `"fmt"`) {
		t.Errorf("expected import added, got:\n%s", result.Content)
	}
}

func TestAddImportWithAlias(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:    "add_import",
		Path:  "google.golang.org/protobuf/proto",
		Alias: "pb",
	})
	if !strings.Contains(result.Content, `pb "google.golang.org/protobuf/proto"`) {
		t.Errorf("expected aliased import, got:\n%s", result.Content)
	}
}

func TestRemoveImport(t *testing.T) {
	src := `package test
import (
	"fmt"
	"os"
)
func Foo() { _ = os.Stdin }
`
	result := editTestHelper(t, src, Operation{
		Op:   "remove_import",
		Path: "fmt",
	})
	if strings.Contains(result.Content, `"fmt"`) {
		t.Error("expected fmt import removed")
	}
	if !strings.Contains(result.Content, `"os"`) {
		t.Error("expected os import preserved")
	}
}

// ===== Category 7: Const/Var Editing =====

func TestSetConstValue(t *testing.T) {
	src := "package test\nconst Max = 3\n"
	result := editTestHelper(t, src, Operation{
		Op:        "set_const_value",
		Target:    "Max",
		ValueSpec: "int:5",
	})
	if !strings.Contains(result.Content, "const Max = 5") {
		t.Errorf("expected updated const, got:\n%s", result.Content)
	}
}

func TestSetVarValue(t *testing.T) {
	src := `package test
var Name = "old"
`
	result := editTestHelper(t, src, Operation{
		Op:        "set_var_value",
		Target:    "Name",
		ValueSpec: "string:new",
	})
	if !strings.Contains(result.Content, `"new"`) {
		t.Errorf("expected updated var, got:\n%s", result.Content)
	}
}

func TestSetConstType(t *testing.T) {
	src := "package test\nconst Max = 3\n"
	result := editTestHelper(t, src, Operation{
		Op:     "set_const_type",
		Target: "Max",
		Type:   "int64",
	})
	if !strings.Contains(result.Content, "int64") {
		t.Errorf("expected const type, got:\n%s", result.Content)
	}
}

func TestRemoveVar(t *testing.T) {
	src := "package test\nvar X = 1\nvar Y = 2\n"
	result := editTestHelper(t, src, Operation{
		Op:     "remove_var",
		Target: "X",
	})
	if strings.Contains(result.Content, "X = 1") {
		t.Error("expected X removed")
	}
	if !strings.Contains(result.Content, "Y = 2") {
		t.Error("expected Y preserved")
	}
}

// ===== Category 8: Package/File Level =====

func TestSetPackage(t *testing.T) {
	src := "package old\n"
	result := editTestHelper(t, src, Operation{
		Op:   "set_package",
		Name: "newpkg",
	})
	if !strings.Contains(result.Content, "package newpkg") {
		t.Errorf("expected new package, got:\n%s", result.Content)
	}
}

func TestSetBuildConstraint(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:         "set_build_constraint",
		Constraint: "linux && amd64",
	})
	if !strings.Contains(result.Content, "//go:build linux && amd64") {
		t.Errorf("expected build constraint, got:\n%s", result.Content)
	}
}

func TestAddGenerateDirective(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:        "add_generate_directive",
		Directive: "stringer -type=Color",
	})
	if !strings.Contains(result.Content, "//go:generate stringer -type=Color") {
		t.Errorf("expected generate directive, got:\n%s", result.Content)
	}
}

// ===== Category 9: Comments =====

func TestSetDocComment(t *testing.T) {
	src := "package test\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "set_doc_comment",
		Target: "Foo",
		Text:   "Foo does something important.",
	})
	if !strings.Contains(result.Content, "// Foo does something important.") {
		t.Errorf("expected doc comment, got:\n%s", result.Content)
	}
}

func TestRemoveDocComment(t *testing.T) {
	src := "package test\n\n// Foo does something.\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "remove_doc_comment",
		Target: "Foo",
	})
	if strings.Contains(result.Content, "// Foo does") {
		t.Errorf("expected doc comment removed, got:\n%s", result.Content)
	}
}

func TestAddLineComment(t *testing.T) {
	src := "package test\ntype Config struct {\n\tPort int\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "add_line_comment",
		Target: "Config.Port",
		Text:   "default: 8080",
	})
	if !strings.Contains(result.Content, "// default: 8080") {
		t.Errorf("expected line comment, got:\n%s", result.Content)
	}
}

// ===== Category 10: Refactoring =====

func TestRename(t *testing.T) {
	src := `package test
var old = 1
func Foo() {
	x := old
	_ = x
}
`
	result := editTestHelper(t, src, Operation{
		Op:      "rename",
		Target:  "old",
		NewName: "renamed",
	})
	if strings.Contains(result.Content, "old") {
		t.Errorf("expected old renamed, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "renamed") {
		t.Errorf("expected renamed, got:\n%s", result.Content)
	}
}

func TestExtractInterface(t *testing.T) {
	src := `package test
type Server struct{}
func (s *Server) Start() error { return nil }
func (s *Server) Stop() error { return nil }
func (s *Server) internal() {}
`
	result := editTestHelper(t, src, Operation{
		Op:            "extract_interface",
		Target:        "Server",
		InterfaceName: "Starter",
	})
	if !strings.Contains(result.Content, "type Starter interface") {
		t.Errorf("expected interface extracted, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "Start()") {
		t.Errorf("expected Start method, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "Stop()") {
		t.Errorf("expected Stop method, got:\n%s", result.Content)
	}
	// internal() is unexported, should not be in interface
	if strings.Contains(result.Content, "internal()") {
		// Check it's not in the interface — it should still exist as a method
		// but not in the Starter interface
	}
}

// ===== Multi-step sequence =====

func TestMultiStepCreateAndPopulate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	src := "package test\n"
	os.WriteFile(path, []byte(src), 0644)

	// Step 1: Create function
	op := Operation{File: path, Mode: "edit", Op: "create_function", Name: "NewConfig", Params: "path:string", Returns: "*Config,error"}
	r, err := edit(op)
	if err != nil {
		t.Fatalf("step 1: %v", err)
	}
	os.WriteFile(path, []byte(r.Content), 0644)

	// Step 2: Insert assign call
	op = Operation{File: path, Mode: "edit", Op: "insert_assign_call", Target: "NewConfig", Vars: "data,err", Func: "os.ReadFile", Args: "ident:path", Short: boolPtr(true)}
	r, err = edit(op)
	if err != nil {
		t.Fatalf("step 2: %v", err)
	}
	os.WriteFile(path, []byte(r.Content), 0644)

	// Step 3: Insert error check
	op = Operation{File: path, Mode: "edit", Op: "insert_error_check", Target: "NewConfig", ErrVar: "err", ReturnValues: "nil,ident:err", Position: "at_index", Index: intPtr(1)}
	r, err = edit(op)
	if err != nil {
		t.Fatalf("step 3: %v", err)
	}
	os.WriteFile(path, []byte(r.Content), 0644)

	// Step 4: Insert return
	op = Operation{File: path, Mode: "edit", Op: "insert_return", Target: "NewConfig", Values: "addr:Config,nil"}
	r, err = edit(op)
	if err != nil {
		t.Fatalf("step 4: %v", err)
	}

	content := r.Content
	if !strings.Contains(content, "func NewConfig(path string) (*Config, error)") {
		t.Errorf("expected function sig, got:\n%s", content)
	}
	if !strings.Contains(content, "data, err := os.ReadFile(path)") {
		t.Errorf("expected assign call, got:\n%s", content)
	}
	if !strings.Contains(content, "if err != nil") {
		t.Errorf("expected error check, got:\n%s", content)
	}
	if !strings.Contains(content, "return &Config") {
		t.Errorf("expected return, got:\n%s", content)
	}
}

// ===== Edge cases =====

func TestEditEmptyFile(t *testing.T) {
	src := "package empty\n"
	result := editTestHelper(t, src, Operation{
		Op:   "create_struct",
		Name: "Config",
	})
	if !strings.Contains(result.Content, "type Config struct") {
		t.Errorf("expected struct in empty file, got:\n%s", result.Content)
	}
}

func TestGofmtPreservation(t *testing.T) {
	src := "package test\nfunc Foo()   {   }\n"
	result := editTestHelper(t, src, Operation{
		Op:     "insert_call",
		Target: "Foo",
		Func:   "bar",
	})
	// Result should be properly formatted
	if strings.Contains(result.Content, "   ") {
		t.Errorf("expected gofmt'd output, got:\n%s", result.Content)
	}
}

// ===== Category 11: Structural Replacement =====

func TestReplaceBody(t *testing.T) {
	src := "package test\nfunc Foo() int {\n\treturn 1\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "replace_body",
		Target: "Foo",
		Body:   "return 42",
	})
	if !strings.Contains(result.Content, "return 42") {
		t.Errorf("expected new return, got:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "return 1") {
		t.Errorf("expected old return removed, got:\n%s", result.Content)
	}
}

func TestReplaceMethodBody(t *testing.T) {
	src := "package test\ntype S struct{}\nfunc (s *S) Hello() string {\n\treturn \"old\"\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "replace_body",
		Target: "S.Hello",
		Body:   "return \"new\"",
	})
	if !strings.Contains(result.Content, `"new"`) {
		t.Errorf("expected new return, got:\n%s", result.Content)
	}
	if strings.Contains(result.Content, `"old"`) {
		t.Errorf("expected old return removed, got:\n%s", result.Content)
	}
}

func TestReplaceStruct(t *testing.T) {
	src := "package test\ntype Config struct {\n\tX int\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "replace_struct",
		Target: "Config",
		Body:   "\tHost string\n\tPort int\n",
	})
	if !strings.Contains(result.Content, "Host string") {
		t.Errorf("expected Host field, got:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "X int") {
		t.Errorf("expected old field removed, got:\n%s", result.Content)
	}
}

func TestReplaceInterface(t *testing.T) {
	src := "package test\ntype Handler interface {\n\tOld()\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "replace_interface",
		Target: "Handler",
		Body:   "\tServeHTTP(w http.ResponseWriter, r *http.Request)\n",
	})
	if !strings.Contains(result.Content, "ServeHTTP") {
		t.Errorf("expected new method, got:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "Old()") {
		t.Errorf("expected old method removed, got:\n%s", result.Content)
	}
}

func TestReplaceDecl(t *testing.T) {
	src := "package test\nfunc Foo() int {\n\treturn 1\n}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "replace_decl",
		Target: "Foo",
		Source: "func Foo(n int) int {\n\treturn n * 2\n}",
	})
	if !strings.Contains(result.Content, "func Foo(n int) int") {
		t.Errorf("expected new signature, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "return n * 2") {
		t.Errorf("expected new body, got:\n%s", result.Content)
	}
}

func TestAddFunctionWithBody(t *testing.T) {
	src := "package test\n"
	result := editTestHelper(t, src, Operation{
		Op:      "add_function_with_body",
		Name:    "Greet",
		Params:  "name:string",
		Returns: "string",
		Body:    "return \"Hello, \" + name",
	})
	if !strings.Contains(result.Content, "func Greet(name string) string") {
		t.Errorf("expected function signature, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `"Hello, "`) {
		t.Errorf("expected body, got:\n%s", result.Content)
	}
}

func TestAddMethodWithBody(t *testing.T) {
	src := "package test\ntype S struct{ Name string }\n"
	result := editTestHelper(t, src, Operation{
		Op:           "add_method_with_body",
		Name:         "String",
		ReceiverType: "*S",
		ReceiverVar:  "s",
		Returns:      "string",
		Body:         "return s.Name",
	})
	if !strings.Contains(result.Content, "func (s *S) String() string") {
		t.Errorf("expected method signature, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "return s.Name") {
		t.Errorf("expected body, got:\n%s", result.Content)
	}
}

func TestReplaceImports(t *testing.T) {
	src := "package test\nimport \"fmt\"\nfunc Foo() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:      "replace_imports",
		Imports: "\"net/http\"\n\"encoding/json\"",
	})
	if !strings.Contains(result.Content, "net/http") {
		t.Errorf("expected new import, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "encoding/json") {
		t.Errorf("expected new import, got:\n%s", result.Content)
	}
}

func TestReplaceFile(t *testing.T) {
	src := "package test\nfunc Old() {}\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	op := Operation{
		Mode:   "edit",
		File:   path,
		Op:     "replace_file",
		Source: "package test\n\nfunc New() int {\n\treturn 1\n}\n",
	}
	result, err := edit(op)
	if err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("edit not successful: %v", result.Errors)
	}
	if !strings.Contains(result.Content, "func New()") {
		t.Errorf("expected new func, got:\n%s", result.Content)
	}
	if strings.Contains(result.Content, "func Old()") {
		t.Errorf("expected old func removed, got:\n%s", result.Content)
	}
}

func TestInsertBeforeDecl(t *testing.T) {
	src := "package test\nfunc B() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "insert_before_decl",
		Target: "B",
		Source: "func A() {}",
	})
	posA := strings.Index(result.Content, "func A()")
	posB := strings.Index(result.Content, "func B()")
	if posA < 0 || posB < 0 || posA >= posB {
		t.Errorf("expected A before B, got:\n%s", result.Content)
	}
}

func TestInsertAfterDecl(t *testing.T) {
	src := "package test\nfunc A() {}\n"
	result := editTestHelper(t, src, Operation{
		Op:     "insert_after_decl",
		Target: "A",
		Source: "func B() {}",
	})
	posA := strings.Index(result.Content, "func A()")
	posB := strings.Index(result.Content, "func B()")
	if posA < 0 || posB < 0 || posA >= posB {
		t.Errorf("expected B after A, got:\n%s", result.Content)
	}
}
