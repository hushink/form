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

// Package output converts Go values into HTTP-oriented output formats.
//
// Form, Query, and Header use the form, query, and header struct tags
// respectively. They include only fields carrying that source's tag by
// default; pass form.WithMode(form.ModeImplicit) to include untagged exported
// fields, or supply other form options to change naming and namespace behavior.
// As with binding, named nested parent fields need the source tag for nested
// fields to be included.
//
// JSON and XML use encoding/json and encoding/xml with their native tags and
// matching rules.
//
// The WriteForm, WriteJSON, and WriteXML helpers write a response body and set
// Content-Type only when it is not already set. SetHeader applies header-tagged
// fields to an http.ResponseWriter. Status codes are left to the caller.
package output
