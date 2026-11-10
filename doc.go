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

// Package form converts between Go values and map[string][]string, which is
// the shape of url.Values used by HTTP forms and query parameters. Structs are
// encoded by field; other values use the root path (the empty key, or "[key]"
// for maps and "[i]" for slices/arrays).
//
// # Basic usage
//
//	values, err := form.Marshal(user)
//	err = form.Unmarshal(values, &decoded)
//
// # Paths
//
// Nested values use dotted paths and bracket indexes:
//
//	Name
//	Address.City
//	Items[0].Name
//	Filter[key]
//	NestedMap[key][key2]
//
// # Struct tags
//
//	form:"name"
//	form:"name,omitempty"
//	form:"name,default=value"
//	form:"name,default=a|b|c"
//	form:"-"
//
// # Naming and case
//
// Field names come from struct tags. WithNamingStrategy applies SnakeCase,
// KebabCase, LowerCamelCase, or a custom conversion to fields that do not
// provide an explicit tag name; explicit names always win. WithCaseInsensitive
// optionally matches decoding keys case-insensitively, with exact matches
// taking precedence.
//
// # Errors
//
// Encoding and decoding continue after field errors. All errors are returned
// as DecodeErrors or EncodeErrors, keyed by field path. Both types implement
// Unwrap() []error, so errors.Is and errors.As work.
//
// # Custom types
//
// Types may implement Marshaler and Unmarshaler to control their own
// conversion. Nil pointers are never passed to MarshalValues.
//
// # HTTP binding
//
// The binding subpackage (github.com/hushink/form/binding) decodes net/http
// requests from param, form, JSON, XML, query, header, and cookie sources. A
// field may carry tags for several sources, and binding.WithTagPriority
// controls precedence.
//
// The output subpackage (github.com/hushink/form/output) encodes structs into
// form values, query strings, headers, JSON, and XML, with HTTP response
// helpers.
//
// # Concurrency
//
// Encoder and Decoder cache struct metadata and are safe for concurrent use
// after construction. Reuse them when encoding or decoding the same types.
// binding.Binder is also safe for concurrent use after construction.
package form
