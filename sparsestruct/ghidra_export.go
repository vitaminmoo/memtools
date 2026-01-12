package sparsestruct

import (
	"fmt"
	"io"
	"reflect"
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

	// Scan fields for dependencies
	for i := 0; i < t.NumField(); i++ {
		if err := c.scanDependencies(t.Field(i).Type); err != nil {
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

func (c *genContext) generateStruct(w io.Writer, t reflect.Type) error {
	name := c.knownTypes[t]
	fmt.Fprintf(w, "struct %s {\n", name)

	var currentOffset uintptr

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		_, tagOffset, err := parseTag(field)
		if err != nil {
			return err
		}

		if tagOffset > currentOffset {
			padding := tagOffset - currentOffset
			fmt.Fprintf(w, "    uint8_t undefined_0x%x[%d];\n", currentOffset, padding)
			currentOffset = tagOffset
		}

		cType, dims, size, err := c.mapType(field.Type)
		if err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}

		fmt.Fprintf(w, "    %s %s", cType, field.Name)
		for _, d := range dims {
			fmt.Fprintf(w, "[%d]", d)
		}
		fmt.Fprintf(w, ";\n")
		
		currentOffset += size
	}
	fmt.Fprintf(w, "};\n\n")
	return nil
}

func (c *genContext) mapType(t reflect.Type) (string, []int, uintptr, error) {
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
	case reflect.Uint64, reflect.Uint, reflect.Uintptr:
		return "uint64_t", nil, 8, nil
		
	case reflect.Array:
		elemType, elemDims, elemSize, err := c.mapType(t.Elem())
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
		return "", nil, 0, fmt.Errorf("dynamic string type not supported")

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

	var totalSize uintptr
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		_, offset, err := parseTag(field)
		if err != nil {
			return 0, err
		}

		sparseSize, err := c.getSparseSize(field.Type)
		if err != nil {
			return 0, err
		}

		end := offset + uintptr(sparseSize)
		if end > totalSize {
			totalSize = end
		}
	}
	return int(totalSize), nil
}

func (c *genContext) getSparseSize(t reflect.Type) (int, error) {
	switch t.Kind() {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		return 1, nil
	case reflect.Int16, reflect.Uint16:
		return 2, nil
	case reflect.Int32, reflect.Uint32:
		return 4, nil
	case reflect.Int64, reflect.Uint64, reflect.Int, reflect.Uint, reflect.Uintptr:
		return 8, nil
	case reflect.Array:
		elemSize, err := c.getSparseSize(t.Elem())
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
	}
	return 0, fmt.Errorf("unknown size for %v", t)
}
