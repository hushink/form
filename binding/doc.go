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

// Package binding binds net/http requests to Go values with multiple tag
// sources and configurable priority.
//
// A field may carry tags for several sources. The first source listed in
// WithTagPriority has the highest priority; lower-priority values are bound
// first and overwritten by higher-priority values.
//
//	type Search struct {
//		ID      int    `param:"id" json:"id"`
//		Keyword string `form:"keyword" query:"q"`
//		Page    int    `form:"page" json:"page"`
//		Token   string `header:"X-Token"`
//	}
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//		var search Search
//		if err := binding.Bind(r, &search); err != nil {
//			http.Error(w, err.Error(), http.StatusBadRequest)
//			return
//		}
//		// ...
//	}
//
// The default priority is param, form, json, xml, query, header, cookie.
// Map-style sources bind only fields tagged for that source; JSON and XML use
// standard encoding/json and encoding/xml field matching, where tags are
// optional. Use json:"-" or xml:"-" to exclude fields from body sources.
//
// Binding names come from each source's tag, so naming style is explicit per
// source; form.NamingStrategy is not applied to binding. WithCaseInsensitive
// optionally matches map-style sources without regard to case. Path parameters
// are still fetched with PathValue using the tag name, JSON is already
// case-insensitive, and XML remains case-sensitive.
//
// # Path parameters
//
// Path parameters are read through http.Request.PathValue, so the router must
// populate them (the standard library does this for ServeMux patterns). Empty
// path values are treated as absent, and param tags should target flat scalar
// fields.
//
// # JSON and XML
//
// JSON and XML sources apply only when the request Content-Type matches. The
// body is read fully, restored for downstream handlers, and decoded with
// encoding/json or encoding/xml when that source's turn in the priority order
// arrives. Empty bodies are skipped; malformed bodies stop binding. The read
// size is limited by WithMaxBodyBytes (32 MiB by default).
//
// # Defaults and required fields
//
// Defaults are applied before source values, so any actual value, including an
// explicit empty value, overrides a default.
//
// When a field has tags for several sources, put `required` on the
// highest-priority tag. A non-zero value already bound from a lower-priority
// source or a default satisfies it. The `default` and `required` options are
// recognized on map-style tags (param, form, query, header, cookie); JSON/XML
// fields can add a form tag when those checks are needed.
//
// # Errors
//
// Decode errors from every source are merged into a form.DecodeErrors. Request
// parsing errors stop binding and wrap the underlying error. ErrBodyTooLarge
// reports a body that exceeds WithMaxBodyBytes.
package binding
