/**
 * Go AST operation registry and runtime knowledge generators for Gemma 4.
 *
 * All knowledge documents are generated dynamically from the registry.
 * No static markdown files — the registry is the single source of truth.
 * All tool call examples use wrapToolCall() from gemma4-tool-format.ts.
 */

import { wrapToolCall, Gemma4Tokens } from "./gemma4-tool-format"

// ---------------------------------------------------------------------------
// Operation Registry
// ---------------------------------------------------------------------------

export interface OpDef {
  op: string
  description: string
  hint: string
  required: string[]
  optional: string[]
  exampleParams: Record<string, unknown>
}

export interface CategoryDef {
  name: string
  operations: OpDef[]
  programmingConcept: string
  golangSyntax: string
}

export const GO_AST_REGISTRY: CategoryDef[] = [
  // ---- Category 1: Declaration Creation ----
  {
    name: "Declaration Creation",
    operations: [
      {
        op: "create_function",
        description: "Create a new empty function. Fill body with insert_* ops.",
        hint: "name, params (name:type,...), returns (type,...)",
        required: ["name"],
        optional: ["params", "returns", "anchor", "position"],
        exampleParams: { op: "create_function", name: "NewConfig", params: "path:string", returns: "*Config,error" },
      },
      {
        op: "create_method",
        description: "Create a new empty method with a receiver. Fill body with insert_* ops.",
        hint: "receiverType, receiverVar, name, params, returns",
        required: ["name", "receiverType", "receiverVar"],
        optional: ["params", "returns", "anchor", "position"],
        exampleParams: { op: "create_method", receiverType: "*Server", receiverVar: "s", name: "Start", params: "ctx:context.Context", returns: "error" },
      },
      {
        op: "create_struct",
        description: "Create a new empty struct type. Add fields with add_struct_field.",
        hint: "name",
        required: ["name"],
        optional: ["anchor", "position"],
        exampleParams: { op: "create_struct", name: "Config" },
      },
      {
        op: "create_interface",
        description: "Create a new empty interface type. Add methods with add_interface_method.",
        hint: "name",
        required: ["name"],
        optional: ["anchor", "position"],
        exampleParams: { op: "create_interface", name: "Handler" },
      },
      {
        op: "create_type_alias",
        description: "Create type Name = TargetType.",
        hint: "name, targetType",
        required: ["name", "targetType"],
        optional: ["anchor", "position"],
        exampleParams: { op: "create_type_alias", name: "HandlerFunc", targetType: "func(http.ResponseWriter, *http.Request)" },
      },
      {
        op: "create_named_type",
        description: "Create type Name UnderlyingType (e.g. type Duration int64).",
        hint: "name, underlyingType",
        required: ["name", "underlyingType"],
        optional: ["anchor", "position"],
        exampleParams: { op: "create_named_type", name: "Duration", underlyingType: "int64" },
      },
      {
        op: "add_const",
        description: "Add a constant. Use valueSpec for literals, valueFunc+valueArgs for calls.",
        hint: "name, valueSpec OR valueFunc+valueArgs, type?, block?",
        required: ["name"],
        optional: ["type", "valueSpec", "valueFunc", "valueArgs", "block"],
        exampleParams: { op: "add_const", name: "DefaultPort", type: "int", valueSpec: "int:8080" },
      },
      {
        op: "add_var",
        description: "Add a package-level variable.",
        hint: "name, valueSpec OR valueFunc+valueArgs, type?, block?",
        required: ["name"],
        optional: ["type", "valueSpec", "valueFunc", "valueArgs", "block"],
        exampleParams: { op: "add_var", name: "ErrNotFound", valueFunc: "errors.New", valueArgs: "string:not found" },
      },
      {
        op: "delete",
        description: "Delete any top-level declaration including its doc comments.",
        hint: "target",
        required: ["target"],
        optional: [],
        exampleParams: { op: "delete", target: "OldHandler" },
      },
    ],
    programmingConcept: `Source code is organized into declarations — named things at the top level of a file.

A function is a named block of code that can be called. It has a name, parameters (inputs), return types (outputs), and a body (the code inside).
A method is a function attached to a type via a receiver.
A struct is a type that groups related fields together.
An interface is a type that defines a set of method signatures.
A constant is a named value that never changes.
A variable is a named value that can change.
A type alias creates a new name for an existing type.
A named type creates a distinct type from an underlying type.

To create any of these, use the appropriate create_* or add_* operation. To remove one, use delete.`,
    golangSyntax: `Go declarations:

func NewConfig(path string) (*Config, error) { }
func (s *Server) Start(ctx context.Context) error { }
type Config struct { }
type Handler interface { }
type HandlerFunc = func(http.ResponseWriter, *http.Request)
type Duration int64
const DefaultPort int = 8080
var ErrNotFound = errors.New("not found")`,
  },

  // ---- Category 2: Function/Method Signature ----
  {
    name: "Signature Editing",
    operations: [
      {
        op: "add_parameter",
        description: "Add a parameter to a function/method signature.",
        hint: "target, name, type, position? (first/last/after), anchor?",
        required: ["target", "name", "type"],
        optional: ["position", "anchor"],
        exampleParams: { op: "add_parameter", target: "NewConfig", name: "logger", type: "*log.Logger", position: "last" },
      },
      {
        op: "remove_parameter",
        description: "Remove a parameter by dotted name: Func.ParamName.",
        hint: "target (Func.Param)",
        required: ["target"],
        optional: [],
        exampleParams: { op: "remove_parameter", target: "NewConfig.logger" },
      },
      {
        op: "set_return_types",
        description: "Set return types. Empty string = no returns.",
        hint: "target, returns (comma-separated types)",
        required: ["target", "returns"],
        optional: [],
        exampleParams: { op: "set_return_types", target: "NewConfig", returns: "*Config,error" },
      },
      {
        op: "change_receiver",
        description: "Change the receiver type of a method.",
        hint: "target, receiverType (*T or T), receiverVar?",
        required: ["target", "receiverType"],
        optional: ["receiverVar"],
        exampleParams: { op: "change_receiver", target: "Server.Start", receiverType: "*Server", receiverVar: "s" },
      },
      {
        op: "set_function_name",
        description: "Rename just the function/method name (does not rename call sites).",
        hint: "target, name",
        required: ["target", "name"],
        optional: [],
        exampleParams: { op: "set_function_name", target: "Server.Start", name: "Run" },
      },
    ],
    programmingConcept: `A function signature defines the function's interface: its name, what inputs it accepts (parameters), and what outputs it produces (return types).

Parameters are name-type pairs listed in parentheses after the function name.
Return types are listed after the parameters. Multiple return types are wrapped in parentheses.
A method receiver is a special first parameter that attaches the method to a type.`,
    golangSyntax: `Go function signatures:

func NewConfig(path string) (*Config, error)     // function with params and returns
func NewConfig(path string, logger *log.Logger)   // added parameter
func (s *Server) Start(ctx context.Context) error // method with pointer receiver
func (s Server) String() string                   // method with value receiver`,
  },

  // ---- Category 3: Function/Method Body ----
  {
    name: "Body Statements",
    operations: [
      {
        op: "clear_body",
        description: "Remove all statements from a function body.",
        hint: "target",
        required: ["target"],
        optional: [],
        exampleParams: { op: "clear_body", target: "Server.Start" },
      },
      {
        op: "remove_statement",
        description: "Remove a statement by 0-based index in the function body.",
        hint: "target, index",
        required: ["target", "index"],
        optional: [],
        exampleParams: { op: "remove_statement", target: "Server.Start", index: 2 },
      },
      {
        op: "insert_call",
        description: "Insert a function call statement: func(args...).",
        hint: "target, func, args? (kind:val,...), position?, index?",
        required: ["target", "func"],
        optional: ["args", "position", "index"],
        exampleParams: { op: "insert_call", target: "main", func: "fmt.Println", args: "string:hello world", position: "last" },
      },
      {
        op: "insert_method_call",
        description: "Insert a method call statement: receiver.method(args...).",
        hint: "target, receiver, method, args?, position?, index?",
        required: ["target", "receiver", "method"],
        optional: ["args", "position", "index"],
        exampleParams: { op: "insert_method_call", target: "Server.Start", receiver: "s", method: "init", position: "first" },
      },
      {
        op: "insert_assign_call",
        description: "Insert vars := func(args...) or vars = func(args...).",
        hint: "target, vars (name,...), func, short?, args?, position?, index?",
        required: ["target", "vars", "func"],
        optional: ["short", "args", "position", "index"],
        exampleParams: { op: "insert_assign_call", target: "NewConfig", vars: "data,err", short: true, func: "os.ReadFile", args: "ident:path", position: "first" },
      },
      {
        op: "insert_assign_value",
        description: "Insert varName = value (simple assignment).",
        hint: "target, varName, valueSpec (kind:val), position?, index?",
        required: ["target", "varName", "valueSpec"],
        optional: ["position", "index"],
        exampleParams: { op: "insert_assign_value", target: "Server.Start", varName: "s.running", valueSpec: "true", position: "last" },
      },
      {
        op: "insert_var_decl",
        description: "Insert var name Type inside a function body.",
        hint: "target, varName, varType, position?, index?",
        required: ["target", "varName", "varType"],
        optional: ["position", "index"],
        exampleParams: { op: "insert_var_decl", target: "NewConfig", varName: "cfg", varType: "Config", position: "first" },
      },
      {
        op: "insert_return",
        description: "Insert return statement with optional values.",
        hint: "target, values? (kind:val,...), position?, index?",
        required: ["target"],
        optional: ["values", "position", "index"],
        exampleParams: { op: "insert_return", target: "NewConfig", values: "addr:cfg,nil", position: "last" },
      },
      {
        op: "insert_if",
        description: "Insert if left op right { } with empty body.",
        hint: "target, condLeft, operator, condRight, position?, index?",
        required: ["target", "condLeft", "operator", "condRight"],
        optional: ["position", "index"],
        exampleParams: { op: "insert_if", target: "NewConfig", condLeft: "ident:err", operator: "!=", condRight: "nil", position: "at_index", index: 1 },
      },
      {
        op: "insert_if_init",
        description: "Insert if vars := func(args); left op right { } with init statement.",
        hint: "target, initVars, initFunc, initArgs?, condLeft, operator, condRight, position?, index?",
        required: ["target", "initVars", "initFunc", "condLeft", "operator", "condRight"],
        optional: ["initArgs", "position", "index"],
        exampleParams: { op: "insert_if_init", target: "Server.Start", initVars: "err", initFunc: "s.validate", condLeft: "ident:err", operator: "!=", condRight: "nil", position: "first" },
      },
      {
        op: "insert_for_range",
        description: "Insert for key, value := range iterable { } with empty body.",
        hint: "target, key?, value?, iterable (kind:val), position?, index?",
        required: ["target", "iterable"],
        optional: ["key", "value", "position", "index"],
        exampleParams: { op: "insert_for_range", target: "Server.Start", key: "_", value: "client", iterable: "selector:s.clients", position: "last" },
      },
      {
        op: "insert_defer",
        description: "Insert defer func(args...).",
        hint: "target, func, args?, position?, index?",
        required: ["target", "func"],
        optional: ["args", "position", "index"],
        exampleParams: { op: "insert_defer", target: "NewConfig", func: "f.Close", position: "at_index", index: 1 },
      },
      {
        op: "insert_go",
        description: "Insert go func(args...) goroutine launch.",
        hint: "target, func, args?, position?, index?",
        required: ["target", "func"],
        optional: ["args", "position", "index"],
        exampleParams: { op: "insert_go", target: "Server.Start", func: "s.serve", args: "ident:ln", position: "last" },
      },
      {
        op: "insert_error_check",
        description: "Insert if errVar != nil { return values... } — shorthand for if + return.",
        hint: "target, errVar, returnValues (kind:val,...), position?, index?",
        required: ["target", "errVar", "returnValues"],
        optional: ["position", "index"],
        exampleParams: { op: "insert_error_check", target: "NewConfig", errVar: "err", returnValues: "nil,ident:err", position: "at_index", index: 1 },
      },
      {
        op: "push_arg",
        description: "Append an argument to a call expression at stmtIndex. callIndex targets a call within return/assign.",
        hint: "target, stmtIndex, arg (kind:val), callIndex?",
        required: ["target", "stmtIndex", "arg"],
        optional: ["callIndex"],
        exampleParams: { op: "push_arg", target: "Server.Start", stmtIndex: 0, arg: "selector:s.port" },
      },
      {
        op: "set_else",
        description: "Add an empty else block to an if statement at stmtIndex.",
        hint: "target, stmtIndex",
        required: ["target", "stmtIndex"],
        optional: [],
        exampleParams: { op: "set_else", target: "NewConfig", stmtIndex: 1 },
      },
    ],
    programmingConcept: `A function body is a sequence of statements executed in order. Each statement is one instruction.

Common statement types:
- Function call: execute a function, e.g. fmt.Println("hello")
- Method call: call a method on a receiver, e.g. s.init()
- Assignment: store a value, e.g. data, err := os.ReadFile(path)
- Return: exit the function with values, e.g. return nil, err
- If: conditional execution, e.g. if err != nil { return err }
- For-range: iterate over a collection, e.g. for _, item := range items { }
- Defer: schedule code to run when the function exits
- Go: launch a concurrent goroutine
- Variable declaration: declare a typed variable, e.g. var cfg Config

Statements are added one at a time. Position controls where: first, last, at_index (0-based).
Compound statement bodies (inside if/for) are targeted with dotted paths: Func.if[N], Func.for[N], Func.if[N].else.`,
    golangSyntax: `Go statements:

// Function call
fmt.Println("hello world")

// Method call
s.init()

// Assignment with function call (short := or long =)
data, err := os.ReadFile(path)
s.running = true

// Variable declaration inside function
var cfg Config

// Return with values
return &cfg, nil

// If statement
if err != nil {
    return nil, err
}

// If with init statement
if err := s.validate(); err != nil {
    return err
}

// For-range loop
for _, client := range s.clients {
    client.Close()
}

// Defer (runs when function exits)
defer f.Close()

// Goroutine
go s.serve(ln)

// Error check pattern (if err != nil { return ... })
data, err := os.ReadFile(path)
if err != nil {
    return nil, err
}`,
  },

  // ---- Category 4: Struct Field Editing ----
  {
    name: "Struct Fields",
    operations: [
      {
        op: "add_struct_field",
        description: "Add a field to a struct.",
        hint: "target (struct), fieldName, fieldType, tag?, position?, anchor?",
        required: ["target", "fieldName", "fieldType"],
        optional: ["tag", "position", "anchor"],
        exampleParams: { op: "add_struct_field", target: "Config", fieldName: "Port", fieldType: "int", tag: "`json:\"port\"`" },
      },
      {
        op: "remove_struct_field",
        description: "Remove a field from a struct.",
        hint: "target (Struct.Field)",
        required: ["target"],
        optional: [],
        exampleParams: { op: "remove_struct_field", target: "Config.Port" },
      },
      {
        op: "set_field_type",
        description: "Change the type of a struct field.",
        hint: "target (Struct.Field), fieldType",
        required: ["target", "fieldType"],
        optional: [],
        exampleParams: { op: "set_field_type", target: "Config.Port", fieldType: "uint16" },
      },
      {
        op: "set_field_name",
        description: "Rename a struct field.",
        hint: "target (Struct.Field), fieldName",
        required: ["target", "fieldName"],
        optional: [],
        exampleParams: { op: "set_field_name", target: "Config.Port", fieldName: "ListenPort" },
      },
      {
        op: "set_struct_tag",
        description: "Set or update one struct tag key. Other keys are preserved.",
        hint: "target (Struct.Field), tagKey, tagValue",
        required: ["target", "tagKey", "tagValue"],
        optional: [],
        exampleParams: { op: "set_struct_tag", target: "Config.Port", tagKey: "json", tagValue: "port,omitempty" },
      },
      {
        op: "remove_struct_tag",
        description: "Remove one tag key, or the entire tag (omit tagKey).",
        hint: "target (Struct.Field), tagKey?",
        required: ["target"],
        optional: ["tagKey"],
        exampleParams: { op: "remove_struct_tag", target: "Config.Port", tagKey: "json" },
      },
    ],
    programmingConcept: `A struct groups related data into named fields. Each field has a name, a type, and optionally a tag.

Struct tags are metadata attached to fields, used by libraries for serialization (JSON, database mapping, etc.).
Tags use a key:"value" format inside backticks: \`json:"port" db:"port_num"\`

Fields can be added, removed, renamed, retyped, and their tags can be set or removed individually.`,
    golangSyntax: `Go struct fields:

type Config struct {
    Port    int    \`json:"port"\`
    Host    string \`json:"host,omitempty"\`
    handler http.Handler
}

// Tags use key:"value" format in backticks
// Common tag keys: json, db, yaml, xml, mapstructure`,
  },

  // ---- Category 5: Interface Editing ----
  {
    name: "Interface Editing",
    operations: [
      {
        op: "add_interface_method",
        description: "Add a method signature to an interface.",
        hint: "target (interface), methodName, params, returns",
        required: ["target", "methodName"],
        optional: ["params", "returns"],
        exampleParams: { op: "add_interface_method", target: "Handler", methodName: "ServeHTTP", params: "w:http.ResponseWriter,r:*http.Request", returns: "" },
      },
      {
        op: "remove_interface_method",
        description: "Remove a method from an interface.",
        hint: "target (Interface.Method)",
        required: ["target"],
        optional: [],
        exampleParams: { op: "remove_interface_method", target: "Handler.ServeHTTP" },
      },
      {
        op: "add_interface_embed",
        description: "Embed another interface type.",
        hint: "target (interface), embedType",
        required: ["target", "embedType"],
        optional: [],
        exampleParams: { op: "add_interface_embed", target: "Handler", embedType: "io.Closer" },
      },
    ],
    programmingConcept: `An interface defines a set of method signatures that a type must implement. Interfaces enable polymorphism — different types can satisfy the same interface.

Embedding includes all methods from another interface. For example, embedding io.Reader adds the Read method.`,
    golangSyntax: `Go interfaces:

type Handler interface {
    ServeHTTP(w http.ResponseWriter, r *http.Request)
    io.Closer  // embed: includes Close() error
}`,
  },

  // ---- Category 6: Import Management ----
  {
    name: "Import Management",
    operations: [
      {
        op: "add_import",
        description: "Add an import path. Use alias for named imports.",
        hint: "path, alias?",
        required: ["path"],
        optional: ["alias"],
        exampleParams: { op: "add_import", path: "fmt" },
      },
      {
        op: "remove_import",
        description: "Remove an import path. Use this to clean up unused imports.",
        hint: "path",
        required: ["path"],
        optional: [],
        exampleParams: { op: "remove_import", path: "os" },
      },
    ],
    programmingConcept: `Imports make external packages available in the file. Each import specifies a package path.

An alias gives an import a different local name (e.g. import pb "google.golang.org/protobuf").
Common imports: "fmt" (formatting), "os" (operating system), "net/http" (HTTP), "context", "errors", "strings", "strconv", "io", "log".`,
    golangSyntax: `Go imports:

import "fmt"                              // single import
import (                                  // grouped imports
    "fmt"
    "os"
    pb "google.golang.org/protobuf"       // aliased import
    _ "net/http/pprof"                    // blank import (side effects only)
)`,
  },

  // ---- Category 7: Const/Var Editing ----
  {
    name: "Const/Var Editing",
    operations: [
      {
        op: "set_const_value",
        description: "Change the value of a constant.",
        hint: "target, valueSpec OR valueFunc+valueArgs",
        required: ["target"],
        optional: ["valueSpec", "valueFunc", "valueArgs"],
        exampleParams: { op: "set_const_value", target: "DefaultPort", valueSpec: "int:9090" },
      },
      {
        op: "set_var_value",
        description: "Change the value of a variable.",
        hint: "target, valueSpec OR valueFunc+valueArgs",
        required: ["target"],
        optional: ["valueSpec", "valueFunc", "valueArgs"],
        exampleParams: { op: "set_var_value", target: "ErrNotFound", valueFunc: "errors.New", valueArgs: "string:resource not found" },
      },
      {
        op: "set_const_type",
        description: "Change the type annotation of a constant.",
        hint: "target, type",
        required: ["target", "type"],
        optional: [],
        exampleParams: { op: "set_const_type", target: "DefaultPort", type: "uint16" },
      },
      {
        op: "set_var_type",
        description: "Change the type annotation of a variable.",
        hint: "target, type",
        required: ["target", "type"],
        optional: [],
        exampleParams: { op: "set_var_type", target: "ErrNotFound", type: "error" },
      },
      {
        op: "remove_var",
        description: "Remove a variable from its block.",
        hint: "target",
        required: ["target"],
        optional: [],
        exampleParams: { op: "remove_var", target: "ErrNotFound" },
      },
      {
        op: "remove_const",
        description: "Remove a constant from its block.",
        hint: "target",
        required: ["target"],
        optional: [],
        exampleParams: { op: "remove_const", target: "DefaultPort" },
      },
    ],
    programmingConcept: `Constants and variables store named values at the package level.

A constant's value is fixed at compile time and cannot change.
A variable's value can change at runtime.
Both can have explicit types and values. Values use flat encoding: int:42, string:hello, nil, ident:x.
For function call values (e.g. errors.New("not found")), use valueFunc + valueArgs instead of valueSpec.`,
    golangSyntax: `Go constants and variables:

const DefaultPort int = 8080
const MaxRetries = 3                      // type inferred
var ErrNotFound = errors.New("not found") // function call value
var globalLogger *log.Logger              // typed, no initial value

const (   // const block
    StatusOK    = 200
    StatusError = 500
)`,
  },

  // ---- Category 8: Package/File Level ----
  {
    name: "Package/File",
    operations: [
      {
        op: "set_package",
        description: "Change the package name.",
        hint: "name",
        required: ["name"],
        optional: [],
        exampleParams: { op: "set_package", name: "server" },
      },
      {
        op: "set_build_constraint",
        description: "Set or replace the //go:build constraint line.",
        hint: "constraint",
        required: ["constraint"],
        optional: [],
        exampleParams: { op: "set_build_constraint", constraint: "linux && amd64" },
      },
      {
        op: "add_generate_directive",
        description: "Add a //go:generate directive.",
        hint: "directive, anchor?, position?",
        required: ["directive"],
        optional: ["anchor", "position"],
        exampleParams: { op: "add_generate_directive", directive: "stringer -type=Color" },
      },
    ],
    programmingConcept: `Every Go file starts with a package declaration. All files in the same directory must have the same package name.

Build constraints control which files are compiled on which platforms.
Generate directives trigger code generation tools.`,
    golangSyntax: `Go file-level declarations:

//go:build linux && amd64

package server

//go:generate stringer -type=Color`,
  },

  // ---- Category 9: Comments/Documentation ----
  {
    name: "Comments",
    operations: [
      {
        op: "set_doc_comment",
        description: "Set or replace the doc comment above a declaration.",
        hint: "target, text",
        required: ["target", "text"],
        optional: [],
        exampleParams: { op: "set_doc_comment", target: "Server", text: "Server handles incoming HTTP requests." },
      },
      {
        op: "remove_doc_comment",
        description: "Remove the doc comment from a declaration.",
        hint: "target",
        required: ["target"],
        optional: [],
        exampleParams: { op: "remove_doc_comment", target: "Server" },
      },
      {
        op: "add_line_comment",
        description: "Add an inline comment at the end of a declaration's line.",
        hint: "target, text",
        required: ["target", "text"],
        optional: [],
        exampleParams: { op: "add_line_comment", target: "DefaultPort", text: "default HTTP port" },
      },
    ],
    programmingConcept: `Comments document code. Doc comments appear directly above a declaration and are used by documentation tools.

Line comments appear at the end of a line of code.
All comments use // prefix in Go.`,
    golangSyntax: `Go comments:

// Server handles incoming HTTP requests.
type Server struct { }

const DefaultPort = 8080 // default HTTP port`,
  },

  // ---- Category 10: Refactoring ----
  {
    name: "Refactoring",
    operations: [
      {
        op: "rename",
        description: "Rename all references to target within the file.",
        hint: "target, newName",
        required: ["target", "newName"],
        optional: [],
        exampleParams: { op: "rename", target: "Server", newName: "HTTPServer" },
      },
      {
        op: "extract_interface",
        description: "Generate an interface from a struct type's exported methods.",
        hint: "target, interfaceName",
        required: ["target", "interfaceName"],
        optional: [],
        exampleParams: { op: "extract_interface", target: "Server", interfaceName: "ServerInterface" },
      },
    ],
    programmingConcept: `Refactoring restructures code without changing its behavior.

Renaming changes the name of a declaration and all its references within the file.
Extracting an interface creates a new interface type from a struct's public methods.`,
    golangSyntax: `Go refactoring examples:

// Before rename: Server → HTTPServer
type Server struct { }       →  type HTTPServer struct { }
func (s *Server) Start()     →  func (s *HTTPServer) Start()

// Extract interface from struct methods
type ServerInterface interface {
    Start(ctx context.Context) error
    Stop() error
}`,
  },
]

