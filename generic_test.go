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
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type genericBox[T any] struct {
	Value T `form:"value"`
}

type genericPair[A, B any] struct {
	First  A `form:"first"`
	Second B `form:"second"`
}

type genericOuter[T any] struct {
	Name  string        `form:"name"`
	Inner genericBox[T] `form:"inner"`
}

type genericList[T any] []T

type genericDict[K comparable, V any] map[K]V

type genericNode[T any] struct {
	Value T               `form:"value"`
	Next  *genericNode[T] `form:"next"`
}

type genericWrapped[T any] struct {
	V T
}

func (w genericWrapped[T]) MarshalValues() ([]string, error) {
	return []string{fmt.Sprint(w.V)}, nil
}

func (w *genericWrapped[T]) UnmarshalValues(values []string) error {
	if len(values) == 0 {
		return nil
	}

	var zero T
	switch any(zero).(type) {
	case int:
		n, err := strconv.Atoi(values[0])
		if err != nil {
			return err
		}
		w.V = any(n).(T)
	case string:
		w.V = any(values[0]).(T)
	default:
		return fmt.Errorf("genericWrapped: unsupported type %T", zero)
	}
	return nil
}

type genericBoxAlias[T any] = genericBox[T]

func TestGenericStruct(t *testing.T) {
	encoded, err := Marshal(genericBox[int]{Value: 42})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["value"]; !reflect.DeepEqual(got, []string{"42"}) {
		t.Errorf("value = %v, want [42]", got)
	}

	var decoded genericBox[int]
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value != 42 {
		t.Errorf("Value = %d, want 42", decoded.Value)
	}

	encodedString, err := Marshal(genericBox[string]{Value: "hello"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var decodedString genericBox[string]
	if err := Unmarshal(encodedString, &decodedString); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decodedString.Value != "hello" {
		t.Errorf("Value = %q, want hello", decodedString.Value)
	}
}

func TestGenericMultipleParameters(t *testing.T) {
	input := genericPair[int, string]{First: 1, Second: "two"}
	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["first"]; !reflect.DeepEqual(got, []string{"1"}) {
		t.Errorf("first = %v", got)
	}
	if got := encoded["second"]; !reflect.DeepEqual(got, []string{"two"}) {
		t.Errorf("second = %v", got)
	}

	var decoded genericPair[int, string]
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != input {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}
}

func TestGenericNestedStruct(t *testing.T) {
	input := genericOuter[int]{
		Name:  "outer",
		Inner: genericBox[int]{Value: 7},
	}
	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["inner.value"]; !reflect.DeepEqual(got, []string{"7"}) {
		t.Errorf("inner.value = %v, want [7]", got)
	}

	var decoded genericOuter[int]
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded, input) {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}
}

func TestGenericNamedSlice(t *testing.T) {
	type S struct {
		Values genericList[int] `form:"values"`
	}

	encoded, err := Marshal(S{Values: genericList[int]{1, 2, 3}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["values"]; !reflect.DeepEqual(got, []string{"1", "2", "3"}) {
		t.Errorf("values = %v, want [1 2 3]", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded.Values, genericList[int]{1, 2, 3}) {
		t.Errorf("Values = %v", decoded.Values)
	}
}

func TestGenericNamedMap(t *testing.T) {
	type S struct {
		M genericDict[string, int] `form:"m"`
	}

	encoded, err := Marshal(S{M: genericDict[string, int]{"a": 1, "b": 2}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["m[a]"]; !reflect.DeepEqual(got, []string{"1"}) {
		t.Errorf("m[a] = %v", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	want := genericDict[string, int]{"a": 1, "b": 2}
	if !reflect.DeepEqual(decoded.M, want) {
		t.Errorf("M = %#v, want %#v", decoded.M, want)
	}
}

func TestGenericRecursiveNode(t *testing.T) {
	type S struct {
		Head *genericNode[int] `form:"head"`
	}

	data := map[string][]string{
		"head.value":      {"1"},
		"head.next.value": {"2"},
	}

	var decoded S
	if err := Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Head == nil || decoded.Head.Value != 1 {
		t.Fatalf("Head = %#v", decoded.Head)
	}
	if decoded.Head.Next == nil || decoded.Head.Next.Value != 2 {
		t.Errorf("Head.Next = %#v", decoded.Head.Next)
	}

	encoded, err := Marshal(decoded)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["head.value"]; !reflect.DeepEqual(got, []string{"1"}) {
		t.Errorf("head.value = %v", got)
	}
	if got := encoded["head.next.value"]; !reflect.DeepEqual(got, []string{"2"}) {
		t.Errorf("head.next.value = %v", got)
	}
}

func TestGenericCustomType(t *testing.T) {
	input := genericBox[genericWrapped[int]]{
		Value: genericWrapped[int]{V: 99},
	}

	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["value"]; !reflect.DeepEqual(got, []string{"99"}) {
		t.Errorf("value = %v, want [99]", got)
	}

	var decoded genericBox[genericWrapped[int]]
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value.V != 99 {
		t.Errorf("Value.V = %d, want 99", decoded.Value.V)
	}
}

func TestGenericAlias(t *testing.T) {
	input := genericBoxAlias[int]{Value: 5}
	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["value"]; !reflect.DeepEqual(got, []string{"5"}) {
		t.Errorf("value = %v, want [5]", got)
	}

	var decoded genericBoxAlias[int]
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value != 5 {
		t.Errorf("Value = %d, want 5", decoded.Value)
	}
}

func TestGenericByteSlice(t *testing.T) {
	type S struct {
		Data genericList[byte] `form:"data"`
	}

	encoded, err := Marshal(S{Data: genericList[byte]("hello")})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["data"]; !reflect.DeepEqual(got, []string{"hello"}) {
		t.Errorf("data = %v, want [hello]", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if string(decoded.Data) != "hello" {
		t.Errorf("Data = %q, want hello", decoded.Data)
	}
}

func TestGenericTimeAndPointer(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	input := genericPair[time.Time, *int]{First: now, Second: intPtr(7)}

	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var decoded genericPair[time.Time, *int]
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !decoded.First.Equal(now) {
		t.Errorf("First = %v, want %v", decoded.First, now)
	}
	if decoded.Second == nil || *decoded.Second != 7 {
		t.Errorf("Second = %v, want 7", decoded.Second)
	}
}

func TestGenericAnyField(t *testing.T) {
	// Encoding an interface field unwraps the dynamic value.
	encoded, err := Marshal(genericBox[any]{Value: "hello"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["value"]; !reflect.DeepEqual(got, []string{"hello"}) {
		t.Errorf("value = %v, want [hello]", got)
	}

	// A nil interface receives the first value as a string.
	var decoded genericBox[any]
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value != "hello" {
		t.Errorf("Value = %#v, want hello", decoded.Value)
	}
}

func TestGenericCacheIsolation(t *testing.T) {
	for i := 0; i < 10; i++ {
		intEncoded, err := Marshal(genericBox[int]{Value: i})
		if err != nil {
			t.Fatalf("Marshal int error: %v", err)
		}
		if got := intEncoded["value"]; len(got) != 1 || got[0] != strconv.Itoa(i) {
			t.Fatalf("int value = %v, want [%d]", got, i)
		}

		stringValue := "s" + strconv.Itoa(i)
		stringEncoded, err := Marshal(genericBox[string]{Value: stringValue})
		if err != nil {
			t.Fatalf("Marshal string error: %v", err)
		}
		if got := stringEncoded["value"]; len(got) != 1 || got[0] != stringValue {
			t.Fatalf("string value = %v, want [%s]", got, stringValue)
		}

		var decodedInt genericBox[int]
		if err := Unmarshal(intEncoded, &decodedInt); err != nil {
			t.Fatalf("Unmarshal int error: %v", err)
		}
		if decodedInt.Value != i {
			t.Fatalf("decoded int = %d, want %d", decodedInt.Value, i)
		}
	}
}

func intPtr(v int) *int { return &v }
