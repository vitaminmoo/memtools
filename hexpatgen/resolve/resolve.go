package resolve

import (
	"fmt"
	"strings"

	"github.com/vitaminmoo/memtools/hexpat"
)

// Resolve transforms a parsed hexpat AST into a resolved Package IR.
func Resolve(file *hexpat.File) (*Package, error) {
	r := &resolver{
		symbols:      make(map[string]hexpat.Item),
		usingDefs:    make(map[string]*hexpat.UsingDef),
		structDefs:   make(map[string]*hexpat.StructDef),
		enumDefs:     make(map[string]*hexpat.EnumDef),
		bitfieldDefs: make(map[string]*hexpat.BitfieldDef),
		unionNames:   make(map[string]bool),
		resolved:     make(map[string]*StructType),
		resolvedE:    make(map[string]*EnumType),
		resolvedBF:   make(map[string]*BitfieldType),
		pkg: &Package{
			Name:   "generated",
			Endian: LittleEndian,
		},
	}

	// Collect phase
	for _, item := range file.Items {
		switch it := item.(type) {
		case hexpat.Pragma:
			if it.Key == "endian" {
				switch it.Value {
				case "big":
					r.pkg.Endian = BigEndian
				case "little":
					r.pkg.Endian = LittleEndian
				}
			}
		case hexpat.StructDef:
			r.symbols[it.Name] = it
			r.structDefs[it.Name] = &it
		case hexpat.UnionDef:
			// Convert union to struct-like for unified processing
			sd := hexpat.StructDef{
				Name:       it.Name,
				TypeParams: it.TypeParams,
				Body:       it.Body,
				Attrs:      it.Attrs,
			}
			r.symbols[it.Name] = it
			r.structDefs[it.Name] = &sd
			r.unionNames[it.Name] = true
		case hexpat.EnumDef:
			r.symbols[it.Name] = it
			r.enumDefs[it.Name] = &it
		case hexpat.BitfieldDef:
			r.symbols[it.Name] = it
			r.bitfieldDefs[it.Name] = &it
		case hexpat.UsingDef:
			r.symbols[it.Name] = it
			r.usingDefs[it.Name] = &it
		}
	}

	// Resolve enums
	for name, ed := range r.enumDefs {
		et, err := r.resolveEnum(name, ed)
		if err != nil {
			return nil, fmt.Errorf("resolving enum %s: %w", name, err)
		}
		r.pkg.Enums = append(r.pkg.Enums, et)
	}

	// Resolve bitfields
	for name, bd := range r.bitfieldDefs {
		bt, err := r.resolveBitfield(name, bd)
		if err != nil {
			return nil, fmt.Errorf("resolving bitfield %s: %w", name, err)
		}
		r.pkg.Bitfields = append(r.pkg.Bitfields, bt)
	}

	// Resolve structs/unions (pass 1: register names, pass 2: resolve fields)
	for name := range r.structDefs {
		r.resolved[name] = &StructType{
			Name:    toPascalCase(name),
			IsUnion: r.unionNames[name],
		}
	}
	for name, sd := range r.structDefs {
		if err := r.resolveStructFields(name, sd); err != nil {
			return nil, fmt.Errorf("resolving struct %s: %w", name, err)
		}
	}

	// Topological sort
	r.pkg.Structs = r.topoSort()

	return r.pkg, nil
}

type resolver struct {
	symbols      map[string]hexpat.Item
	usingDefs    map[string]*hexpat.UsingDef
	structDefs   map[string]*hexpat.StructDef
	enumDefs     map[string]*hexpat.EnumDef
	bitfieldDefs map[string]*hexpat.BitfieldDef
	unionNames   map[string]bool
	resolved     map[string]*StructType
	resolvedE    map[string]*EnumType
	resolvedBF   map[string]*BitfieldType
	pkg          *Package
}

