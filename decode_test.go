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
	"errors"
	"testing"
	"time"
)

// DecodeCustomType implements Unmarshaler interface for testing.
type DecodeCustomType struct {
	Values []string
}

func (c *DecodeCustomType) UnmarshalValues(values []string) error {
	c.Values = values
	return nil
}

// DecodeCustomTypeWithError implements Unmarshaler that returns an error.
type DecodeCustomTypeWithError struct{}

func (c *DecodeCustomTypeWithError) UnmarshalValues(values []string) error {
	return errors.New("custom unmarshal error")
}

func TestDecode_BasicTypes(t *testing.T) {
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

	data := map[string][]string{
		"string":  {"hello"},
		"bool":    {"true"},
		"int":     {"-42"},
		"int8":    {"-8"},
		"int16":   {"-16"},
		"int32":   {"-32"},
		"int64":   {"-64"},
		"uint":    {"42"},
		"uint8":   {"8"},
		"uint16":  {"16"},
		"uint32":  {"32"},
		"uint64":  {"64"},
		"float32": {"3.14"},
		"float64": {"3.14159265359"},
	}

	var result BasicStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.String != "hello" {
		t.Errorf("String = %q, want %q", result.String, "hello")
	}
	if result.Bool != true {
		t.Errorf("Bool = %v, want %v", result.Bool, true)
	}
	if result.Int != -42 {
		t.Errorf("Int = %d, want %d", result.Int, -42)
	}
	if result.Int8 != -8 {
		t.Errorf("Int8 = %d, want %d", result.Int8, -8)
	}
	if result.Int16 != -16 {
		t.Errorf("Int16 = %d, want %d", result.Int16, -16)
	}
	if result.Int32 != -32 {
		t.Errorf("Int32 = %d, want %d", result.Int32, -32)
	}
	if result.Int64 != -64 {
		t.Errorf("Int64 = %d, want %d", result.Int64, -64)
	}
	if result.Uint != 42 {
		t.Errorf("Uint = %d, want %d", result.Uint, 42)
	}
	if result.Uint8 != 8 {
		t.Errorf("Uint8 = %d, want %d", result.Uint8, 8)
	}
	if result.Uint16 != 16 {
		t.Errorf("Uint16 = %d, want %d", result.Uint16, 16)
	}
	if result.Uint32 != 32 {
		t.Errorf("Uint32 = %d, want %d", result.Uint32, 32)
	}
	if result.Uint64 != 64 {
		t.Errorf("Uint64 = %d, want %d", result.Uint64, 64)
	}
	if result.Float32 != 3.14 {
		t.Errorf("Float32 = %f, want %f", result.Float32, 3.14)
	}
	if result.Float64 != 3.14159265359 {
		t.Errorf("Float64 = %f, want %f", result.Float64, 3.14159265359)
	}
}

func TestDecode_PointerTypes(t *testing.T) {
	type PointerStruct struct {
		StringPtr *string `form:"string_ptr"`
		IntPtr    *int    `form:"int_ptr"`
		BoolPtr   *bool   `form:"bool_ptr"`
	}

	data := map[string][]string{
		"string_ptr": {"pointer_value"},
		"int_ptr":    {"123"},
		"bool_ptr":   {"true"},
	}

	var result PointerStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.StringPtr == nil || *result.StringPtr != "pointer_value" {
		t.Errorf("StringPtr = %v, want %q", result.StringPtr, "pointer_value")
	}
	if result.IntPtr == nil || *result.IntPtr != 123 {
		t.Errorf("IntPtr = %v, want %d", result.IntPtr, 123)
	}
	if result.BoolPtr == nil || *result.BoolPtr != true {
		t.Errorf("BoolPtr = %v, want %v", result.BoolPtr, true)
	}
}

