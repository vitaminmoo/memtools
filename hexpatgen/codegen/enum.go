package codegen

import (
	"bytes"
	"fmt"

	"github.com/vitaminmoo/memtools/hexpatgen/resolve"
)

func writeEnum(buf *bytes.Buffer, et *resolve.EnumType) {
	fmt.Fprintf(buf, "type %s %s\n\n", et.Name, et.UnderlyingType.GoType)
	fmt.Fprintf(buf, "const (\n")
	for _, m := range et.Members {
		fmt.Fprintf(buf, "\t%s%s %s = %d\n", et.Name, m.Name, et.Name, m.Value)
	}
	fmt.Fprintf(buf, ")\n\n")
}