// ---------------------------------------------------------------------------
// Workflow Examples
// ---------------------------------------------------------------------------

export interface WorkflowDef {
  name: string
  description: string
  steps: { description: string; params: Record<string, unknown> }[]
}

export const GO_AST_WORKFLOWS: WorkflowDef[] = [
  {
    name: "Create a function that reads a config file",
    description: "Create NewConfig(path string) (*Config, error) that reads a file and returns the config.",
    steps: [
      { description: "Create the function", params: { op: "create_function", name: "NewConfig", params: "path:string", returns: "*Config,error" } },
      { description: "Read the file", params: { op: "insert_assign_call", target: "NewConfig", vars: "data,err", short: true, func: "os.ReadFile", args: "ident:path", position: "first" } },
      { description: "Check for error", params: { op: "insert_error_check", target: "NewConfig", errVar: "err", returnValues: "nil,ident:err", position: "at_index", index: 1 } },
      { description: "Declare config var", params: { op: "insert_var_decl", target: "NewConfig", varName: "cfg", varType: "Config", position: "at_index", index: 2 } },
      { description: "Unmarshal JSON", params: { op: "insert_assign_call", target: "NewConfig", vars: "err", short: false, func: "json.Unmarshal", args: "ident:data,addr:cfg", position: "at_index", index: 3 } },
      { description: "Check unmarshal error", params: { op: "insert_error_check", target: "NewConfig", errVar: "err", returnValues: "nil,ident:err", position: "at_index", index: 4 } },
      { description: "Return result", params: { op: "insert_return", target: "NewConfig", values: "addr:cfg,nil", position: "last" } },
      { description: "Add import", params: { op: "add_import", path: "encoding/json" } },
    ],
  },
  {
    name: "Add a method with error handling",
    description: "Add a Start method to Server that listens on a port.",
    steps: [
      { description: "Create the method", params: { op: "create_method", receiverType: "*Server", receiverVar: "s", name: "Start", params: "ctx:context.Context", returns: "error" } },
      { description: "Format address", params: { op: "insert_assign_call", target: "Server.Start", vars: "addr", short: true, func: "fmt.Sprintf", args: "string::%d,selector:s.port", position: "first" } },
      { description: "Start listener", params: { op: "insert_assign_call", target: "Server.Start", vars: "ln,err", short: true, func: "net.Listen", args: "string:tcp,ident:addr", position: "at_index", index: 1 } },
      { description: "Check error", params: { op: "insert_error_check", target: "Server.Start", errVar: "err", returnValues: "ident:err", position: "at_index", index: 2 } },
      { description: "Store listener", params: { op: "insert_assign_value", target: "Server.Start", varName: "s.listener", valueSpec: "ident:ln", position: "at_index", index: 3 } },
      { description: "Return nil", params: { op: "insert_return", target: "Server.Start", values: "nil", position: "last" } },
    ],
  },
  {
    name: "Create a struct with fields and tags",
    description: "Create a Config struct with JSON-tagged fields.",
    steps: [
      { description: "Create the struct", params: { op: "create_struct", name: "Config" } },
      { description: "Add Host field", params: { op: "add_struct_field", target: "Config", fieldName: "Host", fieldType: "string", tag: "`json:\"host\"`" } },
      { description: "Add Port field", params: { op: "add_struct_field", target: "Config", fieldName: "Port", fieldType: "int", tag: "`json:\"port\"`" } },
      { description: "Add doc comment", params: { op: "set_doc_comment", target: "Config", text: "Config holds server configuration." } },
    ],
  },
  {
    name: "Clean up unused imports",
    description: "Remove unused imports reported by the Go language server.",
    steps: [
      { description: "Remove unused os import", params: { op: "remove_import", path: "os" } },
      { description: "Remove unused time import", params: { op: "remove_import", path: "time" } },
    ],
  },
]

