package codegen

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"

	"github.com/vitaminmoo/memtools/hexpat/resolve"
)

// Options configures code generation.
type Options struct {
	PackageName string // defaults to "generated"
}

// Generate produces Go source code from a resolved Package.
func Generate(pkg *resolve.Package, opts Options) ([]byte, error) {
	pkgName := opts.PackageName
	if pkgName == "" {
		pkgName = pkg.Name
	}

	var buf bytes.Buffer

	// Package declaration
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)

	// Imports
	needsMath := false
	needsBinary := false
	needsRuntime := false
	needsJSON := len(pkg.Enums) > 0
	needsFmt := len(pkg.Enums) > 0
	for _, st := range pkg.Structs {
		fields := st.Fields()
		if len(fields) > 0 {
			needsBinary = true
			needsRuntime = true
		}
		for _, f := range fields {
			if f.Type.Primitive != nil && (f.Type.Primitive.GoType == "float32" || f.Type.Primitive.GoType == "float64") {
				needsMath = true
			}
			if f.Type.Kind == resolve.KindArray && f.Type.Array != nil && f.Type.Array.Element.Primitive != nil {
				p := f.Type.Array.Element.Primitive
				if p.GoType == "float32" || p.GoType == "float64" {
					needsMath = true
				}
			}
		}
	}
	if len(pkg.Bitfields) > 0 {
		needsBinary = true
		needsRuntime = true
	}
	if len(pkg.Placements) > 0 {
		needsBinary = true
		needsRuntime = true
	}
	for _, fb := range pkg.FuncBindings {
		if fb.NeedsCtx {
			needsRuntime = true
		}
		if containsStdFormat(fb.Body) {
			needsFmt = true
		}
	}

	var imports []string
	if needsBinary {
		imports = append(imports, `"encoding/binary"`)
	}
	if needsJSON {
		imports = append(imports, `"encoding/json"`)
	}
	if needsFmt {
		imports = append(imports, `"fmt"`)
	}
	if needsMath {
		imports = append(imports, `"math"`)
	}
	if needsRuntime {
		imports = append(imports, `"github.com/vitaminmoo/memtools/hexpat/runtime"`)
	}

	if len(imports) > 0 {
		fmt.Fprintf(&buf, "import (\n")
		for _, imp := range imports {
			fmt.Fprintf(&buf, "\t%s\n", imp)
		}
		fmt.Fprintf(&buf, ")\n\n")
	}

	// Enums
	for _, et := range pkg.Enums {
		writeEnum(&buf, et)
	}

	// Bitfields
	for _, bt := range pkg.Bitfields {
		writeBitfieldStruct(&buf, bt)
	}

	// Structs
	for _, st := range pkg.Structs {
		writeStruct(&buf, st)
	}

	// Bitfield read functions
	for _, bt := range pkg.Bitfields {
		writeBitfieldReadFunc(&buf, bt, pkg.Endian)
	}

	// Struct read functions
	for _, st := range pkg.Structs {
		if len(st.Fields()) > 0 {
			writeReadFunc(&buf, st, pkg.Endian)
		}
	}

	// Pointer follow methods on eager structs
	for _, st := range pkg.Structs {
		writePointerFollowMethods(&buf, st)
	}

	// Static address placements
	if len(pkg.Placements) > 0 {
		writePlacementConstants(&buf, pkg.Placements)
		for _, p := range pkg.Placements {
			writePlacementReadFunc(&buf, p, pkg.Endian)
		}
	}

	// Transpiled function methods
	needsMemHelpers := false
	for _, fb := range pkg.FuncBindings {
		writeFuncBinding(&buf, fb)
		if fb.NeedsMemHelpers {
			needsMemHelpers = true
		}
	}
	if needsMemHelpers {
		writeMemHelpers(&buf)
	}

	// Lazy reader types — build set of names that have readers
	hasReader := make(map[string]bool)
	for _, bt := range pkg.Bitfields {
		hasReader[bt.Name] = true
	}
	for _, st := range pkg.Structs {
		if readerEligible(st) {
			hasReader[st.Name] = true
		}
	}

	// Lazy reader types — bitfields
	for _, bt := range pkg.Bitfields {
		writeBitfieldReaderStruct(&buf, bt)
	}

	// Lazy reader types — structs
	for _, st := range pkg.Structs {
		if readerEligible(st) {
			writeReaderStruct(&buf, st)
			writeReaderAddr(&buf, st)
			writeReaderMaterialize(&buf, st)
			writeReaderMethods(&buf, st, pkg.Endian, hasReader)
		}
	}

	// Suppress unused import warnings
	if needsMath || needsBinary || needsJSON || needsFmt {
		fmt.Fprintf(&buf, "// Ensure imports are used.\nvar (\n")
		if needsBinary {
			fmt.Fprintf(&buf, "\t_ = binary.LittleEndian\n")
		}
		if needsJSON {
			fmt.Fprintf(&buf, "\t_ = json.Marshal\n")
		}
		if needsFmt {
			fmt.Fprintf(&buf, "\t_ = fmt.Sprintf\n")
		}
		if needsMath {
			fmt.Fprintf(&buf, "\t_ = math.Float32frombits\n")
		}
		fmt.Fprintf(&buf, ")\n")
	}

	return format.Source(buf.Bytes())
}

