// Copyright 2026 hushink
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package form

import (
	"reflect"
	"testing"
	"time"
)

// MarshalerType implements Marshaler interface for testing.
type MarshalerType struct {
	Values []string
}

func (c MarshalerType) MarshalValues() ([]string, error) {
	return c.Values, nil
}

// MarshalerTypePointer implements Marshaler with pointer receiver for testing.
type MarshalerTypePointer struct {
	Value string
}

func (c *MarshalerTypePointer) MarshalValues() ([]string, error) {
	return []string{c.Value + "-marshaled"}, nil
}

func TestEncode_BasicTypes(t *testing.T) {
	type BasicStruct struct {
		String  string  `form:"string"`
		Bool    bool    `form:"bool"`
		Int     int     `form:"int"`
		Int8    int8    `form:"int8"`
		Int16   int16   `form:"int16"`
		Int32   int32   `form:"int32"`
		Int64   int64   `form:"int64"`
		Uint    uint    `form:"uint"`
		Uint8   uint8   `form:"uint8"`
		Uint16  uint16  `form:"uint16"`
		Uint32  uint32  `form:"uint32"`
		Uint64  uint64  `form:"uint64"`
		Float32 float32 `form:"float32"`
		Float64 float64 `form:"float64"`
	}

	input := BasicStruct{
		String:  "hello",
		Bool:    true,
		Int:     -42,
		Int8:    -8,
		Int16:   -16,
		Int32:   -32,
		Int64:   -64,
		Uint:    42,
		Uint8:   8,
		Uint16:  16,
		Uint32:  32,
		Uint64:  64,
		Float32: 3.14,
		Float64: 2.718281828,
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"string", "hello"},
		{"bool", "true"},
		{"int", "-42"},
		{"int8", "-8"},
		{"int16", "-16"},
		{"int32", "-32"},
		{"int64", "-64"},
		{"uint", "42"},
		{"uint8", "8"},
		{"uint16", "16"},
		{"uint32", "32"},
		{"uint64", "64"},
		{"float32", "3.14"},
		{"float64", "2.718281828"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			vals, ok := result[tt.key]
			if !ok {
				t.Errorf("missing key %q", tt.key)
				return
			}
			if len(vals) != 1 || vals[0] != tt.expected {
				t.Errorf("got %v, want [%s]", vals, tt.expected)
			}
		})
	}
}

func TestEncode_PointerTypes(t *testing.T) {
	strVal := "pointer-string"
	intVal := 123

	t.Run("non-nil pointers", func(t *testing.T) {
		type PointerStruct struct {
			PtrString *string `form:"ptr_string"`
			PtrInt    *int    `form:"ptr_int"`
		}

		input := PointerStruct{
			PtrString: &strVal,
			PtrInt:    &intVal,
		}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if vals := result["ptr_string"]; len(vals) != 1 || vals[0] != "pointer-string" {
			t.Errorf("ptr_string: got %v, want [pointer-string]", vals)
		}
		if vals := result["ptr_int"]; len(vals) != 1 || vals[0] != "123" {
			t.Errorf("ptr_int: got %v, want [123]", vals)
		}
	})

	t.Run("nil pointers", func(t *testing.T) {
		type PointerStruct struct {
			PtrString *string `form:"ptr_string"`
			PtrInt    *int    `form:"ptr_int"`
		}

		input := PointerStruct{
			PtrString: nil,
			PtrInt:    nil,
		}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// nil pointers should not be included
		if _, ok := result["ptr_string"]; ok {
			t.Error("ptr_string should not be present for nil pointer")
		}
		if _, ok := result["ptr_int"]; ok {
			t.Error("ptr_int should not be present for nil pointer")
		}
	})
}

