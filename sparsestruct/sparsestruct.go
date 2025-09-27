// Package sparsestruct provides functionality to unmarshal sparse binary data
// into Go structs. It supports specifying offsets and byte orders via struct
// tags, and can handle pointer fields that reference other structures in
// memory.
package sparsestruct

import (
	"context"
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const tag = "offset"

func Size(v any) (int, error) {
	var offset uintptr
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return 0, fmt.Errorf("non-pointer %T", v)
	}
	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return 0, fmt.Errorf("non-struct pointer %T", v)
	}
	t := elem.Type()
	for i := range elem.NumField() {
		field := t.Field(i)
		fieldVal := elem.Field(i)
		_, x, err := parseTag(fieldVal, field, offset, 0)
		if err != nil {
			return 0, fmt.Errorf("parsing tag for field %s: %w", field.Name, err)
		}
		offset += x + uintptr(fieldVal.Type().Size())
	}
	return int(offset), nil
}

// Unmarshal unmarshals a byte slice into a struct.
//
// You may specify an offset tag per v field with either `le` or `be` to specify the byte order.
//
// You may also specify a numeric value in the offset tag per field.
//
// This offset defaults to 0 for the first field, and for any subsequent fields,
// it defaults to the sum of the previous field's offset and its size.
func Unmarshal(data []byte, v any) error {
	return unmarshal(0, data, v)
}

func unmarshal(addr uintptr, data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("non-pointer %T", v)
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("non-struct pointer %T", v)
	}

	t := elem.Type()

	offset := addr

	for i := range elem.NumField() {
		field := t.Field(i)
		fieldVal := elem.Field(i)

		var byteOrder binary.ByteOrder
		var err error
		byteOrder, offset, err = parseTag(fieldVal, field, offset, addr)
		if err != nil {
			return fmt.Errorf("parsing tags: %w", err)
		}

		switch fieldVal.Kind() {
		case reflect.Pointer:
			if field.Type.Elem().Kind() == reflect.Struct && strings.HasPrefix(field.Type.Elem().Name(), "PointerGetter[") {
				if fieldVal.IsNil() {
					// Create a new PointerGetter instance.
					ptrType := reflect.TypeOf(fieldVal.Interface())
					newPtr := reflect.New(ptrType.Elem())
					fieldVal.Set(newPtr)
				}
				pgVal := fieldVal.Elem()
				var pAddr uint64
				n, err := binary.Decode(data[offset:], byteOrder, &pAddr)
				if err != nil {
					return fmt.Errorf("reading address for field %s: %w", field.Name, err)
				}
				// data = data[offset+uintptr(n):]
				offset += uintptr(n)
				// These fields must be exported on PointerGetter.
				pgVal.FieldByName("AddressValue").SetUint(pAddr)
				pgVal.FieldByName("Data").Set(reflect.ValueOf(data))
				continue
			} else {
				return fmt.Errorf("unsupported pointer type for field %s: %s", field.Name, field.Type)
			}
		case reflect.Int8:
			var val int8
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading int8 for field %s: %w", field.Name, err)
			}
			// data = data[offset+uintptr(n):]
			offset += uintptr(n)
			fieldVal.SetInt(int64(val))
		case reflect.Int16:
			var val int16
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading int16 for field %s: %w", field.Name, err)
			}
			// data = data[offset+uintptr(n):]
			offset += uintptr(n)
			fieldVal.SetInt(int64(val))
		case reflect.Int32:
			var val int32
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading int32 for field %s: %w", field.Name, err)
			}
			// data = data[offset+uintptr(n):]
			offset += uintptr(n)
			fieldVal.SetInt(int64(val))
		case reflect.Int, reflect.Int64:
			var val int64
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading int64 for field %s: %w", field.Name, err)
			}
			// data = data[offset+uintptr(n):]
			offset += uintptr(n)
			fieldVal.SetInt(val)
		case reflect.Uint8:
			var val uint8
			fmt.Printf("Reading uint8 at offset %d, address %d\n", offset, addr)
			fmt.Printf("Data: % x\n", data[offset:])
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading uint8 for field %s: %w", field.Name, err)
			}
			// data = data[offset+uintptr(n):]
			offset += uintptr(n)
			fieldVal.SetUint(uint64(val))
		case reflect.Uint16:
			var val uint16
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading uint16 for field %s: %w", field.Name, err)
			}
			// data = data[offset+uintptr(n):]
			offset += uintptr(n)
			fieldVal.SetUint(uint64(val))
		case reflect.Uint32:
			var val uint32
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading uint32 for field %s: %w", field.Name, err)
			}
			// data = data[offset+uintptr(n):]
			offset += uintptr(n)
			fieldVal.SetUint(uint64(val))
		case reflect.Uint, reflect.Uint64:
			var val uint64
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading uint64 for field %s: %w", field.Name, err)
			}
			// data = data[offset+uintptr(n):]
			offset += uintptr(n)
			fieldVal.SetUint(uint64(val))
		default:
			return fmt.Errorf("unsupported type: %s", fieldVal.Type())
		}
	}

	return nil
}

func parseTag(fieldVal reflect.Value, field reflect.StructField, offset uintptr, addr uintptr) (binary.ByteOrder, uintptr, error) {
	if !fieldVal.CanSet() {
		fmt.Printf("%+v: %v\n", fieldVal.Type(), fieldVal.Kind())
		return nil, 0, fmt.Errorf("non-settable field %s", field.Name)
	}

	var byteOrder binary.ByteOrder = binary.NativeEndian

	offsetTag := field.Tag.Get(tag)
	if offsetTag != "" {
		offsetParts := strings.Split(offsetTag, ",")
		if len(offsetParts) < 1 {
			return nil, 0, fmt.Errorf("invalid offset tag on field %s: %s", field.Name, offsetTag)
		}
		offsetStr := offsetParts[0]
		offsetVal, err := strconv.ParseUint(offsetStr, 0, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid offset tag on field %s: %w", field.Name, err)
		}
		offset = addr + uintptr(offsetVal)
		for _, part := range offsetParts[1:] {
			switch part {
			case "le":
				byteOrder = binary.LittleEndian
			case "be":
				byteOrder = binary.BigEndian
			default:
				return nil, 0, fmt.Errorf("invalid offset tag on field %s: %s", field.Name, offsetTag)
			}
		}
	}
	return byteOrder, offset, nil
}

type PointerGetter[T any] struct {
	AddressValue uintptr
	Data         []byte
	Val          *T
}

// Value returns the stored pointer.
func (p *PointerGetter[T]) Value() *T {
	return p.Val
}

// Address returns the value relative the readseeker's base address.
func (p *PointerGetter[T]) Address() uintptr {
	if p.Val == nil {
		p.Val = new(T)
	}
	return p.AddressValue
}

// Read is a no-op for a preloaded value.
func (p *PointerGetter[T]) Read(ctx context.Context) error {
	if p.Val == nil {
		p.Val = new(T)
	}
	return unmarshal(p.AddressValue, p.Data, p.Val)
}
