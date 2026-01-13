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
	"math"
	"reflect"
	"strconv"
	"strings"
)

const (
	tag     = "offset"
	verbose = true
)

// Numeric is a constraint for all numeric types that binary.Decode supports.
type Numeric interface {
	~int8 | ~int16 | ~int32 | ~int64 |
		~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~bool
}

// decodeNumeric decodes a numeric value from data using the given byte order.
// Returns the decoded value and number of bytes consumed.
func decodeNumeric[T Numeric](data []byte, order binary.ByteOrder) (T, int, error) {
	var val T
	n, err := binary.Decode(data, order, &val)
	return val, n, err
}

// decodeIntoValue decodes a numeric value and sets it into a reflect.Value.
// Returns the number of bytes consumed.
func decodeIntoValue(data []byte, order binary.ByteOrder, fieldVal reflect.Value) (int, error) {
	switch fieldVal.Kind() {
	case reflect.Int8:
		val, n, err := decodeNumeric[int8](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetInt(int64(val))
		return n, nil
	case reflect.Int16:
		val, n, err := decodeNumeric[int16](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetInt(int64(val))
		return n, nil
	case reflect.Int32:
		val, n, err := decodeNumeric[int32](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetInt(int64(val))
		return n, nil
	case reflect.Int, reflect.Int64:
		val, n, err := decodeNumeric[int64](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetInt(val)
		return n, nil
	case reflect.Uint8:
		val, n, err := decodeNumeric[uint8](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetUint(uint64(val))
		return n, nil
	case reflect.Uint16:
		val, n, err := decodeNumeric[uint16](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetUint(uint64(val))
		return n, nil
	case reflect.Uint32:
		val, n, err := decodeNumeric[uint32](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetUint(uint64(val))
		return n, nil
	case reflect.Uint, reflect.Uint64, reflect.Uintptr:
		val, n, err := decodeNumeric[uint64](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetUint(val)
		return n, nil
	case reflect.Bool:
		val, n, err := decodeNumeric[bool](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetBool(val)
		return n, nil
	case reflect.Float32:
		bits, n, err := decodeNumeric[uint32](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetFloat(float64(math.Float32frombits(bits)))
		return n, nil
	case reflect.Float64:
		bits, n, err := decodeNumeric[uint64](data, order)
		if err != nil {
			return 0, err
		}
		fieldVal.SetFloat(math.Float64frombits(bits))
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported numeric type: %s", fieldVal.Kind())
	}
}

// decodeString reads a null-terminated string from data with optional max length.
// If maxLen > 0, reads at most maxLen bytes. Always stops at first null byte.
func decodeString(data []byte, maxLen int) (string, int) {
	limit := len(data)
	if maxLen > 0 && maxLen < limit {
		limit = maxLen
	}

	var result strings.Builder
	for i := 0; i < limit; i++ {
		if data[i] == 0 {
			break
		}
		result.WriteByte(data[i])
	}

	// Return the number of bytes consumed (maxLen if specified, else string length)
	consumed := result.Len()
	if maxLen > 0 {
		consumed = maxLen
	}
	return result.String(), consumed
}

// decodePointerField handles decoding of pointer fields (PointerGetter and StringPointer).
// Returns the number of bytes consumed.
func decodePointerField(data []byte, order binary.ByteOrder, fieldVal reflect.Value, field reflect.StructField, r io.ReadSeeker) (int, error) {
	if field.Type.Elem().Kind() != reflect.Struct {
		return 0, fmt.Errorf("unsupported pointer type for field %s: %s", field.Name, field.Type)
	}

	elemName := field.Type.Elem().Name()
	isPointerGetter := strings.HasPrefix(elemName, "PointerGetter[")
	isStringPointer := elemName == "StringPointer"

	if !isPointerGetter && !isStringPointer {
		return 0, fmt.Errorf("unsupported pointer type for field %s: %s", field.Name, field.Type)
	}

	// Create instance if nil
	if fieldVal.IsNil() {
		ptrType := reflect.TypeOf(fieldVal.Interface())
		newPtr := reflect.New(ptrType.Elem())
		fieldVal.Set(newPtr)
	}

	// Decode the address
	var pAddr uint64
	n, err := binary.Decode(data, order, &pAddr)
	if err != nil {
		return 0, fmt.Errorf("reading address for field %s: %w", field.Name, err)
	}

	// Set fields and call Read method
	ptrVal := fieldVal.Elem()
	ptrVal.FieldByName("AddressValue").SetUint(pAddr)
	ptrVal.FieldByName("Data").Set(reflect.ValueOf(data))

	method := fieldVal.MethodByName("Read")
	method.Call([]reflect.Value{reflect.ValueOf(r)})

	return n, nil
}

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

	return sizeOfType(t)
}

// sizeOfType calculates the sparse size of a struct type based on offset tags.
// It handles embedded structs by recursively processing their fields.
func sizeOfType(t reflect.Type) (int, error) {
	var totalSize uintptr

	for i := range t.NumField() {
		field := t.Field(i)

		// Handle embedded (anonymous) structs by recursing into their fields
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			embeddedSize, err := sizeOfType(field.Type)
			if err != nil {
				return 0, fmt.Errorf("embedded struct %s: %w", field.Name, err)
			}
			totalSize = max(totalSize, uintptr(embeddedSize))
			continue
		}

		opts, err := parseTagOptions(field)
		if err != nil {
			return 0, fmt.Errorf("parsing tag for field %s: %w", field.Name, err)
		}

		var fieldSize uintptr
		switch field.Type.Kind() {
		case reflect.String:
			if opts.MaxLen > 0 {
				fieldSize = uintptr(opts.MaxLen)
			} else {
				// For strings without maxlen, we can't determine size statically
				// This is a limitation - caller must ensure buffer is large enough
				fieldSize = 0
			}
		default:
			fieldSize = field.Type.Size()
		}

		end := opts.Offset + fieldSize
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
//
// Embedded (anonymous) struct fields are supported - their fields are processed
// using their own absolute offsets from their tags.
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

	return unmarshalStructFields(data, elem, r)
}

// unmarshalStructFields processes the fields of a struct value, handling embedded structs recursively.
func unmarshalStructFields(data []byte, elem reflect.Value, r io.ReadSeeker) error {
	t := elem.Type()
	var offset uintptr

	for i := range elem.NumField() {
		field := t.Field(i)
		fieldVal := elem.Field(i)

		if !fieldVal.CanSet() {
			fmt.Printf("%+v: %v\n", fieldVal.Type(), fieldVal.Kind())
			return fmt.Errorf("non-settable field %s", field.Name)
		}

		// Handle embedded (anonymous) structs by recursing into their fields
		if field.Anonymous && fieldVal.Kind() == reflect.Struct {
			if err := unmarshalStructFields(data, fieldVal, r); err != nil {
				return fmt.Errorf("embedded struct %s: %w", field.Name, err)
			}
			continue
		}

		opts, err := parseTagOptions(field)
		if err != nil {
			return fmt.Errorf("parsing tags: %w", err)
		}
		if opts.Offset != 0 {
			offset = opts.Offset
		}

		switch fieldVal.Kind() {
		case reflect.Pointer:
			n, err := decodePointerField(data[offset:], opts.ByteOrder, fieldVal, field, r)
			if err != nil {
				return err
			}
			offset += uintptr(n)

		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int, reflect.Int64,
			reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64,
			reflect.Bool:
			n, err := decodeIntoValue(data[offset:], opts.ByteOrder, fieldVal)
			if err != nil {
				return fmt.Errorf("reading %s for field %s: %w", fieldVal.Kind(), field.Name, err)
			}
			offset += uintptr(n)

		case reflect.String:
			val, n := decodeString(data[offset:], opts.MaxLen)
			fieldVal.SetString(val)
			offset += uintptr(n)

		case reflect.Array:
			for j := 0; j < fieldVal.Len(); j++ {
				elem := fieldVal.Index(j)
				n, err := decodeIntoValue(data[offset:], opts.ByteOrder, elem)
				if err != nil {
					return fmt.Errorf("reading %s for field %s[%d]: %w", elem.Kind(), field.Name, j, err)
				}
				offset += uintptr(n)
			}

		default:
			return fmt.Errorf("unsupported type: %s", fieldVal.Type())
		}
	}

	return nil
}

// tagOptions holds parsed struct tag options.
type tagOptions struct {
	ByteOrder binary.ByteOrder
	Offset    uintptr
	MaxLen    int // For strings: maximum bytes to read (0 = unlimited)
}

func parseTag(field reflect.StructField) (binary.ByteOrder, uintptr, error) {
	opts, err := parseTagOptions(field)
	if err != nil {
		return nil, 0, err
	}
	return opts.ByteOrder, opts.Offset, nil
}

func parseTagOptions(field reflect.StructField) (tagOptions, error) {
	opts := tagOptions{
		ByteOrder: binary.NativeEndian,
	}

	offsetTag := field.Tag.Get(tag)
	if offsetTag == "" {
		return opts, nil
	}

	offsetParts := strings.Split(offsetTag, ",")
	if len(offsetParts) < 1 {
		return opts, fmt.Errorf("invalid offset tag on field %s: %s", field.Name, offsetTag)
	}

	offsetStr := offsetParts[0]
	offsetVal, err := strconv.ParseUint(offsetStr, 0, 64)
	if err != nil {
		return opts, fmt.Errorf("invalid offset tag on field %s: %w", field.Name, err)
	}
	opts.Offset = uintptr(offsetVal)

	for _, part := range offsetParts[1:] {
		switch {
		case part == "le":
			opts.ByteOrder = binary.LittleEndian
		case part == "be":
			opts.ByteOrder = binary.BigEndian
		case strings.HasPrefix(part, "maxlen:"):
			maxLenStr := strings.TrimPrefix(part, "maxlen:")
			maxLen, err := strconv.ParseInt(maxLenStr, 0, 64)
			if err != nil {
				return opts, fmt.Errorf("invalid maxlen on field %s: %w", field.Name, err)
			}
			opts.MaxLen = int(maxLen)
		default:
			return opts, fmt.Errorf("invalid offset tag on field %s: %s", field.Name, offsetTag)
		}
	}

	return opts, nil
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
		if verbose {
			p.Error = fmt.Errorf("pointer address is zero")
		}
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
	if p.AddressValue == 0 || !verbose {
		return []byte("null"), nil
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
		if verbose {
			p.Error = fmt.Errorf("pointer address is zero")
		}
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
	if p.AddressValue == 0 || !verbose {
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
		fmt.Fprintf(&result, "%08x  ", i)

		// Write hex bytes
		for j := range 16 {
			if i+j < len(data) {
				fmt.Fprintf(&result, "%02x ", data[i+j])
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