// ---------------------------------------------------------------------------
// Generator: Tool Call Syntax
// ---------------------------------------------------------------------------

/**
 * Generate a tool call example using the individual tool name (go_<op>) instead
 * of the monolithic go_edit. Strips the "op" field from exampleParams and adds
 * filePath so every example shows the correct parameter pattern.
 */
function wrapOpToolCall(op: OpDef): string {
  const { op: _op, ...cleanParams } = op.exampleParams
  return wrapToolCall(`go_${op.op}`, { filePath: "main.go", ...cleanParams })
}

/** Same as wrapOpToolCall but for workflow steps that have an op field in params. */
function wrapStepToolCall(params: Record<string, unknown>): string {
  const { op, ...cleanParams } = params
  const withFile = cleanParams.filePath ? cleanParams : { filePath: "main.go", ...cleanParams }
  return wrapToolCall(`go_${op as string}`, withFile)
}

export function generateToolCallSyntax(): string {
  const lines: string[] = []
  lines.push("When you call a tool, use this format:")
  lines.push(`${Gemma4Tokens.toolCallStart}call:tool_name{param1:value1,param2:value2}${Gemma4Tokens.toolCallEnd}`)
  lines.push("")
  lines.push(`String values must be wrapped in ${Gemma4Tokens.quote} delimiters:`)
  lines.push(`${Gemma4Tokens.toolCallStart}call:go_insert_call{filePath:${Gemma4Tokens.quote}main.go${Gemma4Tokens.quote},target:${Gemma4Tokens.quote}main${Gemma4Tokens.quote},func:${Gemma4Tokens.quote}fmt.Println${Gemma4Tokens.quote}}${Gemma4Tokens.toolCallEnd}`)
  lines.push("")
  lines.push("IMPORTANT: filePath is the file system path (e.g. main.go). target is the declaration inside the file (e.g. main, Server.Start).")
  return lines.join("\n")
}