func TestDecode_ReusesExistingPointer(t *testing.T) {
	type Inner struct {
		A string `form:"a"`
		B string `form:"b"`
	}
	type Outer struct {
		Inner *Inner `form:"inner"`
	}

	existing := &Inner{A: "old-a", B: "old-b"}
	result := Outer{Inner: existing}
	if err := Unmarshal(map[string][]string{"inner.a": {"new-a"}}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result.Inner != existing {
		t.Error("existing pointer was replaced")
	}
	if result.Inner.A != "new-a" || result.Inner.B != "old-b" {
		t.Errorf("Inner = %+v, want A=new-a B=old-b", result.Inner)
	}
}

func TestDecode_SliceTypes(t *testing.T) {
	type SliceStruct struct {
		Strings []string  `form:"strings"`
		Ints    []int     `form:"ints"`
		Floats  []float64 `form:"floats"`
	}

	data := map[string][]string{
		"strings": {"a", "b", "c"},
		"ints":    {"1", "2", "3"},
		"floats":  {"1.1", "2.2", "3.3"},
	}

	var result SliceStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Strings) != 3 || result.Strings[0] != "a" || result.Strings[1] != "b" || result.Strings[2] != "c" {
		t.Errorf("Strings = %v, want [a b c]", result.Strings)
	}
	if len(result.Ints) != 3 || result.Ints[0] != 1 || result.Ints[1] != 2 || result.Ints[2] != 3 {
		t.Errorf("Ints = %v, want [1 2 3]", result.Ints)
	}
	if len(result.Floats) != 3 || result.Floats[0] != 1.1 || result.Floats[1] != 2.2 || result.Floats[2] != 3.3 {
		t.Errorf("Floats = %v, want [1.1 2.2 3.3]", result.Floats)
	}
}

func TestDecode_SkipTag(t *testing.T) {
	type SkipStruct struct {
		Normal  string `form:"normal"`
		Skipped string `form:"-"`
	}

	data := map[string][]string{
		"normal":  {"value"},
		"Skipped": {"should_be_ignored"},
	}

	var result SkipStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Normal != "value" {
		t.Errorf("Normal = %q, want %q", result.Normal, "value")
	}
	if result.Skipped != "" {
		t.Errorf("Skipped = %q, want empty string", result.Skipped)
	}
}

func TestDecode_NoTag(t *testing.T) {
	type NoTagStruct struct {
		FieldName string
		Another   int
	}

	data := map[string][]string{
		"FieldName": {"value"},
		"Another":   {"42"},
	}

	var result NoTagStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FieldName != "value" {
		t.Errorf("FieldName = %q, want %q", result.FieldName, "value")
	}
	if result.Another != 42 {
		t.Errorf("Another = %d, want %d", result.Another, 42)
	}
}

func TestDecode_CustomUnmarshaler(t *testing.T) {
	type CustomStruct struct {
		Custom DecodeCustomType `form:"custom"`
	}

	data := map[string][]string{
		"custom": {"a", "b", "c"},
	}

	var result CustomStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Custom.Values) != 3 {
		t.Errorf("Custom.Values length = %d, want 3", len(result.Custom.Values))
	}
	if result.Custom.Values[0] != "a" || result.Custom.Values[1] != "b" || result.Custom.Values[2] != "c" {
		t.Errorf("Custom.Values = %v, want [a b c]", result.Custom.Values)
	}
}

func TestDecode_CustomTagName(t *testing.T) {
	type QueryStruct struct {
		Name  string `query:"name"`
		Value int    `query:"value"`
		Form  string `form:"form_field"`
	}

	data := map[string][]string{
		"name":       {"custom_name"},
		"value":      {"999"},
		"form_field": {"should_not_match"},
	}

	decoder := NewDecoder(WithTagName("query"))
	var result QueryStruct
	err := decoder.Decode(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "custom_name" {
		t.Errorf("Name = %q, want %q", result.Name, "custom_name")
	}
	if result.Value != 999 {
		t.Errorf("Value = %d, want %d", result.Value, 999)
	}
	if result.Form != "" {
		t.Errorf("Form = %q, want empty (form tag should not match with query tag name)", result.Form)
	}
}

func TestUnmarshal_Function(t *testing.T) {
	type SimpleStruct struct {
		Name string `form:"name"`
	}

	data := map[string][]string{
		"name": {"test"},
	}

	var result SimpleStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("Name = %q, want %q", result.Name, "test")
	}
}

func TestDecode_ErrorNil(t *testing.T) {
	err := Unmarshal(nil, nil)
	if err != ErrNilPointer {
		t.Errorf("error = %v, want ErrNilPointer", err)
	}
}

