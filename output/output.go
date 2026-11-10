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

package output

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/textproto"
	"net/url"

	"github.com/hushink/form"
)

const (
	tagForm   = "form"
	tagQuery  = "query"
	tagHeader = "header"

	contentTypeForm = "application/x-www-form-urlencoded; charset=utf-8"
	contentTypeJSON = "application/json; charset=utf-8"
	contentTypeXML  = "application/xml; charset=utf-8"
)

// Form encodes v using form tags. Only fields carrying a form tag are
// included by default; use form.WithMode(form.ModeImplicit) to include
// untagged exported fields. Additional form options may be supplied to change
// the tag name, mode, naming strategy, or other encoder behavior.
func Form(v any, opts ...form.Option) (url.Values, error) {
	return encodeValues(tagForm, v, opts...)
}

// Query encodes v using query tags. Only fields carrying a query tag are
// included by default; use form.WithMode(form.ModeImplicit) to include
// untagged exported fields.
func Query(v any, opts ...form.Option) (url.Values, error) {
	return encodeValues(tagQuery, v, opts...)
}

// QueryString encodes v using query tags and returns an escaped URL query
// string. Keys are sorted by url.Values.Encode.
func QueryString(v any, opts ...form.Option) (string, error) {
	values, err := Query(v, opts...)
	if err != nil {
		return "", err
	}
	return values.Encode(), nil
}

// Header encodes v using header tags into an http.Header with canonical MIME
// header names. Only fields carrying a header tag are included by default.
// Header names should be flat; nested paths are not valid HTTP header names.
func Header(v any, opts ...form.Option) (http.Header, error) {
	values, err := encodeValues(tagHeader, v, opts...)
	if err != nil {
		return nil, err
	}
	header := make(http.Header, len(values))
	for key, entries := range values {
		canonical := textproto.CanonicalMIMEHeaderKey(key)
		header[canonical] = append(header[canonical], entries...)
	}
	return header, nil
}

// JSON encodes v with encoding/json.
func JSON(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("output: cannot encode JSON: %w", err)
	}
	return data, nil
}

// XML encodes v with encoding/xml.
func XML(v any) ([]byte, error) {
	data, err := xml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("output: cannot encode XML: %w", err)
	}
	return data, nil
}

// WriteForm writes v as an URL-encoded form body. It sets Content-Type only
// when the response does not already have one. The status code is left to the
// caller.
func WriteForm(w http.ResponseWriter, v any, opts ...form.Option) error {
	values, err := Form(v, opts...)
	if err != nil {
		return err
	}
	setContentType(w, contentTypeForm)
	if _, err := io.WriteString(w, values.Encode()); err != nil {
		return fmt.Errorf("output: cannot write response: %w", err)
	}
	return nil
}

// WriteJSON writes v as a JSON response body. It sets Content-Type only when
// the response does not already have one. The status code is left to the
// caller.
func WriteJSON(w http.ResponseWriter, v any) error {
	data, err := JSON(v)
	if err != nil {
		return err
	}
	setContentType(w, contentTypeJSON)
	return writeBody(w, data)
}

// WriteXML writes v as an XML response body. It sets Content-Type only when
// the response does not already have one. The status code is left to the
// caller.
func WriteXML(w http.ResponseWriter, v any) error {
	data, err := XML(v)
	if err != nil {
		return err
	}
	setContentType(w, contentTypeXML)
	return writeBody(w, data)
}

// SetHeader sets response headers from fields tagged with header tags.
// Existing values for the same header are replaced.
func SetHeader(w http.ResponseWriter, v any, opts ...form.Option) error {
	header, err := Header(v, opts...)
	if err != nil {
		return err
	}
	maps.Copy(w.Header(), header)
	return nil
}

var defaultEncoders = map[string]*form.Encoder{
	tagForm:   form.NewEncoder(form.WithTagName(tagForm), form.WithMode(form.ModeExplicitEmbedded)),
	tagQuery:  form.NewEncoder(form.WithTagName(tagQuery), form.WithMode(form.ModeExplicitEmbedded)),
	tagHeader: form.NewEncoder(form.WithTagName(tagHeader), form.WithMode(form.ModeExplicitEmbedded)),
}

func encodeValues(tag string, v any, opts ...form.Option) (url.Values, error) {
	if len(opts) == 0 {
		return defaultEncoders[tag].Encode(v)
	}
	options := make([]form.Option, 0, len(opts)+2)
	options = append(options,
		form.WithTagName(tag),
		form.WithMode(form.ModeExplicitEmbedded),
	)
	options = append(options, opts...)
	return form.NewEncoder(options...).Encode(v)
}

func setContentType(w http.ResponseWriter, contentType string) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", contentType)
	}
}

func writeBody(w http.ResponseWriter, data []byte) error {
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("output: cannot write response: %w", err)
	}
	return nil
}
