package codegen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/vitaminmoo/memtools/hexpat/resolve"
)

// writeBitfieldReaderStruct generates a reader for a bitfield type.
// Bitfields are small (1-8 bytes), so the reader just provides ctx+addr and delegates to ReadFoo.
func writeBitfieldReaderStruct(buf *bytes.Buffer, bt *resolve.BitfieldType) {
	fmt.Fprintf(buf, "type %sReader struct {\n", bt.Name)
	fmt.Fprintf(buf, "\tctx  *runtime.ReadContext\n")
	fmt.Fprintf(buf, "\taddr uintptr\n")
	fmt.Fprintf(buf, "}\n\n")

	fmt.Fprintf(buf, "func New%sReader(ctx *runtime.ReadContext, addr uintptr) *%sReader {\n", bt.Name, bt.Name)
	fmt.Fprintf(buf, "\treturn &%sReader{ctx: ctx, addr: addr}\n", bt.Name)
	fmt.Fprintf(buf, "}\n\n")

	fmt.Fprintf(buf, "func (r *%sReader) Read() (*%s, runtime.Errors) {\n", bt.Name, bt.Name)
	fmt.Fprintf(buf, "\treturn Read%s(r.ctx, r.addr)\n", bt.Name)
	fmt.Fprintf(buf, "}\n\n")

	fmt.Fprintf(buf, "func (r *%sReader) Addr() uintptr {\n", bt.Name)
	fmt.Fprintf(buf, "\treturn r.addr\n")
	fmt.Fprintf(buf, "}\n\n")
}

// readerEligible returns true if a struct can have a lazy reader generated.
// Phase 1: only static-offset structs (no conditionals, no dynamic arrays, no remote fields).
func readerEligible(st *resolve.StructType) bool {
	if len(st.Fields()) == 0 {
		return false
	}
	return !st.HasDynamicFields()
}

// writeReaderStruct emits the FooReader struct and NewFooReader constructor.
func writeReaderStruct(buf *bytes.Buffer, st *resolve.StructType) {
	fmt.Fprintf(buf, "// %sReader provides lazy, field-level access to %s without reading the entire struct.\n", st.Name, st.Name)
	fmt.Fprintf(buf, "type %sReader struct {\n", st.Name)
	fmt.Fprintf(buf, "\tctx  *runtime.ReadContext\n")
	fmt.Fprintf(buf, "\taddr uintptr\n")
	fmt.Fprintf(buf, "}\n\n")

	fmt.Fprintf(buf, "// New%sReader creates a lazy reader for %s at the given address.\n", st.Name, st.Name)
	fmt.Fprintf(buf, "func New%sReader(ctx *runtime.ReadContext, addr uintptr) *%sReader {\n", st.Name, st.Name)
	fmt.Fprintf(buf, "\treturn &%sReader{ctx: ctx, addr: addr}\n", st.Name)
	fmt.Fprintf(buf, "}\n\n")
}

// writeReaderMaterialize emits the Read() escape hatch that delegates to the eager ReadFoo.
func writeReaderMaterialize(buf *bytes.Buffer, st *resolve.StructType) {
	fmt.Fprintf(buf, "// Read materializes the full %s struct eagerly.\n", st.Name)
	fmt.Fprintf(buf, "func (r *%sReader) Read() (*%s, runtime.Errors) {\n", st.Name, st.Name)
	fmt.Fprintf(buf, "\treturn Read%s(r.ctx, r.addr)\n", st.Name)
	fmt.Fprintf(buf, "}\n\n")
}

// writeReaderAddr emits the Addr() method that returns the reader's address.
func writeReaderAddr(buf *bytes.Buffer, st *resolve.StructType) {
	fmt.Fprintf(buf, "// Addr returns the base address of this %s.\n", st.Name)
	fmt.Fprintf(buf, "func (r *%sReader) Addr() uintptr {\n", st.Name)
	fmt.Fprintf(buf, "\treturn r.addr\n")
	fmt.Fprintf(buf, "}\n\n")
}

