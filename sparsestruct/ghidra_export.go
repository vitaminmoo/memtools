package sparsestruct

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
)

// Arch represents target architecture configuration for C code generation.
type Arch struct {
	PointerSize int // Size of pointers in bytes (4 for 32-bit, 8 for 64-bit)
}

// Predefined architectures
var (
	Arch32 = Arch{PointerSize: 4}
	Arch64 = Arch{PointerSize: 8}
)

// GenerateCDefinitions writes C struct definitions compatible with Ghidra to w.
// It recursively processes the types of the provided values (structs).
// Uses 64-bit architecture by default. Use GenerateCDefinitionsWithArch for other architectures.
func GenerateCDefinitions(w io.Writer, values ...any) error {
	return GenerateCDefinitionsWithArch(w, Arch64, values...)
}

// GenerateCDefinitionsWithArch writes C struct definitions with the specified architecture.
func GenerateCDefinitionsWithArch(w io.Writer, arch Arch, values ...any) error {
	ctx := &genContext{
		knownTypes: make(map[reflect.Type]string),
		arch:       arch,
	}

	// 1. Collect all types
	for _, v := range values {
		t := reflect.TypeOf(v)
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return fmt.Errorf("expected struct or pointer to struct, got %v", t.Kind())
		}
		if err := ctx.collectType(t); err != nil {
			return err
		}
	}

	fmt.Fprintln(w, "#include <stdint.h>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "typedef unsigned char bool;")
	fmt.Fprintln(w, "")

	// 2. Forward declarations
	for _, t := range ctx.orderedTypes {
		name := ctx.knownTypes[t]
		fmt.Fprintf(w, "typedef struct %s %s;\n", name, name)
	}
	fmt.Fprintln(w, "")

	// 3. Definitions
	for _, t := range ctx.orderedTypes {
		if err := ctx.generateStruct(w, t); err != nil {
			return err
		}
	}

	return nil
}

type genContext struct {
	knownTypes   map[reflect.Type]string
	orderedTypes []reflect.Type
	anonCount    int
	arch         Arch
}

func (c *genContext) collectType(t reflect.Type) error {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	// Only struct needs definition
	if t.Kind() != reflect.Struct {
		return nil
	}
	if _, ok := c.knownTypes[t]; ok {
		return nil
	}

	name := t.Name()
	if name == "" {
		c.anonCount++
		name = fmt.Sprintf("Struct_%d", c.anonCount)
	} else {
		name = strings.ReplaceAll(name, ".", "_")
	}

	c.knownTypes[t] = name
	c.orderedTypes = append(c.orderedTypes, t)

	// Scan fields for dependencies (including embedded struct fields)
	if err := c.scanFieldsDependencies(t); err != nil {
		return err
	}
	return nil
}

// scanFieldsDependencies recursively scans all fields including embedded structs.
func (c *genContext) scanFieldsDependencies(t reflect.Type) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// For embedded structs, recurse into their fields (don't collect the embedded type itself)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := c.scanFieldsDependencies(field.Type); err != nil {
				return err
			}
			continue
		}

		if err := c.scanDependencies(field.Type); err != nil {
			return err
		}
	}
	return nil
}

func (c *genContext) scanDependencies(t reflect.Type) error {
	switch t.Kind() {
	case reflect.Array:
		return c.scanDependencies(t.Elem())
	case reflect.Pointer:
		if t.Elem().Kind() == reflect.Struct {
			elemName := t.Elem().Name()
			if strings.HasPrefix(elemName, "PointerGetter[") {
				valField, ok := t.Elem().FieldByName("Val")
				if !ok {
					return fmt.Errorf("PointerGetter missing Val field")
				}
				targetType := valField.Type.Elem()
				return c.collectType(targetType)
			}
		}
	case reflect.Struct:
		return c.collectType(t)
	}
	return nil
}

// flatField represents a field with its absolute offset for C generation.
type flatField struct {
	Name   string
	Type   reflect.Type
	Field  reflect.StructField
	Offset uintptr
}

