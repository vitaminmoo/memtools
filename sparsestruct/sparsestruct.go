// Package sparsestruct provides functionality to unmarshal sparse binary data
// into Go structs. It supports specifying offsets and byte orders via struct
// tags, and can handle pointer fields that reference other structures in
// memory.
package sparsestruct

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

const tag = "offset"

func Size(v any) (int, error) {
	t := reflect.TypeOf(v)
	if t == nil {
		return 0, fmt.Errorf("nil interface")
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return 0, fmt.Errorf("expected a struct or a pointer to a struct, but got %v, %T", t.Kind(), t)
	}

	var totalSize uintptr
	for i := range t.NumField() {
		field := t.Field(i)
		_, fieldSize, err := parseTag(field)
		if err != nil {
			return 0, fmt.Errorf("parsing tag for field %s: %w", field.Name, err)
		}
		end := field.Offset + fieldSize
		totalSize = max(totalSize, end)
	}
	return int(totalSize), nil
}

// Unmarshal unmarshals a byte slice into a struct.
//
// You may specify an offset tag per v field with either `le` or `be` to specify the byte order.
//
// You may also specify a numeric value in the offset tag per field.
//
// This offset defaults to 0 for the first field, and for any subsequent fields,
// it defaults to the sum of the previous field's offset and its size.
func Unmarshal(r io.ReadSeeker, addr uintptr, v any) error {
	size, err := Size(v)
	if err != nil {
		return fmt.Errorf("getting size of %T: %w", v, err)
	}
	if size == 0 {
		return fmt.Errorf("zero size for %T", v)
	}
	r.Seek(int64(addr), io.SeekStart)
	data := make([]byte, size)
	n, err := r.Read(data)
	if err != nil {
		return fmt.Errorf("reading %T from process: %w", v, err)
	}
	if n < size {
		return fmt.Errorf("read %d bytes, need %d", n, len(data))
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("non-pointer %T", v)
	}

	elem := rv.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("non-struct pointer %T", v)
	}

	t := elem.Type()

	for i := range elem.NumField() {
		field := t.Field(i)
		fieldVal := elem.Field(i)

		if !fieldVal.CanSet() {
			fmt.Printf("%+v: %v\n", fieldVal.Type(), fieldVal.Kind())
			return fmt.Errorf("non-settable field %s", field.Name)
		}

		var byteOrder binary.ByteOrder
		var err error
		byteOrder, tagOffset, err := parseTag(field)
		if err != nil {
			return fmt.Errorf("parsing tags: %w", err)
		}
		offset := tagOffset

		switch fieldVal.Kind() {
		case reflect.Pointer:
			if field.Type.Elem().Kind() == reflect.Struct {
				if strings.HasPrefix(field.Type.Elem().Name(), "PointerGetter[") {
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
					offset += uintptr(n)
					pgVal.FieldByName("AddressValue").SetUint(pAddr)
					pgVal.FieldByName("Data").Set(reflect.ValueOf(data))

					method := fieldVal.MethodByName("Read")
					method.Call([]reflect.Value{reflect.ValueOf(r)})
					continue
				} else if field.Type.Elem().Name() == "StringPointer" {
					if fieldVal.IsNil() {
						// Create a new StringPointer instance.
						ptrType := reflect.TypeOf(fieldVal.Interface())
						newPtr := reflect.New(ptrType.Elem())
						fieldVal.Set(newPtr)
					}
					spVal := fieldVal.Elem()
					var pAddr uint64
					n, err := binary.Decode(data[offset:], byteOrder, &pAddr)
					if err != nil {
						return fmt.Errorf("reading address for field %s: %w", field.Name, err)
					}
					offset += uintptr(n)
					spVal.FieldByName("AddressValue").SetUint(pAddr)
					spVal.FieldByName("Data").Set(reflect.ValueOf(data))

					method := fieldVal.MethodByName("Read")
					method.Call([]reflect.Value{reflect.ValueOf(r)})
					continue
				} else {
					return fmt.Errorf("unsupported pointer type for field %s: %s", field.Name, field.Type)
				}
			}
		case reflect.Int8:
			var val int8
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading int8 for field %s: %w", field.Name, err)
			}
			offset += uintptr(n)
			fieldVal.SetInt(int64(val))
		case reflect.Int16:
			var val int16
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading int16 for field %s: %w", field.Name, err)
			}
			offset += uintptr(n)
			fieldVal.SetInt(int64(val))
		case reflect.Int32:
			var val int32
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading int32 for field %s: %w", field.Name, err)
			}
			offset += uintptr(n)
			fieldVal.SetInt(int64(val))
		case reflect.Int, reflect.Int64:
			var val int64
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading int64 for field %s: %w", field.Name, err)
			}
			offset += uintptr(n)
			fieldVal.SetInt(val)
		case reflect.Uint8:
			var val uint8
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading uint8 for field %s: %w", field.Name, err)
			}
			offset += uintptr(n)
			fieldVal.SetUint(uint64(val))
		case reflect.Uint16:
			var val uint16
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading uint16 for field %s: %w", field.Name, err)
			}
			offset += uintptr(n)
			fieldVal.SetUint(uint64(val))
		case reflect.Uint32:
			var val uint32
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading uint32 for field %s: %w", field.Name, err)
			}
			offset += uintptr(n)
			fieldVal.SetUint(uint64(val))
		case reflect.Uint, reflect.Uint64, reflect.Uintptr:
			var val uint64
			n, err := binary.Decode(data[offset:], byteOrder, &val)
			if err != nil {
				return fmt.Errorf("reading uint64 for field %s: %w", field.Name, err)
			}
			offset += uintptr(n)
			fieldVal.SetUint(uint64(val))

		default:
			return fmt.Errorf("unsupported type: %s", fieldVal.Type())
		}
	}

	return nil
}