func (r *resolver) resolveEnum(name string, ed *hexpat.EnumDef) (*EnumType, error) {
	underlying := r.resolveBuiltinType(ed.UnderlyingType)
	if underlying == nil {
		return nil, fmt.Errorf("unknown underlying type for enum %s", name)
	}

	et := &EnumType{
		Name:           toPascalCase(name),
		UnderlyingType: underlying,
	}

	var nextVal int64
	for _, m := range ed.Members {
		val := nextVal
		if m.Value != nil {
			if nl, ok := m.Value.(hexpat.NumberLit); ok {
				val = nl.Value
			}
		}
		et.Members = append(et.Members, EnumMember{
			Name:  toPascalCase(m.Name),
			Value: val,
		})
		nextVal = val + 1
	}

	r.resolvedE[name] = et
	return et, nil
}

func (r *resolver) resolveBitfield(name string, bd *hexpat.BitfieldDef) (*BitfieldType, error) {
	goName := toPascalCase(name)
	bt := &BitfieldType{Name: goName}

	var bitOffset int
	for _, entry := range bd.Body {
		bits := 0
		if nl, ok := entry.Bits.(hexpat.NumberLit); ok {
			bits = int(nl.Value)
		} else {
			return nil, fmt.Errorf("bitfield %s: non-constant bit width", name)
		}

		if entry.Name != "" && entry.Name != "padding" {
			bt.Fields = append(bt.Fields, &BitfieldField{
				Name:      toPascalCase(entry.Name),
				Bits:      bits,
				BitOffset: bitOffset,
				GoType:    bitfieldFieldGoType(bits),
			})
		}
		bitOffset += bits
	}

	bt.TotalBits = bitOffset
	bt.Underlying = inferBitfieldUnderlying(bitOffset)

	r.resolvedBF[name] = bt
	return bt, nil
}

func bitfieldFieldGoType(bits int) string {
	if bits == 1 {
		return "bool"
	}
	if bits <= 8 {
		return "uint8"
	}
	if bits <= 16 {
		return "uint16"
	}
	if bits <= 32 {
		return "uint32"
	}
	return "uint64"
}

func inferBitfieldUnderlying(totalBits int) *PrimitiveInfo {
	if totalBits <= 8 {
		return LookupBuiltin("u8")
	}
	if totalBits <= 16 {
		return LookupBuiltin("u16")
	}
	if totalBits <= 32 {
		return LookupBuiltin("u32")
	}
	return LookupBuiltin("u64")
}

func (r *resolver) resolveStructFields(name string, sd *hexpat.StructDef) error {
	st := r.resolved[name]

	// Guard against double resolution — this function can be called both from
	// the main loop and lazily when another struct references this type.
	if len(st.Members) > 0 {
		return nil
	}

	var offset int

	// Build fieldMap for expression transpilation
	fieldMap := make(map[string]string)

	// Handle inheritance
	if sd.Parent != "" {
		parentSt, ok := r.resolved[sd.Parent]
		if !ok {
			return fmt.Errorf("unknown parent struct %s", sd.Parent)
		}
		parentDef := r.structDefs[sd.Parent]
		if parentDef != nil {
			// Ensure parent is resolved first
			if len(parentSt.Fields()) == 0 && len(parentDef.Body) > 0 {
				if err := r.resolveStructFields(sd.Parent, parentDef); err != nil {
					return fmt.Errorf("resolving parent %s: %w", sd.Parent, err)
				}
			}
			// Populate fieldMap from parent body
			for _, stmt := range parentDef.Body {
				if vd, ok := stmt.(hexpat.VarDecl); ok {
					fieldMap[vd.Name] = toPascalCase(vd.Name)
				}
			}
		}
		// Copy parent members
		for _, m := range parentSt.Members {
			st.Members = append(st.Members, m)
		}
		if parentSt.Size > 0 {
			offset = parentSt.Size
		}
	}

	// Skip structs with template parameters (MVP)
	if len(sd.TypeParams) > 0 {
		st.Size = -1
		return nil
	}

	for _, stmt := range sd.Body {
		switch s := stmt.(type) {
		case hexpat.VarDecl:
			// Handle @ offset (not for unions)
			if !st.IsUnion && s.Offset != nil {
				if nl, ok := s.Offset.(hexpat.NumberLit); ok {
					offset = int(nl.Value)
				}
			}

			field, err := r.resolveVarDecl(s, fieldMap)
			if err != nil {
				continue // skip unresolvable types
			}

			if st.IsUnion {
				field.Offset = 0
				if field.Type.Size > 0 && field.Type.Size > offset {
					offset = field.Type.Size
				}
			} else {
				field.Offset = offset
				if field.Type.Size > 0 {
					offset += field.Type.Size
				}
			}

			st.Members = append(st.Members, &FieldMember{Field: field})

		case hexpat.PaddingStmt:
			if nl, ok := s.Size.(hexpat.NumberLit); ok {
				padSize := int(nl.Value)
				st.Members = append(st.Members, &PaddingMember{Size: padSize})
				if !st.IsUnion {
					offset += padSize
				}
			}

		case hexpat.IfStmt:
			cm, err := r.resolveConditional(s, fieldMap)
			if err != nil {
				continue
			}
			st.Members = append(st.Members, cm)
		}
	}

	st.Size = offset
	if st.HasDynamicFields() {
		st.Size = -1
	}
	return nil
}

