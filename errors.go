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
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrInvalidValue is returned when the encode input is nil.
	ErrInvalidValue = errors.New("form: input value is nil")
	// ErrNilPointer is returned when a nil pointer is passed to Decode.
	ErrNilPointer = errors.New("form: destination must be a non-nil pointer")
	// ErrNotPointer is returned when a non-pointer is passed to Decode.
	ErrNotPointer = errors.New("form: destination must be a pointer")

	// ErrDuplicateTag is returned (wrapped) when WithRejectDuplicateTags(true)
	// is enabled and a struct has two fields with the same tag name.
	ErrDuplicateTag = errors.New("form: duplicate tag name")
)

// DecodeErrors collects decoding errors keyed by the form field path
// (for example "Address[0].Name"). Decoding continues after an error so that
// all invalid fields are reported at once.
type DecodeErrors map[string]error

// Error implements the error interface. Keys are sorted for deterministic
// output.
func (e DecodeErrors) Error() string { return formatErrorMap(e) }

// Unwrap returns the underlying errors in sorted key order, enabling
// errors.Is and errors.As.
func (e DecodeErrors) Unwrap() []error {
	keys := sortedKeys(e)
	out := make([]error, 0, len(keys))
	for _, k := range keys {
		out = append(out, e[k])
	}
	return out
}

// EncodeErrors collects encoding errors keyed by the form field path.
// Encoding continues after an error; Encode returns the partially encoded map
// together with these errors.
type EncodeErrors map[string]error

// Error implements the error interface. Keys are sorted for deterministic
// output.
func (e EncodeErrors) Error() string { return formatErrorMap(e) }

// Unwrap returns the underlying errors in sorted key order, enabling
// errors.Is and errors.As.
func (e EncodeErrors) Unwrap() []error {
	keys := sortedKeys(e)
	out := make([]error, 0, len(keys))
	for _, k := range keys {
		out = append(out, e[k])
	}
	return out
}

func formatErrorMap[E error](m map[string]E) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(m[k].Error())
	}
	return b.String()
}

func sortedKeys[E error](m map[string]E) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// UnknownKeyError reports an input key that did not match any destination
// field. It is returned when WithStrictUnknownKeys(true) is enabled.
type UnknownKeyError struct {
	Key string
}

func (e UnknownKeyError) Error() string {
	return fmt.Sprintf("form: unknown key %q", e.Key)
}

// EmptyFieldError reports a field tagged with "required" that was missing or
// had only empty values.
type EmptyFieldError struct {
	Key string
}

func (e EmptyFieldError) Error() string {
	return fmt.Sprintf("form: required field %q is empty", e.Key)
}
