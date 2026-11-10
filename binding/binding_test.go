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

package binding

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/hushink/form"
)

func urlencodedRequest(t *testing.T, target, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func multipartRequest(t *testing.T, target string, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField(%q) error: %v", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart Close error: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func jsonRequest(t *testing.T, target, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func xmlRequest(t *testing.T, target, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/xml")
	return req
}

func TestBindQuery(t *testing.T) {
	type target struct {
		Name string   `query:"name"`
		Page int      `query:"page"`
		Tags []string `query:"tags"`
	}

	req := httptest.NewRequest(http.MethodGet, "/?name=alice&page=2&tags=a&tags=b", nil)
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	want := target{Name: "alice", Page: 2, Tags: []string{"a", "b"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestBindParam(t *testing.T) {
	type target struct {
		ID int `param:"id" form:"id" query:"id"`
	}

	req := urlencodedRequest(t, "/?id=1", "id=2")
	req.SetPathValue("id", "42")
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("ID = %d, want 42 (path parameter has highest priority)", got.ID)
	}

	req = urlencodedRequest(t, "/?id=1", "id=2")
	req.SetPathValue("id", "42")
	got = target{}
	if err := Bind(req, &got, WithTagPriority(SourceQuery, SourceForm, SourceParam)); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 1 {
		t.Errorf("ID = %d, want 1 (custom query priority)", got.ID)
	}

	req = urlencodedRequest(t, "/?id=1", "id=2")
	got = target{}
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 2 {
		t.Errorf("ID = %d, want 2 (missing path parameter falls back to form)", got.ID)
	}
}

func TestBindParamRequired(t *testing.T) {
	type target struct {
		ID int `param:"id,required"`
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	var got target
	err := Bind(req, &got)
	if err == nil {
		t.Fatal("expected required error")
	}
	var empty form.EmptyFieldError
	if !errors.As(err, &empty) {
		t.Fatalf("error = %v, want EmptyFieldError", err)
	}

	req.SetPathValue("id", "5")
	got = target{}
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 5 {
		t.Errorf("ID = %d, want 5", got.ID)
	}
}

func TestBindParamCyclicType(t *testing.T) {
	type node struct {
		ID   int `param:"id"`
		Next *node
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	var got node
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
}

func TestBindParamDefaultAndEmpty(t *testing.T) {
	type target struct {
		ID int `param:"id,default=7" query:"id"`
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 7 {
		t.Errorf("ID = %d, want 7 (param default)", got.ID)
	}

	req = urlencodedRequest(t, "/?id=1", "")
	req.SetPathValue("id", "")
	got = target{}
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 1 {
		t.Errorf("ID = %d, want 1 (empty path value treated as absent)", got.ID)
	}
}

func TestBindJSON(t *testing.T) {
	type target struct {
		Name string `json:"name" query:"name"`
		Page int    `json:"page" query:"page"`
	}

	const body = `{"name":"alice","page":3}`
	req := jsonRequest(t, "/?name=query&page=1", body)
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Name != "alice" || got.Page != 3 {
		t.Errorf("got %+v, want Name=alice Page=3", got)
	}

	restored, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll restored body error: %v", err)
	}
	if string(restored) != body {
		t.Errorf("restored body = %q, want %q", restored, body)
	}

	req = jsonRequest(t, "/?name=query&page=1", body)
	got = target{}
	if err := Bind(req, &got, WithTagPriority(SourceQuery, SourceJSON)); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Name != "query" || got.Page != 1 {
		t.Errorf("got %+v, want query priority Name=query Page=1", got)
	}

	req = jsonRequest(t, "/?name=query&page=1", "")
	got = target{}
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Name != "query" || got.Page != 1 {
		t.Errorf("got %+v, want query fallback for empty JSON body", got)
	}

	req = jsonRequest(t, "/", `{"name":`)
	got = target{}
	err = Bind(req, &got)
	if err == nil || !strings.Contains(err.Error(), "JSON body") {
		t.Errorf("error = %v, want JSON parse error", err)
	}

	req = jsonRequest(t, "/", `{"page":"bad"}`)
	got = target{}
	err = Bind(req, &got)
	if err == nil || !strings.Contains(err.Error(), "JSON body") {
		t.Errorf("error = %v, want JSON type error", err)
	}

	req = jsonRequest(t, "/", body)
	req.Header.Set("Content-Type", "application/problem+json")
	got = target{}
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind +json error: %v", err)
	}
	if got.Name != "alice" || got.Page != 3 {
		t.Errorf("got %+v, want Name=alice Page=3 for +json", got)
	}
}

func TestBindJSONDefaultsAndRequired(t *testing.T) {
	t.Run("default from form tag", func(t *testing.T) {
		type target struct {
			Name string `form:"name,default=guest" json:"name"`
		}
		req := jsonRequest(t, "/", `{}`)
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Name != "guest" {
			t.Errorf("Name = %q, want guest", got.Name)
		}

		req = jsonRequest(t, "/", `{"name":"alice"}`)
		got = target{}
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Name != "alice" {
			t.Errorf("Name = %q, want alice", got.Name)
		}
	})

	t.Run("required from form tag", func(t *testing.T) {
		type target struct {
			Name string `form:"name,required" json:"name"`
		}
		req := jsonRequest(t, "/", `{"name":"alice"}`)
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}

		req = jsonRequest(t, "/", `{}`)
		got = target{}
		err := Bind(req, &got)
		if err == nil {
			t.Fatal("expected required error")
		}
		var empty form.EmptyFieldError
		if !errors.As(err, &empty) {
			t.Fatalf("error = %v, want EmptyFieldError", err)
		}
	})
}

func TestBindXML(t *testing.T) {
	type target struct {
		Name string `xml:"name" query:"name"`
		Page int    `xml:"page" query:"page"`
	}

	const body = `<data><name>alice</name><page>3</page></data>`
	req := xmlRequest(t, "/?name=query&page=1", body)
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Name != "alice" || got.Page != 3 {
		t.Errorf("got %+v, want Name=alice Page=3", got)
	}

	req = xmlRequest(t, "/?name=query&page=1", body)
	got = target{}
	if err := Bind(req, &got, WithTagPriority(SourceQuery, SourceXML)); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Name != "query" || got.Page != 1 {
		t.Errorf("got %+v, want query priority Name=query Page=1", got)
	}

	req = xmlRequest(t, "/?name=query&page=1", "")
	got = target{}
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Name != "query" || got.Page != 1 {
		t.Errorf("got %+v, want query fallback for empty XML body", got)
	}

	req = xmlRequest(t, "/", `<data><name>alice</name>`)
	got = target{}
	err := Bind(req, &got)
	if err == nil || !strings.Contains(err.Error(), "XML body") {
		t.Errorf("error = %v, want XML parse error", err)
	}

	req = xmlRequest(t, "/", body)
	req.Header.Set("Content-Type", "text/xml")
	got = target{}
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind text/xml error: %v", err)
	}
	if got.Name != "alice" || got.Page != 3 {
		t.Errorf("got %+v, want Name=alice Page=3 for text/xml", got)
	}
}

func TestBindBodyTooLarge(t *testing.T) {
	type target struct {
		Name string `json:"name"`
	}
	req := jsonRequest(t, "/", `{"name":"alice"}`)
	var got target
	err := Bind(req, &got, WithMaxBodyBytes(4))
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Errorf("error = %v, want ErrBodyTooLarge", err)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestBindBodyReadError(t *testing.T) {
	type target struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/", failingReader{})
	req.Header.Set("Content-Type", "application/json")
	var got target
	err := Bind(req, &got)
	if err == nil || !strings.Contains(err.Error(), "read request body") {
		t.Errorf("error = %v, want body read error", err)
	}
}

func TestBindJSONOnlyParsesJSONContentType(t *testing.T) {
	type target struct {
		Name string `form:"name" json:"name"`
	}
	req := urlencodedRequest(t, "/", "name=form")
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Name != "form" {
		t.Errorf("Name = %q, want form", got.Name)
	}
}

func TestBindAllSourcesPriority(t *testing.T) {
	type target struct {
		ID int `param:"id" form:"id" json:"id" query:"id" header:"X-Id" cookie:"id"`
	}

	req := jsonRequest(t, "/?id=1", `{"id":4}`)
	req.SetPathValue("id", "5")
	req.Header.Set("X-Id", "3")
	req.AddCookie(&http.Cookie{Name: "id", Value: "2"})

	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 5 {
		t.Errorf("ID = %d, want 5 (default param priority)", got.ID)
	}

	req = jsonRequest(t, "/?id=1", `{"id":4}`)
	req.SetPathValue("id", "5")
	req.Header.Set("X-Id", "3")
	req.AddCookie(&http.Cookie{Name: "id", Value: "2"})
	got = target{}
	if err := Bind(req, &got, WithTagPriority(
		SourceCookie, SourceHeader, SourceQuery, SourceJSON, SourceForm, SourceParam,
	)); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 2 {
		t.Errorf("ID = %d, want 2 (custom cookie priority)", got.ID)
	}
}

func TestBindFormVariants(t *testing.T) {
	type target struct {
		Name string `form:"name"`
		Page int    `form:"page"`
	}
	want := target{Name: "alice", Page: 3}

	t.Run("urlencoded", func(t *testing.T) {
		req := urlencodedRequest(t, "/", "name=alice&page=3")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("multipart", func(t *testing.T) {
		req := multipartRequest(t, "/", map[string]string{"name": "alice", "page": "3"})
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

func TestBindTagPriority(t *testing.T) {
	type target struct {
		Name string `form:"name" query:"name"`
		Page int    `form:"page" query:"page"`
	}

	t.Run("default form wins", func(t *testing.T) {
		req := urlencodedRequest(t, "/?name=query&page=1", "name=form&page=2")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{Name: "form", Page: 2}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("query priority wins", func(t *testing.T) {
		req := urlencodedRequest(t, "/?name=query&page=1", "name=form&page=2")
		var got target
		if err := Bind(req, &got, WithTagPriority(SourceQuery, SourceForm)); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{Name: "query", Page: 1}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("query fallback", func(t *testing.T) {
		req := urlencodedRequest(t, "/?name=query&page=1", "")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{Name: "query", Page: 1}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("mixed fallback", func(t *testing.T) {
		req := urlencodedRequest(t, "/?name=query", "page=2")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{Name: "query", Page: 2}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("empty higher priority overrides", func(t *testing.T) {
		req := urlencodedRequest(t, "/?name=query&page=5", "name=&page=")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{Name: "", Page: 0}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

func TestBindUsesOnlyConfiguredTags(t *testing.T) {
	type target struct {
		FormOnly  string `form:"form_only"`
		QueryOnly string `query:"query_only"`
		Untagged  string
	}

	req := urlencodedRequest(t, "/?form_only=query&query_only=query2&Untagged=query3", "form_only=form")
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	want := target{FormOnly: "form", QueryOnly: "query2"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	var queryOnly target
	if err := Bind(req, &queryOnly, WithTagPriority(SourceQuery)); err != nil {
		t.Fatalf("Bind query-only error: %v", err)
	}
	want = target{QueryOnly: "query2"}
	if queryOnly != want {
		t.Errorf("query-only got %+v, want %+v", queryOnly, want)
	}
}

func TestBindCaseInsensitive(t *testing.T) {
	type target struct {
		Name  string `form:"name" query:"name"`
		Token string `header:"x-token"`
	}

	req := urlencodedRequest(t, "/?NAME=query", "NAME=form")
	req.Header.Set("X-Token", "abc")

	var got target
	if err := Bind(req, &got, WithCaseInsensitive(true)); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Name != "form" {
		t.Errorf("Name = %q, want form", got.Name)
	}
	if got.Token != "abc" {
		t.Errorf("Token = %q, want abc", got.Token)
	}

	req = urlencodedRequest(t, "/?NAME=query", "NAME=form")
	req.Header.Set("X-Token", "abc")
	var exact target
	if err := Bind(req, &exact); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if exact.Name != "" || exact.Token != "" {
		t.Errorf("got %+v, want exact-case matching by default", exact)
	}
}

func TestBindCaseInsensitiveExactMatchWins(t *testing.T) {
	type target struct {
		Name string `query:"name"`
	}

	req := httptest.NewRequest(http.MethodGet, "/?name=exact&NAME=upper", nil)
	var got target
	if err := Bind(req, &got, WithCaseInsensitive(true)); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Name != "exact" {
		t.Errorf("Name = %q, want exact", got.Name)
	}
}

func TestBindHeaderAndCookie(t *testing.T) {
	type target struct {
		Token   string   `header:"X-Token"`
		Session []string `cookie:"session"`
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// net/http normally canonicalizes header names; make sure Binder does too
	// for hand-built requests.
	req.Header["x-token"] = []string{"abc"}
	req.AddCookie(&http.Cookie{Name: "session", Value: "s1"})
	req.AddCookie(&http.Cookie{Name: "session", Value: "s2"})

	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	want := target{Token: "abc", Session: []string{"s1", "s2"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestBindDefaults(t *testing.T) {
	type target struct {
		Name string `form:"name,default=guest" query:"name"`
		Page int    `form:"page,default=1" query:"page"`
	}

	t.Run("no input", func(t *testing.T) {
		req := urlencodedRequest(t, "/", "")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{Name: "guest", Page: 1}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("query overrides form default", func(t *testing.T) {
		req := urlencodedRequest(t, "/?name=alice&page=5", "")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{Name: "alice", Page: 5}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("form overrides default", func(t *testing.T) {
		req := urlencodedRequest(t, "/", "name=bob&page=2")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{Name: "bob", Page: 2}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("explicit empty overrides default and query", func(t *testing.T) {
		req := urlencodedRequest(t, "/?name=alice&page=5", "name=&page=")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{Name: "", Page: 0}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})

	t.Run("default on lower priority tag", func(t *testing.T) {
		type lower struct {
			Name string `form:"name" query:"name,default=guest"`
		}
		req := urlencodedRequest(t, "/", "")
		var got lower
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Name != "guest" {
			t.Errorf("Name = %q, want guest", got.Name)
		}

		req = urlencodedRequest(t, "/?name=", "")
		got = lower{}
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Name != "" {
			t.Errorf("Name = %q, want empty (explicit empty suppresses default)", got.Name)
		}
	})

	t.Run("higher priority default wins", func(t *testing.T) {
		type both struct {
			Name string `form:"name,default=form" query:"name,default=query"`
		}
		req := urlencodedRequest(t, "/", "")
		var got both
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Name != "form" {
			t.Errorf("Name = %q, want form", got.Name)
		}
	})

	t.Run("nested default", func(t *testing.T) {
		type nested struct {
			Inner struct {
				Name string `form:"name,default=guest"`
			} `form:"inner"`
		}
		req := urlencodedRequest(t, "/", "")
		var got nested
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Inner.Name != "guest" {
			t.Errorf("Inner.Name = %q, want guest", got.Inner.Name)
		}
	})
}

func TestBindRequired(t *testing.T) {
	type target struct {
		Name string `form:"name,required" query:"name"`
	}

	t.Run("lower priority value satisfies required", func(t *testing.T) {
		req := urlencodedRequest(t, "/?name=alice", "")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Name != "alice" {
			t.Errorf("Name = %q, want alice", got.Name)
		}
	})

	t.Run("missing required field", func(t *testing.T) {
		req := urlencodedRequest(t, "/", "")
		var got target
		err := Bind(req, &got)
		if err == nil {
			t.Fatal("expected required error")
		}
		var empty form.EmptyFieldError
		if !errors.As(err, &empty) {
			t.Fatalf("error = %v, want EmptyFieldError", err)
		}
		if empty.Key != "name" {
			t.Errorf("Key = %q, want name", empty.Key)
		}
	})

	t.Run("required and default conflict", func(t *testing.T) {
		type conflict struct {
			Name string `form:"name,required,default=x"`
		}
		req := urlencodedRequest(t, "/", "")
		var got conflict
		err := Bind(req, &got)
		if err == nil || !strings.Contains(err.Error(), "cannot have a default") {
			t.Fatalf("error = %v, want required/default conflict", err)
		}
		if got.Name != "x" {
			t.Errorf("Name = %q, want x (the default is still applied)", got.Name)
		}
	})
}

func TestBindNestedAndCollections(t *testing.T) {
	type target struct {
		Filter struct {
			Name string `form:"name" query:"q"`
		} `form:"filter" query:"filter"`
		Tags   []string        `form:"tags" query:"tags"`
		Values map[string]int  `form:"values" query:"values"`
		Ptr    *string         `form:"ptr" query:"ptr"`
		Lookup map[string]*int `form:"lookup" query:"lookup"`
	}

	req := urlencodedRequest(t,
		"/?filter.q=query&tags=q1&tags=q2&values[a]=1&ptr=query&lookup[k]=7",
		"filter.name=form&tags=f1&values[b]=2&ptr=form&lookup[j]=8")

	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	want := target{
		Tags:   []string{"f1"},
		Values: map[string]int{"a": 1, "b": 2},
		Lookup: map[string]*int{"k": intPtr(7), "j": intPtr(8)},
	}
	want.Filter.Name = "form"
	want.Ptr = stringPtr("form")
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	t.Run("query fallback", func(t *testing.T) {
		req := urlencodedRequest(t, "/?filter.q=query&tags=q1&tags=q2&values[a]=1&ptr=query&lookup[k]=7", "")
		var got target
		if err := Bind(req, &got); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		want := target{
			Tags:   []string{"q1", "q2"},
			Values: map[string]int{"a": 1},
			Lookup: map[string]*int{"k": intPtr(7)},
		}
		want.Filter.Name = "query"
		want.Ptr = stringPtr("query")
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
}

type EmbeddedBindBase struct {
	ID   int `param:"id"`
	Page int `form:"page" query:"page"`
}

type embeddedBindTarget struct {
	EmbeddedBindBase
	Name string `form:"name" query:"name"`
}

func TestBindEmbeddedStruct(t *testing.T) {
	req := urlencodedRequest(t, "/?page=1&name=query", "page=2&name=form")
	req.SetPathValue("id", "9")
	var got embeddedBindTarget
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 9 || got.Page != 2 || got.Name != "form" {
		t.Errorf("got %+v, want ID=9 Page=2 Name=form", got)
	}

	req = urlencodedRequest(t, "/?page=1&name=query", "")
	req.SetPathValue("id", "9")
	got = embeddedBindTarget{}
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.ID != 9 || got.Page != 1 || got.Name != "query" {
		t.Errorf("got %+v, want ID=9 Page=1 Name=query", got)
	}
}

type EmbeddedBindPtrBase struct {
	ID   int `param:"id"`
	Page int `form:"page" query:"page"`
}

func TestBindEmbeddedPointer(t *testing.T) {
	type target struct {
		*EmbeddedBindPtrBase
		Name string `form:"name" query:"name"`
	}

	req := urlencodedRequest(t, "/?page=1&name=query", "")
	req.SetPathValue("id", "9")
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.EmbeddedBindPtrBase == nil {
		t.Fatal("embedded pointer was not allocated")
	}
	if got.ID != 9 || got.Page != 1 || got.Name != "query" {
		t.Errorf("got %+v, want ID=9 Page=1 Name=query", got)
	}
}

func TestBindNestedRequiresParentTag(t *testing.T) {
	type target struct {
		Filter struct {
			Name string `form:"name" query:"q"`
		} `form:"filter"`
	}

	req := urlencodedRequest(t, "/?filter.q=query", "")
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.Filter.Name != "" {
		t.Errorf("Filter.Name = %q, want empty without a parent query tag", got.Filter.Name)
	}
}

func intPtr(v int) *int          { return &v }
func stringPtr(v string) *string { return &v }

func TestBindMergesPointerStructSources(t *testing.T) {
	type inner struct {
		A string `form:"a" query:"a"`
		B string `form:"b" query:"b"`
	}
	type target struct {
		In *inner `form:"in" query:"in"`
	}

	req := urlencodedRequest(t, "/?in.a=query-a&in.b=query-b", "in.a=form-a")
	var got target
	if err := Bind(req, &got); err != nil {
		t.Fatalf("Bind error: %v", err)
	}
	if got.In == nil {
		t.Fatal("In was not allocated")
	}
	if got.In.A != "form-a" || got.In.B != "query-b" {
		t.Errorf("In = %+v, want A=form-a B=query-b", got.In)
	}
}

func TestBindErrors(t *testing.T) {
	type target struct {
		Name string `query:"name"`
	}

	t.Run("nil request", func(t *testing.T) {
		var got target
		if err := Bind(nil, &got); !errors.Is(err, ErrNilRequest) {
			t.Errorf("error = %v, want ErrNilRequest", err)
		}
	})

	t.Run("nil target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if err := Bind(req, nil); !errors.Is(err, form.ErrNilPointer) {
			t.Errorf("error = %v, want form.ErrNilPointer", err)
		}
	})

	t.Run("non-pointer target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		var got target
		if err := Bind(req, got); !errors.Is(err, form.ErrNotPointer) {
			t.Errorf("error = %v, want form.ErrNotPointer", err)
		}
	})

	t.Run("invalid target fails before parsing", func(t *testing.T) {
		req := urlencodedRequest(t, "/", "name=%zz")
		if err := Bind(req, nil); !errors.Is(err, form.ErrNilPointer) {
			t.Errorf("error = %v, want form.ErrNilPointer", err)
		}
	})

	t.Run("malformed form", func(t *testing.T) {
		req := urlencodedRequest(t, "/", "name=%zz")
		var got target
		err := Bind(req, &got)
		if err == nil || !strings.Contains(err.Error(), "parse form") {
			t.Errorf("error = %v, want form parse error", err)
		}
	})

	t.Run("malformed query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.URL.RawQuery = "name=%zz"
		var got target
		err := Bind(req, &got, WithTagPriority(SourceQuery))
		if err == nil || !strings.Contains(err.Error(), "parse query string") {
			t.Errorf("error = %v, want query parse error", err)
		}
	})

	t.Run("malformed query ignored when query source is disabled", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.URL.RawQuery = "name=%zz"
		req.Header.Set("X-Token", "abc")
		var got struct {
			Token string `header:"X-Token"`
		}
		if err := Bind(req, &got, WithTagPriority(SourceHeader)); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Token != "abc" {
			t.Errorf("Token = %q, want abc", got.Token)
		}
	})

	t.Run("malformed multipart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not multipart"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=nope")
		var got target
		err := Bind(req, &got)
		if err == nil || !strings.Contains(err.Error(), "parse multipart form") {
			t.Errorf("error = %v, want multipart parse error", err)
		}
	})

	t.Run("unsupported source", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=alice", nil)
		var got target
		err := Bind(req, &got, WithTagPriority("yaml"))
		if !errors.Is(err, ErrUnsupportedSource) {
			t.Errorf("error = %v, want ErrUnsupportedSource", err)
		}
	})

	t.Run("empty priority uses defaults", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=alice", nil)
		var got target
		if err := Bind(req, &got, WithTagPriority()); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Name != "alice" {
			t.Errorf("Name = %q, want alice", got.Name)
		}
	})

	t.Run("duplicate priority is ignored", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?name=alice", nil)
		var got target
		if err := Bind(req, &got, WithTagPriority(SourceQuery, SourceQuery)); err != nil {
			t.Fatalf("Bind error: %v", err)
		}
		if got.Name != "alice" {
			t.Errorf("Name = %q, want alice", got.Name)
		}
	})
}

func TestBindAggregatesSourceErrors(t *testing.T) {
	type target struct {
		A int `form:"a" query:"a"`
		B int `form:"b" query:"b"`
	}
	req := urlencodedRequest(t, "/?a=bad", "b=worse")
	var got target
	err := Bind(req, &got)
	if err == nil {
		t.Fatal("expected decode errors")
	}
	var decodeErrs form.DecodeErrors
	if !errors.As(err, &decodeErrs) {
		t.Fatalf("error type = %T, want DecodeErrors", err)
	}
	if len(decodeErrs) != 2 {
		t.Fatalf("errors = %v, want 2 entries", decodeErrs)
	}
	if _, ok := decodeErrs["a"]; !ok {
		t.Errorf("missing error for a: %v", decodeErrs)
	}
	if _, ok := decodeErrs["b"]; !ok {
		t.Errorf("missing error for b: %v", decodeErrs)
	}
}

func TestBindConcurrent(t *testing.T) {
	type target struct {
		Name string `form:"name" query:"name"`
		Page int    `form:"page" query:"page"`
	}

	binder := NewBinder()
	const workers = 16
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := urlencodedRequest(t, "/?name=query&page=1", "name=form&page=2")
			var got target
			if err := binder.Bind(req, &got); err != nil {
				t.Errorf("Bind error: %v", err)
				return
			}
			if got.Name != "form" || got.Page != 2 {
				t.Errorf("got %+v, want {Name:form Page:2}", got)
			}
		}()
	}
	wg.Wait()
}
