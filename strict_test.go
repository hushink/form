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
)

func TestStrictUnknownKeysDefaultOff(t *testing.T) {
	type S struct {
		Name string `form:"name"`
	}

	var result S
	if err := Unmarshal(map[string][]string{
		"name":    {"n"},
		"unknown": {"x"},
	}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "n" {
		t.Errorf("Name = %q, want n", result.Name)
	}
}

func TestStrictUnknownKeysTopLevel(t *testing.T) {
	type S struct {
		Name string `form:"name"`
	}

	var result S
	err := NewDecoder(WithStrictUnknownKeys(true)).Decode(map[string][]string{
		"name":    {"n"},
		"unknown": {"x"},
	}, &result)
	if err == nil {
		t.Fatal("expected unknown key error")
	}

	de, ok := err.(DecodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if _, ok := de["unknown"]; !ok {
		t.Errorf("missing unknown key error: %v", de)
	}

	var unknownErr UnknownKeyError
	if !errors.As(err, &unknownErr) || unknownErr.Key != "unknown" {
		t.Errorf("errors.As UnknownKeyError = %#v, want key unknown", unknownErr)
	}
}

func TestStrictUnknownKeysNested(t *testing.T) {
	type Inner struct {
		Name string `form:"name"`
	}
	type S struct {
		In Inner `form:"in"`
	}

	var result S
	err := NewDecoder(WithStrictUnknownKeys(true)).Decode(map[string][]string{
		"in.name":    {"n"},
		"in.unknown": {"x"},
	}, &result)
	if err == nil {
		t.Fatal("expected unknown key error")
	}
	de := err.(DecodeErrors)
	if _, ok := de["in.unknown"]; !ok {
		t.Errorf("missing unknown key error: %v", de)
	}
}

func TestStrictUnknownKeysSliceElement(t *testing.T) {
	type Item struct {
		Name string `form:"name"`
	}
	type S struct {
		Items []Item `form:"items"`
	}

	var result S
	err := NewDecoder(WithStrictUnknownKeys(true)).Decode(map[string][]string{
		"items[0].name":    {"n"},
		"items[0].unknown": {"x"},
	}, &result)
	if err == nil {
		t.Fatal("expected unknown key error")
	}
	de := err.(DecodeErrors)
	if _, ok := de["items[0].unknown"]; !ok {
		t.Errorf("missing unknown key error: %v", de)
	}
}

func TestStrictUnknownKeysUnknownPrefix(t *testing.T) {
	type S struct {
		Name string `form:"name"`
	}

	var result S
	err := NewDecoder(WithStrictUnknownKeys(true)).Decode(map[string][]string{
		"unknown.child": {"x"},
	}, &result)
	if err == nil {
		t.Fatal("expected unknown key error")
	}
	de := err.(DecodeErrors)
	if _, ok := de["unknown"]; !ok {
		t.Errorf("expected top-level unknown key, got %v", de)
	}
}

func TestStrictUnknownKeysBracketUnderScalar(t *testing.T) {
	type S struct {
		Name string `form:"name"`
	}

	var result S
	err := NewDecoder(WithStrictUnknownKeys(true)).Decode(map[string][]string{
		"name[0]": {"x"},
	}, &result)
	if err == nil {
		t.Fatal("expected unknown key error")
	}
	de := err.(DecodeErrors)
	if _, ok := de["name[0]"]; !ok {
		t.Errorf("missing unknown bracket error: %v", de)
	}
}

func TestStrictUnknownKeysMapEntriesAllowed(t *testing.T) {
	type S struct {
		M map[string]string `form:"m"`
	}

	var result S
	if err := NewDecoder(WithStrictUnknownKeys(true)).Decode(map[string][]string{
		"m[a]": {"1"},
		"m[b]": {"2"},
	}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.M["a"] != "1" || result.M["b"] != "2" {
		t.Errorf("M = %#v", result.M)
	}
}

func TestStrictUnknownKeysValidInput(t *testing.T) {
	type Inner struct {
		Name string `form:"name"`
	}
	type S struct {
		Name  string `form:"name"`
		In    Inner  `form:"in"`
		Items []int  `form:"items"`
	}

	var result S
	if err := NewDecoder(WithStrictUnknownKeys(true)).Decode(map[string][]string{
		"name":    {"n"},
		"in.name": {"i"},
		"items":   {"1", "2"},
	}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStrictUnknownKeysNonStructTarget(t *testing.T) {
	var result map[string]int
	err := NewDecoder(WithStrictUnknownKeys(true)).Decode(map[string][]string{
		"[a]": {"1"},
		"foo": {"2"},
	}, &result)
	if err == nil {
		t.Fatal("expected unknown key error for non-bracket key")
	}
}
