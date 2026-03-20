package resolve

// builtinPrimitives maps hexpat type names to their Go representation and size.
var builtinPrimitives = map[string]*PrimitiveInfo{
	"u8":  {Name: "u8", GoType: "uint8", Size: 1},
	"u16": {Name: "u16", GoType: "uint16", Size: 2},
	"u24": {Name: "u24", GoType: "[3]byte", Size: 3},
	"u32": {Name: "u32", GoType: "uint32", Size: 4},
	"u48": {Name: "u48", GoType: "[6]byte", Size: 6},
	"u64": {Name: "u64", GoType: "uint64", Size: 8},
	"u96": {Name: "u96", GoType: "[12]byte", Size: 12},
	"u128": {Name: "u128", GoType: "[16]byte", Size: 16},

	"s8":  {Name: "s8", GoType: "int8", Size: 1},
	"s16": {Name: "s16", GoType: "int16", Size: 2},
	"s24": {Name: "s24", GoType: "[3]byte", Size: 3},
	"s32": {Name: "s32", GoType: "int32", Size: 4},
	"s48": {Name: "s48", GoType: "[6]byte", Size: 6},
	"s64": {Name: "s64", GoType: "int64", Size: 8},
	"s96": {Name: "s96", GoType: "[12]byte", Size: 12},
	"s128": {Name: "s128", GoType: "[16]byte", Size: 16},

	"float":  {Name: "float", GoType: "float32", Size: 4},
	"f32":    {Name: "f32", GoType: "float32", Size: 4},
	"double": {Name: "double", GoType: "float64", Size: 8},
	"f64":    {Name: "f64", GoType: "float64", Size: 8},

	"char":   {Name: "char", GoType: "byte", Size: 1},
	"char16": {Name: "char16", GoType: "uint16", Size: 2},
	"bool":   {Name: "bool", GoType: "bool", Size: 1},
}

// LookupBuiltin returns the PrimitiveInfo for a builtin type name, or nil.
func LookupBuiltin(name string) *PrimitiveInfo {
	return builtinPrimitives[name]
}