func parseTag(field reflect.StructField) (binary.ByteOrder, uintptr, error) {
	var byteOrder binary.ByteOrder = binary.NativeEndian
	var offset uintptr

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
		offset = uintptr(offsetVal)
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
	AddressValue uintptr `json:"-"`
	Data         []byte  `json:"-"`
	Val          *T      `json:"val,omitempty"`
	Error        error   `json:"-"`
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

func (p *PointerGetter[T]) Read(r io.ReadSeeker) {
	if p.AddressValue == 0 {
		p.Error = fmt.Errorf("pointer address is zero")
		return
	}
	if p.Val == nil {
		p.Val = new(T)
	}
	p.Error = Unmarshal(r, p.Address(), p.Val)
}

func (p PointerGetter[T]) MarshalJSON() ([]byte, error) {
	type Alias PointerGetter[T]
	addrValue := fmt.Sprintf("0x%x", p.AddressValue)
	if p.AddressValue == 0 {
		addrValue = ""
	}
	var errStr string
	if p.Error != nil {
		errStr = p.Error.Error()
	}
	aux := struct {
		AddressValue string `json:"address_value,omitempty"`
		Error        string `json:"error,omitempty"`
		Alias
	}{
		AddressValue: addrValue,
		Error:        errStr,
		Alias:        (Alias)(p),
	}

	return json.Marshal(aux)
}

type StringPointer struct {
	AddressValue uintptr `json:"-"`
	Data         []byte  `json:"-"`
	Val          *string `json:"val,omitempty"`
	Error        error   `json:"-"`
}

// Value returns the stored pointer.
func (p *StringPointer) Value() *string {
	return p.Val
}

// Address returns the value relative the readseeker's base address.
func (p *StringPointer) Address() uintptr {
	if p.Val == nil {
		p.Val = new(string)
	}
	return p.AddressValue
}

func (p *StringPointer) Read(r io.ReadSeeker) {
	if p.AddressValue == 0 {
		p.Error = fmt.Errorf("pointer address is zero")
		return
	}
	if p.Val == nil {
		p.Val = new(string)
	}
	buf := make([]byte, 1024)
	r.Seek(int64(p.AddressValue), io.SeekStart)
	n, err := r.Read(buf)
	if err != nil {
		p.Error = err
		return
	}
	for _, b := range buf[:n] {
		if b == 0 {
			break
		}
		*p.Val += string(b)
	}
}

func (p StringPointer) MarshalJSON() ([]byte, error) {
	type Alias StringPointer
	addrValue := fmt.Sprintf("0x%x", p.AddressValue)
	if p.AddressValue == 0 {
		addrValue = ""
	}
	var errStr string
	if p.Error != nil {
		errStr = p.Error.Error()
	}
	aux := struct {
		AddressValue string `json:"address_value,omitempty"`
		Error        string `json:"error,omitempty"`
		Alias
	}{
		AddressValue: addrValue,
		Error:        errStr,
		Alias:        (Alias)(p),
	}

	return json.Marshal(aux)
}

// HexDump formats a byte slice in a classic hexdump format with hex and ASCII representation.
func HexDump(data []byte) string {
	var result strings.Builder

	for i := 0; i < len(data); i += 16 {
		// Write offset
		result.WriteString(fmt.Sprintf("%08x  ", i))

		// Write hex bytes
		for j := range 16 {
			if i+j < len(data) {
				result.WriteString(fmt.Sprintf("%02x ", data[i+j]))
			} else {
				result.WriteString("   ")
			}

			// Add extra space after 8 bytes
			if j == 7 {
				result.WriteString(" ")
			}
		}

		// Write ASCII representation
		result.WriteString(" |")
		for j := 0; j < 16 && i+j < len(data); j++ {
			b := data[i+j]
			if b >= 32 && b <= 126 {
				result.WriteByte(b)
			} else {
				result.WriteByte('.')
			}
		}
		result.WriteString("|\n")
	}

	return result.String()
}
