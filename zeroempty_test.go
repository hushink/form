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
	"testing"
	"time"
)

type zeroEmptyStruct struct {
	I    int        `form:"i"`
	U    uint       `form:"u"`
	F    float64    `form:"f"`
	T    time.Time  `form:"t"`
	B    []byte     `form:"b"`
	P    *int       `form:"p"`
	TP   *time.Time `form:"tp"`
	S    string     `form:"s"`
	Bool bool       `form:"bool"`
}

func zeroEmptyData() map[string][]string {
	return map[string][]string{
		"i":    {""},
		"u":    {""},
		"f":    {""},
		"t":    {""},
		"b":    {""},
		"p":    {""},
		"tp":   {""},
		"s":    {""},
		"bool": {""},
	}
}

func TestZeroEmptyDefaultOff(t *testing.T) {
	date := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	pointer := 7

	result := zeroEmptyStruct{
		I: 1, U: 2, F: 3.5, T: date, B: []byte("x"), P: &pointer, TP: &date,
		S: "s", Bool: true,
	}
	if err := Unmarshal(zeroEmptyData(), &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if result.I != 1 || result.U != 2 || result.F != 3.5 {
		t.Errorf("numeric fields = %d/%d/%v, want 1/2/3.5", result.I, result.U, result.F)
	}
	if !result.T.Equal(date) {
		t.Errorf("T = %v, want %v", result.T, date)
	}
	if string(result.B) != "x" {
		t.Errorf("B = %q, want x", result.B)
	}
	if result.P == nil || *result.P != 7 {
		t.Errorf("P = %v, want 7", result.P)
	}
	if result.TP == nil || !result.TP.Equal(date) {
		t.Errorf("TP = %v, want %v", result.TP, date)
	}
	if result.S != "" {
		t.Errorf("S = %q, want empty (strings always receive empty values)", result.S)
	}
	if result.Bool {
		t.Error("Bool = true, want false (empty bool is always false)")
	}
}

func TestZeroEmptyEnabled(t *testing.T) {
	date := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	pointer := 7

	result := zeroEmptyStruct{
		I: 1, U: 2, F: 3.5, T: date, B: []byte("x"), P: &pointer, TP: &date,
		S: "s", Bool: true,
	}
	decoder := NewDecoder(WithZeroEmpty(true))
	if err := decoder.Decode(zeroEmptyData(), &result); err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	if result.I != 0 || result.U != 0 || result.F != 0 {
		t.Errorf("numeric fields = %d/%d/%v, want zeros", result.I, result.U, result.F)
	}
	if !result.T.IsZero() {
		t.Errorf("T = %v, want zero", result.T)
	}
	if result.B != nil {
		t.Errorf("B = %v, want nil", result.B)
	}
	if result.P == nil || *result.P != 0 {
		t.Errorf("P = %v, want pointer to 0", result.P)
	}
	if result.TP == nil || !result.TP.IsZero() {
		t.Errorf("TP = %v, want pointer to zero time", result.TP)
	}
	if result.S != "" || result.Bool {
		t.Errorf("S = %q Bool = %v, want empty/false", result.S, result.Bool)
	}
}

func TestZeroEmptyMissingKeysUntouched(t *testing.T) {
	result := zeroEmptyStruct{I: 1, S: "s", Bool: true}
	decoder := NewDecoder(WithZeroEmpty(true))
	if err := decoder.Decode(map[string][]string{}, &result); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if result.I != 1 || result.S != "s" || !result.Bool {
		t.Errorf("result = %#v, want untouched", result)
	}
}