// writeReaderMethods emits per-field accessor methods on the reader.
// hasReader is the set of struct names that have reader types generated.
func writeReaderMethods(buf *bytes.Buffer, st *resolve.StructType, defaultEndian resolve.Endian, hasReader map[string]bool) {
	for _, m := range st.Members {
		fm, ok := m.(*resolve.FieldMember)
		if !ok {
			continue
		}
		f := fm.Field
		endian := f.Type.Endian
		if endian != resolve.BigEndian && endian != resolve.LittleEndian {
			endian = defaultEndian
		}
		addrExpr := fmt.Sprintf("int64(r.addr)+%d", f.Offset)

		switch f.Type.Kind {
		case resolve.KindPrimitive:
			writeReaderPrimitiveAccessor(buf, st.Name, f, addrExpr, endian)
		case resolve.KindEnum:
			writeReaderEnumAccessor(buf, st.Name, f, addrExpr, endian)
		case resolve.KindStruct:
			if hasReader[f.Type.StructRef.Name] {
				writeReaderStructAccessor(buf, st.Name, f, addrExpr)
			} else {
				writeReaderEagerStructAccessor(buf, st.Name, f, addrExpr)
			}
		case resolve.KindBitfield:
			writeReaderBitfieldAccessor(buf, st.Name, f, addrExpr)
		case resolve.KindPointer:
			writeReaderPointerAccessor(buf, st.Name, f, addrExpr, endian, hasReader)
		case resolve.KindArray:
			writeReaderArrayAccessor(buf, st.Name, f, addrExpr, endian, hasReader)
		}
	}
}

func writeReaderPrimitiveAccessor(buf *bytes.Buffer, structName string, f *resolve.Field, addrExpr string, endian resolve.Endian) {
	prim := f.Type.Primitive
	size := prim.Size
	ev := endianVar(endian)
	goType := prim.GoType

	// For odd-sized types like [3]byte, return the array type
	retType := goType
	if strings.HasPrefix(goType, "[") {
		retType = goType
	}

	fmt.Fprintf(buf, "func (r *%sReader) %s() (%s, error) {\n", structName, f.Name, retType)
	fmt.Fprintf(buf, "\tvar buf [%d]byte\n", size)
	fmt.Fprintf(buf, "\tif _, err := r.ctx.ReadAt(buf[:%d], %s); err != nil {\n", size, addrExpr)

	// Return zero value on error
	if strings.HasPrefix(goType, "[") {
		fmt.Fprintf(buf, "\t\tvar zero %s\n", retType)
		fmt.Fprintf(buf, "\t\treturn zero, err\n")
	} else if goType == "bool" {
		fmt.Fprintf(buf, "\t\treturn false, err\n")
	} else {
		fmt.Fprintf(buf, "\t\treturn 0, err\n")
	}
	fmt.Fprintf(buf, "\t}\n")

	switch goType {
	case "uint8", "byte":
		fmt.Fprintf(buf, "\treturn buf[0], nil\n")
	case "int8":
		fmt.Fprintf(buf, "\treturn int8(buf[0]), nil\n")
	case "bool":
		fmt.Fprintf(buf, "\treturn buf[0] != 0, nil\n")
	case "uint16":
		fmt.Fprintf(buf, "\treturn %s.Uint16(buf[:%d]), nil\n", ev, size)
	case "int16":
		fmt.Fprintf(buf, "\treturn int16(%s.Uint16(buf[:%d])), nil\n", ev, size)
	case "uint32":
		fmt.Fprintf(buf, "\treturn %s.Uint32(buf[:%d]), nil\n", ev, size)
	case "int32":
		fmt.Fprintf(buf, "\treturn int32(%s.Uint32(buf[:%d])), nil\n", ev, size)
	case "uint64":
		fmt.Fprintf(buf, "\treturn %s.Uint64(buf[:%d]), nil\n", ev, size)
	case "int64":
		fmt.Fprintf(buf, "\treturn int64(%s.Uint64(buf[:%d])), nil\n", ev, size)
	case "float32":
		fmt.Fprintf(buf, "\treturn math.Float32frombits(%s.Uint32(buf[:%d])), nil\n", ev, size)
	case "float64":
		fmt.Fprintf(buf, "\treturn math.Float64frombits(%s.Uint64(buf[:%d])), nil\n", ev, size)
	default:
		if strings.HasPrefix(goType, "[") {
			fmt.Fprintf(buf, "\tvar result %s\n", retType)
			fmt.Fprintf(buf, "\tcopy(result[:], buf[:%d])\n", size)
			fmt.Fprintf(buf, "\treturn result, nil\n")
		}
	}

	fmt.Fprintf(buf, "}\n\n")
}