// collectFlatFields gathers all fields from a struct type, flattening embedded structs.
// Fields are returned with their absolute offsets from their tags.
// For fields without explicit offsets, a running offset is maintained.
// When a derived struct has a field at the same offset as an embedded struct,
// the derived struct's field takes precedence (overrides).
func collectFlatFields(t reflect.Type) ([]flatField, error) {
	var fields []flatField
	ctx := &genContext{
		knownTypes: make(map[reflect.Type]string),
		arch:       Arch64, // default for size calculations
	}
	if err := collectFlatFieldsRecursive(t, &fields, ctx); err != nil {
		return nil, err
	}

	// Deduplicate by offset - later fields (from derived structs) override earlier ones
	fieldsByOffset := make(map[uintptr]flatField)
	for _, f := range fields {
		fieldsByOffset[f.Offset] = f // later fields overwrite earlier ones
	}

	// Convert back to slice and sort by offset
	fields = make([]flatField, 0, len(fieldsByOffset))
	for _, f := range fieldsByOffset {
		fields = append(fields, f)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Offset < fields[j].Offset
	})

	return fields, nil
}

func collectFlatFieldsRecursive(t reflect.Type, fields *[]flatField, ctx *genContext) error {
	var runningOffset uintptr

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// For embedded (anonymous) structs, recurse into their fields
		// Embedded structs' fields use their own absolute offsets
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			if err := collectFlatFieldsRecursive(field.Type, fields, ctx); err != nil {
				return err
			}
			continue
		}

		_, tagOffset, err := parseTag(field)
		if err != nil {
			return err
		}

		// Use explicit offset if provided, otherwise use running offset
		offset := runningOffset
		if tagOffset != 0 {
			offset = tagOffset
		}

		*fields = append(*fields, flatField{
			Name:   field.Name,
			Type:   field.Type,
			Field:  field,
			Offset: offset,
		})

		// Calculate field size and update running offset
		fieldSize, err := ctx.getSparseSize(field.Type, &field)
		if err != nil {
			return err
		}
		runningOffset = offset + uintptr(fieldSize)
	}
	return nil
}

func (c *genContext) generateStruct(w io.Writer, t reflect.Type) error {
	name := c.knownTypes[t]
	fmt.Fprintf(w, "struct %s {\n", name)

	// Collect all fields including from embedded structs
	fields, err := collectFlatFields(t)
	if err != nil {
		return err
	}

	var currentOffset uintptr

	for _, ff := range fields {
		if ff.Offset > currentOffset {
			padding := ff.Offset - currentOffset
			fmt.Fprintf(w, "    uint8_t undefined_0x%x[%d];\n", currentOffset, padding)
			currentOffset = ff.Offset
		}

		cType, dims, size, err := c.mapType(ff.Type, &ff.Field)
		if err != nil {
			return fmt.Errorf("field %s: %w", ff.Name, err)
		}

		fmt.Fprintf(w, "    %s %s", cType, ff.Name)
		for _, d := range dims {
			fmt.Fprintf(w, "[%d]", d)
		}
		fmt.Fprintf(w, ";\n")

		currentOffset += size
	}
	fmt.Fprintf(w, "};\n\n")
	return nil
}