func writeStruct(buf *bytes.Buffer, st *resolve.StructType) {
	fields := st.Fields()
	fmt.Fprintf(buf, "type %s struct {\n", st.Name)
	for _, f := range fields {
		goType := fieldGoType(f.Type)
		fmt.Fprintf(buf, "\t%s %s\n", f.Name, goType)
	}
	fmt.Fprintf(buf, "}\n\n")
}

func fieldGoType(rt *resolve.ResolvedType) string {
	switch rt.Kind {
	case resolve.KindPointer:
		// Store raw pointer value (e.g. uint32 for :u32) instead of *PointeeType.
		// Follow methods are generated separately.
		if rt.Pointer != nil {
			return rt.Pointer.SizeType.GoType
		}
		return rt.GoType
	default:
		return rt.GoType
	}
}

func writeReadFunc(buf *bytes.Buffer, st *resolve.StructType, defaultEndian resolve.Endian) {
	fmt.Fprintf(buf, "func Read%s(ctx *runtime.ReadContext, addr uintptr) (*%s, runtime.Errors) {\n", st.Name, st.Name)
	fmt.Fprintf(buf, "\tvar errs runtime.Errors\n")
	fmt.Fprintf(buf, "\tresult := &%s{}\n", st.Name)

	// Compute max field size for buffer
	maxSize := 0
	for _, f := range st.Fields() {
		s := primitiveReadSize(f.Type)
		if s > maxSize {
			maxSize = s
		}
	}
	if maxSize > 0 {
		fmt.Fprintf(buf, "\tvar buf [%d]byte\n", maxSize)
	}

	dynamic := st.HasDynamicFields()
	if dynamic {
		fmt.Fprintf(buf, "\toffset := int64(0)\n")
	}
	fmt.Fprintf(buf, "\n")

	for _, m := range st.Members {
		switch v := m.(type) {
		case *resolve.FieldMember:
			endian := v.Type.Endian
			if endian != resolve.BigEndian && endian != resolve.LittleEndian {
				endian = defaultEndian
			}
			if v.OffsetExpr != "" {
				// Field placed at a remote absolute address (e.g. @ begin_ptr)
				addrExpr := fmt.Sprintf("int64(%s)", v.OffsetExpr)
				writeFieldRead(buf, v.Field, st.Name, endian, addrExpr)
				// No offset advance — remote fields don't consume inline space
			} else if dynamic {
				addrExpr := "int64(addr)+offset"
				writeFieldRead(buf, v.Field, st.Name, endian, addrExpr)
				writeOffsetAdvance(buf, v.Field)
			} else {
				addrExpr := fmt.Sprintf("int64(addr)+%d", v.Offset)
				writeFieldRead(buf, v.Field, st.Name, endian, addrExpr)
			}
		case *resolve.PaddingMember:
			if dynamic {
				fmt.Fprintf(buf, "\toffset += %d // padding\n\n", v.Size)
			}
		case *resolve.ConditionalMember:
			writeConditionalRead(buf, v, st.Name, defaultEndian)
		}
	}

	fmt.Fprintf(buf, "\treturn result, errs\n")
	fmt.Fprintf(buf, "}\n\n")
}

// writeOffsetAdvance emits offset += size for dynamic offset tracking.
func writeOffsetAdvance(buf *bytes.Buffer, f *resolve.Field) {
	if f.Type.Size > 0 {
		fmt.Fprintf(buf, "\toffset += %d\n", f.Type.Size)
	} else if f.Type.Kind == resolve.KindArray && f.Type.Array != nil && f.Type.Array.LengthExpr != "" {
		fmt.Fprintf(buf, "\toffset += int64(len(result.%s)) * %d\n", f.Name, f.Type.Array.Element.Size)
	}
}