func TestDecode_ErrorNonPointer(t *testing.T) {
	type SimpleStruct struct {
		Name string `form:"name"`
	}

	var s SimpleStruct
	err := Unmarshal(nil, s)
	if err != ErrNotPointer {
		t.Errorf("error = %v, want ErrNotPointer", err)
	}
}

func TestDecode_ErrorNilPointer(t *testing.T) {
	var s *struct{ Name string }
	err := Unmarshal(nil, s)
	if err != ErrNilPointer {
		t.Errorf("error = %v, want ErrNilPointer", err)
	}
}

func TestDecode_ErrorInvalidNumber(t *testing.T) {
	type IntStruct struct {
		Value int `form:"value"`
	}

	data := map[string][]string{
		"value": {"abc"},
	}

	var result IntStruct
	err := Unmarshal(data, &result)
	if err == nil {
		t.Error("expected error for invalid number, got nil")
	}
}

func TestDecode_MissingKey(t *testing.T) {
	type MissingStruct struct {
		Present string `form:"present"`
		Missing string `form:"missing"`
	}

	data := map[string][]string{
		"present": {"value"},
	}

	var result MissingStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Present != "value" {
		t.Errorf("Present = %q, want %q", result.Present, "value")
	}
	if result.Missing != "" {
		t.Errorf("Missing = %q, want empty string (zero value)", result.Missing)
	}
}

func TestDecode_BoolValues(t *testing.T) {
	type BoolStruct struct {
		True1  bool `form:"true1"`
		True2  bool `form:"true2"`
		True3  bool `form:"true3"`
		False1 bool `form:"false1"`
		False2 bool `form:"false2"`
		False3 bool `form:"false3"`
	}

	data := map[string][]string{
		"true1":  {"true"},
		"true2":  {"1"},
		"true3":  {"T"},
		"false1": {"false"},
		"false2": {"0"},
		"false3": {"F"},
	}

	var result BoolStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.True1 || !result.True2 || !result.True3 {
		t.Errorf("True values not parsed correctly: %+v", result)
	}
	if result.False1 || result.False2 || result.False3 {
		t.Errorf("False values not parsed correctly: %+v", result)
	}
}

func TestDecode_OmitemptyTag(t *testing.T) {
	type OmitemptyStruct struct {
		Name string `form:"name,omitempty"`
	}

	data := map[string][]string{
		"name": {"test"},
	}

	var result OmitemptyStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("Name = %q, want %q", result.Name, "test")
	}
}

func TestDecode_EmptyTagName(t *testing.T) {
	type EmptyTagStruct struct {
		Name string `form:",omitempty"`
	}

	data := map[string][]string{
		"Name": {"test"},
	}

	var result EmptyTagStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Name != "test" {
		t.Errorf("Name = %q, want %q", result.Name, "test")
	}
}

func TestDecode_UnexportedFields(t *testing.T) {
	type UnexportedStruct struct {
		Exported   string `form:"exported"`
		unexported string `form:"unexported"` //nolint:unused // intentionally unexported for this test
	}

	data := map[string][]string{
		"exported":   {"value"},
		"unexported": {"should_be_ignored"},
	}

	var result UnexportedStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Exported != "value" {
		t.Errorf("Exported = %q, want %q", result.Exported, "value")
	}
}

func TestDecode_PointerNotInMap(t *testing.T) {
	type PointerStruct struct {
		Ptr *string `form:"ptr"`
	}

	data := map[string][]string{}

	var result PointerStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Ptr != nil {
		t.Errorf("Ptr = %v, want nil", result.Ptr)
	}
}

func TestDecode_SliceNotInMap(t *testing.T) {
	type SliceStruct struct {
		Slice []string `form:"slice"`
	}

	data := map[string][]string{}

	var result SliceStruct
	err := Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Slice != nil {
		t.Errorf("Slice = %v, want nil", result.Slice)
	}
}

func TestDecode_InvalidFloat(t *testing.T) {
	type FloatStruct struct {
		Value float64 `form:"value"`
	}

	data := map[string][]string{
		"value": {"not_a_float"},
	}

	var result FloatStruct
	err := Unmarshal(data, &result)
	if err == nil {
		t.Error("expected error for invalid float, got nil")
	}
}

