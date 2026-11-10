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

type nsAddress struct {
	City string `form:"city"`
}

type nsItem struct {
	Name string `form:"name"`
}

type nsUser struct {
	Address nsAddress            `form:"address"`
	Items   []nsItem             `form:"items"`
	Matrix  [][]int              `form:"matrix"`
	Lookup  map[string]nsAddress `form:"lookup"`
}

func nsUserValue() nsUser {
	return nsUser{
		Address: nsAddress{City: "shanghai"},
		Items:   []nsItem{{Name: "a"}, {Name: "b"}},
		Matrix:  [][]int{{1, 2}, {3}},
		Lookup:  map[string]nsAddress{"k": {City: "beijing"}},
	}
}

func TestNamespacePrefix(t *testing.T) {
	encoder := NewEncoder(WithNamespacePrefix("_"))
	decoder := NewDecoder(WithNamespacePrefix("_"))

	input := nsUserValue()
	encoded, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	expected := map[string]string{
		"address_city":   "shanghai",
		"items[0]_name":  "a",
		"items[1]_name":  "b",
		"matrix[0][0]":   "1",
		"matrix[0][1]":   "2",
		"matrix[1][0]":   "3",
		"lookup[k]_city": "beijing",
	}
	for key, want := range expected {
		if got := encoded[key]; len(got) != 1 || got[0] != want {
			t.Errorf("%s = %v, want [%s]", key, got, want)
		}
	}

	var decoded nsUser
	if err := decoder.Decode(encoded, &decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded, input) {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}
}

func TestNamespaceSuffix(t *testing.T) {
	encoder := NewEncoder(WithNamespaceSuffix("!"))
	decoder := NewDecoder(WithNamespaceSuffix("!"))

	input := nsUserValue()
	encoded, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	expected := map[string]string{
		"address.city!":   "shanghai",
		"items[0].name!":  "a",
		"items[1].name!":  "b",
		"matrix[0][0]":    "1",
		"matrix[0][1]":    "2",
		"matrix[1][0]":    "3",
		"lookup[k].city!": "beijing",
	}
	for key, want := range expected {
		if got := encoded[key]; len(got) != 1 || got[0] != want {
			t.Errorf("%s = %v, want [%s]", key, got, want)
		}
	}

	var decoded nsUser
	if err := decoder.Decode(encoded, &decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded, input) {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}
}

func TestNamespacePrefixAndSuffix(t *testing.T) {
	encoder := NewEncoder(WithNamespacePrefix("."), WithNamespaceSuffix("_end"))
	decoder := NewDecoder(WithNamespacePrefix("."), WithNamespaceSuffix("_end"))

	input := nsUserValue()
	encoded, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := encoded["address.city_end"]; !reflect.DeepEqual(got, []string{"shanghai"}) {
		t.Errorf("address.city_end = %v, want [shanghai]", got)
	}
	if got := encoded["items[0].name_end"]; !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("items[0].name_end = %v, want [a]", got)
	}
	if got := encoded["lookup[k].city_end"]; !reflect.DeepEqual(got, []string{"beijing"}) {
		t.Errorf("lookup[k].city_end = %v, want [beijing]", got)
	}

	var decoded nsUser
	if err := decoder.Decode(encoded, &decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded, input) {
		t.Errorf("decoded = %#v, want %#v", decoded, input)
	}
}

func TestNamespaceSuffixContainingPrefixRejected(t *testing.T) {
	encoder := NewEncoder(WithNamespacePrefix("_"), WithNamespaceSuffix("_end"))
	if _, err := encoder.Encode(struct {
		Name string `form:"name"`
	}{Name: "x"}); err == nil {
		t.Fatal("expected namespace configuration error from Encode")
	}

	decoder := NewDecoder(WithNamespacePrefix("_"), WithNamespaceSuffix("_end"))
	var target struct {
		Name string `form:"name"`
	}
	if err := decoder.Decode(map[string][]string{"name": {"x"}}, &target); err == nil {
		t.Fatal("expected namespace configuration error from Decode")
	}
}

func TestNamespaceEmptyPrefixIgnored(t *testing.T) {
	encoder := NewEncoder(WithNamespacePrefix(""))
	encoded, err := encoder.Encode(nsUser{Address: nsAddress{City: "x"}})
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := encoded["address.city"]; !reflect.DeepEqual(got, []string{"x"}) {
		t.Errorf("address.city = %v, want [x]", got)
	}
}

func TestNamespaceWrongSeparatorStrict(t *testing.T) {
	decoder := NewDecoder(WithNamespacePrefix("_"), WithStrictUnknownKeys(true))
	var decoded nsUser
	err := decoder.Decode(map[string][]string{"address.city": {"x"}}, &decoded)
	if err == nil {
		t.Fatal("expected unknown key error with the wrong separator")
	}
	if decoded.Address.City != "" {
		t.Errorf("Address.City = %q, want empty", decoded.Address.City)
	}
}

func TestParseKeyPathCustomSeparators(t *testing.T) {
	tests := []struct {
		key    string
		prefix string
		suffix string
		want   []pathSegment
	}{
		{
			key: "address_city", prefix: "_",
			want: []pathSegment{{field: "address"}, {field: "city"}},
		},
		{
			key: "address.city!", prefix: ".", suffix: "!",
			want: []pathSegment{{field: "address"}, {field: "city"}},
		},
		{
			key: "items[0]_name", prefix: "_",
			want: []pathSegment{{field: "items"}, {bracket: "0", isBracket: true, isNum: true, num: 0}, {field: "name"}},
		},
		{
			key: "items[0].name!", prefix: ".", suffix: "!",
			want: []pathSegment{{field: "items"}, {bracket: "0", isBracket: true, isNum: true, num: 0}, {field: "name"}},
		},
		{
			key: "a.b_end", prefix: ".", suffix: "_end",
			want: []pathSegment{{field: "a"}, {field: "b"}},
		},
		{
			key: "a.b!", prefix: ".", suffix: "!x",
			want: []pathSegment{{field: "a"}, {field: "b!"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := parseKeyPath(tt.key, tt.prefix, tt.suffix)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseKeyPath(%q) = %#v, want %#v", tt.key, got, tt.want)
			}
		})
	}
}
