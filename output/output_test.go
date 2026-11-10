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
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/hushink/form"
)

type testAddress struct {
	City string `form:"city" query:"city"`
}

type testTarget struct {
	Name      string `form:"name" query:"q"`
	Untagged  string
	FormOnly  int         `form:"form_only"`
	QueryOnly int         `query:"query_only"`
	Address   testAddress `form:"address" query:"address"`
}

func testValue() testTarget {
	return testTarget{
		Name:      "alice",
		Untagged:  "u",
		FormOnly:  1,
		QueryOnly: 2,
		Address:   testAddress{City: "shanghai"},
	}
}

func TestFormAndQuery(t *testing.T) {
	value := testValue()

	formValues, err := Form(value)
	if err != nil {
		t.Fatalf("Form error: %v", err)
	}
	wantForm := map[string][]string{
		"name":         {"alice"},
		"form_only":    {"1"},
		"address.city": {"shanghai"},
	}
	if !reflect.DeepEqual(map[string][]string(formValues), wantForm) {
		t.Errorf("Form = %#v, want %#v", formValues, wantForm)
	}

	queryValues, err := Query(value)
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	wantQuery := map[string][]string{
		"q":            {"alice"},
		"query_only":   {"2"},
		"address.city": {"shanghai"},
	}
	if !reflect.DeepEqual(map[string][]string(queryValues), wantQuery) {
		t.Errorf("Query = %#v, want %#v", queryValues, wantQuery)
	}

	queryString, err := QueryString(value)
	if err != nil {
		t.Fatalf("QueryString error: %v", err)
	}
	if want := "address.city=shanghai&q=alice&query_only=2"; queryString != want {
		t.Errorf("QueryString = %q, want %q", queryString, want)
	}
}

func TestFormImplicitModeAndNaming(t *testing.T) {
	values, err := Form(testValue(), form.WithMode(form.ModeImplicit))
	if err != nil {
		t.Fatalf("Form error: %v", err)
	}
	if got := values["Untagged"]; !reflect.DeepEqual(got, []string{"u"}) {
		t.Errorf("Untagged = %v, want [u]", got)
	}
	if got := values["QueryOnly"]; !reflect.DeepEqual(got, []string{"2"}) {
		t.Errorf("QueryOnly = %v, want [2]", got)
	}

	type named struct {
		UserName string `form:",required"`
	}
	namedValues, err := Form(named{UserName: "bob"}, form.WithNamingStrategy(form.SnakeCase))
	if err != nil {
		t.Fatalf("Form error: %v", err)
	}
	if got := namedValues["user_name"]; !reflect.DeepEqual(got, []string{"bob"}) {
		t.Errorf("user_name = %v, want [bob]", got)
	}
}

func TestHeader(t *testing.T) {
	type target struct {
		Token  string   `header:"x-token"`
		Accept []string `header:"Accept"`
	}

	header, err := Header(target{Token: "abc", Accept: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("Header error: %v", err)
	}
	if got := header.Get("X-Token"); got != "abc" {
		t.Errorf("X-Token = %q, want abc", got)
	}
	if got := header["Accept"]; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("Accept = %v, want [a b]", got)
	}
	lowerKey := "x-token"
	if _, ok := header[lowerKey]; ok {
		t.Error("header names should be canonicalized")
	}

	recorder := httptest.NewRecorder()
	if err := SetHeader(recorder, target{Token: "abc"}); err != nil {
		t.Fatalf("SetHeader error: %v", err)
	}
	if got := recorder.Header().Get("X-Token"); got != "abc" {
		t.Errorf("response X-Token = %q, want abc", got)
	}
}

func TestJSONAndXML(t *testing.T) {
	type jsonTarget struct {
		Name string `json:"name"`
		Page int    `json:"page"`
	}
	data, err := JSON(jsonTarget{Name: "alice", Page: 2})
	if err != nil {
		t.Fatalf("JSON error: %v", err)
	}
	if got := string(data); got != `{"name":"alice","page":2}` {
		t.Errorf("JSON = %s", got)
	}

	type xmlTarget struct {
		XMLName xml.Name `xml:"target"`
		Name    string   `xml:"name"`
		Page    int      `xml:"page"`
	}
	data, err = XML(xmlTarget{Name: "alice", Page: 2})
	if err != nil {
		t.Fatalf("XML error: %v", err)
	}
	if got := string(data); got != `<target><name>alice</name><page>2</page></target>` {
		t.Errorf("XML = %s", got)
	}

	type unsupported struct {
		Fn func() `json:"fn"`
	}
	if _, err := JSON(unsupported{}); err == nil {
		t.Error("expected JSON error for function field")
	}
}

