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
	"unicode"
)

// NamingStrategy converts an exported Go field name to the external key used
// for a field whose tag does not specify an explicit name.
//
// A strategy is applied by Encoder and Decoder to fields without a name, for
// example an untagged field or a tag containing only options such as
// `form:",required"`. An explicit tag name always wins. Returning an empty
// string falls back to the original Go field name. A strategy should be
// deterministic and safe for concurrent use, because struct metadata may be
// built concurrently.
type NamingStrategy func(string) string

// SnakeCase converts Go-style names to lower snake_case.
//
// Examples: UserID -> user_id, HTTPServer -> http_server,
// UTF8String -> utf8_string.
func SnakeCase(name string) string {
	words := splitFieldWords(name)
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}
	return strings.Join(words, "_")
}

// KebabCase converts Go-style names to lower kebab-case.
//
// Examples: UserID -> user-id, HTTPServer -> http-server,
// UTF8String -> utf8-string.
func KebabCase(name string) string {
	words := splitFieldWords(name)
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}
	return strings.Join(words, "-")
}

// LowerCamelCase converts Go-style names to lowerCamelCase.
//
// Examples: UserName -> userName, HTTPServer -> httpServer,
// UserID -> userID.
func LowerCamelCase(name string) string {
	words := splitFieldWords(name)
	if len(words) == 0 {
		return ""
	}

	var b strings.Builder
	for i, word := range words {
		if i == 0 {
			b.WriteString(strings.ToLower(word))
			continue
		}
		if isUpperWord(word) {
			b.WriteString(word)
			continue
		}
		b.WriteString(upperFirst(strings.ToLower(word)))
	}
	return b.String()
}

// splitFieldWords splits a Go field name into words. Existing underscores,
// dashes, and whitespace are treated as separators; CamelCase acronym
// boundaries are detected as well.
func splitFieldWords(name string) []string {
	var words []string
	for _, part := range strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	}) {
		words = append(words, splitCamelWord(part)...)
	}
	return words
}

func splitCamelWord(name string) []string {
	if name == "" {
		return nil
	}
	runes := []rune(name)
	words := make([]string, 0, 4)
	start := 0
	for i := 1; i < len(runes); i++ {
		prev := runes[i-1]
		cur := runes[i]
		if !unicode.IsUpper(cur) {
			continue
		}
		boundary := unicode.IsLower(prev) || unicode.IsDigit(prev)
		if !boundary && unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
			boundary = true
		}
		if boundary {
			words = append(words, string(runes[start:i]))
			start = i
		}
	}
	words = append(words, string(runes[start:]))
	return words
}

func isUpperWord(word string) bool {
	hasLetter := false
	for _, r := range word {
		if !unicode.IsLetter(r) {
			continue
		}
		hasLetter = true
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return hasLetter
}

func upperFirst(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func applyNaming(strategy NamingStrategy, name string) string {
	if strategy == nil {
		return name
	}
	if converted := strategy(name); converted != "" {
		return converted
	}
	return name
}
