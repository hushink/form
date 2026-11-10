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
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestDecode_EmptyValues(t *testing.T) {
	type S struct {
		S  string    `form:"s"`
		I  int       `form:"i"`
		B  bool      `form:"b"`
		F  float64   `form:"f"`
		T  time.Time `form:"t"`
		P  *int      `form:"p"`
		PS *string   `form:"ps"`
		PB *bool     `form:"pb"`
	}

	data := map[string][]string{
		"s":  {""},
		"i":  {""},
		"b":  {""},
		"f":  {""},
		"t":  {""},
		"p":  {""},
		"ps": {""},
		"pb": {""},
	}

	var result S
	if err := Unmarshal(data, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.S != "" {
		t.Errorf("S = %q, want empty", result.S)
	}
	if result.I != 0 {
		t.Errorf("I = %d, want 0", result.I)
	}
	if result.B {
		t.Error("B = true, want false")
	}
	if result.F != 0 {
		t.Errorf("F = %v, want 0", result.F)
	}
	if !result.T.IsZero() {
		t.Errorf("T = %v, want zero", result.T)
	}
	if result.P != nil {
		t.Errorf("P = %v, want nil", result.P)
	}
	if result.PS == nil || *result.PS != "" {
		t.Errorf("PS = %v, want pointer to empty string", result.PS)
	}
	if result.PB == nil || *result.PB {
		t.Errorf("PB = %v, want pointer to false", result.PB)
	}
}

func TestDecode_SliceEmptyElements(t *testing.T) {
	type S struct {
		Strings []string `form:"strings"`
		Ints    []int    `form:"ints"`
		Bools   []bool   `form:"bools"`
	}

	data := map[string][]string{
		"strings": {"a", ""},
		"ints":    {"1", ""},
		"bools":   {"true", ""},
	}

	var result S
	if err := Unmarshal(data, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result.Strings, []string{"a", ""}) {
		t.Errorf("Strings = %#v, want [a \"\"]", result.Strings)
	}
	if !reflect.DeepEqual(result.Ints, []int{1, 0}) {
		t.Errorf("Ints = %#v, want [1 0]", result.Ints)
	}
	if !reflect.DeepEqual(result.Bools, []bool{true, false}) {
		t.Errorf("Bools = %#v, want [true false]", result.Bools)
	}
}

func TestDecode_MultiValueScalarUsesFirst(t *testing.T) {
	type S struct {
		S string `form:"s"`
		I int    `form:"i"`
		B bool   `form:"b"`
		P *int   `form:"p"`
	}

	data := map[string][]string{
		"s": {"a", "b"},
		"i": {"5", "6"},
		"b": {"true", "false"},
		"p": {"7", "8"},
	}

	var result S
	if err := Unmarshal(data, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.S != "a" {
		t.Errorf("S = %q, want a", result.S)
	}
	if result.I != 5 {
		t.Errorf("I = %d, want 5", result.I)
	}
	if !result.B {
		t.Error("B = false, want true")
	}
	if result.P == nil || *result.P != 7 {
		t.Errorf("P = %v, want 7", result.P)
	}
}

func TestDecode_BoolTokens(t *testing.T) {
	trueValues := []string{"1", "t", "T", "true", "TRUE", "True", "on", "yes", "ok"}
	falseValues := []string{"0", "f", "F", "false", "FALSE", "False", "off", "no", ""}

	for _, value := range trueValues {
		var result struct {
			B bool `form:"b"`
		}
		if err := Unmarshal(map[string][]string{"b": {value}}, &result); err != nil {
			t.Errorf("%q: unexpected error: %v", value, err)
			continue
		}
		if !result.B {
			t.Errorf("%q: B = false, want true", value)
		}
	}

	for _, value := range falseValues {
		var result struct {
			B bool `form:"b"`
		}
		if err := Unmarshal(map[string][]string{"b": {value}}, &result); err != nil {
			t.Errorf("%q: unexpected error: %v", value, err)
			continue
		}
		if result.B {
			t.Errorf("%q: B = true, want false", value)
		}
	}

	var result struct {
		B bool `form:"b"`
	}
	if err := Unmarshal(map[string][]string{"b": {"maybe"}}, &result); err == nil {
		t.Error("expected error for invalid bool")
	}
}

func TestDecode_EmptyNumericDoesNotError(t *testing.T) {
	type S struct {
		Age  int     `form:"age"`
		Rate float64 `form:"rate"`
	}

	var result S
	if err := Unmarshal(map[string][]string{"age": {""}, "rate": {""}}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Age != 0 || result.Rate != 0 {
		t.Errorf("got Age=%d Rate=%v, want zeros", result.Age, result.Rate)
	}
}

func TestEncode_NilPointerMarshalerDoesNotPanic(t *testing.T) {
	type S struct {
		X *nilPointerMarshaler `form:"x"`
	}

	result, err := Marshal(S{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("result = %v, want empty", result)
	}
}

type nilPointerMarshaler struct {
	Value string
}

func (m *nilPointerMarshaler) MarshalValues() ([]string, error) {
	return []string{m.Value}, nil
}

func TestEncode_ZeroTimeWithoutOmitempty(t *testing.T) {
	zero := time.Time{}
	input := struct {
		Value time.Time  `form:"value"`
		Ptr   *time.Time `form:"ptr"`
	}{Value: zero, Ptr: &zero}

	result, err := Marshal(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := zero.Format(time.RFC3339)
	for _, key := range []string{"value", "ptr"} {
		if values := result[key]; len(values) != 1 || values[0] != want {
			t.Errorf("%s: got %v, want [%s]", key, values, want)
		}
	}
}

func TestEncode_ExplicitZeroWithDefaultNotOmitted(t *testing.T) {
	type S struct {
		Age int `form:"age,omitempty,default=18"`
	}

	result, err := Marshal(S{Age: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values := result["age"]; len(values) != 1 || values[0] != "0" {
		t.Errorf("age: got %v, want [0]", values)
	}
}

func TestDecode_PresentEmptyDoesNotUseDefault(t *testing.T) {
	type S struct {
		Name string `form:"name,default=guest"`
	}

	var result S
	if err := Unmarshal(map[string][]string{"name": {""}}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "" {
		t.Errorf("Name = %q, want empty", result.Name)
	}
}

func TestDecode_ErrorAggregation(t *testing.T) {
	type S struct {
		A int    `form:"a"`
		B int    `form:"b"`
		C string `form:"c"`
	}

	var result S
	err := Unmarshal(map[string][]string{
		"a": {"x"},
		"b": {"y"},
		"c": {"ok"},
	}, &result)
	if err == nil {
		t.Fatal("expected error")
	}

	de, ok := err.(DecodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if len(de) != 2 {
		t.Fatalf("error count = %d, want 2 (%v)", len(de), de)
	}
	if _, ok := de["a"]; !ok {
		t.Errorf("missing error for a: %v", de)
	}
	if _, ok := de["b"]; !ok {
		t.Errorf("missing error for b: %v", de)
	}
	if result.C != "ok" {
		t.Errorf("C = %q, want ok (decoding should continue)", result.C)
	}

	if !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("errors.Is(err, strconv.ErrSyntax) = false: %v", err)
	}

	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Errorf("errors.As(err, *strconv.NumError) = false: %v", err)
	}
}

func TestEncode_ErrorAggregation(t *testing.T) {
	type S struct {
		Bad  chan int `form:"bad"`
		Good string   `form:"good"`
	}

	values, err := Marshal(S{Good: "ok"})
	if err == nil {
		t.Fatal("expected error")
	}
	ee, ok := err.(EncodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want EncodeErrors", err)
	}
	if _, ok := ee["bad"]; !ok {
		t.Errorf("missing error for bad: %v", ee)
	}
	if got := values["good"]; len(got) != 1 || got[0] != "ok" {
		t.Errorf("good: got %v, want [ok] (partial encoding)", got)
	}
}

func TestDecode_UnknownKeysIgnored(t *testing.T) {
	type S struct {
		Known string `form:"known"`
	}

	var result S
	if err := Unmarshal(map[string][]string{
		"known":   {"value"},
		"unknown": {"ignored"},
	}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Known != "value" {
		t.Errorf("Known = %q, want value", result.Known)
	}
}

func TestDecode_InvalidPathAggregated(t *testing.T) {
	type S struct {
		A []string `form:"a"`
	}

	var result S
	err := Unmarshal(map[string][]string{"a[0": {"x"}}, &result)
	if err == nil {
		t.Fatal("expected error")
	}
	de, ok := err.(DecodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if _, ok := de["a[0"]; !ok {
		t.Errorf("missing path error: %v", de)
	}
}

func TestDecode_MaxArraySize(t *testing.T) {
	type S struct {
		Items []string `form:"items"`
	}

	var result S
	err := NewDecoder(WithMaxArraySize(3)).Decode(map[string][]string{"items[3]": {"x"}}, &result)
	if err == nil {
		t.Fatal("expected error for index at max size")
	}

	var okResult S
	if err := NewDecoder(WithMaxArraySize(3)).Decode(map[string][]string{"items[2]": {"x"}}, &okResult); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"", "", "x"}; !reflect.DeepEqual(okResult.Items, want) {
		t.Errorf("Items = %#v, want %#v", okResult.Items, want)
	}
}

func TestDecode_ByteSliceIndexedRejected(t *testing.T) {
	type S struct {
		Data []byte `form:"data"`
	}

	var result S
	if err := Unmarshal(map[string][]string{"data[0]": {"x"}}, &result); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecode_StructDirectValueRejected(t *testing.T) {
	type Inner struct {
		A string `form:"a"`
	}
	type S struct {
		In Inner `form:"in"`
	}

	var result S
	err := Unmarshal(map[string][]string{"in": {"x"}}, &result)
	if err == nil {
		t.Fatal("expected error")
	}
	de, ok := err.(DecodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if _, ok := de["in"]; !ok {
		t.Errorf("missing error for in: %v", de)
	}
}

func TestEncode_CyclicPointerReturnsError(t *testing.T) {
	type node struct {
		Next *node `form:"next"`
	}

	n := &node{}
	n.Next = n

	_, err := Marshal(n)
	if err == nil {
		t.Fatal("expected depth error for cyclic value")
	}
	if _, ok := err.(EncodeErrors); !ok {
		t.Fatalf("error type = %T, want EncodeErrors", err)
	}
}

func TestDecode_PathDepthLimit(t *testing.T) {
	key := "a"
	for i := 0; i < defaultMaxPathDepth; i++ {
		key += ".a"
	}

	var result struct {
		A string `form:"a"`
	}
	err := Unmarshal(map[string][]string{key: {"x"}}, &result)
	if err == nil {
		t.Fatal("expected error for overly deep path")
	}
}

func TestEncode_MapKeySyntaxRejected(t *testing.T) {
	type S struct {
		M map[string]string `form:"m"`
		X string            `form:"x"`
	}

	values, err := Marshal(S{M: map[string]string{"a]b": "v"}, X: "ok"})
	if err == nil {
		t.Fatal("expected error for map key containing ']'")
	}
	if got := values["x"]; len(got) != 1 || got[0] != "ok" {
		t.Errorf("x = %v, want [ok] (partial encoding)", got)
	}
}

func TestDuplicateTagNames(t *testing.T) {
	type S struct {
		A string `form:"x"`
		B string `form:"x"`
	}

	encoded, err := Marshal(S{A: "1", B: "2"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["x"]; !reflect.DeepEqual(got, []string{"1", "2"}) {
		t.Errorf("x = %v, want [1 2]", got)
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"x": {"v"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.A != "v" || decoded.B != "v" {
		t.Errorf("decoded = %#v, want both fields set to v", decoded)
	}
}

func TestConcurrentUse(t *testing.T) {
	type Address struct {
		City string `form:"city"`
	}
	type S struct {
		Name    string  `form:"name"`
		Items   []int   `form:"items"`
		Address Address `form:"address"`
	}

	encoder := NewEncoder()
	decoder := NewDecoder()

	const goroutines = 8
	const iterations = 100

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				input := S{
					Name:    fmt.Sprintf("n%d-%d", g, i),
					Items:   []int{g, i},
					Address: Address{City: "c"},
				}

				values, err := encoder.Encode(input)
				if err != nil {
					errCh <- err
					return
				}

				var decoded S
				if err := decoder.Decode(values, &decoded); err != nil {
					errCh <- err
					return
				}
				if decoded.Name != input.Name || !reflect.DeepEqual(decoded.Items, input.Items) || decoded.Address.City != "c" {
					errCh <- fmt.Errorf("decoded = %#v, want %#v", decoded, input)
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

func TestURLValuesCompatibility(t *testing.T) {
	type S struct {
		Name string `form:"name"`
	}

	values, err := Marshal(S{Name: "x"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var urlValues url.Values = values
	if got := urlValues.Get("name"); got != "x" {
		t.Errorf("urlValues.Get(name) = %q, want x", got)
	}

	var decoded S
	if err := Unmarshal(url.Values{"name": {"y"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Name != "y" {
		t.Errorf("Name = %q, want y", decoded.Name)
	}
}

func TestTimeLayoutOption(t *testing.T) {
	type S struct {
		Date time.Time `form:"date"`
	}

	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	encoded, err := NewEncoder(WithTimeLayout("2006-01-02")).Encode(S{Date: date})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values := encoded["date"]; len(values) != 1 || values[0] != "2024-01-15" {
		t.Errorf("date: got %v, want [2024-01-15]", values)
	}

	var decoded S
	if err := NewDecoder(WithTimeLayout("2006-01-02")).Decode(map[string][]string{"date": {"2024-01-15"}}, &decoded); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !decoded.Date.Equal(date) {
		t.Errorf("Date = %v, want %v", decoded.Date, date)
	}
}