func writeReaderEnumAccessor(buf *bytes.Buffer, structName string, f *resolve.Field, addrExpr string, endian resolve.Endian) {
	et := f.Type.EnumRef
	size := et.UnderlyingType.Size
	ev := endianVar(endian)
	enumGoType := f.Type.GoType

	fmt.Fprintf(buf, "func (r *%sReader) %s() (%s, error) {\n", structName, f.Name, enumGoType)
	fmt.Fprintf(buf, "\tvar buf [%d]byte\n", size)
	fmt.Fprintf(buf, "\tif _, err := r.ctx.ReadAt(buf[:%d], %s); err != nil {\n", size, addrExpr)
	fmt.Fprintf(buf, "\t\treturn 0, err\n")
	fmt.Fprintf(buf, "\t}\n")

	goType := et.UnderlyingType.GoType
	switch goType {
	case "uint8", "byte":
		fmt.Fprintf(buf, "\treturn %s(buf[0]), nil\n", enumGoType)
	case "int8":
		fmt.Fprintf(buf, "\treturn %s(int8(buf[0])), nil\n", enumGoType)
	case "uint16":
		fmt.Fprintf(buf, "\treturn %s(%s.Uint16(buf[:%d])), nil\n", enumGoType, ev, size)
	case "int16":
		fmt.Fprintf(buf, "\treturn %s(int16(%s.Uint16(buf[:%d]))), nil\n", enumGoType, ev, size)
	case "uint32":
		fmt.Fprintf(buf, "\treturn %s(%s.Uint32(buf[:%d])), nil\n", enumGoType, ev, size)
	case "int32":
		fmt.Fprintf(buf, "\treturn %s(int32(%s.Uint32(buf[:%d]))), nil\n", enumGoType, ev, size)
	case "uint64":
		fmt.Fprintf(buf, "\treturn %s(%s.Uint64(buf[:%d])), nil\n", enumGoType, ev, size)
	case "int64":
		fmt.Fprintf(buf, "\treturn %s(int64(%s.Uint64(buf[:%d]))), nil\n", enumGoType, ev, size)
	}

	fmt.Fprintf(buf, "}\n\n")
}

func writeReaderStructAccessor(buf *bytes.Buffer, structName string, f *resolve.Field, addrExpr string) {
	childName := f.Type.StructRef.Name
	fmt.Fprintf(buf, "// %s returns a lazy reader for the nested %s (zero I/O).\n", f.Name, childName)
	fmt.Fprintf(buf, "func (r *%sReader) %s() *%sReader {\n", structName, f.Name, childName)
	fmt.Fprintf(buf, "\treturn New%sReader(r.ctx, uintptr(%s))\n", childName, addrExpr)
	fmt.Fprintf(buf, "}\n\n")
}

// writeReaderEagerStructAccessor emits an accessor that eagerly reads a nested struct
// when that struct doesn't have a reader type (e.g., dynamic-offset structs).
func writeReaderEagerStructAccessor(buf *bytes.Buffer, structName string, f *resolve.Field, addrExpr string) {
	childName := f.Type.StructRef.Name
	fmt.Fprintf(buf, "// %s eagerly reads the nested %s (no lazy reader available for this type).\n", f.Name, childName)
	fmt.Fprintf(buf, "func (r *%sReader) %s() (*%s, runtime.Errors) {\n", structName, f.Name, childName)
	fmt.Fprintf(buf, "\treturn Read%s(r.ctx, uintptr(%s))\n", childName, addrExpr)
	fmt.Fprintf(buf, "}\n\n")
}

