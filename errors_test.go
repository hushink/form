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
	"strings"
	"testing"
)

var errSentinel = errors.New("sentinel error")

type sentinelMarshaler struct{}

func (sentinelMarshaler) MarshalValues() ([]string, error) {
	return nil, errSentinel
}

type sentinelUnmarshaler struct{}

func (*sentinelUnmarshaler) UnmarshalValues([]string) error {
	return errSentinel
}

func TestEncodeErrorsWrappingAndMessage(t *testing.T) {
	type S struct {
		Bad  sentinelMarshaler `form:"bad"`
		Good string            `form:"good"`
	}

	values, err := Marshal(S{Good: "ok"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errSentinel) {
		t.Errorf("errors.Is(err, errSentinel) = false: %v", err)
	}
	if !strings.Contains(err.Error(), "bad:") {
		t.Errorf("Error() = %q, want it to contain %q", err.Error(), "bad:")
	}
	if got := values["good"]; len(got) != 1 || got[0] != "ok" {
		t.Errorf("good = %v, want [ok]", got)
	}
}

func TestDecodeErrorsWrappingAndMessage(t *testing.T) {
	type S struct {
		Bad  sentinelUnmarshaler `form:"bad"`
		Good string              `form:"good"`
	}

	var result S
	err := Unmarshal(map[string][]string{
		"bad":  {"x"},
		"good": {"ok"},
	}, &result)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errSentinel) {
		t.Errorf("errors.Is(err, errSentinel) = false: %v", err)
	}
	if !strings.Contains(err.Error(), "bad:") {
		t.Errorf("Error() = %q, want it to contain %q", err.Error(), "bad:")
	}
	if result.Good != "ok" {
		t.Errorf("Good = %q, want ok", result.Good)
	}
}

func TestDecodeErrorsSortedMessage(t *testing.T) {
	type S struct {
		B int `form:"b"`
		A int `form:"a"`
	}

	var result S
	err := Unmarshal(map[string][]string{
		"b": {"x"},
		"a": {"y"},
	}, &result)
	if err == nil {
		t.Fatal("expected error")
	}

	message := err.Error()
	indexA := strings.Index(message, "a:")
	indexB := strings.Index(message, "b:")
	if indexA < 0 || indexB < 0 {
		t.Fatalf("Error() = %q, want both a: and b:", message)
	}
	if indexA > indexB {
		t.Errorf("Error() = %q, want keys sorted (a before b)", message)
	}
}