// resolveVarDecl resolves a VarDecl into a Field. The Offset is set to -1
// and must be filled in by the caller.
func (r *resolver) resolveVarDecl(s hexpat.VarDecl, fieldMap map[string]string) (*Field, error) {
	rt, err := r.resolveType(s.Type, s.Pointer, r.pkg.Endian)
	if err != nil {
		return nil, err
	}

	// Handle per-field endian override
	if et, ok := s.Type.(hexpat.EndianType); ok {
		switch et.Order {
		case "le":
			rt.Endian = LittleEndian
		case "be":
			rt.Endian = BigEndian
		}
	}

	// Handle array
	if s.Array != nil {
		if nl, ok := s.Array.(hexpat.NumberLit); ok {
			elemType := *rt
			rt = &ResolvedType{
				Kind: KindArray,
				Array: &ArrayInfo{
					Length:  int(nl.Value),
					Element: &elemType,
				},
				Endian: rt.Endian,
				Size:   int(nl.Value) * rt.Size,
				GoType: fmt.Sprintf("[%d]%s", nl.Value, rt.GoType),
			}
		} else {
			lengthExpr, exprErr := exprToGo(s.Array, fieldMap)
			if exprErr != nil {
				return nil, exprErr
			}
			elemType := *rt
			rt = &ResolvedType{
				Kind: KindArray,
				Array: &ArrayInfo{
					Length:     -1,
					LengthExpr: lengthExpr,
					Element:    &elemType,
				},
				Endian: rt.Endian,
				Size:   -1,
				GoType: "[]" + rt.GoType,
			}
		}
	}

	field := &Field{
		Name:   toPascalCase(s.Name),
		Type:   rt,
		Offset: -1,
	}
	fieldMap[s.Name] = toPascalCase(s.Name)
	return field, nil
}

// resolveConditional resolves an IfStmt into a ConditionalMember.
func (r *resolver) resolveConditional(s hexpat.IfStmt, fieldMap map[string]string) (*ConditionalMember, error) {
	condExpr, err := exprToGo(s.Cond, fieldMap)
	if err != nil {
		return nil, err
	}

	cm := &ConditionalMember{}

	// Process "then" branch
	thenBranch := ConditionalBranch{Cond: condExpr}
	for _, stmt := range s.Then {
		if vd, ok := stmt.(hexpat.VarDecl); ok {
			field, err := r.resolveVarDecl(vd, fieldMap)
			if err != nil {
				continue
			}
			thenBranch.Fields = append(thenBranch.Fields, field)
		}
	}
	cm.Branches = append(cm.Branches, thenBranch)

	// Process "else" branch
	if len(s.Else) > 0 {
		// Check for else-if chain
		if len(s.Else) == 1 {
			if elseIf, ok := s.Else[0].(hexpat.IfStmt); ok {
				innerCM, err := r.resolveConditional(elseIf, fieldMap)
				if err != nil {
					return nil, err
				}
				cm.Branches = append(cm.Branches, innerCM.Branches...)
				return cm, nil
			}
		}

		// Plain else
		elseBranch := ConditionalBranch{}
		for _, stmt := range s.Else {
			if vd, ok := stmt.(hexpat.VarDecl); ok {
				field, err := r.resolveVarDecl(vd, fieldMap)
				if err != nil {
					continue
				}
				elseBranch.Fields = append(elseBranch.Fields, field)
			}
		}
		cm.Branches = append(cm.Branches, elseBranch)
	}

	return cm, nil
}

