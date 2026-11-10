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

type AnonBase struct {
	ID int `form:"id"`
}

type AnonHolder struct {
	AnonBase
	Name string `form:"name"`
}

func TestAnonymousModeEmbedDefault(t *testing.T) {
	input := AnonHolder{AnonBase: AnonBase{ID: 7}, Name: "n"}

	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["id"]; !reflect.DeepEqual(got, []string{"7"}) {
		t.Errorf("id = %v, want [7]", got)
	}
	if _, ok := encoded["AnonBase.id"]; ok {
		t.Error("AnonBase.id should not be present in embed mode")
	}

	var decoded AnonHolder
	if err := Unmarshal(map[string][]string{
		"id":   {"7"},
		"name": {"n"},
	}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.ID != 7 || decoded.Name != "n" {
		t.Errorf("decoded = %#v, want ID=7 Name=n", decoded)
	}
}

func TestAnonymousModeSeparate(t *testing.T) {
	encoder := NewEncoder(WithAnonymousMode(AnonymousSeparate))
	decoder := NewDecoder(WithAnonymousMode(AnonymousSeparate))

	input := AnonHolder{AnonBase: AnonBase{ID: 7}, Name: "n"}
	encoded, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := encoded["AnonBase.id"]; !reflect.DeepEqual(got, []string{"7"}) {
		t.Errorf("AnonBase.id = %v, want [7]", got)
	}
	if _, ok := encoded["id"]; ok {
		t.Error("id should not be present in separate mode")
	}
	if got := encoded["name"]; !reflect.DeepEqual(got, []string{"n"}) {
		t.Errorf("name = %v, want [n]", got)
	}

	var decoded AnonHolder
	if err := decoder.Decode(encoded, &decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if decoded.ID != 7 || decoded.Name != "n" {
		t.Errorf("decoded = %#v, want ID=7 Name=n", decoded)
	}

	var flattened AnonHolder
	if err := decoder.Decode(map[string][]string{"id": {"7"}}, &flattened); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if flattened.ID != 0 {
		t.Errorf("ID = %d, want 0 (flattened key must be ignored)", flattened.ID)
	}
}

func TestAnonymousModeSeparateNested(t *testing.T) {
	type Outer struct {
		Inner AnonHolder `form:"inner"`
	}

	encoder := NewEncoder(WithAnonymousMode(AnonymousSeparate))
	decoder := NewDecoder(WithAnonymousMode(AnonymousSeparate))

	input := Outer{Inner: AnonHolder{AnonBase: AnonBase{ID: 9}, Name: "n"}}
	encoded, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := encoded["inner.AnonBase.id"]; !reflect.DeepEqual(got, []string{"9"}) {
		t.Errorf("inner.AnonBase.id = %v, want [9]", got)
	}

	var decoded Outer
	if err := decoder.Decode(encoded, &decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if decoded.Inner.ID != 9 || decoded.Inner.Name != "n" {
		t.Errorf("decoded = %#v", decoded)
	}
}

type hiddenAnonBase struct {
	ID int `form:"id"`
}

type hiddenAnonHolder struct {
	hiddenAnonBase
	Name string `form:"name"`
}

type hiddenPtrBase struct {
	ID int `form:"id"`
}

type hiddenPtrHolder struct {
	*hiddenPtrBase
	Name string `form:"name"`
}

func TestUnexportedEmbeddedPointer(t *testing.T) {
	t.Run("non-nil decode flattened", func(t *testing.T) {
		holder := hiddenPtrHolder{hiddenPtrBase: &hiddenPtrBase{ID: 1}, Name: "old"}
		if err := Unmarshal(map[string][]string{
			"id":   {"7"},
			"name": {"n"},
		}, &holder); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if holder.hiddenPtrBase == nil || holder.ID != 7 || holder.Name != "n" {
			t.Errorf("holder = %#v, want ID=7 Name=n", holder)
		}
	})

	t.Run("non-nil encode", func(t *testing.T) {
		encoded, err := Marshal(hiddenPtrHolder{
			hiddenPtrBase: &hiddenPtrBase{ID: 7},
			Name:          "n",
		})
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		if got := encoded["id"]; !reflect.DeepEqual(got, []string{"7"}) {
			t.Errorf("id = %v, want [7]", got)
		}
	})

	t.Run("nil with input reports error", func(t *testing.T) {
		var holder hiddenPtrHolder
		err := Unmarshal(map[string][]string{"id": {"7"}}, &holder)
		if err == nil {
			t.Fatal("expected nil unexported embedded pointer error")
		}
		if holder.hiddenPtrBase != nil {
			t.Errorf("hiddenPtrBase = %#v, want nil", holder.hiddenPtrBase)
		}
	})

	t.Run("nil without input is ignored", func(t *testing.T) {
		var holder hiddenPtrHolder
		if err := Unmarshal(map[string][]string{"name": {"n"}}, &holder); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if holder.Name != "n" {
			t.Errorf("Name = %q, want n", holder.Name)
		}
	})

	t.Run("separate mode non-nil", func(t *testing.T) {
		decoder := NewDecoder(WithAnonymousMode(AnonymousSeparate))
		holder := hiddenPtrHolder{hiddenPtrBase: &hiddenPtrBase{}}
		if err := decoder.Decode(map[string][]string{"hiddenPtrBase.id": {"9"}}, &holder); err != nil {
			t.Fatalf("Decode error: %v", err)
		}
		if holder.ID != 9 {
			t.Errorf("ID = %d, want 9", holder.ID)
		}
	})

	t.Run("separate mode nil with input reports error", func(t *testing.T) {
		decoder := NewDecoder(WithAnonymousMode(AnonymousSeparate))
		var holder hiddenPtrHolder
		if err := decoder.Decode(map[string][]string{"hiddenPtrBase.id": {"9"}}, &holder); err == nil {
			t.Fatal("expected nil unexported embedded pointer error")
		}
	})
}

func TestAnonymousModeSeparateUnexported(t *testing.T) {
	encoder := NewEncoder(WithAnonymousMode(AnonymousSeparate))
	decoder := NewDecoder(WithAnonymousMode(AnonymousSeparate))

	input := hiddenAnonHolder{hiddenAnonBase: hiddenAnonBase{ID: 5}, Name: "n"}
	encoded, err := encoder.Encode(input)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := encoded["hiddenAnonBase.id"]; !reflect.DeepEqual(got, []string{"5"}) {
		t.Errorf("hiddenAnonBase.id = %v, want [5]", got)
	}

	var decoded hiddenAnonHolder
	if err := decoder.Decode(encoded, &decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if decoded.ID != 5 || decoded.Name != "n" {
		t.Errorf("decoded = %#v, want ID=5 Name=n", decoded)
	}
}
