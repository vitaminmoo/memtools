package sparsestruct

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Unmarshal(data []byte, v any) error {
	return unmarshal(0, data, v)
}

func unmarshal(base uint64, data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("sparsestruct: Unmarshal(non-pointer %T)", v)
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("sparsestruct: Unmarshal(non-struct pointer %T)", v)
	}

	t := elem.Type()

	var next uint64 = 0

	for i := range elem.NumField() {
		field := t.Field(i)
		fieldVal := elem.Field(i)

		if !fieldVal.CanSet() {
			return fmt.Errorf("sparsestruct: Unmarshal(non-settable field %s)", field.Name)
		}

		var offset uint64 = base
		var byteOrder binary.ByteOrder = binary.NativeEndian

		offsetTag := field.Tag.Get("offset")
		if offsetTag != "" {
			offsetParts := strings.Split(offsetTag, ",")
			if len(offsetParts) < 1 {
				return fmt.Errorf("sparsestruct: invalid offset tag on field %s: %s", field.Name, offsetTag)
			}
			offsetStr := offsetParts[0]
			offsetVal, err := strconv.ParseUint(offsetStr, 0, 64)
			if err != nil {
				return fmt.Errorf("sparsestruct: invalid offset tag on field %s: %w", field.Name, err)
			}
			offset += offsetVal
			for _, part := range offsetParts[1:] {
				switch part {
				case "le":
					byteOrder = binary.LittleEndian
				case "be":
					byteOrder = binary.BigEndian
				default:
					return fmt.Errorf("sparsestruct: invalid offset tag on field %s: %s", field.Name, offsetTag)
				}
			}
		} else {
			offset = base + next
		}

		if fieldVal.Type() == reflect.TypeOf(PointerGetter{}) {
			size := uint64(8) // Assuming 64-bit pointers
			end := offset + size
			if end > uint64(len(data)) {
				return fmt.Errorf("sparsestruct: target offset %d for field %s of size %d is out of bounds", offset, field.Name, size)
			}
			val := byteOrder.Uint64(data[offset:end])
			pg := PointerGetter{address: val}
			fieldVal.Set(reflect.ValueOf(pg))
			next = end
			continue
		}

		size := uint64(fieldVal.Type().Size())
		end := offset + size
		if end > uint64(len(data)) {
			return fmt.Errorf("sparsestruct: target offset %d for field %s of size %d is out of bounds", offset, field.Name, size)
		}

		switch fieldVal.Kind() {
		case reflect.Int8:
			val := data[offset]
			fieldVal.SetInt(int64(int8(val)))
		case reflect.Int16:
			val := byteOrder.Uint16(data[offset:end])
			fieldVal.SetInt(int64(int16(val)))
		case reflect.Int32:
			val := byteOrder.Uint32(data[offset:end])
			fieldVal.SetInt(int64(int32(val)))
		case reflect.Int, reflect.Int64:
			val := byteOrder.Uint64(data[offset:end])
			fieldVal.SetInt(int64(val))
		case reflect.Uint8:
			val := data[offset]
			fieldVal.SetUint(uint64(val))
		case reflect.Uint16:
			val := byteOrder.Uint16(data[offset:end])
			fieldVal.SetUint(uint64(val))
		case reflect.Uint32:
			val := byteOrder.Uint32(data[offset:end])
			fieldVal.SetUint(uint64(val))
		case reflect.Uint, reflect.Uint64:
			val := byteOrder.Uint64(data[offset:end])
			fieldVal.SetUint(val)
		default:
			return fmt.Errorf("unsupported type: %s", fieldVal.Type())
		}
		next = end
	}

	return nil
}

// PointerGetter is a utility for lazily chasing pointer chains.
//
// By unmarshalling into this type, the raw address of the pointer is stored.
// This can then be used with PointerGetter.Address() and PointerGetter.Length()
// to determine where and how much to read to feed to PointerGetter.Get(), which will unmarshal the target value.
type PointerGetter struct {
	address uint64
}

// Address returns the raw address of the pointer.
func (pg *PointerGetter) Address() uint64 {
	return pg.address
}

// Length returns the length required to successfully unmarshal the target value.
func (pg *PointerGetter) Length(v any) (uint64, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return 0, fmt.Errorf("sparsestruct: Unmarshal(non-pointer %T)", v)
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return 0, fmt.Errorf("sparsestruct: Unmarshal(non-struct pointer %T)", v)
	}

	t := elem.Type()

	var next uint64
	var end uint64

	for i := range elem.NumField() {
		field := t.Field(i)
		fieldVal := elem.Field(i)

		if !fieldVal.CanSet() {
			return 0, fmt.Errorf("sparsestruct: Unmarshal(non-settable field %s)", field.Name)
		}

		var offset uint64

		offsetTag := field.Tag.Get("offset")
		if offsetTag != "" {
			offsetParts := strings.Split(offsetTag, ",")
			if len(offsetParts) < 1 {
				return 0, fmt.Errorf("sparsestruct: invalid offset tag on field %s: %s", field.Name, offsetTag)
			}
			offsetStr := offsetParts[0]
			offsetVal, err := strconv.ParseUint(offsetStr, 0, 64)
			if err != nil {
				return 0, fmt.Errorf("sparsestruct: invalid offset tag on field %s: %w", field.Name, err)
			}
			offset += offsetVal
			for _, part := range offsetParts[1:] {
				switch part {
				case "le":
				case "be":
				default:
					return 0, fmt.Errorf("sparsestruct: invalid offset tag on field %s: %s", field.Name, offsetTag)
				}
			}
		} else {
			offset = next
		}

		if fieldVal.Type() == reflect.TypeOf(PointerGetter{}) {
			size := uint64(8) // Assuming 64-bit pointers
			end = max(end, offset+size)
			next = end
			continue
		}

		size := uint64(fieldVal.Type().Size())
		end = max(end, offset+size)
		next = end
	}

	return end, nil
}

// Get stores a previously unmarshaled pointer value, and allows for pointer chasing.
//
// base: The base address of data[0] in the virtual address space of the original data.
// data: The byte slice containing at least PointerGetter.Length(v) bytes.
// v: The value to store the pointed to value in.
func (pg *PointerGetter) Get(base uint64, data []byte, v any) error {
	offset := pg.address - base
	unmarshal(offset, data, v)
	return nil
}