// exprToGo transpiles a hexpat expression to a Go source string.
func exprToGo(expr hexpat.Expr, fieldMap map[string]string) (string, error) {
	switch e := expr.(type) {
	case hexpat.NumberLit:
		if e.Raw != "" && (strings.HasPrefix(e.Raw, "0x") || strings.HasPrefix(e.Raw, "0X")) {
			return e.Raw, nil
		}
		return fmt.Sprintf("%d", e.Value), nil
	case hexpat.BoolLit:
		if e.Value {
			return "true", nil
		}
		return "false", nil
	case hexpat.Ident:
		if goName, ok := fieldMap[e.Name]; ok {
			return "result." + goName, nil
		}
		return e.Name, nil
	case hexpat.BinaryExpr:
		left, err := exprToGo(e.Left, fieldMap)
		if err != nil {
			return "", err
		}
		right, err := exprToGo(e.Right, fieldMap)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s)", left, e.Op, right), nil
	case hexpat.UnaryExpr:
		operand, err := exprToGo(e.Operand, fieldMap)
		if err != nil {
			return "", err
		}
		op := e.Op
		if op == "~" {
			op = "^" // Go bitwise NOT
		}
		if e.Prefix {
			return fmt.Sprintf("(%s%s)", op, operand), nil
		}
		return fmt.Sprintf("(%s%s)", operand, op), nil
	case hexpat.MemberAccess:
		obj, err := exprToGo(e.Object, fieldMap)
		if err != nil {
			return "", err
		}
		return obj + "." + toPascalCase(e.Member), nil
	default:
		return "", fmt.Errorf("unsupported expression type %T in transpilation", expr)
	}
}

func (r *resolver) resolveType(tn hexpat.TypeNode, ptr *hexpat.PointerInfo, endian Endian) (*ResolvedType, error) {
	// Unwrap EndianType
	if et, ok := tn.(hexpat.EndianType); ok {
		switch et.Order {
		case "le":
			endian = LittleEndian
		case "be":
			endian = BigEndian
		}
		tn = et.Inner
	}

	// Handle pointer wrapping
	if ptr != nil {
		pointee, err := r.resolveType(tn, nil, endian)
		if err != nil {
			return nil, err
		}
		sizeType := r.resolveBuiltinType(ptr.SizeType)
		if sizeType == nil {
			return nil, fmt.Errorf("unknown pointer size type")
		}
		return &ResolvedType{
			Kind: KindPointer,
			Pointer: &PointerInfo{
				Pointee:  pointee,
				SizeType: sizeType,
			},
			Endian: endian,
			Size:   sizeType.Size,
			GoType: "*" + toPascalCase(pointee.GoType),
		}, nil
	}

	switch t := tn.(type) {
	case hexpat.BuiltinType:
		prim := LookupBuiltin(t.Name)
		if prim == nil {
			return nil, fmt.Errorf("unknown builtin type %s", t.Name)
		}
		return &ResolvedType{
			Kind:      KindPrimitive,
			Primitive: prim,
			Endian:    endian,
			Size:      prim.Size,
			GoType:    prim.GoType,
		}, nil

	case hexpat.NamedType:
		name := t.Name
		// Skip namespaced types and template instantiations for MVP
		if len(t.Namespace) > 0 || len(t.TypeArgs) > 0 {
			return nil, fmt.Errorf("unsupported namespaced/template type %s", name)
		}

		// Check using aliases
		if ud, ok := r.usingDefs[name]; ok {
			return r.resolveType(ud.Type, nil, endian)
		}

		// Check enums
		if et, ok := r.resolvedE[name]; ok {
			return &ResolvedType{
				Kind:    KindEnum,
				EnumRef: et,
				Endian:  endian,
				Size:    et.UnderlyingType.Size,
				GoType:  toPascalCase(name),
			}, nil
		}

		// Check bitfields
		if bt, ok := r.resolvedBF[name]; ok {
			return &ResolvedType{
				Kind:        KindBitfield,
				BitfieldRef: bt,
				Endian:      endian,
				Size:        bt.Underlying.Size,
				GoType:      bt.Name,
			}, nil
		}

		// Check structs
		if st, ok := r.resolved[name]; ok {
			// Ensure referenced struct is resolved first
			if sd, exists := r.structDefs[name]; exists && len(st.Members) == 0 && len(sd.Body) > 0 {
				r.resolveStructFields(name, sd)
			}
			return &ResolvedType{
				Kind:      KindStruct,
				StructRef: st,
				Endian:    endian,
				Size:      st.Size,
				GoType:    toPascalCase(name),
			}, nil
		}

		return nil, fmt.Errorf("unknown type %s", name)

	default:
		return nil, fmt.Errorf("unsupported type node %T", tn)
	}
}