func TestWriters(t *testing.T) {
	t.Run("form", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		if err := WriteForm(recorder, testValue()); err != nil {
			t.Fatalf("WriteForm error: %v", err)
		}
		if got := recorder.Header().Get("Content-Type"); got != contentTypeForm {
			t.Errorf("Content-Type = %q, want %q", got, contentTypeForm)
		}
		if got := recorder.Body.String(); got != "address.city=shanghai&form_only=1&name=alice" {
			t.Errorf("body = %q", got)
		}
	})

	t.Run("json", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		if err := WriteJSON(recorder, map[string]int{"page": 2}); err != nil {
			t.Fatalf("WriteJSON error: %v", err)
		}
		if got := recorder.Header().Get("Content-Type"); got != contentTypeJSON {
			t.Errorf("Content-Type = %q, want %q", got, contentTypeJSON)
		}
		if got := recorder.Body.String(); got != `{"page":2}` {
			t.Errorf("body = %q", got)
		}
	})

	t.Run("xml", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		type target struct {
			Name string `xml:"name"`
		}
		if err := WriteXML(recorder, target{Name: "alice"}); err != nil {
			t.Fatalf("WriteXML error: %v", err)
		}
		if got := recorder.Header().Get("Content-Type"); got != contentTypeXML {
			t.Errorf("Content-Type = %q, want %q", got, contentTypeXML)
		}
		if got := recorder.Body.String(); got != `<target><name>alice</name></target>` {
			t.Errorf("body = %q", got)
		}
	})

	t.Run("existing content type", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		recorder.Header().Set("Content-Type", "text/plain")
		if err := WriteJSON(recorder, map[string]int{"page": 2}); err != nil {
			t.Fatalf("WriteJSON error: %v", err)
		}
		if got := recorder.Header().Get("Content-Type"); got != "text/plain" {
			t.Errorf("Content-Type = %q, want text/plain", got)
		}
	})
}

func TestFormEncodeError(t *testing.T) {
	type target struct {
		Fn func() `form:"fn"`
	}

	_, err := Form(target{})
	if err == nil {
		t.Fatal("expected encode error")
	}
	var encodeErrs form.EncodeErrors
	if !errors.As(err, &encodeErrs) {
		t.Errorf("error = %v, want form.EncodeErrors", err)
	}
}

func TestQueryStringError(t *testing.T) {
	type target struct {
		Fn func() `query:"fn"`
	}

	if _, err := QueryString(target{}); err == nil {
		t.Fatal("expected encode error")
	}
}

func TestHeaderError(t *testing.T) {
	type target struct {
		Fn func() `header:"X-Fn"`
	}

	if _, err := Header(target{}); err == nil {
		t.Fatal("expected encode error")
	}
	if err := SetHeader(httptest.NewRecorder(), target{}); err == nil {
		t.Fatal("expected encode error")
	}
}

func TestWriteMarshalErrors(t *testing.T) {
	type jsonTarget struct {
		Fn func() `json:"fn"`
	}
	recorder := httptest.NewRecorder()
	if err := WriteJSON(recorder, jsonTarget{}); err == nil {
		t.Fatal("expected JSON encode error")
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", recorder.Body.String())
	}

	type xmlTarget struct {
		Fn func() `xml:"fn"`
	}
	if err := WriteXML(recorder, xmlTarget{}); err == nil {
		t.Fatal("expected XML encode error")
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", recorder.Body.String())
	}
}

type failingResponseWriter struct {
	header http.Header
}

func (w *failingResponseWriter) Header() http.Header { return w.header }
func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
func (w *failingResponseWriter) WriteHeader(int) {}

func TestWriteError(t *testing.T) {
	writer := &failingResponseWriter{header: make(http.Header)}
	err := WriteJSON(writer, map[string]int{"page": 2})
	if err == nil || !strings.Contains(err.Error(), "cannot write response") {
		t.Errorf("error = %v, want write error", err)
	}

	writer = &failingResponseWriter{header: make(http.Header)}
	err = WriteForm(writer, testValue())
	if err == nil || !strings.Contains(err.Error(), "cannot write response") {
		t.Errorf("error = %v, want write error", err)
	}
}