func (c *genContext) mapType(t reflect.Type, field *reflect.StructField) (string, []int, uintptr, error) {
	switch t.Kind() {
	case reflect.Bool:
		return "bool", nil, 1, nil
	case reflect.Int8:
		return "int8_t", nil, 1, nil
	case reflect.Uint8:
		return "uint8_t", nil, 1, nil
	case reflect.Int16:
		return "int16_t", nil, 2, nil
	case reflect.Uint16:
		return "uint16_t", nil, 2, nil
	case reflect.Int32:
		return "int32_t", nil, 4, nil
	case reflect.Uint32:
		return "uint32_t", nil, 4, nil
	case reflect.Int64, reflect.Int:
		return "int64_t", nil, 8, nil
	case reflect.Uint64, reflect.Uint:
		return "uint64_t", nil, 8, nil
	case reflect.Uintptr:
		// uintptr maps to void* for generic/variant pointers
		return "void *", nil, uintptr(c.arch.PointerSize), nil

	case reflect.Array:
		elemType, elemDims, elemSize, err := c.mapType(t.Elem(), nil)
		if err != nil {
			return "", nil, 0, err
		}
		newDims := append([]int{t.Len()}, elemDims...)
		return elemType, newDims, elemSize * uintptr(t.Len()), nil

	case reflect.Pointer:
		if t.Elem().Kind() == reflect.Struct {
			elemName := t.Elem().Name()
			if strings.HasPrefix(elemName, "PointerGetter[") {
				valField, ok := t.Elem().FieldByName("Val")
				if !ok {
					return "", nil, 0, fmt.Errorf("PointerGetter missing Val field")
				}
				targetType := valField.Type.Elem()

				typeName := c.knownTypes[targetType]
				if typeName == "" {
					return "", nil, 0, fmt.Errorf("type %v not collected", targetType)
				}

				return "struct " + typeName + " *", nil, uintptr(c.arch.PointerSize), nil
			} else if elemName == "StringPointer" {
				return "char *", nil, uintptr(c.arch.PointerSize), nil
			}
		}
		return "", nil, 0, fmt.Errorf("unsupported pointer type: %v", t)

	case reflect.Struct:
		typeName := c.knownTypes[t]
		if typeName == "" {
			return "", nil, 0, fmt.Errorf("type %v not collected", t)
		}
		size, err := c.calculateSize(t)
		if err != nil {
			return "", nil, 0, err
		}
		return typeName, nil, uintptr(size), nil

	case reflect.String:
		if field == nil {
			return "", nil, 0, fmt.Errorf("string type requires field context")
		}
		opts, err := parseTagOptions(*field)
		if err != nil {
			return "", nil, 0, err
		}
		if opts.MaxLen <= 0 {
			return "", nil, 0, fmt.Errorf("string type requires maxlen tag for C export (e.g., `offset:\"0x0,maxlen:32\"`)")
		}
		return "char", []int{opts.MaxLen}, uintptr(opts.MaxLen), nil

	default:
		return "", nil, 0, fmt.Errorf("unsupported kind: %v", t.Kind())
	}
}

func (c *genContext) calculateSize(t reflect.Type) (int, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return 0, fmt.Errorf("expected struct, got %v", t.Kind())
	}

	// Use collectFlatFields to handle embedded structs
	fields, err := collectFlatFields(t)
	if err != nil {
		return 0, err
	}

	var totalSize uintptr
	for _, ff := range fields {
		sparseSize, err := c.getSparseSize(ff.Type, &ff.Field)
		if err != nil {
			return 0, err
		}

		end := ff.Offset + uintptr(sparseSize)
		if end > totalSize {
			totalSize = end
		}
	}
	return int(totalSize), nil
}

func (c *genContext) getSparseSize(t reflect.Type, field *reflect.StructField) (int, error) {
	switch t.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		return 1, nil
	case reflect.Int16, reflect.Uint16:
		return 2, nil
	case reflect.Int32, reflect.Uint32:
		return 4, nil
	case reflect.Int64, reflect.Uint64, reflect.Int, reflect.Uint:
		return 8, nil
	case reflect.Uintptr:
		// uintptr is a pointer (void*), so use arch pointer size
		return c.arch.PointerSize, nil
	case reflect.Array:
		elemSize, err := c.getSparseSize(t.Elem(), nil)
		if err != nil {
			return 0, err
		}
		return elemSize * t.Len(), nil
	case reflect.Pointer:
		if t.Elem().Kind() == reflect.Struct {
			name := t.Elem().Name()
			if strings.HasPrefix(name, "PointerGetter[") || name == "StringPointer" {
				return c.arch.PointerSize, nil
			}
		}
		return 0, fmt.Errorf("unsupported pointer in size calc")
	case reflect.Struct:
		return c.calculateSize(t)
	case reflect.String:
		if field == nil {
			return 0, fmt.Errorf("string type requires field context")
		}
		opts, err := parseTagOptions(*field)
		if err != nil {
			return 0, err
		}
		if opts.MaxLen <= 0 {
			return 0, fmt.Errorf("string type requires maxlen tag for size calculation")
		}
		return opts.MaxLen, nil
	}
	return 0, fmt.Errorf("unknown size for %v", t)
}
