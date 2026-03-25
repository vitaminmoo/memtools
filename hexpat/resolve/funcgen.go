package resolve

import (
	"fmt"
	"strings"

	"github.com/vitaminmoo/memtools/hexpat/parser"
)

// bindFunctions scans struct attributes for [[format("fn_name")]] and binds
// the named function to the struct with concrete types.
func (r *resolver) bindFunctions() {
	for name, sd := range r.structDefs {
		st := r.resolved[name]
		if st == nil {
			continue
		}
		for _, attr := range sd.Attrs {
			if attr.Name != "format" || len(attr.Args) == 0 {
				continue
			}
			// Extract function name from attribute arg (string literal)
			fnName := attrStringArg(attr.Args[0])
			if fnName == "" {
				continue
			}
			fd, ok := r.fnDefs[fnName]
			if !ok {
				continue
			}
			binding := r.transpileFunc(fd, st)
			if binding != nil {
				r.pkg.FuncBindings = append(r.pkg.FuncBindings, binding)
			}
		}
	}
}

// attrStringArg extracts a string value from an attribute argument expression.
func attrStringArg(expr parser.Expr) string {
	switch e := expr.(type) {
	case parser.StringLit:
		return e.Value
	}
	return ""
}

// transpileFunc transpiles a hexpat function into a FuncBinding with Go code.
func (r *resolver) transpileFunc(fd *parser.FnDef, receiver *StructType) *FuncBinding {
	env := newTypeEnv(receiver, fd.Params, r)

	var stmts []TranspiledStmt
	for _, s := range fd.Body {
		ts, err := env.transpileStmt(s, 0)
		if err != nil {
			// If we can't transpile a statement, skip the whole function
			return nil
		}
		stmts = append(stmts, ts...)
	}

	retType := env.inferReturnType()
	if retType == "" {
		retType = "string" // default for format functions
	}

	return &FuncBinding{
		GoName:          toPascalCase(fd.Name),
		HexpatName:      fd.Name,
		Receiver:        receiver,
		ReturnType:      retType,
		NeedsCtx:        env.needsCtx,
		NeedsMemHelpers: env.needsMemHelpers,
		Body:            stmts,
	}
}

// typeEnv tracks type information during function transpilation.
type typeEnv struct {
	receiver       *StructType
	params         map[string]*envVar // function params → type info
	locals         map[string]*envVar // local variables
	fieldMap       map[string]string  // hexpat field name → Go PascalCase name
	returnTypes    []string           // collected from return statements
	needsCtx       bool               // set when body references ctx-dependent operations
	needsMemHelpers bool              // set when std::mem:: functions are used
	resolver       *resolver
}

type envVar struct {
	goName string
	goType string
}

func newTypeEnv(receiver *StructType, params []parser.FnParam, r *resolver) *typeEnv {
	env := &typeEnv{
		receiver: receiver,
		params:   make(map[string]*envVar),
		locals:   make(map[string]*envVar),
		fieldMap: make(map[string]string),
		resolver: r,
	}

	// Build field map from receiver struct
	for _, f := range receiver.Fields() {
		// Map original snake_case names to Go PascalCase
		// The field's Name is already PascalCase, we need to find the original
		// We build a reverse map from the struct fields
		env.fieldMap[f.Name] = f.Name // PascalCase → PascalCase (identity)
	}

	// Map function params — first param is the receiver
	if len(params) > 0 {
		p := params[0]
		env.params[p.Name] = &envVar{
			goName: "s",
			goType: receiver.Name,
		}
	}

	return env
}

// lookupField looks up a field on the receiver by Go name and returns its Go type.
func (env *typeEnv) lookupField(goName string) string {
	for _, f := range env.receiver.Fields() {
		if f.Name == goName {
			return fieldGoTypeStr(f.Type)
		}
	}
	return ""
}

// fieldGoTypeStr returns a simple Go type string for a resolved type.
func fieldGoTypeStr(rt *ResolvedType) string {
	switch rt.Kind {
	case KindPrimitive:
		return rt.Primitive.GoType
	case KindEnum:
		return rt.GoType
	case KindArray:
		return rt.GoType
	case KindPointer:
		if rt.Pointer != nil {
			return rt.Pointer.SizeType.GoType
		}
	case KindStruct:
		return rt.GoType
	}
	return rt.GoType
}