// writeConditionalRead emits if/else blocks for conditional members.
func writeConditionalRead(buf *bytes.Buffer, cm *resolve.ConditionalMember, structName string, defaultEndian resolve.Endian) {
	for i, br := range cm.Branches {
		if i == 0 {
			fmt.Fprintf(buf, "if %s {\n", br.Cond)
		} else if br.Cond != "" {
			fmt.Fprintf(buf, "} else if %s {\n", br.Cond)
		} else {
			fmt.Fprintf(buf, "} else {\n")
		}

		for _, f := range br.Fields {
			endian := f.Type.Endian
			if endian != resolve.BigEndian && endian != resolve.LittleEndian {
				endian = defaultEndian
			}
			writeFieldRead(buf, f, structName, endian, "int64(addr)+offset")
			writeOffsetAdvance(buf, f)
		}
	}
	fmt.Fprintf(buf, "}\n\n")
}

func primitiveReadSize(rt *resolve.ResolvedType) int {
	switch rt.Kind {
	case resolve.KindPrimitive:
		return rt.Size
	case resolve.KindEnum:
		if rt.EnumRef != nil {
			return rt.EnumRef.UnderlyingType.Size
		}
	case resolve.KindPointer:
		if rt.Pointer != nil {
			return rt.Pointer.SizeType.Size
		}
	case resolve.KindArray:
		if rt.Array != nil {
			return primitiveReadSize(rt.Array.Element)
		}
	case resolve.KindStruct, resolve.KindBitfield:
		return 0 // handled by recursive Read call
	}
	return 0
}

func endianVar(e resolve.Endian) string {
	if e == resolve.BigEndian {
		return "binary.BigEndian"
	}
	return "binary.LittleEndian"
}

func writeFieldRead(buf *bytes.Buffer, f *resolve.Field, structName string, endian resolve.Endian, addrExpr string) {
	path := fmt.Sprintf("%s.%s", structName, f.Name)

	switch f.Type.Kind {
	case resolve.KindPrimitive:
		writePrimitiveRead(buf, f, path, addrExpr, endian)

	case resolve.KindEnum:
		writeEnumRead(buf, f, path, addrExpr, endian)

	case resolve.KindArray:
		writeArrayRead(buf, f, path, addrExpr, endian)

	case resolve.KindStruct:
		writeCompositeFieldRead(buf, f, path, addrExpr, f.Type.StructRef.Name)

	case resolve.KindBitfield:
		writeCompositeFieldRead(buf, f, path, addrExpr, f.Type.BitfieldRef.Name)

	case resolve.KindPointer:
		writePointerRead(buf, f, path, addrExpr, endian)
	}
}

func writePrimitiveRead(buf *bytes.Buffer, f *resolve.Field, path string, addrExpr string, endian resolve.Endian) {
	prim := f.Type.Primitive
	size := prim.Size
	ev := endianVar(endian)

	fmt.Fprintf(buf, "\t// Field: %s at %s\n", f.Name, addrExpr)
	fmt.Fprintf(buf, "\tif _, err := ctx.ReadAt(buf[:%d], %s); err != nil {\n", size, addrExpr)
	fmt.Fprintf(buf, "\t\terrs.Add(%q, uintptr(%s), err)\n", path, addrExpr)
	fmt.Fprintf(buf, "\t} else {\n")

	switch prim.GoType {
	case "uint8", "byte":
		fmt.Fprintf(buf, "\t\tresult.%s = buf[0]\n", f.Name)
	case "int8":
		fmt.Fprintf(buf, "\t\tresult.%s = int8(buf[0])\n", f.Name)
	case "bool":
		fmt.Fprintf(buf, "\t\tresult.%s = buf[0] != 0\n", f.Name)
	case "uint16":
		fmt.Fprintf(buf, "\t\tresult.%s = %s.Uint16(buf[:%d])\n", f.Name, ev, size)
	case "int16":
		fmt.Fprintf(buf, "\t\tresult.%s = int16(%s.Uint16(buf[:%d]))\n", f.Name, ev, size)
	case "uint32":
		fmt.Fprintf(buf, "\t\tresult.%s = %s.Uint32(buf[:%d])\n", f.Name, ev, size)
	case "int32":
		fmt.Fprintf(buf, "\t\tresult.%s = int32(%s.Uint32(buf[:%d]))\n", f.Name, ev, size)
	case "uint64":
		fmt.Fprintf(buf, "\t\tresult.%s = %s.Uint64(buf[:%d])\n", f.Name, ev, size)
	case "int64":
		fmt.Fprintf(buf, "\t\tresult.%s = int64(%s.Uint64(buf[:%d]))\n", f.Name, ev, size)
	case "float32":
		fmt.Fprintf(buf, "\t\tresult.%s = math.Float32frombits(%s.Uint32(buf[:%d]))\n", f.Name, ev, size)
	case "float64":
		fmt.Fprintf(buf, "\t\tresult.%s = math.Float64frombits(%s.Uint64(buf[:%d]))\n", f.Name, ev, size)
	default:
		// Odd-sized types like [3]byte, [6]byte, [12]byte, [16]byte
		if strings.HasPrefix(prim.GoType, "[") {
			fmt.Fprintf(buf, "\t\tcopy(result.%s[:], buf[:%d])\n", f.Name, size)
		}
	}

	fmt.Fprintf(buf, "\t}\n\n")
}

