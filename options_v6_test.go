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

func TestWithoutDefaults(t *testing.T) {
	type target struct {
		Name string `form:"name,default=guest"`
	}

	var got target
	if err := NewDecoder(WithoutDefaults()).Decode(nil, &got); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if got.Name != "" {
		t.Errorf("Name = %q, want empty (defaults disabled)", got.Name)
	}
}

func TestWithoutRequiredChecks(t *testing.T) {
	type target struct {
		Name string `form:"name,required"`
	}

	var got target
	if err := NewDecoder(WithoutRequiredChecks()).Decode(nil, &got); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	type conflict struct {
		Name string `form:"name,required,default=x"`
	}
	var gotConflict conflict
	if err := NewDecoder(WithoutRequiredChecks()).Decode(nil, &gotConflict); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if gotConflict.Name != "x" {
		t.Errorf("Name = %q, want x", gotConflict.Name)
	}
}

func TestWithRequiredFallback(t *testing.T) {
	type target struct {
		Name string `form:"name,required"`
	}

	existing := target{Name: "existing"}
	if err := NewDecoder(WithRequiredFallback()).Decode(nil, &existing); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	var empty target
	err := NewDecoder(WithRequiredFallback()).Decode(map[string][]string{"name": {""}}, &empty)
	if err == nil {
		t.Fatal("expected empty required value to fail")
	}
	var emptyErr EmptyFieldError
	if !errors.As(err, &emptyErr) {
		t.Fatalf("error = %v, want EmptyFieldError", err)
	}
}