func TestDecode_InvalidBool(t *testing.T) {
	type BoolStruct struct {
		Value bool `form:"value"`
	}

	data := map[string][]string{
		"value": {"not_a_bool"},
	}

	var result BoolStruct
	err := Unmarshal(data, &result)
	if err == nil {
		t.Error("expected error for invalid bool, got nil")
	}
}

func TestDecode_InvalidUint(t *testing.T) {
	type UintStruct struct {
		Value uint `form:"value"`
	}

	data := map[string][]string{
		"value": {"-1"},
	}

	var result UintStruct
	err := Unmarshal(data, &result)
	if err == nil {
		t.Error("expected error for negative uint, got nil")
	}
}

func TestDecode_TimeType(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	timeStr := testTime.Format(time.RFC3339)

	t.Run("time.Time decode", func(t *testing.T) {
		type TimeStruct struct {
			Created time.Time `form:"created"`
		}

		data := map[string][]string{
			"created": {timeStr},
		}

		var result TimeStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Created.Equal(testTime) {
			t.Errorf("Created = %v, want %v", result.Created, testTime)
		}
	})

	t.Run("*time.Time decode non-nil", func(t *testing.T) {
		type TimeStruct struct {
			Created *time.Time `form:"created"`
		}

		data := map[string][]string{
			"created": {timeStr},
		}

		var result TimeStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Created == nil {
			t.Fatal("Created should not be nil")
		}
		if !result.Created.Equal(testTime) {
			t.Errorf("Created = %v, want %v", *result.Created, testTime)
		}
	})

	t.Run("*time.Time remains nil when not provided", func(t *testing.T) {
		type TimeStruct struct {
			Created *time.Time `form:"created"`
		}

		data := map[string][]string{}

		var result TimeStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Created != nil {
			t.Errorf("Created = %v, want nil", result.Created)
		}
	})

	t.Run("invalid time string returns error", func(t *testing.T) {
		type TimeStruct struct {
			Created time.Time `form:"created"`
		}

		data := map[string][]string{
			"created": {"not-a-valid-time"},
		}

		var result TimeStruct
		err := Unmarshal(data, &result)
		if err == nil {
			t.Error("expected error for invalid time string, got nil")
		}
	})

	t.Run("empty time string is ignored", func(t *testing.T) {
		type TimeStruct struct {
			Created time.Time `form:"created"`
		}

		data := map[string][]string{
			"created": {""},
		}

		var result TimeStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Empty string should leave time as zero value
		if !result.Created.IsZero() {
			t.Errorf("Created = %v, want zero time", result.Created)
		}
	})
}

func TestDecode_ByteSlice(t *testing.T) {
	t.Run("[]byte from string", func(t *testing.T) {
		type ByteStruct struct {
			Data []byte `form:"data"`
		}

		data := map[string][]string{
			"data": {"hello world"},
		}

		var result ByteStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if string(result.Data) != "hello world" {
			t.Errorf("Data = %q, want %q", result.Data, "hello world")
		}
	})

	t.Run("[]byte remains nil when not provided", func(t *testing.T) {
		type ByteStruct struct {
			Data []byte `form:"data"`
		}

		data := map[string][]string{}

		var result ByteStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Data != nil {
			t.Errorf("Data = %v, want nil", result.Data)
		}
	})

	t.Run("empty string is ignored", func(t *testing.T) {
		type ByteStruct struct {
			Data []byte `form:"data"`
		}

		data := map[string][]string{
			"data": {""},
		}

		var result ByteStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Empty string should leave data as nil
		if result.Data != nil {
			t.Errorf("Data = %v, want nil", result.Data)
		}
	})
}