func (r *resolver) resolveBuiltinType(tn hexpat.TypeNode) *PrimitiveInfo {
	// Unwrap EndianType
	if et, ok := tn.(hexpat.EndianType); ok {
		tn = et.Inner
	}
	switch t := tn.(type) {
	case hexpat.BuiltinType:
		return LookupBuiltin(t.Name)
	case hexpat.NamedType:
		// Check using aliases
		if ud, ok := r.usingDefs[t.Name]; ok {
			return r.resolveBuiltinType(ud.Type)
		}
		return LookupBuiltin(t.Name)
	default:
		return nil
	}
}

// topoSort returns structs in dependency order (dependencies first).
// Pointer refs don't count as dependencies.
func (r *resolver) topoSort() []*StructType {
	visited := make(map[string]bool)
	var result []*StructType

	var visit func(st *StructType)
	visit = func(st *StructType) {
		if visited[st.Name] {
			return
		}
		visited[st.Name] = true
		for _, f := range st.Fields() {
			if f.Type.Kind == KindStruct && f.Type.StructRef != nil {
				visit(f.Type.StructRef)
			}
			if f.Type.Kind == KindArray && f.Type.Array != nil && f.Type.Array.Element.Kind == KindStruct {
				visit(f.Type.Array.Element.StructRef)
			}
		}
		result = append(result, st)
	}

	for _, sd := range r.structDefs {
		st := r.resolved[sd.Name]
		if st != nil {
			visit(st)
		}
	}

	return result
}

// toPascalCase converts snake_case to PascalCase.
func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	result := make([]byte, 0, len(s))
	upper := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			upper = true
			continue
		}
		if upper && c >= 'a' && c <= 'z' {
			c -= 32
		}
		upper = false
		result = append(result, c)
	}
	// Handle Go keyword collisions (compare lowercase since PascalCase won't
	// collide, but single-word inputs like "type" become "Type" which is fine;
	// only check the lowercase original)
	goKeywords := map[string]bool{
		"break": true, "default": true, "func": true, "interface": true, "select": true,
		"case": true, "defer": true, "go": true, "map": true, "struct": true,
		"chan": true, "else": true, "goto": true, "package": true, "switch": true,
		"const": true, "fallthrough": true, "if": true, "range": true, "type": true,
		"continue": true, "for": true, "import": true, "return": true, "var": true,
	}
	rs := string(result)
	if goKeywords[strings.ToLower(rs)] {
		return rs + "_"
	}
	return rs
}