// ---------------------------------------------------------------------------
// Generator: Operation Reference
// ---------------------------------------------------------------------------

export function generateGoAstReference(): string {
  const lines: string[] = []
  lines.push("Go AST editing tools reference:")
  lines.push("")

  for (const cat of GO_AST_REGISTRY) {
    lines.push(`## ${cat.name}`)
    lines.push("")
    for (const op of cat.operations) {
      lines.push(`go_${op.op}: ${op.description}`)
      lines.push(`  Required: ${op.required.join(", ") || "(none)"}`)
      if (op.optional.length > 0) {
        lines.push(`  Optional: ${op.optional.join(", ")}`)
      }
      lines.push(`  Example: ${wrapOpToolCall(op)}`)
      lines.push("")
    }
  }

  return lines.join("\n")
}

// ---------------------------------------------------------------------------
// Generator: Workflow Examples
// ---------------------------------------------------------------------------

export function generateGoAstWorkflows(): string {
  const lines: string[] = []
  lines.push("Multi-step workflow examples:")
  lines.push("")

  for (const wf of GO_AST_WORKFLOWS) {
    lines.push(`### ${wf.name}`)
    lines.push(wf.description)
    lines.push("")
    for (let i = 0; i < wf.steps.length; i++) {
      const step = wf.steps[i]!
      lines.push(`Step ${i + 1}: ${step.description}`)
      lines.push(wrapStepToolCall(step.params))
      lines.push("")
    }
  }

  return lines.join("\n")
}