func (env *typeEnv) inferReturnType() string {
	if len(env.returnTypes) == 0 {
		return ""
	}
	// If all return types agree, use that
	t := env.returnTypes[0]
	for _, rt := range env.returnTypes[1:] {
		if rt != t {
			return "string" // fallback
		}
	}
	return t
}

// transpileStmt converts a hexpat statement to Go code lines.
func (env *typeEnv) transpileStmt(stmt parser.Statement, depth int) ([]TranspiledStmt, error) {
	switch s := stmt.(type) {
	case parser.ReturnStmt:
		return env.transpileReturn(s)
	case parser.IfStmt:
		return env.transpileIf(s, depth)
	case parser.ForStmt:
		return env.transpileFor(s, depth)
	case parser.WhileStmt:
		return env.transpileWhile(s, depth)
	case parser.VarDecl:
		return env.transpileVarDecl(s)
	case parser.AssignStmt:
		return env.transpileAssign(s)
	case parser.BreakStmt:
		return []TranspiledStmt{{Code: "break"}}, nil
	case parser.ContinueStmt:
		return []TranspiledStmt{{Code: "continue"}}, nil
	case parser.ExprStmt:
		code, _, err := env.transpileExpr(s.Expr)
		if err != nil {
			return nil, err
		}
		return []TranspiledStmt{{Code: code}}, nil
	default:
		return nil, fmt.Errorf("unsupported statement type %T", stmt)
	}
}

func (env *typeEnv) transpileReturn(s parser.ReturnStmt) ([]TranspiledStmt, error) {
	if s.Value == nil {
		return []TranspiledStmt{{Code: "return"}}, nil
	}
	code, typ, err := env.transpileExpr(s.Value)
	if err != nil {
		return nil, err
	}
	env.returnTypes = append(env.returnTypes, typ)
	return []TranspiledStmt{{Code: "return " + code}}, nil
}

func (env *typeEnv) transpileIf(s parser.IfStmt, depth int) ([]TranspiledStmt, error) {
	cond, _, err := env.transpileExpr(s.Cond)
	if err != nil {
		return nil, err
	}

	var thenStmts []TranspiledStmt
	for _, st := range s.Then {
		ts, err := env.transpileStmt(st, depth+1)
		if err != nil {
			return nil, err
		}
		thenStmts = append(thenStmts, ts...)
	}

	result := []TranspiledStmt{{
		Code:     "if " + cond,
		Children: thenStmts,
	}}

	if len(s.Else) > 0 {
		// Check for else-if chain
		if len(s.Else) == 1 {
			if elseIf, ok := s.Else[0].(parser.IfStmt); ok {
				elseIfStmts, err := env.transpileIf(elseIf, depth)
				if err != nil {
					return nil, err
				}
				if len(elseIfStmts) > 0 {
					elseIfStmts[0].Code = "} else " + elseIfStmts[0].Code
					result = append(result, elseIfStmts...)
				}
				return result, nil
			}
		}
		var elseStmts []TranspiledStmt
		for _, st := range s.Else {
			ts, err := env.transpileStmt(st, depth+1)
			if err != nil {
				return nil, err
			}
			elseStmts = append(elseStmts, ts...)
		}
		result = append(result, TranspiledStmt{
			Code:     "} else",
			Children: elseStmts,
		})
	}

	return result, nil
}

func (env *typeEnv) transpileFor(s parser.ForStmt, depth int) ([]TranspiledStmt, error) {
	// Transpile init
	initCode := ""
	if s.Init != nil {
		if vd, ok := s.Init.(parser.VarDecl); ok {
			goType := env.resolveVarType(vd)
			goName := lowerCamelCase(vd.Name)
			env.locals[vd.Name] = &envVar{goName: goName, goType: goType}
			if vd.Init != nil {
				initExpr, _, err := env.transpileExpr(vd.Init)
				if err != nil {
					return nil, err
				}
				initCode = fmt.Sprintf("%s := %s(%s)", goName, goType, initExpr)
			} else {
				initCode = fmt.Sprintf("var %s %s", goName, goType)
			}
		} else if as, ok := s.Init.(parser.AssignStmt); ok {
			target, _, _ := env.transpileExpr(as.Target)
			value, _, _ := env.transpileExpr(as.Value)
			initCode = target + " " + as.Op + " " + value
		}
	}

	// Transpile condition
	condCode := ""
	if s.Cond != nil {
		c, _, err := env.transpileExpr(s.Cond)
		if err != nil {
			return nil, err
		}
		condCode = c
	}

	// Transpile post
	postCode := ""
	if s.Post != nil {
		if as, ok := s.Post.(parser.AssignStmt); ok {
			target, _, _ := env.transpileExpr(as.Target)
			value, _, _ := env.transpileExpr(as.Value)
			postCode = target + " " + as.Op + " " + value
		}
	}

	var bodyStmts []TranspiledStmt
	for _, st := range s.Body {
		ts, err := env.transpileStmt(st, depth+1)
		if err != nil {
			return nil, err
		}
		bodyStmts = append(bodyStmts, ts...)
	}

	return []TranspiledStmt{{
		Code:     fmt.Sprintf("for %s; %s; %s", initCode, condCode, postCode),
		Children: bodyStmts,
	}}, nil
}

