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

func TestParseKeyPath(t *testing.T) {
	tests := []struct {
		key  string
		want []pathSegment
	}{
		{key: "name", want: []pathSegment{{field: "name"}}},
		{key: "address.city", want: []pathSegment{{field: "address"}, {field: "city"}}},
		{key: "items[0]", want: []pathSegment{{field: "items"}, {bracket: "0", isBracket: true, isNum: true, num: 0}}},
		{key: "items[0].name", want: []pathSegment{{field: "items"}, {bracket: "0", isBracket: true, isNum: true, num: 0}, {field: "name"}}},
		{key: "filter[key]", want: []pathSegment{{field: "filter"}, {bracket: "key", isBracket: true}}},
		{key: "nested[a][b]", want: []pathSegment{{field: "nested"}, {bracket: "a", isBracket: true}, {bracket: "b", isBracket: true}}},
		{key: "matrix[0][1]", want: []pathSegment{{field: "matrix"}, {bracket: "0", isBracket: true, isNum: true, num: 0}, {bracket: "1", isBracket: true, isNum: true, num: 1}}},
		{key: "a[b.c]", want: []pathSegment{{field: "a"}, {bracket: "b.c", isBracket: true}}},
		{key: "", want: nil},
		{key: "[0]", want: []pathSegment{{bracket: "0", isBracket: true, isNum: true, num: 0}}},
		{key: "[key]", want: []pathSegment{{bracket: "key", isBracket: true}}},
		{key: "[0].name", want: []pathSegment{{bracket: "0", isBracket: true, isNum: true, num: 0}, {field: "name"}}},
		{key: "[0][1]", want: []pathSegment{{bracket: "0", isBracket: true, isNum: true, num: 0}, {bracket: "1", isBracket: true, isNum: true, num: 1}}},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := parseKeyPath(tt.key, ".", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseKeyPath(%q) = %#v, want %#v", tt.key, got, tt.want)
			}
		})
	}
}

func TestParseKeyPathInvalid(t *testing.T) {
	invalid := []string{
		".a",
		"a.",
		"a..b",
		"a[",
		"a[]",
		"a[0]b",
		"[0",
		"[]",
		"a[0]]",
		"[0]b",
		"a[999999999999999999999999]",
	}

	for _, key := range invalid {
		t.Run(key, func(t *testing.T) {
			if _, err := parseKeyPath(key, ".", ""); err == nil {
				t.Errorf("parseKeyPath(%q) = nil error, want error", key)
			}
		})
	}
}

func TestBuildPathTreeDeterministic(t *testing.T) {
	data := map[string][]string{
		"items[1]": {"b"},
		"items[0]": {"a"},
		"name":     {"n"},
	}

	root, errs := buildPathTree(data, ".", "")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	items := root.field("items")
	if items == nil || len(items.brackets) != 2 {
		t.Fatalf("items node = %#v", items)
	}
	if items.brackets[0].num != 0 || items.brackets[1].num != 1 {
		t.Errorf("bracket order = [%d %d], want [0 1]", items.brackets[0].num, items.brackets[1].num)
	}
	if got := items.brackets[0].node.values; !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("items[0] = %v, want [a]", got)
	}
}

func FuzzParseKeyPath(f *testing.F) {
	seeds := []string{
		"name",
		"address.city",
		"items[0].name",
		"filter[key]",
		"nested[a][b]",
		"a[0]",
		"",
		"a[",
		"a[0]b",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, key string) {
		segments, err := parseKeyPath(key, ".", "")
		if err == nil && key != "" && len(segments) == 0 {
			t.Fatalf("parseKeyPath(%q) returned no segments and no error", key)
		}
	})
}

func FuzzDecode(f *testing.F) {
	f.Add("items[0].name", "a")
	f.Add("filter[key]", "v")
	f.Add("name", "n")
	f.Add("items[1000000]", "x")
	f.Add("a[", "x")

	f.Fuzz(func(t *testing.T, key, value string) {
		type Item struct {
			Name string `form:"name"`
		}
		type S struct {
			Name   string            `form:"name"`
			Items  []Item            `form:"items"`
			Names  []string          `form:"names"`
			Filter map[string]string `form:"filter"`
			Nested map[string]Item   `form:"nested"`
			Array  [2]int            `form:"array"`
		}

		var result S
		_ = Unmarshal(map[string][]string{key: {value}}, &result)
	})
}
