package resolve

// Endian represents byte order.
type Endian int

const (
	LittleEndian Endian = iota
	BigEndian
)

// Package is the resolved IR for an entire hexpat file.
type Package struct {
	Name      string
	Endian    Endian
	Structs   []*StructType // dependency order
	Enums     []*EnumType
	Bitfields []*BitfieldType
}

// StructType is a resolved struct/union definition.
type StructType struct {
	Name    string
	IsUnion bool
	Members []StructMember
	Size    int // -1 if dynamic
}

// Fields returns all fields, including those inside conditionals.
func (st *StructType) Fields() []*Field {
	var fields []*Field
	for _, m := range st.Members {
		switch v := m.(type) {
		case *FieldMember:
			fields = append(fields, v.Field)
		case *ConditionalMember:
			for _, br := range v.Branches {
				fields = append(fields, br.Fields...)
			}
		}
	}
	return fields
}

// HasConditionals returns true if the struct has any conditional members.
func (st *StructType) HasConditionals() bool {
	for _, m := range st.Members {
		if _, ok := m.(*ConditionalMember); ok {
			return true
		}
	}
	return false
}

// HasDynamicFields returns true if the struct needs dynamic offset tracking.
func (st *StructType) HasDynamicFields() bool {
	if st.HasConditionals() {
		return true
	}
	for _, f := range st.Fields() {
		if f.Type.Kind == KindArray && f.Type.Array != nil && f.Type.Array.LengthExpr != "" {
			return true
		}
	}
	return false
}

// StructMember is an item in a struct body.
type StructMember interface {
	structMember()
}

// FieldMember wraps a Field as a StructMember.
type FieldMember struct {
	*Field
}

func (*FieldMember) structMember() {}

// PaddingMember represents padding in a struct body.
type PaddingMember struct {
	Size int // static padding size in bytes
}

func (*PaddingMember) structMember() {}

// ConditionalMember represents an if/else chain in a struct body.
type ConditionalMember struct {
	Branches []ConditionalBranch
}

func (*ConditionalMember) structMember() {}

// ConditionalBranch is one branch of a conditional.
type ConditionalBranch struct {
	Cond   string   // Go expression string, empty for else
	Fields []*Field
}

// Field is a resolved struct field.
type Field struct {
	Name   string
	Type   *ResolvedType
	Offset int // byte offset from struct start, -1 if dynamic
}

// TypeKind classifies a resolved type.
type TypeKind int

const (
	KindPrimitive TypeKind = iota
	KindStruct
	KindEnum
	KindArray
	KindPointer
	KindBitfield
)

// ResolvedType is the fully resolved form of a type reference.
type ResolvedType struct {
	Kind        TypeKind
	Primitive   *PrimitiveInfo
	StructRef   *StructType
	EnumRef     *EnumType
	BitfieldRef *BitfieldType
	Array       *ArrayInfo
	Pointer     *PointerInfo
	Endian      Endian
	Size        int
	GoType      string // e.g. "uint32", "[10]uint8", "[]uint8"
}

// PrimitiveInfo describes a builtin primitive type.
type PrimitiveInfo struct {
	Name   string // hexpat name (u32, s16, etc.)
	GoType string // Go type (uint32, int16, etc.)
	Size   int
}

// ArrayInfo describes a fixed-size or expression-sized array.
type ArrayInfo struct {
	Length     int    // fixed length, -1 if dynamic
	LengthExpr string // Go expression for dynamic length, empty if fixed
	Element   *ResolvedType
}

// PointerInfo describes a pointer field.
type PointerInfo struct {
	Pointee  *ResolvedType
	SizeType *PrimitiveInfo // type encoding the pointer value (e.g. u32 → 4 bytes)
}

// EnumType is a resolved enum definition.
type EnumType struct {
	Name           string
	UnderlyingType *PrimitiveInfo
	Members        []EnumMember
}

// EnumMember is a single enum constant.
type EnumMember struct {
	Name  string
	Value int64
}

// BitfieldType is a resolved bitfield definition.
type BitfieldType struct {
	Name       string
	TotalBits  int
	Underlying *PrimitiveInfo // u8/u16/u32/u64 inferred from TotalBits
	Fields     []*BitfieldField
}

// BitfieldField is a single field within a bitfield.
type BitfieldField struct {
	Name      string // empty = padding
	Bits      int
	BitOffset int    // from bit 0 of underlying
	GoType    string // bool for 1-bit, else uint8/16/32/64
}