func writeEnumRead(buf *bytes.Buffer, f *resolve.Field, path string, addrExpr string, endian resolve.Endian) {
	et := f.Type.EnumRef
	size := et.UnderlyingType.Size
	ev := endianVar(endian)

	fmt.Fprintf(buf, "\t// Field: %s (enum) at %s\n", f.Name, addrExpr)
	fmt.Fprintf(buf, "\tif _, err := ctx.ReadAt(buf[:%d], %s); err != nil {\n", size, addrExpr)
	fmt.Fprintf(buf, "\t\terrs.Add(%q, uintptr(%s), err)\n", path, addrExpr)
	fmt.Fprintf(buf, "\t} else {\n")

	goType := et.UnderlyingType.GoType
	switch goType {
	case "uint8", "byte":
		fmt.Fprintf(buf, "\t\tresult.%s = %s(buf[0])\n", f.Name, f.Type.GoType)
	case "int8":
		fmt.Fprintf(buf, "\t\tresult.%s = %s(int8(buf[0]))\n", f.Name, f.Type.GoType)
	case "uint16":
		fmt.Fprintf(buf, "\t\tresult.%s = %s(%s.Uint16(buf[:%d]))\n", f.Name, f.Type.GoType, ev, size)
	case "int16":
		fmt.Fprintf(buf, "\t\tresult.%s = %s(int16(%s.Uint16(buf[:%d])))\n", f.Name, f.Type.GoType, ev, size)
	case "uint32":
		fmt.Fprintf(buf, "\t\tresult.%s = %s(%s.Uint32(buf[:%d]))\n", f.Name, f.Type.GoType, ev, size)
	case "int32":
		fmt.Fprintf(buf, "\t\tresult.%s = %s(int32(%s.Uint32(buf[:%d])))\n", f.Name, f.Type.GoType, ev, size)
	case "uint64":
		fmt.Fprintf(buf, "\t\tresult.%s = %s(%s.Uint64(buf[:%d]))\n", f.Name, f.Type.GoType, ev, size)
	case "int64":
		fmt.Fprintf(buf, "\t\tresult.%s = %s(int64(%s.Uint64(buf[:%d])))\n", f.Name, f.Type.GoType, ev, size)
	}

	fmt.Fprintf(buf, "\t}\n\n")
}

