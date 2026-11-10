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

func TestTopLevelScalar(t *testing.T) {
	encoded, err := Marshal(42)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded[""]; len(got) != 1 || got[0] != "42" {
		t.Errorf("root value = %v, want [42]", got)
	}

	var decoded int
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != 42 {
		t.Errorf("decoded = %d, want 42", decoded)
	}
}

func TestTopLevelStringAndBool(t *testing.T) {
	encoded, err := Marshal("hello")
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var s string
	if err := Unmarshal(encoded, &s); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if s != "hello" {
		t.Errorf("s = %q, want hello", s)
	}

	encodedBool, err := Marshal(true)
	if err != nil {
		t.Fatalf("Marshal bool error: %v", err)
	}
	var b bool
	if err := Unmarshal(encodedBool, &b); err != nil {
		t.Fatalf("Unmarshal bool error: %v", err)
	}
	if !b {
		t.Error("b = false, want true")
	}
}

func TestTopLevelMap(t *testing.T) {
	input := map[string]int{"a": 1, "b": 2}

	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["[a]"]; !reflect.DeepEqual(got, []string{"1"}) {
		t.Errorf("[a] = %v, want [1]", got)
	}
	if got := encoded["[b]"]; !reflect.DeepEqual(got, []string{"2"}) {
		t.Errorf("[b] = %v, want [2]", got)
	}

	var decoded map[string]int
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded, input) {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}
}

func TestTopLevelSlice(t *testing.T) {
	input := []string{"a", "b"}

	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded[""]; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("root values = %v, want [a b]", got)
	}

	var decoded []string
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded, input) {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}

	var indexed []string
	if err := Unmarshal(map[string][]string{
		"[0]": {"a"},
		"[1]": {"b"},
	}, &indexed); err != nil {
		t.Fatalf("Unmarshal indexed error: %v", err)
	}
	if !reflect.DeepEqual(indexed, input) {
		t.Errorf("indexed = %#v, want %#v", indexed, input)
	}

	var sparse []int
	if err := Unmarshal(map[string][]string{"[2]": {"3"}}, &sparse); err != nil {
		t.Fatalf("Unmarshal sparse error: %v", err)
	}
	if !reflect.DeepEqual(sparse, []int{0, 0, 3}) {
		t.Errorf("sparse = %#v, want [0 0 3]", sparse)
	}
}

func TestTopLevelStructSlice(t *testing.T) {
	type Item struct {
		Name string `form:"name"`
	}
	input := []Item{{Name: "a"}, {Name: "b"}}

	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["[0].name"]; !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("[0].name = %v, want [a]", got)
	}
	if got := encoded["[1].name"]; !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("[1].name = %v, want [b]", got)
	}

	var decoded []Item
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded, input) {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}
}

func TestTopLevelMapOfStructs(t *testing.T) {
	type Item struct {
		Name string `form:"name"`
	}
	input := map[string]Item{"a": {Name: "x"}}

	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["[a].name"]; !reflect.DeepEqual(got, []string{"x"}) {
		t.Errorf("[a].name = %v, want [x]", got)
	}

	var decoded map[string]Item
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !reflect.DeepEqual(decoded, input) {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}
}

func TestTopLevelTimeAndBytes(t *testing.T) {
	date := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	encoded, err := Marshal(date)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded[""]; len(got) != 1 || got[0] != date.Format(time.RFC3339) {
		t.Errorf("root time = %v, want [%s]", got, date.Format(time.RFC3339))
	}

	var decoded time.Time
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !decoded.Equal(date) {
		t.Errorf("decoded = %v, want %v", decoded, date)
	}

	encodedBytes, err := Marshal([]byte("hi"))
	if err != nil {
		t.Fatalf("Marshal bytes error: %v", err)
	}
	if got := encodedBytes[""]; len(got) != 1 || got[0] != "hi" {
		t.Errorf("root bytes = %v, want [hi]", got)
	}

	var decodedBytes []byte
	if err := Unmarshal(encodedBytes, &decodedBytes); err != nil {
		t.Fatalf("Unmarshal bytes error: %v", err)
	}
	if string(decodedBytes) != "hi" {
		t.Errorf("decoded bytes = %q, want hi", decodedBytes)
	}
}

func TestTopLevelInterface(t *testing.T) {
	var value any
	if err := Unmarshal(map[string][]string{"": {"hello"}}, &value); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if value != "hello" {
		t.Errorf("value = %#v, want hello", value)
	}

	encoded, err := Marshal("hello")
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var decoded any
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != "hello" {
		t.Errorf("decoded = %#v, want hello", decoded)
	}
}

func TestTopLevelCustomUnmarshaler(t *testing.T) {
	var value topLevelCustom
	if err := Unmarshal(map[string][]string{"": {"x"}}, &value); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if value.V != "x" {
		t.Errorf("V = %q, want x", value.V)
	}
}

type topLevelCustom struct {
	V string
}

func (c *topLevelCustom) UnmarshalValues(values []string) error {
	if len(values) > 0 {
		c.V = values[0]
	}
	return nil
}

func TestTopLevelNilMapAndSlice(t *testing.T) {
	var nilMap map[string]string
	encoded, err := Marshal(nilMap)
	if err != nil {
		t.Fatalf("Marshal nil map error: %v", err)
	}
	if len(encoded) != 0 {
		t.Errorf("encoded = %v, want empty", encoded)
	}

	var nilSlice []string
	encoded, err = Marshal(nilSlice)
	if err != nil {
		t.Fatalf("Marshal nil slice error: %v", err)
	}
	if len(encoded) != 0 {
		t.Errorf("encoded = %v, want empty", encoded)
	}

	var targetMap map[string]string
	if err := Unmarshal(nil, &targetMap); err != nil {
		t.Fatalf("Unmarshal nil data error: %v", err)
	}
	if targetMap != nil {
		t.Errorf("targetMap = %#v, want nil", targetMap)
	}
}
