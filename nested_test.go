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

type testAddress struct {
	City string `form:"city"`
	Zip  int    `form:"zip"`
}

type testUser struct {
	Name    string      `form:"name"`
	Address testAddress `form:"address"`
}

func TestNestedStructRoundTrip(t *testing.T) {
	original := testUser{
		Name:    "alice",
		Address: testAddress{City: "shanghai", Zip: 200000},
	}

	encoded, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["name"]; !reflect.DeepEqual(got, []string{"alice"}) {
		t.Errorf("name = %v, want [alice]", got)
	}
	if got := encoded["address.city"]; !reflect.DeepEqual(got, []string{"shanghai"}) {
		t.Errorf("address.city = %v, want [shanghai]", got)
	}
	if got := encoded["address.zip"]; !reflect.DeepEqual(got, []string{"200000"}) {
		t.Errorf("address.zip = %v, want [200000]", got)
	}

	var decoded testUser
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Errorf("round trip mismatch: got %#v, want %#v", decoded, original)
	}
}

func TestNestedPointerStruct(t *testing.T) {
	type S struct {
		Address *testAddress `form:"address"`
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"address.city": {"beijing"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Address == nil {
		t.Fatal("Address should be allocated")
	}
	if decoded.Address.City != "beijing" || decoded.Address.Zip != 0 {
		t.Errorf("Address = %#v, want {beijing 0}", *decoded.Address)
	}

	var missing S
	if err := Unmarshal(map[string][]string{}, &missing); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if missing.Address != nil {
		t.Errorf("Address = %#v, want nil", missing.Address)
	}

	encoded, err := Marshal(missing)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if len(encoded) != 0 {
		t.Errorf("encoded = %v, want empty", encoded)
	}
}

type TestBase struct {
	ID int `form:"id"`
}

type testEmbedded struct {
	TestBase
	Name string `form:"name"`
}

func TestEmbeddedStruct(t *testing.T) {
	var decoded testEmbedded
	if err := Unmarshal(map[string][]string{
		"id":   {"7"},
		"name": {"alice"},
	}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.ID != 7 || decoded.Name != "alice" {
		t.Errorf("decoded = %#v, want ID=7 Name=alice", decoded)
	}

	var separate testEmbedded
	if err := Unmarshal(map[string][]string{"TestBase.id": {"9"}}, &separate); err != nil {
		t.Fatalf("Unmarshal separate error: %v", err)
	}
	if separate.ID != 9 {
		t.Errorf("ID = %d, want 9", separate.ID)
	}

	encoded, err := Marshal(testEmbedded{TestBase: TestBase{ID: 7}, Name: "alice"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["id"]; !reflect.DeepEqual(got, []string{"7"}) {
		t.Errorf("id = %v, want [7]", got)
	}
	if got := encoded["name"]; !reflect.DeepEqual(got, []string{"alice"}) {
		t.Errorf("name = %v, want [alice]", got)
	}
}

func TestEmbeddedPointerStruct(t *testing.T) {
	type S struct {
		*TestBase
		Name string `form:"name"`
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"id": {"3"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.TestBase == nil || decoded.ID != 3 {
		t.Errorf("decoded = %#v, want embedded ID=3", decoded)
	}

	encoded, err := Marshal(S{TestBase: &TestBase{ID: 3}, Name: "x"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["id"]; !reflect.DeepEqual(got, []string{"3"}) {
		t.Errorf("id = %v, want [3]", got)
	}
}

type OnlyUnmarshalEmbedded struct {
	ID int `form:"id"`
}

func (*OnlyUnmarshalEmbedded) UnmarshalValues([]string) error { return nil }

type EmbeddedCustom struct {
	V string
}

func (e EmbeddedCustom) MarshalValues() ([]string, error) {
	return []string{e.V}, nil
}

func (e *EmbeddedCustom) UnmarshalValues(values []string) error {
	if len(values) > 0 {
		e.V = values[0]
	}
	return nil
}

func TestEmbeddedCustomTypesUseNamedNamespace(t *testing.T) {
	t.Run("unmarshaler only", func(t *testing.T) {
		type S struct {
			OnlyUnmarshalEmbedded
			Name string `form:"name"`
		}

		input := S{OnlyUnmarshalEmbedded: OnlyUnmarshalEmbedded{ID: 7}, Name: "n"}
		encoded, err := Marshal(input)
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		if got := encoded["OnlyUnmarshalEmbedded.id"]; !reflect.DeepEqual(got, []string{"7"}) {
			t.Errorf("OnlyUnmarshalEmbedded.id = %v, want [7]", got)
		}

		var decoded S
		if err := Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.ID != 7 || decoded.Name != "n" {
			t.Errorf("decoded = %#v, want ID=7 Name=n", decoded)
		}
	})

	t.Run("marshaler and unmarshaler", func(t *testing.T) {
		type S struct {
			EmbeddedCustom
		}

		encoded, err := Marshal(S{EmbeddedCustom: EmbeddedCustom{V: "x"}})
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		if got := encoded["EmbeddedCustom"]; !reflect.DeepEqual(got, []string{"x"}) {
			t.Errorf("EmbeddedCustom = %v, want [x]", got)
		}

		var decoded S
		if err := Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.V != "x" {
			t.Errorf("V = %q, want x", decoded.V)
		}
	})
}

type hiddenBase struct {
	ID int `form:"id"`
}

type hiddenHolder struct {
	hiddenBase
	Name string `form:"name"`
}

func TestUnexportedEmbeddedStruct(t *testing.T) {
	var decoded hiddenHolder
	if err := Unmarshal(map[string][]string{
		"id":   {"9"},
		"name": {"alice"},
	}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.ID != 9 || decoded.Name != "alice" {
		t.Errorf("decoded = %#v, want ID=9 Name=alice", decoded)
	}

	encoded, err := Marshal(hiddenHolder{hiddenBase: hiddenBase{ID: 9}, Name: "alice"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["id"]; !reflect.DeepEqual(got, []string{"9"}) {
		t.Errorf("id = %v, want [9]", got)
	}
	if got := encoded["name"]; !reflect.DeepEqual(got, []string{"alice"}) {
		t.Errorf("name = %v, want [alice]", got)
	}
}

func TestDecode_SliceIndexed(t *testing.T) {
	type S struct {
		Items []string `form:"items"`
	}

	var result S
	if err := Unmarshal(map[string][]string{
		"items[1]": {"b"},
		"items[3]": {"d"},
	}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	want := []string{"", "b", "", "d"}
	if !reflect.DeepEqual(result.Items, want) {
		t.Errorf("Items = %#v, want %#v", result.Items, want)
	}
}

func TestDecode_SliceMixedDirectAndIndexed(t *testing.T) {
	type S struct {
		Items []string `form:"items"`
	}

	var result S
	if err := Unmarshal(map[string][]string{
		"items":    {"a"},
		"items[2]": {"c"},
	}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	want := []string{"a", "", "c"}
	if !reflect.DeepEqual(result.Items, want) {
		t.Errorf("Items = %#v, want %#v", result.Items, want)
	}
}

func TestEncode_SliceOfStructs(t *testing.T) {
	type Item struct {
		Name string `form:"name"`
	}
	type S struct {
		Items []Item `form:"items"`
	}

	encoded, err := Marshal(S{Items: []Item{{Name: "a"}, {Name: "b"}}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["items[0].name"]; !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("items[0].name = %v, want [a]", got)
	}
	if got := encoded["items[1].name"]; !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("items[1].name = %v, want [b]", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded.Items, []Item{{Name: "a"}, {Name: "b"}}) {
		t.Errorf("Items = %#v", decoded.Items)
	}
}

func TestSliceOfPointers(t *testing.T) {
	type S struct {
		Items []*int `form:"items"`
	}

	a, c := 1, 3
	encoded, err := Marshal(S{Items: []*int{&a, nil, &c}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["items[0]"]; !reflect.DeepEqual(got, []string{"1"}) {
		t.Errorf("items[0] = %v, want [1]", got)
	}
	if _, ok := encoded["items[1]"]; ok {
		t.Errorf("items[1] should be absent for nil pointer")
	}
	if got := encoded["items[2]"]; !reflect.DeepEqual(got, []string{"3"}) {
		t.Errorf("items[2] = %v, want [3]", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(decoded.Items) != 3 || decoded.Items[0] == nil || *decoded.Items[0] != 1 {
		t.Fatalf("Items = %#v", decoded.Items)
	}
	if decoded.Items[1] != nil {
		t.Errorf("Items[1] = %v, want nil", decoded.Items[1])
	}
	if decoded.Items[2] == nil || *decoded.Items[2] != 3 {
		t.Errorf("Items[2] = %v, want 3", decoded.Items[2])
	}
}

func TestSliceOfTime(t *testing.T) {
	type S struct {
		Times []time.Time `form:"times"`
	}

	t1 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 2, 20, 11, 0, 0, 0, time.UTC)

	encoded, err := Marshal(S{Times: []time.Time{t1, t2}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["times[0]"]; len(got) != 1 || got[0] != t1.Format(time.RFC3339) {
		t.Errorf("times[0] = %v, want [%s]", got, t1.Format(time.RFC3339))
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(decoded.Times) != 2 || !decoded.Times[0].Equal(t1) || !decoded.Times[1].Equal(t2) {
		t.Errorf("Times = %v, want [%v %v]", decoded.Times, t1, t2)
	}
}

func TestNestedSlices(t *testing.T) {
	type S struct {
		Matrix [][]string `form:"matrix"`
	}

	encoded, err := Marshal(S{Matrix: [][]string{{"a", "b"}, {"c"}}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["matrix[0][0]"]; !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("matrix[0][0] = %v, want [a]", got)
	}
	if got := encoded["matrix[0][1]"]; !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("matrix[0][1] = %v, want [b]", got)
	}
	if got := encoded["matrix[1][0]"]; !reflect.DeepEqual(got, []string{"c"}) {
		t.Errorf("matrix[1][0] = %v, want [c]", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded.Matrix, [][]string{{"a", "b"}, {"c"}}) {
		t.Errorf("Matrix = %#v", decoded.Matrix)
	}
}

func TestNestedSliceInsideStructSlice(t *testing.T) {
	type Item struct {
		Tags []string `form:"tags"`
	}
	type S struct {
		Items []Item `form:"items"`
	}

	input := S{Items: []Item{{Tags: []string{"a", "b"}}, {Tags: []string{"c"}}}}
	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["items[0].tags[0]"]; !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("items[0].tags[0] = %v, want [a]", got)
	}
	if got := encoded["items[0].tags[1]"]; !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("items[0].tags[1] = %v, want [b]", got)
	}
	if got := encoded["items[1].tags[0]"]; !reflect.DeepEqual(got, []string{"c"}) {
		t.Errorf("items[1].tags[0] = %v, want [c]", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded.Items, input.Items) {
		t.Errorf("Items = %#v, want %#v", decoded.Items, input.Items)
	}
}

func TestMapValueSlice(t *testing.T) {
	type S struct {
		M map[string][]string `form:"m"`
	}

	input := S{M: map[string][]string{"k": {"a", "b"}}}
	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["m[k][0]"]; !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("m[k][0] = %v, want [a]", got)
	}
	if got := encoded["m[k][1]"]; !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("m[k][1] = %v, want [b]", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded.M, input.M) {
		t.Errorf("M = %#v, want %#v", decoded.M, input.M)
	}
}

func TestSliceInTopLevelNestedStructUsesRepeatedValues(t *testing.T) {
	type Inner struct {
		Tags []string `form:"tags"`
	}
	type S struct {
		Inner Inner `form:"inner"`
	}

	encoded, err := Marshal(S{Inner: Inner{Tags: []string{"a", "b"}}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["inner.tags"]; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("inner.tags = %v, want [a b]", got)
	}
	if _, ok := encoded["inner.tags[0]"]; ok {
		t.Error("inner.tags[0] should not be present for a top-level nested struct")
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded.Inner.Tags, []string{"a", "b"}) {
		t.Errorf("Tags = %v, want [a b]", decoded.Inner.Tags)
	}
}

func TestComplexNestedRoundTrip(t *testing.T) {
	type Item struct {
		Name string            `form:"name"`
		Tags []string          `form:"tags"`
		Meta map[string]string `form:"meta"`
	}
	type S struct {
		Items  []Item          `form:"items"`
		Lookup map[string]Item `form:"lookup"`
		Matrix [][]int         `form:"matrix"`
	}

	input := S{
		Items: []Item{{
			Name: "a",
			Tags: []string{"t1", "t2"},
			Meta: map[string]string{"k": "v"},
		}},
		Lookup: map[string]Item{
			"x": {Name: "b", Tags: []string{"t3"}, Meta: map[string]string{"m": "n"}},
		},
		Matrix: [][]int{{1, 2}, {3}},
	}

	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	expected := map[string]string{
		"items[0].name":     "a",
		"items[0].tags[0]":  "t1",
		"items[0].tags[1]":  "t2",
		"items[0].meta[k]":  "v",
		"lookup[x].name":    "b",
		"lookup[x].tags[0]": "t3",
		"lookup[x].meta[m]": "n",
		"matrix[0][0]":      "1",
		"matrix[0][1]":      "2",
		"matrix[1][0]":      "3",
	}
	for key, want := range expected {
		if got := encoded[key]; len(got) != 1 || got[0] != want {
			t.Errorf("%s = %v, want [%s]", key, got, want)
		}
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded, input) {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}
}

func TestArrayDecode(t *testing.T) {
	type S struct {
		Nums [3]int `form:"nums"`
	}

	var indexed S
	if err := Unmarshal(map[string][]string{
		"nums[0]": {"1"},
		"nums[2]": {"3"},
	}, &indexed); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if indexed.Nums != [3]int{1, 0, 3} {
		t.Errorf("Nums = %v, want [1 0 3]", indexed.Nums)
	}

	var direct S
	if err := Unmarshal(map[string][]string{"nums": {"1", "2", "3"}}, &direct); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if direct.Nums != [3]int{1, 2, 3} {
		t.Errorf("Nums = %v, want [1 2 3]", direct.Nums)
	}

	var overflow S
	if err := Unmarshal(map[string][]string{"nums[3]": {"4"}}, &overflow); err == nil {
		t.Error("expected error for array index out of range")
	}

	var directOverflow S
	if err := Unmarshal(map[string][]string{"nums": {"1", "2", "3", "4"}}, &directOverflow); err == nil {
		t.Error("expected error for too many array values")
	}
}

func TestMapDecode(t *testing.T) {
	type S struct {
		M map[string]string `form:"m"`
	}

	var result S
	if err := Unmarshal(map[string][]string{
		"m[a]": {"1"},
		"m[b]": {"2"},
	}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	want := map[string]string{"a": "1", "b": "2"}
	if !reflect.DeepEqual(result.M, want) {
		t.Errorf("M = %#v, want %#v", result.M, want)
	}

	type SInt struct {
		M map[string]int `form:"m"`
	}
	var invalid SInt
	err := Unmarshal(map[string][]string{"m[a]": {"x"}}, &invalid)
	if err == nil {
		t.Fatal("expected conversion error")
	}
	de, ok := err.(DecodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if _, ok := de["m[a]"]; !ok {
		t.Errorf("missing map path error: %v", de)
	}
}

func TestMapIntKeys(t *testing.T) {
	type S struct {
		M map[int]string `form:"m"`
	}

	var result S
	if err := Unmarshal(map[string][]string{
		"m[1]": {"a"},
		"m[2]": {"b"},
	}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	want := map[int]string{1: "a", 2: "b"}
	if !reflect.DeepEqual(result.M, want) {
		t.Errorf("M = %#v, want %#v", result.M, want)
	}
}

func TestNestedMap(t *testing.T) {
	type S struct {
		M map[string]map[string]string `form:"m"`
	}

	var result S
	if err := Unmarshal(map[string][]string{
		"m[outer][inner]": {"v"},
	}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	want := map[string]map[string]string{"outer": {"inner": "v"}}
	if !reflect.DeepEqual(result.M, want) {
		t.Errorf("M = %#v, want %#v", result.M, want)
	}

	encoded, err := Marshal(S{M: want})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["m[outer][inner]"]; !reflect.DeepEqual(got, []string{"v"}) {
		t.Errorf("m[outer][inner] = %v, want [v]", got)
	}
}

func TestMapOfStructs(t *testing.T) {
	type Item struct {
		Name string `form:"name"`
	}
	type S struct {
		Items map[string]Item `form:"items"`
	}

	var decoded S
	if err := Unmarshal(map[string][]string{
		"items[a].name": {"alice"},
	}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Items["a"].Name != "alice" {
		t.Errorf("Items = %#v", decoded.Items)
	}

	encoded, err := Marshal(S{Items: map[string]Item{"a": {Name: "alice"}}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["items[a].name"]; !reflect.DeepEqual(got, []string{"alice"}) {
		t.Errorf("items[a].name = %v, want [alice]", got)
	}
}

func TestMapEncodeSortedKeys(t *testing.T) {
	type S struct {
		M map[string]string `form:"m"`
	}

	encoded, err := Marshal(S{M: map[string]string{"b": "2", "a": "1"}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["m[a]"]; !reflect.DeepEqual(got, []string{"1"}) {
		t.Errorf("m[a] = %v", got)
	}
	if got := encoded["m[b]"]; !reflect.DeepEqual(got, []string{"2"}) {
		t.Errorf("m[b] = %v", got)
	}
}

func TestMapUnsupportedKey(t *testing.T) {
	type key struct{ A int }
	type S struct {
		M map[key]string `form:"m"`
	}

	// Encoding a map with a struct key must report an EncodeErrors entry and
	// keep encoding other fields, but constructing the map is enough here.
	_ = S{M: map[key]string{{A: 1}: "v"}}

	values, err := Marshal(struct {
		M map[key]string `form:"m"`
		X string         `form:"x"`
	}{M: map[key]string{{A: 1}: "v"}, X: "ok"})
	if err == nil {
		t.Fatal("expected error for unsupported map key")
	}
	if got := values["x"]; !reflect.DeepEqual(got, []string{"ok"}) {
		t.Errorf("x = %v, want [ok] (partial encoding)", got)
	}
}

func TestDefaultInsideNestedStruct(t *testing.T) {
	type Inner struct {
		Mode string `form:"mode,default=debug"`
	}
	type S struct {
		In Inner `form:"in"`
	}

	var result S
	if err := Unmarshal(map[string][]string{}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result.In.Mode != "debug" {
		t.Errorf("In.Mode = %q, want debug", result.In.Mode)
	}
}

func TestDefaultInsideEmbeddedStruct(t *testing.T) {
	type Base struct {
		Mode string `form:"mode,default=debug"`
	}
	type S struct {
		Base
	}

	var result S
	if err := Unmarshal(map[string][]string{}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result.Mode != "debug" {
		t.Errorf("Mode = %q, want debug", result.Mode)
	}
}

func TestEncode_SliceOfMaps(t *testing.T) {
	type S struct {
		Items []map[string]string `form:"items"`
	}

	encoded, err := Marshal(S{Items: []map[string]string{{"a": "1"}, {"b": "2"}}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["items[0][a]"]; !reflect.DeepEqual(got, []string{"1"}) {
		t.Errorf("items[0][a] = %v, want [1]", got)
	}
	if got := encoded["items[1][b]"]; !reflect.DeepEqual(got, []string{"2"}) {
		t.Errorf("items[1][b] = %v, want [2]", got)
	}
}

func TestDecode_SliceOfStructsPointerElements(t *testing.T) {
	type Item struct {
		Name string `form:"name"`
	}
	type S struct {
		Items []*Item `form:"items"`
	}

	var result S
	if err := Unmarshal(map[string][]string{
		"items[0].name": {"a"},
		"items[1].name": {"b"},
	}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(result.Items) != 2 || result.Items[0] == nil || result.Items[0].Name != "a" {
		t.Fatalf("Items = %#v", result.Items)
	}
	if result.Items[1] == nil || result.Items[1].Name != "b" {
		t.Errorf("Items[1] = %#v", result.Items[1])
	}
}
