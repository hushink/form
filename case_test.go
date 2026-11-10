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

func TestCaseInsensitiveDecode(t *testing.T) {
	type Address struct {
		CityName string `form:"city"`
	}
	type target struct {
		Name    string            `form:"name"`
		Count   int               `form:"count,default=5"`
		Req     string            `form:"req,required"`
		Address Address           `form:"address"`
		Filter  map[string]string `form:"filter"`
	}

	data := map[string][]string{
		"NAME":         {"alice"},
		"COUNT":        {"7"},
		"REQ":          {"yes"},
		"ADDRESS.CITY": {"shanghai"},
		"FILTER[Key]":  {"value"},
	}

	var got target
	if err := NewDecoder(WithCaseInsensitive(true)).Decode(data, &got); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	want := target{
		Name:    "alice",
		Count:   7,
		Req:     "yes",
		Address: Address{CityName: "shanghai"},
		Filter:  map[string]string{"Key": "value"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCaseInsensitiveExactMatchWins(t *testing.T) {
	type target struct {
		Name string `form:"name"`
	}

	var got target
	if err := NewDecoder(WithCaseInsensitive(true)).Decode(map[string][]string{
		"name": {"exact"},
		"NAME": {"upper"},
	}, &got); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if got.Name != "exact" {
		t.Errorf("Name = %q, want exact", got.Name)
	}
}

func TestCaseInsensitiveStrictUnknownKeys(t *testing.T) {
	type target struct {
		Name string `form:"name"`
	}

	var got target
	err := NewDecoder(
		WithCaseInsensitive(true),
		WithStrictUnknownKeys(true),
	).Decode(map[string][]string{"NAME": {"alice"}}, &got)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if got.Name != "alice" {
		t.Errorf("Name = %q, want alice", got.Name)
	}
}

func TestCaseInsensitiveDisabledByDefault(t *testing.T) {
	type target struct {
		Name string `form:"name"`
	}

	var got target
	if err := Unmarshal(map[string][]string{"NAME": {"alice"}}, &got); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if got.Name != "" {
		t.Errorf("Name = %q, want empty when case-insensitive matching is disabled", got.Name)
	}
}

func TestCaseInsensitiveDoesNotAffectEncoding(t *testing.T) {
	type target struct {
		Name string `form:"name"`
	}

	values, err := NewEncoder(WithCaseInsensitive(true)).Encode(target{Name: "alice"})
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := values["name"]; !reflect.DeepEqual(got, []string{"alice"}) {
		t.Errorf("name = %v, want [alice]", got)
	}
	if _, ok := values["NAME"]; ok {
		t.Error("encoding must not use case-insensitive matching")
	}
}