func writeArrayRead(buf *bytes.Buffer, f *resolve.Field, path string, addrExpr string, endian resolve.Endian) {
	arr := f.Type.Array
	elem := arr.Element

	isDynamic := arr.LengthExpr != ""

	if isDynamic {
		fmt.Fprintf(buf, "\t// Field: %s (dynamic array) at %s\n", f.Name, addrExpr)
		fmt.Fprintf(buf, "\tresult.%s = make(%s, int(%s))\n", f.Name, f.Type.GoType, arr.LengthExpr)
	} else {
		fmt.Fprintf(buf, "\t// Field: %s (array[%d]) at %s\n", f.Name, arr.Length, addrExpr)
	}

	switch elem.Kind {
	case resolve.KindPrimitive:
		if elem.Size == 1 && !isDynamic {
			// Byte arrays: read all at once (fixed size only)
			fmt.Fprintf(buf, "\tif _, err := ctx.ReadAt(result.%s[:], %s); err != nil {\n", f.Name, addrExpr)
			fmt.Fprintf(buf, "\t\terrs.Add(%q, uintptr(%s), err)\n", path, addrExpr)
			fmt.Fprintf(buf, "\t}\n\n")
		} else if elem.Size == 1 && isDynamic {
			// Dynamic byte slice: read all at once
			fmt.Fprintf(buf, "\tif len(result.%s) > 0 {\n", f.Name)
			fmt.Fprintf(buf, "\t\tif _, err := ctx.ReadAt(result.%s, %s); err != nil {\n", f.Name, addrExpr)
			fmt.Fprintf(buf, "\t\t\terrs.Add(%q, uintptr(%s), err)\n", path, addrExpr)
			fmt.Fprintf(buf, "\t\t}\n")
			fmt.Fprintf(buf, "\t}\n\n")
		} else {
			// Multi-byte element arrays
			ev := endianVar(endian)
			fmt.Fprintf(buf, "\tfor i := range result.%s {\n", f.Name)
			elemOffset := fmt.Sprintf("%s+int64(i)*%d", addrExpr, elem.Size)
			fmt.Fprintf(buf, "\t\tif _, err := ctx.ReadAt(buf[:%d], %s); err != nil {\n", elem.Size, elemOffset)
			fmt.Fprintf(buf, "\t\t\terrs.Add(%q, uintptr(%s), err)\n", path, elemOffset)
			fmt.Fprintf(buf, "\t\t} else {\n")
			writeArrayElemDecode(buf, f.Name, elem, ev)
			fmt.Fprintf(buf, "\t\t}\n")
			fmt.Fprintf(buf, "\t}\n\n")
		}

	case resolve.KindStruct:
		fmt.Fprintf(buf, "\tfor i := range result.%s {\n", f.Name)
		elemOffset := fmt.Sprintf("%s+int64(i)*%d", addrExpr, elem.Size)
		fmt.Fprintf(buf, "\t\telem, elemErrs := Read%s(ctx, uintptr(%s))\n", elem.StructRef.Name, elemOffset)
		fmt.Fprintf(buf, "\t\tif elem != nil {\n")
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = *elem\n", f.Name)
		fmt.Fprintf(buf, "\t\t}\n")
		fmt.Fprintf(buf, "\t\terrs = append(errs, elemErrs...)\n")
		fmt.Fprintf(buf, "\t}\n\n")

	default:
		// Unsupported array element, read raw bytes
		if !isDynamic {
			totalSize := arr.Length * elem.Size
			fmt.Fprintf(buf, "\t{\n")
			fmt.Fprintf(buf, "\t\tvar tmp [%d]byte\n", totalSize)
			fmt.Fprintf(buf, "\t\tif _, err := ctx.ReadAt(tmp[:], %s); err != nil {\n", addrExpr)
			fmt.Fprintf(buf, "\t\t\terrs.Add(%q, uintptr(%s), err)\n", path, addrExpr)
			fmt.Fprintf(buf, "\t\t}\n")
			fmt.Fprintf(buf, "\t}\n\n")
		}
	}
}

func writeArrayElemDecode(buf *bytes.Buffer, fieldName string, elem *resolve.ResolvedType, ev string) {
	prim := elem.Primitive
	switch prim.GoType {
	case "uint8", "byte":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = buf[0]\n", fieldName)
	case "int8":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = int8(buf[0])\n", fieldName)
	case "uint16":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = %s.Uint16(buf[:%d])\n", fieldName, ev, elem.Size)
	case "int16":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = int16(%s.Uint16(buf[:%d]))\n", fieldName, ev, elem.Size)
	case "uint32":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = %s.Uint32(buf[:%d])\n", fieldName, ev, elem.Size)
	case "int32":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = int32(%s.Uint32(buf[:%d]))\n", fieldName, ev, elem.Size)
	case "uint64":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = %s.Uint64(buf[:%d])\n", fieldName, ev, elem.Size)
	case "int64":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = int64(%s.Uint64(buf[:%d]))\n", fieldName, ev, elem.Size)
	case "float32":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = math.Float32frombits(%s.Uint32(buf[:%d]))\n", fieldName, ev, elem.Size)
	case "float64":
		fmt.Fprintf(buf, "\t\t\tresult.%s[i] = math.Float64frombits(%s.Uint64(buf[:%d]))\n", fieldName, ev, elem.Size)
	}
}

func writeCompositeFieldRead(buf *bytes.Buffer, f *resolve.Field, path string, addrExpr string, readName string) {
	fmt.Fprintf(buf, "\t// Field: %s at %s\n", f.Name, addrExpr)
	fmt.Fprintf(buf, "\t{\n")
	fmt.Fprintf(buf, "\t\tchild, childErrs := Read%s(ctx, uintptr(%s))\n", readName, addrExpr)
	fmt.Fprintf(buf, "\t\tif child != nil {\n")
	fmt.Fprintf(buf, "\t\t\tresult.%s = *child\n", f.Name)
	fmt.Fprintf(buf, "\t\t}\n")
	fmt.Fprintf(buf, "\t\terrs = append(errs, childErrs...)\n")
	fmt.Fprintf(buf, "\t}\n\n")
}

