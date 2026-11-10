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

func TestRequiredMissing(t *testing.T) {
	type S struct {
		Name string `form:"name,required"`
		Age  int    `form:"age,required"`
	}

	var result S
	err := Unmarshal(map[string][]string{}, &result)
	if err == nil {
		t.Fatal("expected required field errors")
	}
	de := err.(DecodeErrors)
	for _, key := range []string{"name", "age"} {
		if _, ok := de[key]; !ok {
			t.Errorf("missing required error for %s: %v", key, de)
		}
	}

	var emptyErr EmptyFieldError
	if !errors.As(err, &emptyErr) {
		t.Errorf("errors.As EmptyFieldError = false: %v", err)
	}
}

func TestRequiredPresent(t *testing.T) {
	type S struct {
		Name string   `form:"name,required"`
		Age  int      `form:"age,required"`
		Tags []string `form:"tags,required"`
	}

	var result S
	if err := Unmarshal(map[string][]string{
		"name": {"n"},
		"age":  {"0"}, // explicit zero is a provided value
		"tags": {"a"},
	}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "n" || result.Age != 0 || len(result.Tags) != 1 {
		t.Errorf("result = %#v", result)
	}
}

func TestRequiredEmptyValues(t *testing.T) {
	type S struct {
		Name string   `form:"name,required"`
		Age  int      `form:"age,required"`
		Tags []string `form:"tags,required"`
	}

	var result S
	err := Unmarshal(map[string][]string{
		"name": {""},
		"age":  {""},
		"tags": {""},
	}, &result)
	if err == nil {
		t.Fatal("expected required field errors")
	}
	de := err.(DecodeErrors)
	for _, key := range []string{"name", "age", "tags"} {
		if _, ok := de[key]; !ok {
			t.Errorf("missing required error for %s: %v", key, de)
		}
	}
}

func TestRequiredMapAndStruct(t *testing.T) {
	type Inner struct {
		Name string `form:"name"`
	}
	type S struct {
		M  map[string]string `form:"m,required"`
		In Inner             `form:"in,required"`
	}

	var missing S
	err := Unmarshal(map[string][]string{}, &missing)
	if err == nil {
		t.Fatal("expected required errors")
	}
	de := err.(DecodeErrors)
	for _, key := range []string{"m", "in"} {
		if _, ok := de[key]; !ok {
			t.Errorf("missing required error for %s: %v", key, de)
		}
	}

	var present S
	if err := Unmarshal(map[string][]string{
		"m[k]":    {"v"},
		"in.name": {"n"},
	}, &present); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if present.M["k"] != "v" || present.In.Name != "n" {
		t.Errorf("result = %#v", present)
	}
}

func TestRequiredAnonymousEmbed(t *testing.T) {
	type Base struct {
		ID int `form:"id"`
	}
	type S struct {
		Base `form:"base,required"`
	}

	var flattened S
	if err := Unmarshal(map[string][]string{"id": {"7"}}, &flattened); err != nil {
		t.Fatalf("unexpected error for flattened key: %v", err)
	}
	if flattened.ID != 7 {
		t.Errorf("ID = %d, want 7", flattened.ID)
	}

	var missing S
	if err := Unmarshal(map[string][]string{}, &missing); err == nil {
		t.Fatal("expected required error")
	}

	separate := NewDecoder(WithAnonymousMode(AnonymousSeparate))
	var separateResult S
	if err := separate.Decode(map[string][]string{"base.id": {"7"}}, &separateResult); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if separateResult.ID != 7 {
		t.Errorf("ID = %d, want 7", separateResult.ID)
	}
}

type ReqInt int

func TestRequiredEmbeddedNonStruct(t *testing.T) {
	type S struct {
		ReqInt `form:"req,required"`
	}

	var present S
	if err := Unmarshal(map[string][]string{"req": {"5"}}, &present); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if present.ReqInt != 5 {
		t.Errorf("ReqInt = %d, want 5", present.ReqInt)
	}

	var missing S
	if err := Unmarshal(map[string][]string{}, &missing); err == nil {
		t.Fatal("expected required error")
	}
}

func TestRequiredPointerEmpty(t *testing.T) {
	type S struct {
		P *int `form:"p,required"`
	}

	var present S
	if err := Unmarshal(map[string][]string{"p": {"0"}}, &present); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if present.P == nil || *present.P != 0 {
		t.Errorf("P = %v, want pointer to 0", present.P)
	}

	var empty S
	if err := Unmarshal(map[string][]string{"p": {""}}, &empty); err == nil {
		t.Fatal("expected required error for empty value")
	}
}

func TestRequiredAnonymousSeparateMissing(t *testing.T) {
	type Base struct {
		ID int `form:"id"`
	}
	type S struct {
		Base `form:"base,required"`
	}

	var result S
	decoder := NewDecoder(WithAnonymousMode(AnonymousSeparate))
	if err := decoder.Decode(map[string][]string{}, &result); err == nil {
		t.Fatal("expected required error")
	}
}

func TestRequiredWithDefaultIsError(t *testing.T) {
	type S struct {
		Name string `form:"name,required,default=guest"`
	}

	var result S
	err := Unmarshal(map[string][]string{}, &result)
	if err == nil {
		t.Fatal("expected required/default configuration error")
	}
	de := err.(DecodeErrors)
	if _, ok := de["name"]; !ok {
		t.Errorf("missing error for name: %v", de)
	}
	if result.Name != "guest" {
		t.Errorf("Name = %q, want guest (default still applied)", result.Name)
	}
}