func writeReaderBitfieldAccessor(buf *bytes.Buffer, structName string, f *resolve.Field, addrExpr string) {
	childName := f.Type.BitfieldRef.Name
	fmt.Fprintf(buf, "// %s returns a lazy reader for the nested %s (zero I/O).\n", f.Name, childName)
	fmt.Fprintf(buf, "func (r *%sReader) %s() *%sReader {\n", structName, f.Name, childName)
	fmt.Fprintf(buf, "\treturn New%sReader(r.ctx, uintptr(%s))\n", childName, addrExpr)
	fmt.Fprintf(buf, "}\n\n")
}

func writeReaderPointerAccessor(buf *bytes.Buffer, structName string, f *resolve.Field, addrExpr string, endian resolve.Endian, hasReader map[string]bool) {
	ptr := f.Type.Pointer
	ptrSize := ptr.SizeType.Size
	ev := endianVar(endian)
	storageType := ptr.SizeType.GoType

	// Raw value accessor — returns the pointer address as uint32/uint64
	fmt.Fprintf(buf, "func (r *%sReader) %s() (%s, error) {\n", structName, f.Name, storageType)
	fmt.Fprintf(buf, "\tvar buf [%d]byte\n", ptrSize)
	fmt.Fprintf(buf, "\tif _, err := r.ctx.ReadAt(buf[:%d], %s); err != nil {\n", ptrSize, addrExpr)
	fmt.Fprintf(buf, "\t\treturn 0, err\n")
	fmt.Fprintf(buf, "\t}\n")
	switch ptrSize {
	case 4:
		fmt.Fprintf(buf, "\treturn %s.Uint32(buf[:%d]), nil\n", ev, ptrSize)
	case 8:
		fmt.Fprintf(buf, "\treturn %s.Uint64(buf[:%d]), nil\n", ev, ptrSize)
	}
	fmt.Fprintf(buf, "}\n\n")

	// Follow method — reads the pointer and follows it to the target struct
	pointee := ptr.Pointee
	if pointee.Kind != resolve.KindStruct || pointee.StructRef == nil {
		return
	}
	childName := pointee.StructRef.Name

	fmt.Fprintf(buf, "// Follow%s reads the %s pointer and follows it to the target %s.\n", f.Name, f.Name, childName)
	fmt.Fprintf(buf, "func (r *%sReader) Follow%s() (*%s, runtime.Errors) {\n", structName, f.Name, childName)
	fmt.Fprintf(buf, "\tptr, err := r.%s()\n", f.Name)
	fmt.Fprintf(buf, "\tif err != nil || ptr == 0 {\n")
	fmt.Fprintf(buf, "\t\tif err != nil {\n")
	fmt.Fprintf(buf, "\t\t\tvar errs runtime.Errors\n")
	fmt.Fprintf(buf, "\t\t\terrs.Add(%q, r.addr, err)\n", structName+"."+f.Name)
	fmt.Fprintf(buf, "\t\t\treturn nil, errs\n")
	fmt.Fprintf(buf, "\t\t}\n")
	fmt.Fprintf(buf, "\t\treturn nil, nil\n")
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "\treturn Read%s(r.ctx, uintptr(ptr))\n", childName)
	fmt.Fprintf(buf, "}\n\n")
}

func writeReaderArrayAccessor(buf *bytes.Buffer, structName string, f *resolve.Field, addrExpr string, endian resolve.Endian, hasReader map[string]bool) {
	arr := f.Type.Array
	elem := arr.Element

	// Phase 1: only fixed-length arrays with static offsets
	if arr.LengthExpr != "" {
		return // dynamic arrays not supported in readers yet
	}

	switch elem.Kind {
	case resolve.KindPrimitive:
		writeReaderPrimitiveArrayAccessor(buf, structName, f, addrExpr, endian)
	case resolve.KindStruct:
		if hasReader[elem.StructRef.Name] {
			writeReaderStructArrayAccessor(buf, structName, f, addrExpr)
		}
		// If child struct has no reader, skip — caller can use Read() to materialize
	}
}