func writePointerRead(buf *bytes.Buffer, f *resolve.Field, path string, addrExpr string, endian resolve.Endian) {
	ptr := f.Type.Pointer
	ptrSize := ptr.SizeType.Size
	ev := endianVar(endian)

	fmt.Fprintf(buf, "\t// Field: %s (pointer) at %s\n", f.Name, addrExpr)
	fmt.Fprintf(buf, "\tif _, err := ctx.ReadAt(buf[:%d], %s); err != nil {\n", ptrSize, addrExpr)
	fmt.Fprintf(buf, "\t\terrs.Add(%q, uintptr(%s), err)\n", path, addrExpr)
	fmt.Fprintf(buf, "\t} else {\n")

	// Store raw pointer value
	switch ptrSize {
	case 4:
		fmt.Fprintf(buf, "\t\tresult.%s = %s.Uint32(buf[:%d])\n", f.Name, ev, ptrSize)
	case 8:
		fmt.Fprintf(buf, "\t\tresult.%s = %s.Uint64(buf[:%d])\n", f.Name, ev, ptrSize)
	}

	fmt.Fprintf(buf, "\t}\n\n")
}

// writePointerFollowMethods emits Read<Field>(ctx) methods on eager structs
// for each pointer field, allowing consumers to follow pointers conveniently.
func writePointerFollowMethods(buf *bytes.Buffer, st *resolve.StructType) {
	for _, f := range st.Fields() {
		if f.Type.Kind != resolve.KindPointer || f.Type.Pointer == nil {
			continue
		}
		ptr := f.Type.Pointer
		pointee := ptr.Pointee
		if pointee.Kind != resolve.KindStruct || pointee.StructRef == nil {
			continue
		}
		childName := pointee.StructRef.Name
		fmt.Fprintf(buf, "// Read%s follows the %s pointer and reads the target %s.\n", f.Name, f.Name, childName)
		fmt.Fprintf(buf, "func (s *%s) Read%s(ctx *runtime.ReadContext) (*%s, runtime.Errors) {\n", st.Name, f.Name, childName)
		fmt.Fprintf(buf, "\tif s.%s == 0 {\n", f.Name)
		fmt.Fprintf(buf, "\t\treturn nil, nil\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\treturn Read%s(ctx, uintptr(s.%s))\n", childName, f.Name)
		fmt.Fprintf(buf, "}\n\n")
	}
}

// writePlacementConstants emits a const block for all static address placements.
func writePlacementConstants(buf *bytes.Buffer, placements []*resolve.Placement) {
	fmt.Fprintf(buf, "// Static address constants for top-level placements.\nconst (\n")
	for _, p := range placements {
		fmt.Fprintf(buf, "\tAddr%s uint32 = 0x%08X\n", p.Name, p.Address)
	}
	fmt.Fprintf(buf, ")\n\n")
}

