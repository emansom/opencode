package main

// Operation represents a single AST editing operation.
type Operation struct {
	Mode string `json:"mode"` // "edit" or "inspect"
	File string `json:"file"`
	Op   string `json:"op,omitempty"`

	// Target: dotted path (e.g. "Server.Start", "Config.Port", "NewConfig.if[1]")
	Target string `json:"target,omitempty"`

	// Declaration creation
	Name           string `json:"name,omitempty"`
	ReceiverType   string `json:"receiverType,omitempty"`
	ReceiverVar    string `json:"receiverVar,omitempty"`
	TargetType     string `json:"targetType,omitempty"`
	UnderlyingType string `json:"underlyingType,omitempty"`

	// Parameters and returns (comma-separated)
	Params  string `json:"params,omitempty"`
	Returns string `json:"returns,omitempty"`

	// Values (flat "kind:content" encoding)
	ValueSpec string `json:"valueSpec,omitempty"`
	ValueFunc string `json:"valueFunc,omitempty"`
	ValueArgs string `json:"valueArgs,omitempty"`
	Values    string `json:"values,omitempty"`
	Args      string `json:"args,omitempty"`
	Arg       string `json:"arg,omitempty"`

	// Call parameters
	Func     string `json:"func,omitempty"`
	Receiver string `json:"receiver,omitempty"`
	Method   string `json:"method,omitempty"`

	// Assignment parameters
	Vars    string `json:"vars,omitempty"`
	VarName string `json:"varName,omitempty"`
	VarType string `json:"varType,omitempty"`
	Short   *bool  `json:"short,omitempty"`

	// If condition parameters
	CondLeft  string `json:"condLeft,omitempty"`
	Operator  string `json:"operator,omitempty"`
	CondRight string `json:"condRight,omitempty"`

	// If-init parameters
	InitVars string `json:"initVars,omitempty"`
	InitFunc string `json:"initFunc,omitempty"`
	InitArgs string `json:"initArgs,omitempty"`

	// For-range parameters
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
	Iterable string `json:"iterable,omitempty"`

	// Error check
	ErrVar       string `json:"errVar,omitempty"`
	ReturnValues string `json:"returnValues,omitempty"`

	// Statement index
	Index     *int `json:"index,omitempty"`
	StmtIndex *int `json:"stmtIndex,omitempty"`
	CallIndex *int `json:"callIndex,omitempty"`

	// Struct field parameters
	FieldName string `json:"fieldName,omitempty"`
	FieldType string `json:"fieldType,omitempty"`

	// Interface method parameters
	MethodName string `json:"methodName,omitempty"`

	// Import parameters
	Path  string `json:"path,omitempty"`
	Alias string `json:"alias,omitempty"`

	// Positioning
	Position string `json:"position,omitempty"`
	Anchor   string `json:"anchor,omitempty"`

	// Tag parameters
	TagKey   string `json:"tagKey,omitempty"`
	TagValue string `json:"tagValue,omitempty"`
	Tag      string `json:"tag,omitempty"`

	// Type parameter (for set_const_type, set_var_type, set_field_type)
	Type string `json:"type,omitempty"`

	// Comment parameters
	Text string `json:"text,omitempty"`

	// Refactoring parameters
	NewName       string `json:"newName,omitempty"`
	InterfaceName string `json:"interfaceName,omitempty"`
	EmbedType     string `json:"embedType,omitempty"`

	// Package/file parameters
	Constraint string `json:"constraint,omitempty"`
	Directive  string `json:"directive,omitempty"`
	Block      string `json:"block,omitempty"`
}

// EditResult is the output of an edit operation.
type EditResult struct {
	Success bool     `json:"success"`
	Diff    string   `json:"diff,omitempty"`
	Content string   `json:"content,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

// InspectResult is the output of an inspect operation.
type InspectResult struct {
	Package            string           `json:"package"`
	BuildConstraint    string           `json:"buildConstraint,omitempty"`
	GenerateDirectives []string         `json:"generateDirectives,omitempty"`
	Imports            []ImportInfo     `json:"imports"`
	Types              []TypeInfo       `json:"types"`
	Functions          []FunctionInfo   `json:"functions"`
	Consts             []VarConstInfo   `json:"consts"`
	Vars               []VarConstInfo   `json:"vars"`
}

type ImportInfo struct {
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
}

type TypeInfo struct {
	Name       string            `json:"name"`
	Kind       string            `json:"kind"` // "struct", "interface", "alias", "named"
	Exported   bool              `json:"exported"`
	Doc        string            `json:"doc,omitempty"`
	Fields     []FieldInfo       `json:"fields,omitempty"`
	Methods    []InterfaceMethod `json:"methods,omitempty"`
	Embeds     []string          `json:"embeds,omitempty"`
	Target     string            `json:"target,omitempty"` // for aliases
	Occurrence int               `json:"occurrence,omitempty"` // set when multiple declarations share this name; use Name#N to target
}

type FieldInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Tag  string `json:"tag,omitempty"`
	Doc  string `json:"doc,omitempty"`
}

type InterfaceMethod struct {
	Name      string `json:"name"`
	Signature string `json:"signature"`
}

type FunctionInfo struct {
	Name        string        `json:"name"`
	Receiver    string        `json:"receiver,omitempty"`
	ReceiverVar string        `json:"receiverVar,omitempty"`
	Signature   string        `json:"signature"`
	Params      []ParamInfo   `json:"params"`
	Returns     []string      `json:"returns"`
	Doc         string        `json:"doc,omitempty"`
	Body        []BodyStmt    `json:"body"`
	Occurrence  int           `json:"occurrence,omitempty"` // set when multiple declarations share this name; use Name#N to target
}

type ParamInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type BodyStmt struct {
	Index     int    `json:"index"`
	Statement string `json:"statement"`
}

type VarConstInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type,omitempty"`
	Value      string `json:"value,omitempty"`
	Block      string `json:"block,omitempty"`
	Occurrence int    `json:"occurrence,omitempty"` // set when multiple declarations share this name; use Name#N to target
}
