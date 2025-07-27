package sparsestruct

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	EndianBig = iota
	EndianLittle
)

func Unmarshal(data []byte, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("sparsestruct: Unmarshal(non-pointer %T)", v)
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("sparsestruct: Unmarshal(non-struct pointer %T)", v)
	}

	t := elem.Type()

	for i := range elem.NumField() {
		field := t.Field(i)
		fieldVal := elem.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		offsetTag := field.Tag.Get("offset")
		if offsetTag == "" {
			// Fields without offset are ignored.
			continue
		}

		offsetParts := strings.Split(offsetTag, ",")
		if len(offsetParts) < 1 {
			return fmt.Errorf("sparsestruct: invalid offset tag on field %s: %s", field.Name, offsetTag)
		}
		offsetStr := offsetParts[0]
		offset, err := strconv.ParseUint(offsetStr, 0, 64)
		if err != nil {
			return fmt.Errorf("sparsestruct: invalid offset tag on field %s: %w", field.Name, err)
		}

		var byteOrder binary.ByteOrder = binary.NativeEndian
		ptr := false
		for _, part := range offsetParts[1:] {
			switch part {
			case "ptr":
				ptr = true
			case "le":
				byteOrder = binary.LittleEndian
			case "be":
				byteOrder = binary.BigEndian
			default:
				return fmt.Errorf("sparsestruct: invalid offset tag on field %s: %s", field.Name, offsetTag)
			}
		}
		_ = ptr

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
	}

	return nil
}
