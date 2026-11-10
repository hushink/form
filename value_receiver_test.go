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
	"strings"
	"testing"
)

type valueReceiverUnmarshaler struct {
	V string
}

func (valueReceiverUnmarshaler) UnmarshalValues([]string) error { return nil }

type pointerReceiverUnmarshaler struct {
	V string
}

func (p *pointerReceiverUnmarshaler) UnmarshalValues(values []string) error {
	if len(values) > 0 {
		p.V = values[0]
	}
	return nil
}

type valueReceiverText string

func (valueReceiverText) UnmarshalText([]byte) error { return nil }

type PtrReceiverTarget struct {
	V string
}

func (p *PtrReceiverTarget) UnmarshalValues(values []string) error {
	if len(values) > 0 {
		p.V = values[0]
	}
	return nil
}

type EmbeddedPtrReceiverHolder struct {
	*PtrReceiverTarget
}

func TestValueReceiverUnmarshalerDefaultNoop(t *testing.T) {
	type S struct {
		Value valueReceiverUnmarshaler `form:"value"`
	}

	var result S
	if err := Unmarshal(map[string][]string{"value": {"x"}}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value.V != "" {
		t.Errorf("V = %q, want empty (value receiver cannot mutate)", result.Value.V)
	}
}

func TestRejectValueReceiverUnmarshaler(t *testing.T) {
	type S struct {
		Value valueReceiverUnmarshaler `form:"value"`
	}

	var result S
	err := NewDecoder(WithRejectValueReceiverUnmarshalers(true)).Decode(map[string][]string{"value": {"x"}}, &result)
	if err == nil {
		t.Fatal("expected value-receiver error")
	}
	if !strings.Contains(err.Error(), "value-receiver") {
		t.Errorf("error = %v, want value-receiver message", err)
	}
}

func TestPointerReceiverUnmarshalerStillWorks(t *testing.T) {
	type S struct {
		Value pointerReceiverUnmarshaler `form:"value"`
	}

	var result S
	decoder := NewDecoder(WithRejectValueReceiverUnmarshalers(true))
	if err := decoder.Decode(map[string][]string{"value": {"x"}}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value.V != "x" {
		t.Errorf("V = %q, want x", result.Value.V)
	}
}

func TestRejectValueReceiverTextUnmarshaler(t *testing.T) {
	type S struct {
		Value valueReceiverText `form:"value"`
	}

	var result S
	err := NewDecoder(WithRejectValueReceiverUnmarshalers(true)).Decode(map[string][]string{"value": {"x"}}, &result)
	if err == nil {
		t.Fatal("expected value-receiver TextUnmarshaler error")
	}
	if !strings.Contains(err.Error(), "value-receiver") {
		t.Errorf("error = %v, want value-receiver message", err)
	}

	type P struct {
		Value *valueReceiverText `form:"value"`
	}
	var pointer P
	if err := NewDecoder(WithRejectValueReceiverUnmarshalers(true)).Decode(map[string][]string{"value": {"x"}}, &pointer); err == nil {
		t.Fatal("expected value-receiver error for pointer to value receiver")
	}
}

func TestPromotedPointerReceiverNotRejected(t *testing.T) {
	result := EmbeddedPtrReceiverHolder{PtrReceiverTarget: &PtrReceiverTarget{}}
	decoder := NewDecoder(WithRejectValueReceiverUnmarshalers(true))
	if err := decoder.Decode(map[string][]string{"PtrReceiverTarget": {"x"}}, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.V != "x" {
		t.Errorf("V = %q, want x", result.V)
	}
}
