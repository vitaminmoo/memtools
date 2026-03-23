package codegen

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/vitaminmoo/memtools/hexpat/resolve"
)

func writeEnum(buf *bytes.Buffer, et *resolve.EnumType) {
	fmt.Fprintf(buf, "type %s %s\n\n", et.Name, et.UnderlyingType.GoType)
	fmt.Fprintf(buf, "const (\n")
	for _, m := range et.Members {
		fmt.Fprintf(buf, "\t%s%s %s = %d\n", et.Name, m.Name, et.Name, m.Value)
	}
	fmt.Fprintf(buf, ")\n\n")

	writeEnumStringMethod(buf, et)
	writeEnumMarshalJSON(buf, et)
}

func writeEnumStringMethod(buf *bytes.Buffer, et *resolve.EnumType) {
	// Sort members alphabetically by name to match ImHex's std::map iteration
	// order: when duplicate values exist, the alphabetically-first name wins.
	sorted := make([]resolve.EnumMember, len(et.Members))
	copy(sorted, et.Members)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	fmt.Fprintf(buf, "func (e %s) String() string {\n", et.Name)
	fmt.Fprintf(buf, "\tswitch e {\n")
	seen := make(map[int64]bool)
	for _, m := range sorted {
		if seen[m.Value] {
			continue // skip duplicate values to avoid compile errors
		}
		seen[m.Value] = true
		fmt.Fprintf(buf, "\tcase %s%s:\n", et.Name, m.Name)
		fmt.Fprintf(buf, "\t\treturn fmt.Sprintf(\"%s (%%d)\", %s(e))\n", m.Name, et.UnderlyingType.GoType)
	}
	fmt.Fprintf(buf, "\tdefault:\n")
	fmt.Fprintf(buf, "\t\treturn fmt.Sprintf(\"unknown (%%d)\", %s(e))\n", et.UnderlyingType.GoType)
	fmt.Fprintf(buf, "\t}\n")
	fmt.Fprintf(buf, "}\n\n")
}

func writeEnumMarshalJSON(buf *bytes.Buffer, et *resolve.EnumType) {
	fmt.Fprintf(buf, "func (e %s) MarshalJSON() ([]byte, error) {\n", et.Name)
	fmt.Fprintf(buf, "\treturn json.Marshal(e.String())\n")
	fmt.Fprintf(buf, "}\n\n")
}