// writePlacementReadFunc emits a read function for a top-level placement.
func writePlacementReadFunc(buf *bytes.Buffer, p *resolve.Placement, defaultEndian resolve.Endian) {
	endian := p.Type.Endian
	if endian != resolve.BigEndian && endian != resolve.LittleEndian {
		endian = defaultEndian
	}
	ev := endianVar(endian)

	if p.Type.Kind == resolve.KindPointer && p.Type.Pointer != nil {
		// Pointer placement: read the pointer value at the static address, then follow it
		ptr := p.Type.Pointer
		ptrSize := ptr.SizeType.Size
		pointee := ptr.Pointee

		if pointee.Kind == resolve.KindStruct && pointee.StructRef != nil {
			childName := pointee.StructRef.Name
			fmt.Fprintf(buf, "// Read%s reads the pointer at Addr%s and follows it to %s.\n", p.Name, p.Name, childName)
			fmt.Fprintf(buf, "func Read%s(ctx *runtime.ReadContext) (*%s, runtime.Errors) {\n", p.Name, childName)
			fmt.Fprintf(buf, "\tvar buf [%d]byte\n", ptrSize)
			fmt.Fprintf(buf, "\tif _, err := ctx.ReadAt(buf[:], int64(Addr%s)); err != nil {\n", p.Name)
			fmt.Fprintf(buf, "\t\tvar errs runtime.Errors\n")
			fmt.Fprintf(buf, "\t\terrs.Add(%q, uintptr(Addr%s), err)\n", p.RawName, p.Name)
			fmt.Fprintf(buf, "\t\treturn nil, errs\n")
			fmt.Fprintf(buf, "\t}\n")
			switch ptrSize {
			case 4:
				fmt.Fprintf(buf, "\tptr := %s.Uint32(buf[:])\n", ev)
			case 8:
				fmt.Fprintf(buf, "\tptr := %s.Uint64(buf[:])\n", ev)
			}
			fmt.Fprintf(buf, "\tif ptr == 0 {\n")
			fmt.Fprintf(buf, "\t\treturn nil, nil\n")
			fmt.Fprintf(buf, "\t}\n")
			fmt.Fprintf(buf, "\treturn Read%s(ctx, uintptr(ptr))\n", childName)
			fmt.Fprintf(buf, "}\n\n")
		}
	} else if p.Type.Kind == resolve.KindPrimitive && p.Type.Primitive != nil {
		// Non-pointer primitive placement: just read the value
		prim := p.Type.Primitive
		size := prim.Size
		fmt.Fprintf(buf, "// Read%s reads the %s at Addr%s.\n", p.Name, prim.GoType, p.Name)
		fmt.Fprintf(buf, "func Read%s(ctx *runtime.ReadContext) (%s, error) {\n", p.Name, prim.GoType)
		fmt.Fprintf(buf, "\tvar buf [%d]byte\n", size)
		fmt.Fprintf(buf, "\tif _, err := ctx.ReadAt(buf[:], int64(Addr%s)); err != nil {\n", p.Name)
		fmt.Fprintf(buf, "\t\treturn 0, err\n")
		fmt.Fprintf(buf, "\t}\n")

		switch prim.GoType {
		case "uint8", "byte":
			fmt.Fprintf(buf, "\treturn buf[0], nil\n")
		case "int8":
			fmt.Fprintf(buf, "\treturn int8(buf[0]), nil\n")
		case "bool":
			fmt.Fprintf(buf, "\treturn buf[0] != 0, nil\n")
		case "uint16":
			fmt.Fprintf(buf, "\treturn %s.Uint16(buf[:]), nil\n", ev)
		case "int16":
			fmt.Fprintf(buf, "\treturn int16(%s.Uint16(buf[:])), nil\n", ev)
		case "uint32":
			fmt.Fprintf(buf, "\treturn %s.Uint32(buf[:]), nil\n", ev)
		case "int32":
			fmt.Fprintf(buf, "\treturn int32(%s.Uint32(buf[:])), nil\n", ev)
		case "uint64":
			fmt.Fprintf(buf, "\treturn %s.Uint64(buf[:]), nil\n", ev)
		case "int64":
			fmt.Fprintf(buf, "\treturn int64(%s.Uint64(buf[:])), nil\n", ev)
		case "float32":
			fmt.Fprintf(buf, "\treturn math.Float32frombits(%s.Uint32(buf[:])), nil\n", ev)
		case "float64":
			fmt.Fprintf(buf, "\treturn math.Float64frombits(%s.Uint64(buf[:])), nil\n", ev)
		}
		fmt.Fprintf(buf, "}\n\n")
	} else if p.Type.Kind == resolve.KindStruct && p.Type.StructRef != nil {
		// Non-pointer struct placement: read the struct directly at that address
		structName := p.Type.StructRef.Name
		fmt.Fprintf(buf, "// Read%s reads %s at Addr%s.\n", p.Name, structName, p.Name)
		fmt.Fprintf(buf, "func Read%s(ctx *runtime.ReadContext) (*%s, runtime.Errors) {\n", p.Name, structName)
		fmt.Fprintf(buf, "\treturn Read%s(ctx, uintptr(Addr%s))\n", structName, p.Name)
		fmt.Fprintf(buf, "}\n\n")
	}
}

// writeFuncBinding emits a Go method for a transpiled hexpat function.
func writeFuncBinding(buf *bytes.Buffer, fb *resolve.FuncBinding) {
	receiverName := fb.Receiver.Name

	// Build parameter list
	params := ""
	if fb.NeedsCtx {
		params = "ctx *runtime.ReadContext"
	}

	fmt.Fprintf(buf, "// %s is transpiled from hexpat function %s.\n", fb.GoName, fb.HexpatName)
	fmt.Fprintf(buf, "func (s *%s) %s(%s) %s {\n", receiverName, fb.GoName, params, fb.ReturnType)

	writeTranspiledStmts(buf, fb.Body, 1)

	fmt.Fprintf(buf, "}\n\n")
}

