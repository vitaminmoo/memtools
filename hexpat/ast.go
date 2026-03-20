// Package hexpat provides a parser for the ImHex pattern language (.hexpat files).
// It produces an AST that can be consumed by code generators.
package hexpat

// File is the top-level AST node representing an entire .hexpat file.
type File struct {
	Items []Item
}

// Item is anything that can appear at the top level of a file.
type Item interface {
	itemNode()
}

// --- Preprocessor / metadata ---

type Pragma struct {
	Key   string
	Value string
}

func (Pragma) itemNode() {}

type Include struct {
	Path    string
	System  bool // <path> vs "path"
}

func (Include) itemNode() {}

type Define struct {
	Name  string
	Value string
}

func (Define) itemNode() {}

type IfDef struct {
	Name    string
	Negated bool // true for #ifndef
	Body    []Item
	Else    []Item // optional #else block
}

func (IfDef) itemNode() {}

// --- Imports ---

type Import struct {
	Path  []string // e.g. ["std", "mem"]
	Alias string   // optional alias
}

func (Import) itemNode() {}

// --- Type definitions ---

type StructDef struct {
	Name       string
	TypeParams []string // template parameters
	Parent     string   // inheritance
	Body       []Statement
	Attrs      []Attribute
}

func (StructDef) itemNode() {}

type UnionDef struct {
	Name       string
	TypeParams []string
	Body       []Statement
	Attrs      []Attribute
}

func (UnionDef) itemNode() {}

type EnumDef struct {
	Name           string
	UnderlyingType TypeNode
	Members        []EnumMember
	Attrs          []Attribute
}

func (EnumDef) itemNode() {}

type EnumMember struct {
	Name     string
	Value    Expr   // nil if auto-increment
	RangeEnd Expr   // non-nil for range entries (A = 0x00 ... 0x7F)
}

type BitfieldDef struct {
	Name       string
	TypeParams []string
	Body       []BitfieldEntry
	Attrs      []Attribute
}

func (BitfieldDef) itemNode() {}

type BitfieldEntry struct {
	Name  string // empty for padding
	Bits  Expr
	Type  TypeNode // optional enum/bool type
	Attrs []Attribute
}

type UsingDef struct {
	Name       string
	TypeParams []string
	Type       TypeNode
	Attrs      []Attribute
}

func (UsingDef) itemNode() {}

type FnDef struct {
	Name   string
	Params []FnParam
	Body   []Statement
}

func (FnDef) itemNode() {}

type FnParam struct {
	Type    TypeNode
	Name    string
	Ref     bool // pass by reference
	Default Expr // default value (= expr), nil if none
}

type NamespaceDef struct {
	Name  string
	Auto  bool // "auto namespace"
	Items []Item
}

func (NamespaceDef) itemNode() {}

// --- Statements (inside struct/union/function bodies) ---

type Statement interface {
	stmtNode()
}

type VarDecl struct {
	Type     TypeNode
	Name     string
	Array    Expr // array size expression, nil if not an array
	Pointer  *PointerInfo
	Offset   Expr        // @ expression, nil if sequential
	Init     Expr        // initializer expression (= expr), nil if none
	Attrs    []Attribute
}

func (VarDecl) itemNode() {}
func (VarDecl) stmtNode() {}

type PointerInfo struct {
	SizeType TypeNode // the type encoding the pointer value (e.g. u32)
}

type IfStmt struct {
	Cond Expr
	Then []Statement
	Else []Statement // may contain a single IfStmt for else-if chains
}

func (IfStmt) stmtNode() {}

type WhileStmt struct {
	Cond Expr
	Body []Statement
}

func (WhileStmt) stmtNode() {}

type ForStmt struct {
	Init Statement
	Cond Expr
	Post Statement
	Body []Statement
}

func (ForStmt) stmtNode() {}

type MatchStmt struct {
	Args []Expr
	Arms []MatchArm
}

func (MatchStmt) stmtNode() {}

type MatchArm struct {
	Patterns []MatchPattern // one per match arg
	Body     []Statement
}

type MatchPattern struct {
	Value    Expr // nil for wildcard (_)
	RangeEnd Expr // non-nil for range patterns (0 ... 63)
	Wildcard bool
}

type ReturnStmt struct {
	Value Expr
}

func (ReturnStmt) stmtNode() {}

type BreakStmt struct{}

func (BreakStmt) stmtNode() {}

type ContinueStmt struct{}

func (ContinueStmt) stmtNode() {}

type AssignStmt struct {
	Target Expr
	Op     string // "=", "+=", "-=", etc.
	Value  Expr
}

func (AssignStmt) stmtNode() {}

type ExprStmt struct {
	Expr Expr
}

func (ExprStmt) stmtNode() {}

type PaddingStmt struct {
	Size Expr
}

func (PaddingStmt) stmtNode() {}

type TryCatchStmt struct {
	Try   []Statement
	Catch []Statement
}

func (TryCatchStmt) stmtNode() {}

// --- Expressions ---

type Expr interface {
	exprNode()
}

type NumberLit struct {
	Value  int64
	Raw    string // original text for preserving hex/octal/binary
}

func (NumberLit) exprNode() {}

type FloatLit struct {
	Value float64
}

func (FloatLit) exprNode() {}

type StringLit struct {
	Value string
}

func (StringLit) exprNode() {}

type BoolLit struct {
	Value bool
}

func (BoolLit) exprNode() {}

type CharLit struct {
	Value rune
}

func (CharLit) exprNode() {}

type Ident struct {
	Name string
}

func (Ident) exprNode() {}

type MemberAccess struct {
	Object Expr
	Member string
}

func (MemberAccess) exprNode() {}

type NamespaceAccess struct {
	Namespace string
	Member    Expr
}

func (NamespaceAccess) exprNode() {}

type IndexAccess struct {
	Object Expr
	Index  Expr
}

func (IndexAccess) exprNode() {}

type BinaryExpr struct {
	Op    string
	Left  Expr
	Right Expr
}

func (BinaryExpr) exprNode() {}

type UnaryExpr struct {
	Op      string
	Operand Expr
	Prefix  bool
}

func (UnaryExpr) exprNode() {}

type TernaryExpr struct {
	Cond Expr
	Then Expr
	Else Expr
}

func (TernaryExpr) exprNode() {}

type FnCall struct {
	Func Expr
	Args []Expr
}

func (FnCall) exprNode() {}

type SizeOfExpr struct {
	Operand Expr
}

func (SizeOfExpr) exprNode() {}

type AddressOfExpr struct {
	Operand Expr
}

func (AddressOfExpr) exprNode() {}

type CastExpr struct {
	Type    TypeNode
	Operand Expr
}

func (CastExpr) exprNode() {}

type DollarExpr struct{}

func (DollarExpr) exprNode() {}

type WhileExpr struct {
	Cond Expr
}

func (WhileExpr) exprNode() {}

type ArrayInitExpr struct {
	Elements []Expr
}

func (ArrayInitExpr) exprNode() {}

// --- Type references ---

type TypeNode interface {
	typeNode()
}

type BuiltinType struct {
	Name string // u8, u16, u32, s8, float, double, char, bool, str, auto, etc.
}

func (BuiltinType) typeNode() {}

type NamedType struct {
	Namespace []string // e.g. ["std", "ptr"]
	Name      string
	TypeArgs  []TypeNode
}

func (NamedType) typeNode() {}

type EndianType struct {
	Order string // "le" or "be"
	Inner TypeNode
}

func (EndianType) typeNode() {}

// --- Attributes ---

type Attribute struct {
	Name string
	Args []Expr // may be empty
}
