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

func TestDurationEncodeDecode(t *testing.T) {
	type S struct {
		D time.Duration `form:"d"`
	}

	encoded, err := Marshal(S{D: 90 * time.Second})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["d"]; len(got) != 1 || got[0] != "1m30s" {
		t.Errorf("d = %v, want [1m30s]", got)
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"d": {"1m30s"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.D != 90*time.Second {
		t.Errorf("D = %v, want 1m30s", decoded.D)
	}

	var numeric S
	if err := Unmarshal(map[string][]string{"d": {"1500000000"}}, &numeric); err != nil {
		t.Fatalf("Unmarshal numeric error: %v", err)
	}
	if numeric.D != 1500*time.Millisecond {
		t.Errorf("D = %v, want 1.5s", numeric.D)
	}

	var zero S
	encodedZero, err := Marshal(zero)
	if err != nil {
		t.Fatalf("Marshal zero error: %v", err)
	}
	if got := encodedZero["d"]; len(got) != 1 || got[0] != "0s" {
		t.Errorf("zero duration = %v, want [0s]", got)
	}
}

func TestDurationFormatOption(t *testing.T) {
	type S struct {
		D time.Duration `form:"d"`
	}

	encoded, err := NewEncoder(WithDurationFormat(DurationNanoseconds)).Encode(S{D: time.Second})
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := encoded["d"]; len(got) != 1 || got[0] != "1000000000" {
		t.Errorf("d = %v, want [1000000000]", got)
	}

	var decoded S
	if err := NewDecoder(WithDurationFormat(DurationNanoseconds)).Decode(map[string][]string{"d": {"1s"}}, &decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if decoded.D != time.Second {
		t.Errorf("D = %v, want 1s", decoded.D)
	}
}

func TestDurationContainers(t *testing.T) {
	type S struct {
		P     *time.Duration           `form:"p"`
		Slice []time.Duration          `form:"slice"`
		Map   map[string]time.Duration `form:"m"`
	}

	duration := 2 * time.Second
	input := S{
		P:     &duration,
		Slice: []time.Duration{time.Second, 2 * time.Second},
		Map:   map[string]time.Duration{"a": time.Minute},
	}

	encoded, err := Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["p"]; len(got) != 1 || got[0] != "2s" {
		t.Errorf("p = %v, want [2s]", got)
	}
	if got := encoded["slice"]; !reflect.DeepEqual(got, []string{"1s", "2s"}) {
		t.Errorf("slice = %v, want [1s 2s]", got)
	}
	if got := encoded["m[a]"]; len(got) != 1 || got[0] != "1m0s" {
		t.Errorf("m[a] = %v, want [1m0s]", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.P == nil || *decoded.P != duration {
		t.Errorf("P = %v, want %v", decoded.P, duration)
	}
	if !reflect.DeepEqual(decoded.Slice, input.Slice) {
		t.Errorf("Slice = %v, want %v", decoded.Slice, input.Slice)
	}
	if decoded.Map["a"] != time.Minute {
		t.Errorf("Map[a] = %v, want 1m", decoded.Map["a"])
	}
}

func TestDurationDefault(t *testing.T) {
	type S struct {
		D time.Duration `form:"d,default=1s"`
	}

	var decoded S
	if err := Unmarshal(map[string][]string{}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.D != time.Second {
		t.Errorf("D = %v, want 1s", decoded.D)
	}

	type E struct {
		D time.Duration `form:"d,omitempty,default=1s"`
	}
	encoded, err := Marshal(E{D: time.Second})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if _, ok := encoded["d"]; ok {
		t.Errorf("d should be omitted when it equals the default, got %v", encoded["d"])
	}
}

func TestDurationEmptyAndErrors(t *testing.T) {
	type S struct {
		D time.Duration `form:"d"`
	}

	present := S{D: 5 * time.Second}
	if err := Unmarshal(map[string][]string{"d": {""}}, &present); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if present.D != 5*time.Second {
		t.Errorf("D = %v, want unchanged 5s", present.D)
	}

	var zero S
	if err := NewDecoder(WithZeroEmpty(true)).Decode(map[string][]string{"d": {""}}, &zero); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if zero.D != 0 {
		t.Errorf("D = %v, want zero", zero.D)
	}

	var invalid S
	if err := Unmarshal(map[string][]string{"d": {"not-a-duration"}}, &invalid); err == nil {
		t.Fatal("expected duration parse error")
	}
}

func TestDurationTopLevel(t *testing.T) {
	encoded, err := Marshal(time.Second)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded[""]; len(got) != 1 || got[0] != "1s" {
		t.Errorf("root = %v, want [1s]", got)
	}

	var decoded time.Duration
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != time.Second {
		t.Errorf("decoded = %v, want 1s", decoded)
	}
}