func (env *typeEnv) transpileWhile(s parser.WhileStmt, depth int) ([]TranspiledStmt, error) {
	cond, _, err := env.transpileExpr(s.Cond)
	if err != nil {
		return nil, err
	}

	var bodyStmts []TranspiledStmt
	for _, st := range s.Body {
		ts, err := env.transpileStmt(st, depth+1)
		if err != nil {
			return nil, err
		}
		bodyStmts = append(bodyStmts, ts...)
	}

	return []TranspiledStmt{{
		Code:     "for " + cond,
		Children: bodyStmts,
	}}, nil
}

func (env *typeEnv) transpileVarDecl(vd parser.VarDecl) ([]TranspiledStmt, error) {
	goType := env.resolveVarType(vd)
	goName := lowerCamelCase(vd.Name)
	env.locals[vd.Name] = &envVar{goName: goName, goType: goType}

	if vd.Init != nil {
		initExpr, exprType, err := env.transpileExpr(vd.Init)
		if err != nil {
			return nil, err
		}
		if goType == exprType || (goType == "string" && exprType == "string") {
			return []TranspiledStmt{{Code: fmt.Sprintf("%s := %s", goName, initExpr)}}, nil
		}
		// Type mismatch — cast the init expression to the declared type
		if exprType != "" && exprType != goType {
			return []TranspiledStmt{{Code: fmt.Sprintf("%s := %s(%s)", goName, goType, initExpr)}}, nil
		}
		return []TranspiledStmt{{Code: fmt.Sprintf("var %s %s = %s", goName, goType, initExpr)}}, nil
	}

	return []TranspiledStmt{{Code: fmt.Sprintf("var %s %s", goName, goType)}}, nil
}

func (env *typeEnv) transpileAssign(s parser.AssignStmt) ([]TranspiledStmt, error) {
	target, _, err := env.transpileExpr(s.Target)
	if err != nil {
		return nil, err
	}
	value, _, err := env.transpileExpr(s.Value)
	if err != nil {
		return nil, err
	}

	return []TranspiledStmt{{Code: target + " " + s.Op + " " + value}}, nil
}