func TestRoundTrip_TimeType(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	type TimeStruct struct {
		Created time.Time `form:"created"`
	}

	original := TimeStruct{Created: testTime}

	// Encode
	encoded, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Decode
	var decoded TimeStruct
	err = Unmarshal(encoded, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Compare (ignore monotonic clock)
	if !original.Created.Equal(decoded.Created) {
		t.Errorf("Round trip failed: original = %v, decoded = %v", original.Created, decoded.Created)
	}
}

func TestRoundTrip_ByteSlice(t *testing.T) {
	type ByteStruct struct {
		Data []byte `form:"data"`
	}

	original := ByteStruct{Data: []byte("hello world")}

	// Encode
	encoded, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Decode
	var decoded ByteStruct
	err = Unmarshal(encoded, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Compare
	if string(original.Data) != string(decoded.Data) {
		t.Errorf("Round trip failed: original = %q, decoded = %q", original.Data, decoded.Data)
	}
}

func TestDecode_PointerToByteSlice(t *testing.T) {
	t.Run("*[]byte decode with value", func(t *testing.T) {
		type ByteStruct struct {
			Data *[]byte `form:"data"`
		}

		data := map[string][]string{
			"data": {"hello pointer"},
		}

		var result ByteStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Data == nil {
			t.Fatal("Data should not be nil")
		}
		if string(*result.Data) != "hello pointer" {
			t.Errorf("Data = %q, want %q", *result.Data, "hello pointer")
		}
	})

	t.Run("*[]byte remains nil when not provided", func(t *testing.T) {
		type ByteStruct struct {
			Data *[]byte `form:"data"`
		}

		data := map[string][]string{}

		var result ByteStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Data != nil {
			t.Errorf("Data = %v, want nil", result.Data)
		}
	})

	t.Run("*[]byte with empty string", func(t *testing.T) {
		type ByteStruct struct {
			Data *[]byte `form:"data"`
		}

		data := map[string][]string{
			"data": {""},
		}

		var result ByteStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Empty string should keep pointer as nil
		if result.Data != nil {
			t.Errorf("Data = %v, want nil for empty string", result.Data)
		}
	})
}

func TestRoundTrip_PointerToByteSlice(t *testing.T) {
	t.Run("non-nil *[]byte", func(t *testing.T) {
		type ByteStruct struct {
			Data *[]byte `form:"data"`
		}

		originalData := []byte("roundtrip test")
		original := ByteStruct{Data: &originalData}

		// Encode
		encoded, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		// Decode
		var decoded ByteStruct
		err = Unmarshal(encoded, &decoded)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		// Compare
		if decoded.Data == nil {
			t.Fatal("decoded.Data should not be nil")
		}
		if string(*original.Data) != string(*decoded.Data) {
			t.Errorf("Round trip failed: original = %q, decoded = %q", *original.Data, *decoded.Data)
		}
	})

	t.Run("nil *[]byte", func(t *testing.T) {
		type ByteStruct struct {
			Data *[]byte `form:"data"`
		}

		original := ByteStruct{Data: nil}

		// Encode
		encoded, err := Marshal(original)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}

		// Decode
		var decoded ByteStruct
		err = Unmarshal(encoded, &decoded)
		if err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}

		// Both should be nil
		if decoded.Data != nil {
			t.Errorf("decoded.Data = %v, want nil", decoded.Data)
		}
	})
}

// ValueReceiverUnmarshaler implements Unmarshaler with value receiver for testing.
type ValueReceiverUnmarshaler struct {
	ParsedValue string
}

func (v ValueReceiverUnmarshaler) UnmarshalValues(values []string) error {
	if len(values) > 0 {
		v.ParsedValue = values[0] + "-value-unmarshaled" //nolint:staticcheck // demonstrates value receiver semantics
	}
	return nil
}

// PointerReceiverUnmarshaler implements Unmarshaler with pointer receiver for comparison.
type PointerReceiverUnmarshaler struct {
	ParsedValue string
}

func (v *PointerReceiverUnmarshaler) UnmarshalValues(values []string) error {
	if len(values) > 0 {
		v.ParsedValue = values[0] + "-pointer-unmarshaled"
	}
	return nil
}

func TestDecode_ValueReceiverUnmarshalerWithPointerField(t *testing.T) {
	t.Run("*CustomType field with pointer receiver Unmarshaler", func(t *testing.T) {
		type TestStruct struct {
			Custom *PointerReceiverUnmarshaler `form:"custom"`
		}

		data := map[string][]string{
			"custom": {"test"},
		}

		var result TestStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Custom == nil {
			t.Fatal("Custom should not be nil")
		}
		if result.Custom.ParsedValue != "test-pointer-unmarshaled" {
			t.Errorf("Custom.ParsedValue = %q, want %q", result.Custom.ParsedValue, "test-pointer-unmarshaled")
		}
	})

	t.Run("*CustomType field with value receiver Unmarshaler - should work via pointer", func(t *testing.T) {
		// Note: Value receiver methods are promoted to pointer types in Go
		// So *ValueReceiverUnmarshaler should also implement Unmarshaler
		// However, calling UnmarshalValues on pointer won't modify the struct
		// because the method has a value receiver
		type TestStruct struct {
			Custom *ValueReceiverUnmarshaler `form:"custom"`
		}

		data := map[string][]string{
			"custom": {"test"},
		}

		var result TestStruct
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The pointer should be created (not nil)
		if result.Custom == nil {
			t.Fatal("Custom should not be nil")
		}
		// Note: Value receiver Unmarshaler won't actually modify the struct
		// because it operates on a copy. This is expected Go behavior.
		// The value will be empty string (zero value)
		// This test documents the current behavior.
	})
}