func writeReaderPrimitiveArrayAccessor(buf *bytes.Buffer, structName string, f *resolve.Field, addrExpr string, endian resolve.Endian) {
	arr := f.Type.Array
	elem := arr.Element
	goType := f.Type.GoType // e.g. "[4]uint8"

	if elem.Size == 1 {
		// Byte arrays: read all at once
		fmt.Fprintf(buf, "func (r *%sReader) %s() (%s, error) {\n", structName, f.Name, goType)
		fmt.Fprintf(buf, "\tvar result %s\n", goType)
		fmt.Fprintf(buf, "\tif _, err := r.ctx.ReadAt(result[:], %s); err != nil {\n", addrExpr)
		fmt.Fprintf(buf, "\t\treturn result, err\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\treturn result, nil\n")
		fmt.Fprintf(buf, "}\n\n")
	} else {
		// Multi-byte element arrays
		ev := endianVar(endian)
		prim := elem.Primitive

		fmt.Fprintf(buf, "func (r *%sReader) %s() (%s, error) {\n", structName, f.Name, goType)
		fmt.Fprintf(buf, "\tvar result %s\n", goType)
		fmt.Fprintf(buf, "\tvar buf [%d]byte\n", elem.Size)
		fmt.Fprintf(buf, "\tfor i := range result {\n")
		elemOffset := fmt.Sprintf("%s+int64(i)*%d", addrExpr, elem.Size)
		fmt.Fprintf(buf, "\t\tif _, err := r.ctx.ReadAt(buf[:%d], %s); err != nil {\n", elem.Size, elemOffset)
		fmt.Fprintf(buf, "\t\t\treturn result, err\n")
		fmt.Fprintf(buf, "\t\t}\n")

		switch prim.GoType {
		case "uint16":
			fmt.Fprintf(buf, "\t\tresult[i] = %s.Uint16(buf[:%d])\n", ev, elem.Size)
		case "int16":
			fmt.Fprintf(buf, "\t\tresult[i] = int16(%s.Uint16(buf[:%d]))\n", ev, elem.Size)
		case "uint32":
			fmt.Fprintf(buf, "\t\tresult[i] = %s.Uint32(buf[:%d])\n", ev, elem.Size)
		case "int32":
			fmt.Fprintf(buf, "\t\tresult[i] = int32(%s.Uint32(buf[:%d]))\n", ev, elem.Size)
		case "uint64":
			fmt.Fprintf(buf, "\t\tresult[i] = %s.Uint64(buf[:%d])\n", ev, elem.Size)
		case "int64":
			fmt.Fprintf(buf, "\t\tresult[i] = int64(%s.Uint64(buf[:%d]))\n", ev, elem.Size)
		case "float32":
			fmt.Fprintf(buf, "\t\tresult[i] = math.Float32frombits(%s.Uint32(buf[:%d]))\n", ev, elem.Size)
		case "float64":
			fmt.Fprintf(buf, "\t\tresult[i] = math.Float64frombits(%s.Uint64(buf[:%d]))\n", ev, elem.Size)
		}

		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\treturn result, nil\n")
		fmt.Fprintf(buf, "}\n\n")
	}
}

func writeReaderStructArrayAccessor(buf *bytes.Buffer, structName string, f *resolve.Field, addrExpr string) {
	arr := f.Type.Array
	elem := arr.Element
	childName := elem.StructRef.Name

	fmt.Fprintf(buf, "// %s returns lazy readers for each element (zero I/O).\n", f.Name)
	fmt.Fprintf(buf, "func (r *%sReader) %s() [%d]%sReader {\n", structName, f.Name, arr.Length, childName)
	fmt.Fprintf(buf, "\tvar result [%d]%sReader\n", arr.Length, childName)
	fmt.Fprintf(buf, "\tfor i := range result {\n")
	fmt.Fprintf(buf, "\t\tresult[i] = *New%sReader(r.ctx, uintptr(%s+int64(i)*%d))\n", childName, addrExpr, elem.Size)
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "\treturn result\n")
	fmt.Fprintf(buf, "}\n\n")
}