// ---------------------------------------------------------------------------
// Generator: Quick Reference Table
// ---------------------------------------------------------------------------

export function generateQuickReference(): string {
  const lines: string[] = []
  lines.push("Go AST tools quick reference:")
  lines.push("")

  for (const cat of GO_AST_REGISTRY) {
    lines.push(`${cat.name}:`)
    for (const op of cat.operations) {
      lines.push(`  go_${op.op} — ${op.hint}`)
    }
    lines.push("")
  }

  return lines.join("\n")
}

// ---------------------------------------------------------------------------
// Generator: Programming Concepts
// ---------------------------------------------------------------------------

export function generateProgrammingConcepts(): string {
  const lines: string[] = []
  lines.push("Programming concepts for Go AST editing:")
  lines.push("")

  for (const cat of GO_AST_REGISTRY) {
    lines.push(`## ${cat.name}`)
    lines.push("")
    lines.push(cat.programmingConcept)
    lines.push("")
    lines.push("Examples:")
    for (const op of cat.operations) {
      lines.push(wrapOpToolCall(op))
    }
    lines.push("")
  }

  return lines.join("\n")
}

// ---------------------------------------------------------------------------
// Generator: Go Syntax Guide
// ---------------------------------------------------------------------------

export function generateGolangSyntax(): string {
  const lines: string[] = []
  lines.push("Go syntax reference with tool call examples:")
  lines.push("")

  for (const cat of GO_AST_REGISTRY) {
    lines.push(`## ${cat.name}`)
    lines.push("")
    lines.push(cat.golangSyntax)
    lines.push("")
    lines.push("Produce this with:")
    for (const op of cat.operations) {
      lines.push(wrapOpToolCall(op))
    }
    lines.push("")
  }

  // Static reference tables
  lines.push("## Common Go Types")
  lines.push("")
  lines.push("string, int, int64, float64, bool, error, byte, rune")
  lines.push("[]byte, []string, map[string]string, map[string]interface{}")
  lines.push("*T (pointer), []T (slice), chan T (channel)")
  lines.push("context.Context, io.Reader, io.Writer, http.Handler")
  lines.push("")

  lines.push("## Common Import Paths")
  lines.push("")
  lines.push("fmt, os, io, log, errors, strings, strconv, bytes")
  lines.push("context, sync, time, math, sort, regexp")
  lines.push("net, net/http, net/url, path, path/filepath")
  lines.push("encoding/json, encoding/xml, encoding/base64")
  lines.push("database/sql, html/template, text/template")
  lines.push("")

  // Flat value encoding reference
  lines.push("## Flat Value Encoding")
  lines.push("")
  lines.push("All values in Go AST tools use kind:content format:")
  lines.push("  ident:name   → identifier (e.g. ident:err)")
  lines.push("  nil          → nil literal")
  lines.push("  true / false → boolean literals")
  lines.push("  int:N        → integer (e.g. int:42)")
  lines.push("  float:N      → float (e.g. float:3.14)")
  lines.push("  string:text  → string literal (e.g. string:hello)")
  lines.push("  selector:a.b → dotted access (e.g. selector:s.Port)")
  lines.push("  addr:name    → address-of (e.g. addr:cfg → &cfg)")
  lines.push("  call:func    → zero-arg call (e.g. call:fmt.Errorf)")
  lines.push("")
  lines.push("Multiple values are comma-separated: nil,ident:err")
  lines.push("Parameters use name:type format: ctx:context.Context,id:string")

  return lines.join("\n")
}

