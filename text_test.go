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
	"net"
	"strings"
	"testing"
)

type textUpper string

func (t textUpper) MarshalText() ([]byte, error) {
	return []byte(strings.ToUpper(string(t))), nil
}

func (t *textUpper) UnmarshalText(data []byte) error {
	*t = textUpper(strings.ToLower(string(data)))
	return nil
}

type textBytes []byte

func (t textBytes) MarshalText() ([]byte, error) {
	return []byte(strings.ToUpper(string(t))), nil
}

func (t *textBytes) UnmarshalText(data []byte) error {
	*t = textBytes(strings.ToLower(string(data)))
	return nil
}

type textStruct struct {
	V string
}

func (t textStruct) MarshalText() ([]byte, error) {
	return []byte(t.V), nil
}

func (t *textStruct) UnmarshalText(data []byte) error {
	t.V = string(data)
	return nil
}

func TestTextMarshalerField(t *testing.T) {
	type S struct {
		Value textUpper `form:"value"`
	}

	encoded, err := Marshal(S{Value: "hello"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["value"]; len(got) != 1 || got[0] != "HELLO" {
		t.Errorf("value = %v, want [HELLO]", got)
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"value": {"WORLD"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value != "world" {
		t.Errorf("Value = %q, want world", decoded.Value)
	}
}

func TestTextMarshalerNilPointer(t *testing.T) {
	type S struct {
		Value *textUpper `form:"value"`
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"value": {"ABC"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value == nil || *decoded.Value != "abc" {
		t.Errorf("Value = %v, want pointer to abc", decoded.Value)
	}
}

func TestTextByteSlicePrecedence(t *testing.T) {
	type S struct {
		Value textBytes `form:"value"`
	}

	encoded, err := Marshal(S{Value: textBytes("hi")})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["value"]; len(got) != 1 || got[0] != "HI" {
		t.Errorf("value = %v, want [HI]", got)
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"value": {"HELLO"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if string(decoded.Value) != "hello" {
		t.Errorf("Value = %q, want hello", decoded.Value)
	}

	// Plain []byte still uses the string extension.
	type Plain struct {
		Value []byte `form:"value"`
	}
	plain, err := Marshal(Plain{Value: []byte("hi")})
	if err != nil {
		t.Fatalf("Marshal plain error: %v", err)
	}
	if got := plain["value"]; len(got) != 1 || got[0] != "hi" {
		t.Errorf("plain value = %v, want [hi]", got)
	}
}

func TestTextNetIP(t *testing.T) {
	type S struct {
		IP net.IP `form:"ip"`
	}

	ip := net.ParseIP("192.168.1.1")
	encoded, err := Marshal(S{IP: ip})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["ip"]; len(got) != 1 || got[0] != "192.168.1.1" {
		t.Errorf("ip = %v, want [192.168.1.1]", got)
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"ip": {"10.0.0.1"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !decoded.IP.Equal(net.ParseIP("10.0.0.1")) {
		t.Errorf("IP = %v, want 10.0.0.1", decoded.IP)
	}
}

func TestTextStruct(t *testing.T) {
	type S struct {
		Value textStruct `form:"value"`
	}

	encoded, err := Marshal(S{Value: textStruct{V: "hello"}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["value"]; len(got) != 1 || got[0] != "hello" {
		t.Errorf("value = %v, want [hello]", got)
	}
	if _, ok := encoded["value.V"]; ok {
		t.Error("value.V should not be encoded when TextMarshaler is implemented")
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"value": {"world"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value.V != "world" {
		t.Errorf("Value.V = %q, want world", decoded.Value.V)
	}
}

func TestCustomInterfacesBeatText(t *testing.T) {
	type S struct {
		Value textBoth `form:"value"`
	}

	// textBoth implements Marshaler/Unmarshaler and TextMarshaler/TextUnmarshaler.
	encoded, err := Marshal(S{Value: textBoth{M: "m", T: "t"}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["value"]; len(got) != 1 || got[0] != "custom" {
		t.Errorf("value = %v, want [custom]", got)
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"value": {"x"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value.M != "custom" || decoded.Value.T != "" {
		t.Errorf("Value = %#v, want M=custom T=empty", decoded.Value)
	}
}

type textBoth struct {
	M string
	T string
}

func (b textBoth) MarshalValues() ([]string, error) { return []string{"custom"}, nil }

func (b *textBoth) UnmarshalValues([]string) error {
	b.M = "custom"
	return nil
}

func (b textBoth) MarshalText() ([]byte, error) { return []byte("text"), nil }

func (b *textBoth) UnmarshalText([]byte) error {
	b.T = "text"
	return nil
}

func TestTextErrors(t *testing.T) {
	type S struct {
		Value failingText `form:"value"`
	}

	_, err := Marshal(S{})
	if err == nil {
		t.Fatal("expected text marshal error")
	}
	if !errors.Is(err, errSentinel) {
		t.Errorf("errors.Is(err, errSentinel) = false: %v", err)
	}

	var decoded S
	err = Unmarshal(map[string][]string{"value": {"x"}}, &decoded)
	if err == nil {
		t.Fatal("expected text unmarshal error")
	}
	if !errors.Is(err, errSentinel) {
		t.Errorf("errors.Is(err, errSentinel) = false: %v", err)
	}
}

type failingText struct{}

func (failingText) MarshalText() ([]byte, error) { return nil, errSentinel }

func (*failingText) UnmarshalText([]byte) error { return errSentinel }

type textPtrMarshaler string

func (t *textPtrMarshaler) MarshalText() ([]byte, error) {
	return []byte(strings.ToUpper(string(*t))), nil
}

func (t *textPtrMarshaler) UnmarshalText(data []byte) error {
	*t = textPtrMarshaler(strings.ToLower(string(data)))
	return nil
}

func TestTextPointerReceiver(t *testing.T) {
	type S struct {
		Value textPtrMarshaler `form:"value"`
	}

	encoded, err := Marshal(S{Value: "hi"})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["value"]; len(got) != 1 || got[0] != "HI" {
		t.Errorf("value = %v, want [HI]", got)
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"value": {"HELLO"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value != "hello" {
		t.Errorf("Value = %q, want hello", decoded.Value)
	}
}

func TestTextPointerField(t *testing.T) {
	type S struct {
		Value *textPtrMarshaler `form:"value"`
	}

	var decoded S
	if err := Unmarshal(map[string][]string{"value": {"ABC"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value == nil || *decoded.Value != "abc" {
		t.Errorf("Value = %v, want pointer to abc", decoded.Value)
	}

	existing := textPtrMarshaler("old")
	decoded.Value = &existing
	if err := Unmarshal(map[string][]string{"value": {"NEW"}}, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Value != &existing || *decoded.Value != "new" {
		t.Errorf("Value = %v, want existing pointer updated to new", decoded.Value)
	}
}

func TestTextMapValueCopy(t *testing.T) {
	type S struct {
		M map[string]textPtrMarshaler `form:"m"`
	}

	encoded, err := Marshal(S{M: map[string]textPtrMarshaler{"k": "hi"}})
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if got := encoded["m[k]"]; len(got) != 1 || got[0] != "HI" {
		t.Errorf("m[k] = %v, want [HI]", got)
	}

	var decoded S
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.M["k"] != "hi" {
		t.Errorf("M[k] = %q, want hi", decoded.M["k"])
	}
}