func TestDecode_DefaultValue(t *testing.T) {
	t.Run("single default value when key missing", func(t *testing.T) {
		type S struct {
			Name string `form:"name,default=guest"`
		}
		data := map[string][]string{} // no name key
		var result S
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "guest" {
			t.Errorf("Name = %q, want %q", result.Name, "guest")
		}
	})

	t.Run("int default value when key missing", func(t *testing.T) {
		type S struct {
			Age int `form:"age,default=18"`
		}
		data := map[string][]string{}
		var result S
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Age != 18 {
			t.Errorf("Age = %d, want %d", result.Age, 18)
		}
	})

	t.Run("bool default value when key missing", func(t *testing.T) {
		type S struct {
			Active bool `form:"active,default=true"`
		}
		data := map[string][]string{}
		var result S
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Active != true {
			t.Errorf("Active = %v, want %v", result.Active, true)
		}
	})

	t.Run("float default value when key missing", func(t *testing.T) {
		type S struct {
			Score float64 `form:"score,default=9.5"`
		}
		data := map[string][]string{}
		var result S
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Score != 9.5 {
			t.Errorf("Score = %f, want %f", result.Score, 9.5)
		}
	})

	t.Run("multi-value default for slice field", func(t *testing.T) {
		type S struct {
			Tags []string `form:"tags,default=go|web"`
		}
		data := map[string][]string{}
		var result S
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Tags) != 2 || result.Tags[0] != "go" || result.Tags[1] != "web" {
			t.Errorf("Tags = %v, want [go web]", result.Tags)
		}
	})

	t.Run("key exists - default not used", func(t *testing.T) {
		type S struct {
			Name string `form:"name,default=guest"`
		}
		data := map[string][]string{"name": {"alice"}}
		var result S
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "alice" {
			t.Errorf("Name = %q, want %q", result.Name, "alice")
		}
	})

	t.Run("default with omitempty tag", func(t *testing.T) {
		type S struct {
			Name string `form:"name,omitempty,default=guest"`
		}
		data := map[string][]string{}
		var result S
		err := Unmarshal(data, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "guest" {
			t.Errorf("Name = %q, want %q", result.Name, "guest")
		}
	})
}

func TestDecode_WithTagName(t *testing.T) {
	type S struct {
		Name string `query:"name"`
	}
	dec := NewDecoder(WithTagName("query"))
	data := map[string][]string{"name": {"test"}}
	var result S
	err := dec.Decode(data, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("Name = %q, want %q", result.Name, "test")
	}
}

func TestRoundTrip_DefaultValue(t *testing.T) {
	// Test struct with default values: Marshal -> Unmarshal consistency
	type S struct {
		Name string   `form:"name,default=guest"`
		Tags []string `form:"tags,default=go|web"`
	}

	original := S{Name: "alice", Tags: []string{"api", "rest"}}

	// Encode
	encoded, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Decode
	var decoded S
	err = Unmarshal(encoded, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Compare
	if original.Name != decoded.Name {
		t.Errorf("Name mismatch: original = %q, decoded = %q", original.Name, decoded.Name)
	}
	if len(original.Tags) != len(decoded.Tags) {
		t.Errorf("Tags length mismatch: original = %d, decoded = %d", len(original.Tags), len(decoded.Tags))
	}
	for i := range original.Tags {
		if original.Tags[i] != decoded.Tags[i] {
			t.Errorf("Tags[%d] mismatch: original = %q, decoded = %q", i, original.Tags[i], decoded.Tags[i])
		}
	}
}
