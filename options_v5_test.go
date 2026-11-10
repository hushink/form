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
	"reflect"
	"testing"
)

func TestRejectDuplicateTags(t *testing.T) {
	type S struct {
		A string `form:"x"`
		B string `form:"x"`
	}

	values, err := NewEncoder(WithRejectDuplicateTags(true)).Encode(S{A: "1", B: "2"})
	if err == nil {
		t.Fatal("expected duplicate tag error")
	}
	if !errors.Is(err, ErrDuplicateTag) {
		t.Errorf("errors.Is(err, ErrDuplicateTag) = false: %v", err)
	}
	if got := values["x"]; !reflect.DeepEqual(got, []string{"1", "2"}) {
		t.Errorf("x = %v, want [1 2]", got)
	}

	var decoded S
	err = NewDecoder(WithRejectDuplicateTags(true)).Decode(map[string][]string{"x": {"v"}}, &decoded)
	if err == nil {
		t.Fatal("expected duplicate tag error")
	}
	if !errors.Is(err, ErrDuplicateTag) {
		t.Errorf("errors.Is(err, ErrDuplicateTag) = false: %v", err)
	}

	type Inner struct {
		A string `form:"y"`
		B string `form:"y"`
	}
	type Outer struct {
		In Inner `form:"in"`
	}
	var outer Outer
	err = NewDecoder(WithRejectDuplicateTags(true)).Decode(map[string][]string{"in.y": {"v"}}, &outer)
	if err == nil {
		t.Fatal("expected nested duplicate tag error")
	}
	de, ok := err.(DecodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if _, ok := de["in"]; !ok {
		t.Errorf("missing nested duplicate error: %v", de)
	}
}

func TestAppendSlices(t *testing.T) {
	type S struct {
		Items []int `form:"items"`
	}

	appendDecoder := NewDecoder(WithAppendSlices(true))

	var appended S
	appended.Items = []int{9}
	if err := appendDecoder.Decode(map[string][]string{"items": {"1", "2"}}, &appended); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !reflect.DeepEqual(appended.Items, []int{9, 1, 2}) {
		t.Errorf("Items = %v, want [9 1 2]", appended.Items)
	}

	var replaced S
	replaced.Items = []int{9}
	if err := Unmarshal(map[string][]string{"items": {"1", "2"}}, &replaced); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(replaced.Items, []int{1, 2}) {
		t.Errorf("Items = %v, want [1 2] (replace is the default)", replaced.Items)
	}

	var indexedAppend S
	indexedAppend.Items = []int{9}
	if err := appendDecoder.Decode(map[string][]string{"items[1]": {"20"}}, &indexedAppend); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !reflect.DeepEqual(indexedAppend.Items, []int{9, 20}) {
		t.Errorf("Items = %v, want [9 20]", indexedAppend.Items)
	}

	var indexedReplace S
	indexedReplace.Items = []int{9}
	if err := Unmarshal(map[string][]string{"items[1]": {"20"}}, &indexedReplace); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(indexedReplace.Items, []int{0, 20}) {
		t.Errorf("Items = %v, want [0 20]", indexedReplace.Items)
	}

	var mixed S
	mixed.Items = []int{9}
	if err := appendDecoder.Decode(map[string][]string{
		"items":    {"1"},
		"items[2]": {"3"},
	}, &mixed); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !reflect.DeepEqual(mixed.Items, []int{9, 1, 3}) {
		t.Errorf("Items = %v, want [9 1 3]", mixed.Items)
	}
}

type topLevelMarshaler struct {
	Name string `form:"name"`
}

func (topLevelMarshaler) MarshalValues() ([]string, error) {
	return []string{"custom"}, nil
}

func TestTopLevelMarshalerOption(t *testing.T) {
	input := topLevelMarshaler{Name: "n"}

	values, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := values["name"]; !reflect.DeepEqual(got, []string{"n"}) {
		t.Errorf("name = %v, want [n] (default traverses fields)", got)
	}
	if _, ok := values[""]; ok {
		t.Error("root key should not be used by default")
	}

	values, err = NewEncoder(WithTopLevelMarshaler(true)).Encode(input)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := values[""]; !reflect.DeepEqual(got, []string{"custom"}) {
		t.Errorf("root = %v, want [custom]", got)
	}
	if _, ok := values["name"]; ok {
		t.Error("name should not be encoded when the top-level Marshaler is used")
	}
}
