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
	"reflect"
	"testing"
)

func TestNamingStrategies(t *testing.T) {
	tests := []struct {
		name       string
		snake      string
		kebab      string
		lowerCamel string
	}{
		{name: "UserName", snake: "user_name", kebab: "user-name", lowerCamel: "userName"},
		{name: "UserID", snake: "user_id", kebab: "user-id", lowerCamel: "userID"},
		{name: "HTTPServer", snake: "http_server", kebab: "http-server", lowerCamel: "httpServer"},
		{name: "APIClient", snake: "api_client", kebab: "api-client", lowerCamel: "apiClient"},
		{name: "UTF8String", snake: "utf8_string", kebab: "utf8-string", lowerCamel: "utf8String"},
		{name: "ID", snake: "id", kebab: "id", lowerCamel: "id"},
		{name: "user_name", snake: "user_name", kebab: "user-name", lowerCamel: "userName"},
		{name: "user-id", snake: "user_id", kebab: "user-id", lowerCamel: "userId"},
		{name: "", snake: "", kebab: "", lowerCamel: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := SnakeCase(test.name); got != test.snake {
				t.Errorf("SnakeCase(%q) = %q, want %q", test.name, got, test.snake)
			}
			if got := KebabCase(test.name); got != test.kebab {
				t.Errorf("KebabCase(%q) = %q, want %q", test.name, got, test.kebab)
			}
			if got := LowerCamelCase(test.name); got != test.lowerCamel {
				t.Errorf("LowerCamelCase(%q) = %q, want %q", test.name, got, test.lowerCamel)
			}
		})
	}
}

func TestNamingStrategyRoundTrip(t *testing.T) {
	type Address struct {
		CityName string
	}
	type target struct {
		UserID     int
		HTTPServer string
		Address    Address
		Explicit   string `form:"custom_name"`
		WithOpts   string `form:",omitempty"`
		Skipped    string `form:"-"`
	}

	input := target{
		UserID:     7,
		HTTPServer: "api",
		Address:    Address{CityName: "shanghai"},
		Explicit:   "e",
		WithOpts:   "o",
		Skipped:    "s",
	}

	values, err := NewEncoder(WithNamingStrategy(SnakeCase)).Encode(input)
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	wantValues := map[string][]string{
		"user_id":           {"7"},
		"http_server":       {"api"},
		"address.city_name": {"shanghai"},
		"custom_name":       {"e"},
		"with_opts":         {"o"},
	}
	if !reflect.DeepEqual(values, wantValues) {
		t.Errorf("values = %#v, want %#v", values, wantValues)
	}

	wantDecoded := input
	wantDecoded.Skipped = ""
	var decoded target
	if err := NewDecoder(WithNamingStrategy(SnakeCase)).Decode(values, &decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if !reflect.DeepEqual(decoded, wantDecoded) {
		t.Errorf("decoded = %+v, want %+v", decoded, wantDecoded)
	}

	var legacy target
	if err := NewDecoder(WithNamingStrategy(SnakeCase)).Decode(map[string][]string{
		"UserID": {"9"},
	}, &legacy); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if legacy.UserID != 0 {
		t.Errorf("UserID = %d, want 0 when only the Go field name is present", legacy.UserID)
	}
}

func TestNamingStrategyWithOptionsAndRequired(t *testing.T) {
	type target struct {
		UserName string `form:",required"`
		Page     int    `form:",default=1"`
	}

	var missing target
	err := NewDecoder(WithNamingStrategy(SnakeCase)).Decode(nil, &missing)
	if err == nil {
		t.Fatal("expected required error")
	}
	var empty EmptyFieldError
	if !errors.As(err, &empty) {
		t.Fatalf("error = %v, want EmptyFieldError", err)
	}
	if empty.Key != "user_name" {
		t.Errorf("Key = %q, want user_name", empty.Key)
	}

	var decoded target
	if err := NewDecoder(WithNamingStrategy(SnakeCase)).Decode(map[string][]string{
		"user_name": {"alice"},
	}, &decoded); err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if decoded.UserName != "alice" || decoded.Page != 1 {
		t.Errorf("decoded = %+v, want UserName=alice Page=1", decoded)
	}
}

func TestNamingStrategyDuplicateDetection(t *testing.T) {
	type target struct {
		Alpha int
		Beta  int
	}

	collide := func(string) string { return "same" }
	_, err := NewEncoder(WithNamingStrategy(collide), WithRejectDuplicateTags(true)).Encode(target{})
	if !errors.Is(err, ErrDuplicateTag) {
		t.Errorf("error = %v, want ErrDuplicateTag", err)
	}
}

func TestNamingStrategyExplicitMode(t *testing.T) {
	type target struct {
		Untagged string
		WithOpts string `form:",omitempty"`
		Named    string `form:"named"`
	}

	values, err := NewEncoder(
		WithMode(ModeExplicit),
		WithNamingStrategy(SnakeCase),
	).Encode(target{Untagged: "u", WithOpts: "o", Named: "n"})
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	want := map[string][]string{"with_opts": {"o"}, "named": {"n"}}
	if !reflect.DeepEqual(values, want) {
		t.Errorf("values = %#v, want %#v", values, want)
	}
}

func TestNamingStrategyEmptyResultFallsBack(t *testing.T) {
	type target struct {
		UserName string
	}

	values, err := NewEncoder(WithNamingStrategy(func(string) string {
		return ""
	})).Encode(target{UserName: "alice"})
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}
	if got := values["UserName"]; !reflect.DeepEqual(got, []string{"alice"}) {
		t.Errorf("UserName = %v, want [alice]", got)
	}
}
