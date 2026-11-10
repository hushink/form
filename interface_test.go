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
)

type testValuer interface {
	Value() int
}

type testInt int

func (v testInt) Value() int { return int(v) }

type testStringer interface {
	String() string
}

func TestDecodeInterfaceNil(t *testing.T) {
	type S struct {
		Value any `form:"value"`
	}

	var result S
	if err := Unmarshal(map[string][]string{"value": {"hello"}}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result.Value != "hello" {
		t.Errorf("Value = %#v, want hello", result.Value)
	}

	var multi S
	if err := Unmarshal(map[string][]string{"value": {"a", "b"}}, &multi); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if multi.Value != "a" {
		t.Errorf("Value = %#v, want a (first value)", multi.Value)
	}
}

func TestDecodeInterfacePreservesDynamicType(t *testing.T) {
	type S struct {
		Value any `form:"value"`
	}

	original := S{Value: 7}
	if err := Unmarshal(map[string][]string{"value": {"42"}}, &original); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if value, ok := original.Value.(int); !ok || value != 42 {
		t.Errorf("Value = %#v, want int(42)", original.Value)
	}

	pointer := new(int)
	withPointer := S{Value: pointer}
	if err := Unmarshal(map[string][]string{"value": {"99"}}, &withPointer); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	value, ok := withPointer.Value.(*int)
	if !ok || value == nil || *value != 99 {
		t.Errorf("Value = %#v, want *int(99)", withPointer.Value)
	}
}

func TestDecodeNonEmptyInterface(t *testing.T) {
	type S struct {
		Value testValuer `form:"value"`
	}

	original := S{Value: testInt(1)}
	if err := Unmarshal(map[string][]string{"value": {"42"}}, &original); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if got := original.Value.Value(); got != 42 {
		t.Errorf("Value.Value() = %d, want 42", got)
	}

	var invalid S
	err := Unmarshal(map[string][]string{"value": {"hello"}}, &invalid)
	if err == nil {
		t.Fatal("expected error when string does not implement the interface")
	}

	type Stringer struct {
		Value testStringer `form:"value"`
	}
	var stringer Stringer
	if err := Unmarshal(map[string][]string{"value": {"hello"}}, &stringer); err == nil {
		t.Error("expected error for nil interface not assignable from string")
	}
}

func TestDecodeSliceOfInterfaces(t *testing.T) {
	type S struct {
		Values []any `form:"values"`
	}

	var result S
	if err := Unmarshal(map[string][]string{"values": {"a", "b"}}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	want := []any{"a", "b"}
	if !reflect.DeepEqual(result.Values, want) {
		t.Errorf("Values = %#v, want %#v", result.Values, want)
	}
}

func TestDecodeMapOfInterfaces(t *testing.T) {
	type S struct {
		M map[string]any `form:"m"`
	}

	var result S
	if err := Unmarshal(map[string][]string{"m[k]": {"v"}}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result.M["k"] != "v" {
		t.Errorf("M = %#v, want map[k:v]", result.M)
	}
}

func TestDecodeInterfaceNestedFieldsRequiresConcreteType(t *testing.T) {
	type S struct {
		Value any `form:"value"`
	}

	var result S
	err := Unmarshal(map[string][]string{"value.x": {"1"}}, &result)
	if err == nil {
		t.Fatal("expected error for nested fields into a nil interface")
	}
	if _, ok := err.(DecodeErrors); !ok {
		t.Errorf("error type = %T, want DecodeErrors", err)
	}
}

func TestDecodeInterfaceWithConcreteStruct(t *testing.T) {
	type Inner struct {
		Name string `form:"name"`
	}
	type S struct {
		Value any `form:"value"`
	}

	result := S{Value: Inner{Name: "old"}}
	if err := Unmarshal(map[string][]string{"value.name": {"new"}}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	inner, ok := result.Value.(Inner)
	if !ok || inner.Name != "new" {
		t.Errorf("Value = %#v, want Inner{Name:new}", result.Value)
	}
}