// transpileExpr converts a hexpat expression to Go code and returns its inferred type.
func (env *typeEnv) transpileExpr(expr parser.Expr) (string, string, error) {
	switch e := expr.(type) {
	case parser.NumberLit:
		if e.Raw != "" && (strings.HasPrefix(e.Raw, "0x") || strings.HasPrefix(e.Raw, "0X")) {
			return e.Raw, "int64", nil
		}
		return fmt.Sprintf("%d", e.Value), "int64", nil

	case parser.FloatLit:
		return fmt.Sprintf("%g", e.Value), "float64", nil

	case parser.StringLit:
		return fmt.Sprintf("%q", e.Value), "string", nil

	case parser.BoolLit:
		if e.Value {
			return "true", "bool", nil
		}
		return "false", "bool", nil

	case parser.CharLit:
		return fmt.Sprintf("byte(%d)", e.Value), "byte", nil

	case parser.Ident:
		return env.resolveIdent(e.Name)

	case parser.MemberAccess:
		return env.resolveMemberAccess(e)

	case parser.IndexAccess:
		obj, objType, err := env.transpileExpr(e.Object)
		if err != nil {
			return "", "", err
		}
		idx, _, err := env.transpileExpr(e.Index)
		if err != nil {
			return "", "", err
		}
		// Array indexing: element type
		elemType := arrayElementType(objType)
		return obj + "[" + idx + "]", elemType, nil

	case parser.BinaryExpr:
		left, leftType, err := env.transpileExpr(e.Left)
		if err != nil {
			return "", "", err
		}
		right, rightType, err := env.transpileExpr(e.Right)
		if err != nil {
			return "", "", err
		}

		// String concatenation
		if e.Op == "+" && leftType == "string" {
			if rightType == "byte" || rightType == "uint8" {
				return left + " + string(" + right + ")", "string", nil
			}
			return left + " + " + right, "string", nil
		}

		// Widen byte-typed left operands in bit-shift expressions to prevent overflow.
		// In Go, shift operations preserve the type of the left operand, so uint8 << 8 == 0.
		if (e.Op == "<<" || e.Op == ">>") && (leftType == "byte" || leftType == "uint8") {
			left = "uint32(" + left + ")"
			leftType = "uint32"
		}

		// Widen byte operands in bitwise expressions when the other operand is wider,
		// to avoid Go type mismatch errors (e.g. byte | uint32 is invalid).
		if e.Op == "|" || e.Op == "&" || e.Op == "^" {
			if (leftType == "byte" || leftType == "uint8") && leftType != rightType {
				left = rightType + "(" + left + ")"
				leftType = rightType
			} else if (rightType == "byte" || rightType == "uint8") && leftType != rightType {
				right = leftType + "(" + right + ")"
				rightType = leftType
			}
		}

		// Comparison operators return bool
		resultType := leftType
		switch e.Op {
		case "==", "!=", "<", ">", "<=", ">=", "&&", "||":
			resultType = "bool"
		}

		return "(" + left + " " + e.Op + " " + right + ")", resultType, nil

	case parser.UnaryExpr:
		operand, opType, err := env.transpileExpr(e.Operand)
		if err != nil {
			return "", "", err
		}
		op := e.Op
		if op == "~" {
			op = "^" // Go bitwise NOT
		}
		if op == "!" {
			opType = "bool"
		}
		if e.Prefix {
			return "(" + op + operand + ")", opType, nil
		}
		return "(" + operand + op + ")", opType, nil

	case parser.TernaryExpr:
		// Go doesn't have ternary — this would need to be an if/else expression.
		// For now, unsupported in expression context.
		return "", "", fmt.Errorf("ternary expressions not supported in function transpilation")

	case parser.FnCall:
		return env.transpileFnCall(e)

	case parser.CastExpr:
		operand, _, err := env.transpileExpr(e.Operand)
		if err != nil {
			return "", "", err
		}
		castType := typeNodeToGoType(e.Type)
		return castType + "(" + operand + ")", castType, nil

	default:
		return "", "", fmt.Errorf("unsupported expression type %T in function transpilation", expr)
	}
}

func (env *typeEnv) resolveIdent(name string) (string, string, error) {
	// Check locals first
	if v, ok := env.locals[name]; ok {
		return v.goName, v.goType, nil
	}
	// Check params
	if v, ok := env.params[name]; ok {
		return v.goName, v.goType, nil
	}
	// Unknown — pass through as-is
	return name, "", nil
}

func (env *typeEnv) resolveMemberAccess(e parser.MemberAccess) (string, string, error) {
	obj, objType, err := env.transpileExpr(e.Object)
	if err != nil {
		return "", "", err
	}

	memberGoName := toPascalCase(e.Member)

	// If the object is the receiver, look up the field type
	if objType == env.receiver.Name {
		fieldType := env.lookupField(memberGoName)
		return obj + "." + memberGoName, fieldType, nil
	}

	return obj + "." + memberGoName, "", nil
}

func (env *typeEnv) transpileFnCall(fc parser.FnCall) (string, string, error) {
	// Resolve the function name
	fnName := flattenFnName(fc.Func)

	// Check for known stdlib mappings
	if mapping, ok := stdlibFuncs[fnName]; ok {
		return env.transpileStdlibCall(mapping, fc.Args)
	}

	// Unknown function call — emit as-is
	var args []string
	for _, arg := range fc.Args {
		a, _, err := env.transpileExpr(arg)
		if err != nil {
			return "", "", err
		}
		args = append(args, a)
	}
	return fnName + "(" + strings.Join(args, ", ") + ")", "", nil
}

// stdlibMapping describes how to translate a hexpat stdlib call to Go.
type stdlibMapping struct {
	goFunc     string
	goImport   string
	returnType string
	// formatStr true means first arg is a format string that needs {} → %v translation
	formatStr bool
	// needsCtx true means this call requires a *runtime.ReadContext
	needsCtx bool
	// prependCtx true means "ctx" is prepended as the first Go arg
	prependCtx bool
}