// writeTranspiledStmts emits transpiled statements with proper indentation.
func writeTranspiledStmts(buf *bytes.Buffer, stmts []resolve.TranspiledStmt, depth int) {
	indent := strings.Repeat("\t", depth)
	for i, s := range stmts {
		if len(s.Children) > 0 {
			// Block statement (if, for, etc.)
			// Check if this is a continuation (} else { or } else if)
			if strings.HasPrefix(s.Code, "} else") {
				fmt.Fprintf(buf, "%s%s {\n", indent, s.Code)
			} else {
				fmt.Fprintf(buf, "%s%s {\n", indent, s.Code)
			}
			writeTranspiledStmts(buf, s.Children, depth+1)
			// Only close the block if the next stmt isn't a continuation
			if i+1 < len(stmts) && strings.HasPrefix(stmts[i+1].Code, "} else") {
				// Don't close — the next line starts with "} else"
			} else {
				fmt.Fprintf(buf, "%s}\n", indent)
			}
		} else {
			fmt.Fprintf(buf, "%s%s\n", indent, s.Code)
		}
	}
}

// containsStdFormat checks if any transpiled statement references fmt.Sprintf.
func containsStdFormat(stmts []resolve.TranspiledStmt) bool {
	for _, s := range stmts {
		if strings.Contains(s.Code, "fmt.Sprintf") {
			return true
		}
		if containsStdFormat(s.Children) {
			return true
		}
	}
	return false
}

// writeMemHelpers emits package-level helper functions for std::mem:: operations.
func writeMemHelpers(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "// _memReadString reads a string of the given length from an address in process memory.\n")
	fmt.Fprintf(buf, "func _memReadString(ctx *runtime.ReadContext, addr, length uint64) string {\n")
	fmt.Fprintf(buf, "\tif length == 0 {\n")
	fmt.Fprintf(buf, "\t\treturn \"\"\n")
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "\tif length > 4096 {\n")
	fmt.Fprintf(buf, "\t\tlength = 4096\n")
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "\tbuf := make([]byte, length)\n")
	fmt.Fprintf(buf, "\tif _, err := ctx.ReadAt(buf, int64(addr)); err != nil {\n")
	fmt.Fprintf(buf, "\t\treturn \"\"\n")
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "\treturn string(buf)\n")
	fmt.Fprintf(buf, "}\n\n")

	fmt.Fprintf(buf, "// _memReadUnsigned reads an unsigned integer of the given byte size from an address.\n")
	fmt.Fprintf(buf, "func _memReadUnsigned(ctx *runtime.ReadContext, addr, size uint64) uint64 {\n")
	fmt.Fprintf(buf, "\tvar buf [8]byte\n")
	fmt.Fprintf(buf, "\tif size > 8 {\n")
	fmt.Fprintf(buf, "\t\tsize = 8\n")
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "\tif _, err := ctx.ReadAt(buf[:size], int64(addr)); err != nil {\n")
	fmt.Fprintf(buf, "\t\treturn 0\n")
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "\tswitch size {\n")
	fmt.Fprintf(buf, "\tcase 1:\n")
	fmt.Fprintf(buf, "\t\treturn uint64(buf[0])\n")
	fmt.Fprintf(buf, "\tcase 2:\n")
	fmt.Fprintf(buf, "\t\treturn uint64(binary.LittleEndian.Uint16(buf[:2]))\n")
	fmt.Fprintf(buf, "\tcase 4:\n")
	fmt.Fprintf(buf, "\t\treturn uint64(binary.LittleEndian.Uint32(buf[:4]))\n")
	fmt.Fprintf(buf, "\tcase 8:\n")
	fmt.Fprintf(buf, "\t\treturn binary.LittleEndian.Uint64(buf[:8])\n")
	fmt.Fprintf(buf, "\tdefault:\n")
	fmt.Fprintf(buf, "\t\treturn 0\n")
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "}\n\n")

	fmt.Fprintf(buf, "// _memReadSigned reads a signed integer of the given byte size from an address.\n")
	fmt.Fprintf(buf, "func _memReadSigned(ctx *runtime.ReadContext, addr, size uint64) int64 {\n")
	fmt.Fprintf(buf, "\treturn int64(_memReadUnsigned(ctx, addr, size))\n")
	fmt.Fprintf(buf, "}\n\n")
}