func TestEncode_SliceTypes(t *testing.T) {
	type SliceStruct struct {
		Strings []string  `form:"strings"`
		Ints    []int     `form:"ints"`
		Floats  []float64 `form:"floats"`
	}

	input := SliceStruct{
		Strings: []string{"a", "b", "c"},
		Ints:    []int{1, 2, 3},
		Floats:  []float64{1.1, 2.2, 3.3},
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vals := result["strings"]; !reflect.DeepEqual(vals, []string{"a", "b", "c"}) {
		t.Errorf("strings: got %v, want [a b c]", vals)
	}
	if vals := result["ints"]; !reflect.DeepEqual(vals, []string{"1", "2", "3"}) {
		t.Errorf("ints: got %v, want [1 2 3]", vals)
	}
	if vals := result["floats"]; !reflect.DeepEqual(vals, []string{"1.1", "2.2", "3.3"}) {
		t.Errorf("floats: got %v, want [1.1 2.2 3.3]", vals)
	}
}

func TestEncode_Omitempty(t *testing.T) {
	type OmitStruct struct {
		Name    string `form:"name,omitempty"`
		Age     int    `form:"age,omitempty"`
		Active  bool   `form:"active,omitempty"`
		Present string `form:"present"`
	}

	input := OmitStruct{
		Name:    "",    // zero value, should be omitted
		Age:     0,     // zero value, should be omitted
		Active:  false, // zero value, should be omitted
		Present: "",    // zero value but no omitempty, should be present
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result["name"]; ok {
		t.Error("name should be omitted")
	}
	if _, ok := result["age"]; ok {
		t.Error("age should be omitted")
	}
	if _, ok := result["active"]; ok {
		t.Error("active should be omitted")
	}
	if _, ok := result["present"]; !ok {
		t.Error("present should be present even if empty")
	}
}

func TestEncode_SkipTag(t *testing.T) {
	type SkipStruct struct {
		Visible string `form:"visible"`
		Ignored string `form:"-"`
	}

	input := SkipStruct{
		Visible: "show",
		Ignored: "hide",
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vals := result["visible"]; len(vals) != 1 || vals[0] != "show" {
		t.Errorf("visible: got %v, want [show]", vals)
	}
	if _, ok := result["Ignored"]; ok {
		t.Error("Ignored field should not be present")
	}
	if _, ok := result["-"]; ok {
		t.Error("- key should not be present")
	}
}

func TestEncode_NoTag(t *testing.T) {
	type NoTagStruct struct {
		Name  string
		Value int
	}

	input := NoTagStruct{
		Name:  "test",
		Value: 42,
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use field names when no tag is present
	if vals := result["Name"]; len(vals) != 1 || vals[0] != "test" {
		t.Errorf("Name: got %v, want [test]", vals)
	}
	if vals := result["Value"]; len(vals) != 1 || vals[0] != "42" {
		t.Errorf("Value: got %v, want [42]", vals)
	}
}

func TestEncode_CustomMarshaler(t *testing.T) {
	t.Run("value receiver marshaler", func(t *testing.T) {
		type MarshalerStruct struct {
			Custom MarshalerType `form:"custom"`
		}

		input := MarshalerStruct{
			Custom: MarshalerType{Values: []string{"x", "y", "z"}},
		}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if vals := result["custom"]; !reflect.DeepEqual(vals, []string{"x", "y", "z"}) {
			t.Errorf("custom: got %v, want [x y z]", vals)
		}
	})

	t.Run("pointer receiver marshaler", func(t *testing.T) {
		type MarshalerStruct struct {
			Custom MarshalerTypePointer `form:"custom"`
		}

		input := MarshalerStruct{
			Custom: MarshalerTypePointer{Value: "test"},
		}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if vals := result["custom"]; len(vals) != 1 || vals[0] != "test-marshaled" {
			t.Errorf("custom: got %v, want [test-marshaled]", vals)
		}
	})
}

func TestEncode_CustomTagName(t *testing.T) {
	type QueryStruct struct {
		Name  string `query:"name"`
		Value int    `query:"value"`
		Form  string `form:"form"` // should be ignored with query tag
	}

	input := QueryStruct{
		Name:  "test",
		Value: 123,
		Form:  "ignored",
	}

	encoder := NewEncoder(WithTagName("query"))
	result, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vals := result["name"]; len(vals) != 1 || vals[0] != "test" {
		t.Errorf("name: got %v, want [test]", vals)
	}
	if vals := result["value"]; len(vals) != 1 || vals[0] != "123" {
		t.Errorf("value: got %v, want [123]", vals)
	}
	// Form should use field name since query tag doesn't exist
	if vals := result["Form"]; len(vals) != 1 || vals[0] != "ignored" {
		t.Errorf("Form: got %v, want [ignored]", vals)
	}
}

func TestMarshal_PackageFunction(t *testing.T) {
	type SimpleStruct struct {
		Name string `form:"name"`
	}

	input := SimpleStruct{Name: "test"}

	// Test package-level Marshal function
	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test Encoder.Encode for comparison
	encoder := NewEncoder()
	expected, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Marshal() and Encoder.Encode() results differ: got %v, want %v", result, expected)
	}
}

func TestEncode_ErrorScenarios(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		_, err := Marshal(nil)
		if err != ErrInvalidValue {
			t.Errorf("got error %v, want ErrInvalidValue", err)
		}
	})

	t.Run("nil pointer", func(t *testing.T) {
		var ptr *struct{ Name string }
		_, err := Marshal(ptr)
		if err != ErrInvalidValue {
			t.Errorf("got error %v, want ErrInvalidValue", err)
		}
	})

	t.Run("int input", func(t *testing.T) {
		values, err := Marshal(42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := values[""]; len(got) != 1 || got[0] != "42" {
			t.Errorf("root value = %v, want [42]", got)
		}
	})

	t.Run("string input", func(t *testing.T) {
		values, err := Marshal("hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := values[""]; len(got) != 1 || got[0] != "hello" {
			t.Errorf("root value = %v, want [hello]", got)
		}
	})

	t.Run("map input", func(t *testing.T) {
		values, err := Marshal(map[string]string{"a": "b"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := values["[a]"]; len(got) != 1 || got[0] != "b" {
			t.Errorf("[a] = %v, want [b]", got)
		}
	})

	t.Run("slice input", func(t *testing.T) {
		values, err := Marshal([]string{"a", "b"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := values[""]; !reflect.DeepEqual(got, []string{"a", "b"}) {
			t.Errorf("root values = %v, want [a b]", got)
		}
	})
}

func TestEncode_PointerToStruct(t *testing.T) {
	type SimpleStruct struct {
		Name string `form:"name"`
	}

	input := &SimpleStruct{Name: "test"}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vals := result["name"]; len(vals) != 1 || vals[0] != "test" {
		t.Errorf("name: got %v, want [test]", vals)
	}
}

func TestEncode_EmptyTagName(t *testing.T) {
	type EmptyTagStruct struct {
		Name  string `form:",omitempty"`
		Value int    `form:""`
	}

	input := EmptyTagStruct{
		Name:  "test",
		Value: 42,
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty tag name should use field name
	if vals := result["Name"]; len(vals) != 1 || vals[0] != "test" {
		t.Errorf("Name: got %v, want [test]", vals)
	}
	if vals := result["Value"]; len(vals) != 1 || vals[0] != "42" {
		t.Errorf("Value: got %v, want [42]", vals)
	}
}

func TestEncode_NilSlice(t *testing.T) {
	type SliceStruct struct {
		Items []string `form:"items"`
	}

	input := SliceStruct{
		Items: nil,
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nil slice should not be present in result
	if _, ok := result["items"]; ok {
		t.Error("items should not be present for nil slice")
	}
}

func TestEncode_EmptySlice(t *testing.T) {
	type SliceStruct struct {
		Items []string `form:"items"`
	}

	input := SliceStruct{
		Items: []string{},
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nil and empty slices are omitted (go-playground/form behavior).
	if vals, ok := result["items"]; ok {
		t.Errorf("items should be omitted for an empty slice, got %v", vals)
	}
}

func TestEncode_SliceWithMarshalerElements(t *testing.T) {
	type SliceStruct struct {
		Items []MarshalerType `form:"items"`
	}

	input := SliceStruct{
		Items: []MarshalerType{
			{Values: []string{"a", "b"}},
			{Values: []string{"c"}},
		},
	}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Custom marshaler elements use explicit indexes, matching
	// go-playground/form's custom type handling.
	if vals := result["items[0]"]; !reflect.DeepEqual(vals, []string{"a", "b"}) {
		t.Errorf("items[0]: got %v, want [a b]", vals)
	}
	if vals := result["items[1]"]; !reflect.DeepEqual(vals, []string{"c"}) {
		t.Errorf("items[1]: got %v, want [c]", vals)
	}
}

func TestEncode_TimeType(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	t.Run("time.Time non-zero", func(t *testing.T) {
		type TimeStruct struct {
			Created time.Time `form:"created"`
		}

		input := TimeStruct{Created: testTime}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := testTime.Format(time.RFC3339)
		if vals := result["created"]; len(vals) != 1 || vals[0] != expected {
			t.Errorf("created: got %v, want [%s]", vals, expected)
		}
	})

	t.Run("time.Time zero value", func(t *testing.T) {
		type TimeStruct struct {
			Created time.Time `form:"created"`
		}

		input := TimeStruct{} // zero value

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Without omitempty, a zero time is encoded like any other struct
		// value (go-playground/form behavior).
		want := time.Time{}.Format(time.RFC3339)
		if vals := result["created"]; len(vals) != 1 || vals[0] != want {
			t.Errorf("created: got %v, want [%s]", vals, want)
		}
	})

	t.Run("*time.Time non-nil", func(t *testing.T) {
		type TimeStruct struct {
			Created *time.Time `form:"created"`
		}

		input := TimeStruct{Created: &testTime}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := testTime.Format(time.RFC3339)
		if vals := result["created"]; len(vals) != 1 || vals[0] != expected {
			t.Errorf("created: got %v, want [%s]", vals, expected)
		}
	})

	t.Run("*time.Time nil", func(t *testing.T) {
		type TimeStruct struct {
			Created *time.Time `form:"created"`
		}

		input := TimeStruct{Created: nil}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// nil pointer should not be present
		if _, ok := result["created"]; ok {
			t.Error("created should not be present for nil pointer")
		}
	})

	t.Run("time.Time zero value with omitempty", func(t *testing.T) {
		type TimeStruct struct {
			Created time.Time `form:"created,omitempty"`
		}

		input := TimeStruct{} // zero value

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// zero time with omitempty should be omitted
		if _, ok := result["created"]; ok {
			t.Error("created should be omitted for zero time with omitempty")
		}
	})

	t.Run("time.Time non-zero with omitempty", func(t *testing.T) {
		type TimeStruct struct {
			Created time.Time `form:"created,omitempty"`
		}

		input := TimeStruct{Created: testTime}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := testTime.Format(time.RFC3339)
		if vals := result["created"]; len(vals) != 1 || vals[0] != expected {
			t.Errorf("created: got %v, want [%s]", vals, expected)
		}
	})
}

func TestEncode_ByteSlice(t *testing.T) {
	t.Run("[]byte with content", func(t *testing.T) {
		type ByteStruct struct {
			Data []byte `form:"data"`
		}

		input := ByteStruct{Data: []byte("hello world")}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if vals := result["data"]; len(vals) != 1 || vals[0] != "hello world" {
			t.Errorf("data: got %v, want [hello world]", vals)
		}
	})

	t.Run("nil []byte", func(t *testing.T) {
		type ByteStruct struct {
			Data []byte `form:"data"`
		}

		input := ByteStruct{Data: nil}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// nil []byte should not be present
		if _, ok := result["data"]; ok {
			t.Error("data should not be present for nil []byte")
		}
	})

	t.Run("empty []byte", func(t *testing.T) {
		type ByteStruct struct {
			Data []byte `form:"data"`
		}

		input := ByteStruct{Data: []byte{}}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// empty []byte should not be present
		if _, ok := result["data"]; ok {
			t.Error("data should not be present for empty []byte")
		}
	})
}

func TestEncode_PointerToByteSlice(t *testing.T) {
	t.Run("*[]byte non-nil", func(t *testing.T) {
		type ByteStruct struct {
			Data *[]byte `form:"data"`
		}

		data := []byte("hello pointer")
		input := ByteStruct{Data: &data}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if vals := result["data"]; len(vals) != 1 || vals[0] != "hello pointer" {
			t.Errorf("data: got %v, want [hello pointer]", vals)
		}
	})

	t.Run("*[]byte nil", func(t *testing.T) {
		type ByteStruct struct {
			Data *[]byte `form:"data"`
		}

		input := ByteStruct{Data: nil}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// nil pointer should not be present
		if _, ok := result["data"]; ok {
			t.Error("data should not be present for nil *[]byte")
		}
	})
}

// ValueReceiverMarshaler implements Marshaler with value receiver for testing.
type ValueReceiverMarshaler struct {
	Value string
}

func (v ValueReceiverMarshaler) MarshalValues() ([]string, error) {
	return []string{v.Value + "-value-marshaled"}, nil
}

func TestEncode_ValueReceiverMarshalerWithPointerField(t *testing.T) {
	t.Run("*CustomType field with value receiver Marshaler", func(t *testing.T) {
		type TestStruct struct {
			Custom *ValueReceiverMarshaler `form:"custom"`
		}

		input := TestStruct{
			Custom: &ValueReceiverMarshaler{Value: "test"},
		}

		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if vals := result["custom"]; len(vals) != 1 || vals[0] != "test-value-marshaled" {
			t.Errorf("custom: got %v, want [test-value-marshaled]", vals)
		}
	})
}

func TestEncode_DefaultValue(t *testing.T) {
	t.Run("omitempty with default - value equals default should be skipped", func(t *testing.T) {
		type S struct {
			Name string `form:"name,omitempty,default=guest"`
		}
		input := S{Name: "guest"}
		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result["name"]; ok {
			t.Error("name should not be present when value equals default with omitempty")
		}
	})

	t.Run("omitempty with default - value differs from default should be kept", func(t *testing.T) {
		type S struct {
			Name string `form:"name,omitempty,default=guest"`
		}
		input := S{Name: "alice"}
		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if vals := result["name"]; len(vals) != 1 || vals[0] != "alice" {
			t.Errorf("name: got %v, want [alice]", vals)
		}
	})

	t.Run("default without omitempty - value equals default should still output", func(t *testing.T) {
		type S struct {
			Name string `form:"name,default=guest"`
		}
		input := S{Name: "guest"}
		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if vals := result["name"]; len(vals) != 1 || vals[0] != "guest" {
			t.Errorf("name: got %v, want [guest] (no omitempty so should output)", vals)
		}
	})

	t.Run("multi-value default with pipe separator", func(t *testing.T) {
		type S struct {
			Tags []string `form:"tags,omitempty,default=go|web"`
		}
		input := S{Tags: []string{"go", "web"}}
		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result["tags"]; ok {
			t.Error("tags should not be present when value equals default with omitempty")
		}
	})

	t.Run("multi-value default - value differs", func(t *testing.T) {
		type S struct {
			Tags []string `form:"tags,omitempty,default=go|web"`
		}
		input := S{Tags: []string{"rust", "wasm"}}
		result, err := Marshal(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if vals := result["tags"]; !reflect.DeepEqual(vals, []string{"rust", "wasm"}) {
			t.Errorf("tags: got %v, want [rust wasm]", vals)
		}
	})

}

func TestEncode_WithTagName(t *testing.T) {
	type S struct {
		Name string `query:"name"`
	}
	enc := NewEncoder(WithTagName("query"))
	input := S{Name: "test"}
	result, err := enc.Encode(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals := result["name"]; len(vals) != 1 || vals[0] != "test" {
		t.Errorf("name: got %v, want [test]", vals)
	}
}