var stdlibFuncs = map[string]stdlibMapping{
	"std::format":            {goFunc: "fmt.Sprintf", goImport: "fmt", returnType: "string", formatStr: true},
	"std::mem::read_string":  {goFunc: "_memReadString", returnType: "string", needsCtx: true, prependCtx: true},
	"std::mem::read_unsigned": {goFunc: "_memReadUnsigned", returnType: "uint64", needsCtx: true, prependCtx: true},
	"std::mem::read_signed":  {goFunc: "_memReadSigned", returnType: "int64", needsCtx: true, prependCtx: true},
}

func (env *typeEnv) transpileStdlibCall(m stdlibMapping, args []parser.Expr) (string, string, error) {
	if m.needsCtx {
		env.needsCtx = true
		env.needsMemHelpers = true
	}

	var goArgs []string
	if m.prependCtx {
		goArgs = append(goArgs, "ctx")
	}
	for i, arg := range args {
		a, argType, err := env.transpileExpr(arg)
		if err != nil {
			return "", "", err
		}
		if i == 0 && m.formatStr {
			a = convertFormatString(a)
		}
		// Memory helpers take uint64 — cast if the arg is a smaller numeric type
		if m.prependCtx && argType != "uint64" && argType != "" && argType != "string" && argType != "bool" {
			a = "uint64(" + a + ")"
		}
		goArgs = append(goArgs, a)
	}
	return m.goFunc + "(" + strings.Join(goArgs, ", ") + ")", m.returnType, nil
}

// convertFormatString converts a hexpat format string to Go fmt format.
// "{}" → "%v", "{:02X}" → "%02X", etc.
func convertFormatString(quoted string) string {
	// The input is a Go quoted string like `"<heap len={}>"`.
	// We need to replace {} and {:fmt} patterns inside it.
	s := quoted
	// Simple {} → %v
	s = strings.ReplaceAll(s, "{}", "%v")
	// {:format} → %format — scan for {: and replace
	for {
		idx := strings.Index(s, "{:")
		if idx < 0 {
			break
		}
		end := strings.Index(s[idx:], "}")
		if end < 0 {
			break
		}
		fmtSpec := s[idx+2 : idx+end]
		s = s[:idx] + "%" + fmtSpec + s[idx+end+1:]
	}
	return s
}

// flattenFnName extracts a dotted function name from a call expression.
func flattenFnName(expr parser.Expr) string {
	switch e := expr.(type) {
	case parser.Ident:
		return e.Name
	case parser.NamespaceAccess:
		inner := flattenFnName(e.Member)
		return e.Namespace + "::" + inner
	case parser.MemberAccess:
		obj := flattenFnName(e.Object)
		return obj + "::" + e.Member
	}
	return ""
}

// resolveVarType determines the Go type for a local variable declaration.
func (env *typeEnv) resolveVarType(vd parser.VarDecl) string {
	if vd.Type != nil {
		goType := typeNodeToGoType(vd.Type)
		if goType != "" {
			return goType
		}
	}
	// Infer from initializer if type is auto or unknown
	if vd.Init != nil {
		_, exprType, err := env.transpileExpr(vd.Init)
		if err == nil && exprType != "" {
			return exprType
		}
	}
	return "string" // default fallback
}

// typeNodeToGoType converts a parser TypeNode to a Go type string.
func typeNodeToGoType(tn parser.TypeNode) string {
	switch t := tn.(type) {
	case parser.BuiltinType:
		switch t.Name {
		case "str", "auto":
			return "string"
		}
		prim := LookupBuiltin(t.Name)
		if prim != nil {
			return prim.GoType
		}
	case parser.NamedType:
		return toPascalCase(t.Name)
	case parser.EndianType:
		return typeNodeToGoType(t.Inner)
	}
	return ""
}

// arrayElementType returns the element type of a Go array type string.
func arrayElementType(goType string) string {
	// "[16]byte" → "byte", "[3]uint32" → "uint32", "[]uint8" → "uint8"
	if strings.HasPrefix(goType, "[]") {
		return goType[2:]
	}
	if strings.HasPrefix(goType, "[") {
		idx := strings.Index(goType, "]")
		if idx >= 0 {
			return goType[idx+1:]
		}
	}
	return "byte" // fallback
}

// lowerCamelCase converts snake_case to lowerCamelCase for local variables.
func lowerCamelCase(s string) string {
	pc := toPascalCase(s)
	if pc == "" {
		return s
	}
	// Lowercase first letter
	return strings.ToLower(pc[:1]) + pc[1:]
}