// ---------------------------------------------------------------------------
// Generator: Combined System Prompt
// ---------------------------------------------------------------------------

export function generateGoAstSystemPrompt(): string {
  const sections: string[] = []

  sections.push("# Go AST Editing Guide")
  sections.push("")
  sections.push("Use go_inspect to understand a file's structure, then call the appropriate editing tool (e.g. go_create_function, go_add_struct_field) to modify it.")
  sections.push("Each tool call makes ONE atomic change. Build complex edits with multiple calls.")
  sections.push("All parameters are basic types only: strings, numbers, booleans. No arrays, no objects.")
  sections.push("")

  sections.push(generateQuickReference())
  sections.push(generateGoAstReference())
  sections.push(generateGoAstWorkflows())
  sections.push(generateGolangSyntax())
  sections.push(generateProgrammingConcepts())

  return sections.join("\n")
}

/**
 * Generate skill activation content for a specific Go AST tool.
 * Used by the skill tool when the model calls skill("go_edit") etc.
 * Returns the full operation reference + examples for on-demand loading.
 */
export function generateGoAstSkillContent(toolId: string): string {
  if (toolId === "go_inspect") {
    return [
      "## Go AST Inspect",
      "",
      "Returns comprehensive file structure: package, imports, functions with",
      "signatures and body statements, types with fields, const/var declarations.",
      "Use the output to understand targets for go_edit operations.",
    ].join("\n")
  }

  // For go_edit and individual go_* tools, return the full reference
  const sections: string[] = []
  sections.push(generateQuickReference())
  sections.push(generateGoAstReference())
  sections.push(generateGoAstWorkflows())
  sections.push(generateGolangSyntax())
  return sections.join("\n")
}