func TestDecodeErrorsJoinSamePath(t *testing.T) {
	type S struct {
		Nums []int `form:"nums"`
	}

	var result S
	err := Unmarshal(map[string][]string{"nums": {"x", "y"}}, &result)
	if err == nil {
		t.Fatal("expected error")
	}

	de, ok := err.(DecodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if len(de) != 1 {
		t.Fatalf("error count = %d, want 1: %v", len(de), de)
	}
	message := de["nums"].Error()
	if !strings.Contains(message, "x") || !strings.Contains(message, "y") {
		t.Errorf("joined error = %q, want both x and y", message)
	}
}

func TestDecodeStructIndexedPathRejected(t *testing.T) {
	type Inner struct {
		A string `form:"a"`
	}
	type S struct {
		In Inner `form:"in"`
	}

	var result S
	err := Unmarshal(map[string][]string{"in[0]": {"x"}}, &result)
	if err == nil {
		t.Fatal("expected error")
	}
	de, ok := err.(DecodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if _, ok := de["in"]; !ok {
		t.Errorf("missing error for in: %v", de)
	}
}

func TestInvalidTagName(t *testing.T) {
	type S struct {
		A string `form:"x.y"`
	}

	var decoded S
	err := Unmarshal(map[string][]string{"x.y": {"value"}}, &decoded)
	if err == nil {
		t.Fatal("expected error for tag name containing '.'")
	}
	de, ok := err.(DecodeErrors)
	if !ok {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if _, ok := de["x.y"]; !ok {
		t.Errorf("missing tag name error: %v", de)
	}

	values, err := Marshal(S{A: "value"})
	if err == nil {
		t.Fatal("expected error for tag name containing '.'")
	}
	if len(values) != 0 {
		t.Errorf("values = %v, want empty", values)
	}
}

type pointerCustom struct {
	V string
}

func (p *pointerCustom) UnmarshalValues(values []string) error {
	if len(values) > 0 {
		p.V = values[0]
	}
	return nil
}

type failingPointerCustom struct{}

func (*failingPointerCustom) UnmarshalValues([]string) error {
	return errSentinel
}

func TestDecodePointerCustomUnmarshaler(t *testing.T) {
	type S struct {
		P *pointerCustom `form:"p"`
	}

	var result S
	if err := Unmarshal(map[string][]string{"p": {"x"}}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result.P == nil || result.P.V != "x" {
		t.Fatalf("P = %#v, want V=x", result.P)
	}

	existing := &pointerCustom{V: "old"}
	result.P = existing
	if err := Unmarshal(map[string][]string{"p": {"new"}}, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if result.P != existing || result.P.V != "new" {
		t.Errorf("P = %#v, want existing pointer updated to new", result.P)
	}
}

func TestDecodeNilPointerCustomUnmarshalerError(t *testing.T) {
	type S struct {
		P *failingPointerCustom `form:"p"`
	}

	var result S
	err := Unmarshal(map[string][]string{"p": {"x"}}, &result)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errSentinel) {
		t.Errorf("errors.Is(err, errSentinel) = false: %v", err)
	}
	if result.P != nil {
		t.Errorf("P = %#v, want nil after failed decode", result.P)
	}
}

func TestErrorMessages(t *testing.T) {
	if got, want := (UnknownKeyError{Key: "x"}).Error(), `form: unknown key "x"`; got != want {
		t.Errorf("UnknownKeyError = %q, want %q", got, want)
	}
	if got, want := (EmptyFieldError{Key: "x"}).Error(), `form: required field "x" is empty`; got != want {
		t.Errorf("EmptyFieldError = %q, want %q", got, want)
	}
}

func TestMapKeyKinds(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		type S struct {
			M map[bool]string `form:"m"`
		}

		var decoded S
		if err := Unmarshal(map[string][]string{"m[true]": {"yes"}}, &decoded); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.M[true] != "yes" {
			t.Errorf("M = %#v", decoded.M)
		}

		encoded, err := Marshal(S{M: map[bool]string{true: "yes"}})
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		if got := encoded["m[true]"]; len(got) != 1 || got[0] != "yes" {
			t.Errorf("m[true] = %v", got)
		}

		var invalid S
		if err := Unmarshal(map[string][]string{"m[maybe]": {"x"}}, &invalid); err == nil {
			t.Error("expected error for invalid bool map key")
		}
	})

	t.Run("uint", func(t *testing.T) {
		type S struct {
			M map[uint]string `form:"m"`
		}

		var decoded S
		if err := Unmarshal(map[string][]string{"m[7]": {"seven"}}, &decoded); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.M[7] != "seven" {
			t.Errorf("M = %#v", decoded.M)
		}

		encoded, err := Marshal(S{M: map[uint]string{7: "seven"}})
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		if got := encoded["m[7]"]; len(got) != 1 || got[0] != "seven" {
			t.Errorf("m[7] = %v", got)
		}
	})

	t.Run("float", func(t *testing.T) {
		type S struct {
			M map[float64]string `form:"m"`
		}

		var decoded S
		if err := Unmarshal(map[string][]string{"m[1.5]": {"one and a half"}}, &decoded); err != nil {
			t.Fatalf("Unmarshal error: %v", err)
		}
		if decoded.M[1.5] != "one and a half" {
			t.Errorf("M = %#v", decoded.M)
		}

		encoded, err := Marshal(S{M: map[float64]string{1.5: "one and a half"}})
		if err != nil {
			t.Fatalf("Marshal error: %v", err)
		}
		if got := encoded["m[1.5]"]; len(got) != 1 || got[0] != "one and a half" {
			t.Errorf("m[1.5] = %v", got)
		}
	})
}
