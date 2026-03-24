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
		if rt.Pointer != nil && rt.Pointer.Pointee != nil {
			return "*" + fieldGoType(rt.Pointer.Pointee)
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

	// Read pointer address
	switch ptrSize {
	case 4:
		fmt.Fprintf(buf, "\t\tptrAddr := uintptr(%s.Uint32(buf[:%d]))\n", ev, ptrSize)
	case 8:
		fmt.Fprintf(buf, "\t\tptrAddr := uintptr(%s.Uint64(buf[:%d]))\n", ev, ptrSize)
	default:
		fmt.Fprintf(buf, "\t\t_ = buf // unsupported pointer size %d\n", ptrSize)
		fmt.Fprintf(buf, "\t}\n\n")
		return
	}

	fmt.Fprintf(buf, "\t\tif ptrAddr != 0 && !ctx.Visit(ptrAddr) {\n")

	pointee := ptr.Pointee
	if pointee.Kind == resolve.KindStruct && pointee.StructRef != nil {
		fmt.Fprintf(buf, "\t\t\tchild, childErrs := Read%s(ctx, ptrAddr)\n", pointee.StructRef.Name)
		fmt.Fprintf(buf, "\t\t\tresult.%s = child\n", f.Name)
		fmt.Fprintf(buf, "\t\t\terrs = append(errs, childErrs...)\n")
	}

	fmt.Fprintf(buf, "\t\t}\n")
	fmt.Fprintf(buf, "\t}\n\n")
}
