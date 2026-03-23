package codegen

import (
	"bytes"
	"fmt"

	"github.com/vitaminmoo/memtools/hexpat/resolve"
)

func writeBitfieldStruct(buf *bytes.Buffer, bt *resolve.BitfieldType) {
	fmt.Fprintf(buf, "type %s struct {\n", bt.Name)
	for _, f := range bt.Fields {
		fmt.Fprintf(buf, "\t%s %s\n", f.Name, f.GoType)
	}
	fmt.Fprintf(buf, "}\n\n")
}

func writeBitfieldReadFunc(buf *bytes.Buffer, bt *resolve.BitfieldType, defaultEndian resolve.Endian) {
	underlying := bt.Underlying
	ev := endianVar(defaultEndian)

	fmt.Fprintf(buf, "func Read%s(ctx *runtime.ReadContext, addr uintptr) (*%s, runtime.Errors) {\n", bt.Name, bt.Name)
	fmt.Fprintf(buf, "\tvar errs runtime.Errors\n")
	fmt.Fprintf(buf, "\tresult := &%s{}\n", bt.Name)
	fmt.Fprintf(buf, "\tvar buf [%d]byte\n", underlying.Size)
	fmt.Fprintf(buf, "\tif _, err := ctx.ReadAt(buf[:], int64(addr)); err != nil {\n")
	fmt.Fprintf(buf, "\t\terrs.Add(%q, addr, err)\n", bt.Name)
	fmt.Fprintf(buf, "\t\treturn result, errs\n")
	fmt.Fprintf(buf, "\t}\n")

	// Decode underlying value
	switch underlying.Size {
	case 1:
		fmt.Fprintf(buf, "\traw := uint64(buf[0])\n")
	case 2:
		fmt.Fprintf(buf, "\traw := uint64(%s.Uint16(buf[:]))\n", ev)
	case 4:
		fmt.Fprintf(buf, "\traw := uint64(%s.Uint32(buf[:]))\n", ev)
	case 8:
		fmt.Fprintf(buf, "\traw := %s.Uint64(buf[:])\n", ev)
	}

	// Extract fields
	for _, f := range bt.Fields {
		mask := (uint64(1) << uint(f.Bits)) - 1
		if f.GoType == "bool" {
			fmt.Fprintf(buf, "\tresult.%s = (raw >> %d) & 1 != 0\n", f.Name, f.BitOffset)
		} else {
			fmt.Fprintf(buf, "\tresult.%s = %s((raw >> %d) & 0x%x)\n", f.Name, f.GoType, f.BitOffset, mask)
		}
	}

	fmt.Fprintf(buf, "\treturn result, errs\n")
	fmt.Fprintf(buf, "}\n\n")
}
