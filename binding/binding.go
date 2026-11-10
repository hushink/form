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
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/textproto"
	"net/url"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/hushink/form"
)

// HTTP binding source names. A source name is also the struct tag name used to
// select that source, for example `query:"q"` or `header:"X-Token"`.
const (
	// SourceParam binds path parameters read through http.Request.PathValue.
	SourceParam = "param"
	// SourceForm binds the request body's URL-encoded or multipart form values.
	SourceForm = "form"
	// SourceJSON binds an application/json (or +json) request body using
	// encoding/json.
	SourceJSON = "json"
	// SourceXML binds an application/xml, text/xml, or +xml request body using
	// encoding/xml.
	SourceXML = "xml"
	// SourceQuery binds the URL query string.
	SourceQuery = "query"
	// SourceHeader binds request headers. Header tag names must use canonical
	// MIME header form, for example `header:"X-Request-ID"`.
	SourceHeader = "header"
	// SourceCookie binds request cookies. Multiple cookies with the same name
	// are exposed as multiple values.
	SourceCookie = "cookie"
)

var (
	// ErrNilRequest is returned by Binder.Bind when the request is nil.
	ErrNilRequest = errors.New("binding: request is nil")
	// ErrUnsupportedSource is returned by Binder for a tag that is not a
	// supported HTTP binding source.
	ErrUnsupportedSource = errors.New("binding: unsupported bind source")
	// ErrBodyTooLarge is returned when a JSON or XML request body exceeds
	// WithMaxBodyBytes.
	ErrBodyTooLarge = errors.New("binding: request body too large")
)

// defaultMultipartMaxMemory matches the memory limit used by net/http's
// multipart parser.
const defaultMultipartMaxMemory int64 = 32 << 20

// defaultMaxBodyBytes limits how much of a JSON or XML body is read.
const defaultMaxBodyBytes int64 = 32 << 20

// defaultBindPriority gives path parameters the highest priority, then body
// values, then the query string and other request metadata.
var defaultBindPriority = []string{
	SourceParam,
	SourceForm,
	SourceJSON,
	SourceXML,
	SourceQuery,
	SourceHeader,
	SourceCookie,
}

// BindOption configures a Binder.
type BindOption func(*bindOptions)

type bindOptions struct {
	priority        []string
	maxMemory       int64
	maxBodyBytes    int64
	caseInsensitive bool
}

func defaultBindOptions() bindOptions {
	return bindOptions{
		priority:     append([]string(nil), defaultBindPriority...),
		maxMemory:    defaultMultipartMaxMemory,
		maxBodyBytes: defaultMaxBodyBytes,
	}
}

// WithTagPriority sets the binding sources and their priority. The first tag
// has the highest priority: values from lower-priority sources are bound
// first, then values from higher-priority sources overwrite them.
//
// Supported source names are SourceParam, SourceForm, SourceJSON, SourceXML,
// SourceQuery, SourceHeader, and SourceCookie. An empty list restores the
// default priority (param, form, json, xml, query, header, cookie). Duplicate
// names are ignored. Unsupported names are reported by Bind.
func WithTagPriority(tags ...string) BindOption {
	priority := append([]string(nil), tags...)
	return func(o *bindOptions) {
		o.priority = priority
	}
}

// WithMultipartMaxMemory sets the maxMemory argument passed to
// http.Request.ParseMultipartForm. Values less than or equal to zero are
// ignored. The default is 32 MiB.
//
// Binding stores no file uploads; the request still owns any temporary files
// created by ParseMultipartForm. Call MultipartForm.RemoveAll when the request
// is finished if uploads are present.
func WithMultipartMaxMemory(maxMemory int64) BindOption {
	return func(o *bindOptions) {
		if maxMemory > 0 {
			o.maxMemory = maxMemory
		}
	}
}

// WithMaxBodyBytes limits how many bytes of a JSON or XML request body Bind
// reads. Values less than or equal to zero are ignored. The default is 32 MiB.
//
// The body is restored after reading, so downstream handlers can consume it.
// When the limit is exceeded, Bind returns ErrBodyTooLarge.
func WithMaxBodyBytes(maxBytes int64) BindOption {
	return func(o *bindOptions) {
		if maxBytes > 0 {
			o.maxBodyBytes = maxBytes
		}
	}
}

// WithCaseInsensitive makes map-style sources (form, query, header, cookie,
// and param) match tag names without regard to case. An exact key wins;
// otherwise the lexicographically smallest case-insensitive match is used.
// The default is false.
//
// Path parameters are still read through http.Request.PathValue using the tag
// name, so a router that stores a differently-cased path parameter cannot be
// matched case-insensitively. JSON uses encoding/json's own case-insensitive
// matching, and XML remains case-sensitive.
func WithCaseInsensitive(caseInsensitive bool) BindOption {
	return func(o *bindOptions) {
		o.caseInsensitive = caseInsensitive
	}
}

// Binder decodes HTTP request path parameters, form values, JSON/XML bodies,
// query parameters, headers, and cookies into Go values. A Binder caches
// struct metadata and is safe for concurrent use across requests. Bind mutates
// the request while parsing its form and reading its body, so do not call it
// concurrently on the same request.
//
// Map-style sources bind only fields carrying a tag for that source; untagged
// fields are ignored. JSON and XML follow standard encoding/json and
// encoding/xml matching, where tags are optional; use json:"-" or xml:"-" to
// exclude a field from those body sources. A field may carry tags for several
// sources:
//
//	type Search struct {
//		ID      int    `param:"id" json:"id"`
//		Keyword string `form:"keyword" query:"q"`
//		Page    int    `form:"page" json:"page"`
//		Token   string `header:"X-Token"`
//	}
//
// For named nested structs, every parent field on the path must also carry a
// tag for that source. For example, a nested query field binds only when its
// parent has a query tag as well as a form tag. Anonymous embedded structs are
// traversed automatically, so their tagged fields bind without tagging the
// embedded field itself.
//
// Path parameters are read through http.Request.PathValue, so the router must
// populate them (the standard library does this for ServeMux patterns). An
// empty path value is treated as absent, and param tags should target flat
// scalar fields rather than nested structs.
//
// JSON and XML sources apply only when the request Content-Type matches. The
// body is read fully, restored for downstream handlers, and decoded with
// encoding/json or encoding/xml when that source's turn in the priority order
// arrives. An empty body is skipped. Malformed JSON or XML stops binding and
// returns a wrapped decode error. Use WithMaxBodyBytes to limit the read size.
//
// Binding names come from each source's tag, so naming style is explicit per
// source; form.WithNamingStrategy is not applied to binding. Use
// WithCaseInsensitive to match map-style sources without regard to case.
//
// With the default priority (param, form, json, xml, query, header, cookie),
// path parameters win, body values beat the query string, and headers/cookies
// are fallbacks. Use WithTagPriority to change the order.
//
// Defaults declared with `default=...` are applied before source values, so an
// actual value (including an explicit empty value) always overrides a default.
// When a field has tags for several sources, put `required` on the
// highest-priority tag; a non-zero value already bound from a lower-priority
// source or a default satisfies it. The `default` and `required` options are
// recognized on map-style tags (param, form, query, header, cookie); JSON and
// XML fields can add a form tag when those checks are needed.
//
// Every configured source is decoded when applicable. An error in a
// lower-priority source is reported even if a higher-priority source supplies
// a valid replacement.
type Binder struct {
	priority        []string // highest priority first
	maxMemory       int64
	maxBodyBytes    int64
	caseInsensitive bool

	// Both pass lists run in ascending priority order (lowest first), so
	// higher-priority values and defaults overwrite lower-priority ones.
	sourcePasses  []bindPass
	defaultPasses []bindPass

	// paramCache maps a struct type to the path parameter names referenced by
	// its param tags.
	paramCache sync.Map

	err error
}

type bindPass struct {
	tag      string
	decoder  *form.Decoder // nil for JSON/XML body passes
	bodyKind string        // SourceJSON or SourceXML for body passes
}

// NewBinder creates a Binder. Unsupported source names are reported by Bind
// instead of panicking, so NewBinder keeps the usual option-constructor shape.
func NewBinder(opts ...BindOption) *Binder {
	bo := defaultBindOptions()
	for _, opt := range opts {
		opt(&bo)
	}

	b := &Binder{
		maxMemory:       bo.maxMemory,
		maxBodyBytes:    bo.maxBodyBytes,
		caseInsensitive: bo.caseInsensitive,
	}
	priority, err := normalizeBindPriority(bo.priority)
	if err != nil {
		b.err = err
		return b
	}
	b.priority = priority

	for _, tag := range slices.Backward(priority) {

		if isBodySource(tag) {
			b.sourcePasses = append(b.sourcePasses, bindPass{tag: tag, bodyKind: tag})
			continue
		}

		sourceOptions := []form.Option{
			form.WithTagName(tag),
			form.WithMode(form.ModeExplicitEmbedded),
			// An empty value from a higher-priority source must override a
			// value from a lower-priority one, including for numeric, time,
			// and []byte fields.
			form.WithZeroEmpty(true),
			// Defaults run in a separate pre-pass; applying them here would
			// make a higher-priority default overwrite a lower-priority actual
			// value.
			form.WithoutDefaults(),
			// Values bound from other sources may satisfy a required marker.
			form.WithRequiredFallback(),
		}
		defaultOpts := []form.Option{
			form.WithTagName(tag),
			form.WithMode(form.ModeExplicitEmbedded),
			// Required fields are checked by the source passes, where actual
			// input is available.
			form.WithoutRequiredChecks(),
		}
		if b.caseInsensitive {
			sourceOptions = append(sourceOptions, form.WithCaseInsensitive(true))
			defaultOpts = append(defaultOpts, form.WithCaseInsensitive(true))
		}

		sourceDecoder := form.NewDecoder(sourceOptions...)
		b.sourcePasses = append(b.sourcePasses, bindPass{tag: tag, decoder: sourceDecoder})

		defaultDecoder := form.NewDecoder(defaultOpts...)
		b.defaultPasses = append(b.defaultPasses, bindPass{tag: tag, decoder: defaultDecoder})
	}

	return b
}

// Bind decodes the request into v. v must be a non-nil pointer. The form body
// is parsed only when the form source is configured, and the query string is
// parsed only when the query source is configured. JSON and XML bodies are
// read only when the matching source is configured and the Content-Type
// matches. Headers and cookies need no body parsing.
//
// Decode errors from all sources and default passes are merged into a single
// form.DecodeErrors value. Request parsing errors stop binding and wrap the
// underlying error.
func (b *Binder) Bind(r *http.Request, v any) error {
	if b.err != nil {
		return b.err
	}
	if r == nil {
		return ErrNilRequest
	}
	if err := validateTarget(v); err != nil {
		return err
	}
	if err := b.parseRequest(r); err != nil {
		return err
	}
	body, err := b.readBody(r)
	if err != nil {
		return err
	}
	sources, err := b.collectSources(r, v)
	if err != nil {
		return err
	}

	errs := make(form.DecodeErrors)
	for _, pass := range b.defaultPasses {
		if err := mergePassError(errs, pass.decoder.Decode(nil, v)); err != nil {
			return err
		}
	}
	for _, pass := range b.sourcePasses {
		if pass.bodyKind != "" {
			if body.kind != pass.bodyKind || len(bytes.TrimSpace(body.data)) == 0 {
				continue
			}
			if err := decodeBody(pass.bodyKind, body.data, v); err != nil {
				return err
			}
			continue
		}
		if err := mergePassError(errs, pass.decoder.Decode(sources[pass.tag], v)); err != nil {
			return err
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Bind decodes the request using a new Binder. Reuse a Binder when binding the
// same types repeatedly to share struct metadata caches.
func Bind(r *http.Request, v any, opts ...BindOption) error {
	if len(opts) == 0 {
		return defaultBinder.Bind(r, v)
	}
	return NewBinder(opts...).Bind(r, v)
}

var defaultBinder = NewBinder()

func validateTarget(v any) error {
	if v == nil {
		return form.ErrNilPointer
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return form.ErrNotPointer
	}
	if rv.IsNil() {
		return form.ErrNilPointer
	}
	return nil
}

func normalizeBindPriority(tags []string) ([]string, error) {
	seen := make(map[string]struct{}, len(tags))
	priority := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if !supportedBindSource(tag) {
			return nil, fmt.Errorf("%w: %q", ErrUnsupportedSource, tag)
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		priority = append(priority, tag)
	}
	if len(priority) == 0 {
		return append([]string(nil), defaultBindPriority...), nil
	}
	return priority, nil
}

func supportedBindSource(tag string) bool {
	switch tag {
	case SourceParam, SourceForm, SourceJSON, SourceXML, SourceQuery, SourceHeader, SourceCookie:
		return true
	default:
		return false
	}
}

func isBodySource(tag string) bool {
	return tag == SourceJSON || tag == SourceXML
}

func (b *Binder) parseRequest(r *http.Request) error {
	// Query parsing has its own error path in collectSources. Skipping
	// ParseForm here also means a malformed query string does not block
	// header- or cookie-only binding.
	if !b.hasSource(SourceForm) {
		return nil
	}
	// A JSON or XML body is not form data. Skipping ParseForm keeps the body
	// untouched and avoids "missing form body" errors for bodyless requests.
	if requestBodyKind(r) != "" {
		return nil
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err == nil && mediaType == "multipart/form-data" {
		if err := r.ParseMultipartForm(b.maxMemory); err != nil {
			return fmt.Errorf("binding: cannot parse multipart form: %w", err)
		}
		return nil
	}
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("binding: cannot parse form: %w", err)
	}
	return nil
}

func (b *Binder) hasSource(tag string) bool {
	return slices.Contains(b.priority, tag)
}

func (b *Binder) collectSources(r *http.Request, v any) (map[string]map[string][]string, error) {
	sources := make(map[string]map[string][]string, len(b.priority))
	for _, tag := range b.priority {
		switch tag {
		case SourceParam:
			sources[tag] = b.paramValues(r, v)
		case SourceForm:
			sources[tag] = r.PostForm
		case SourceQuery:
			query, err := queryValues(r)
			if err != nil {
				return nil, err
			}
			sources[tag] = query
		case SourceHeader:
			sources[tag] = headerValues(r.Header)
		case SourceCookie:
			sources[tag] = cookieValues(r)
		}
	}
	return sources, nil
}

type requestBody struct {
	kind string
	data []byte
}

// readBody reads a JSON or XML body once, restores it for downstream handlers,
// and returns the bytes for the matching priority pass.
func (b *Binder) readBody(r *http.Request) (requestBody, error) {
	kind := requestBodyKind(r)
	if kind == "" || !b.hasSource(kind) {
		return requestBody{}, nil
	}
	if r.Body == nil || r.Body == http.NoBody {
		return requestBody{}, nil
	}

	limit := b.maxBodyBytes
	if limit <= 0 {
		limit = defaultMaxBodyBytes
	}
	readLimit := limit
	if readLimit < math.MaxInt64 {
		readLimit++
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, readLimit))
	if err != nil {
		return requestBody{}, fmt.Errorf("binding: cannot read request body: %w", err)
	}
	if int64(len(data)) > limit {
		return requestBody{}, ErrBodyTooLarge
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	return requestBody{kind: kind, data: data}, nil
}

func requestBodyKind(r *http.Request) string {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return ""
	}
	switch {
	case mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"):
		return SourceJSON
	case mediaType == "application/xml" || mediaType == "text/xml" || strings.HasSuffix(mediaType, "+xml"):
		return SourceXML
	default:
		return ""
	}
}

func decodeBody(kind string, data []byte, v any) error {
	var err error
	switch kind {
	case SourceJSON:
		err = json.Unmarshal(data, v)
	case SourceXML:
		err = xml.Unmarshal(data, v)
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedSource, kind)
	}
	if err != nil {
		return fmt.Errorf("binding: cannot decode %s body: %w", strings.ToUpper(kind), err)
	}
	return nil
}

func (b *Binder) paramValues(r *http.Request, v any) map[string][]string {
	t := reflect.TypeOf(v)
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	names := b.paramNames(t)
	if len(names) == 0 {
		return nil
	}
	values := make(map[string][]string, len(names))
	for _, name := range names {
		if value := r.PathValue(name); value != "" {
			values[name] = []string{value}
		}
	}
	return values
}

func (b *Binder) paramNames(t reflect.Type) []string {
	if cached, ok := b.paramCache.Load(t); ok {
		return cached.([]string)
	}

	names := make(map[string]struct{})
	collectParamNames(t, make(map[reflect.Type]struct{}), names)

	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	sort.Strings(list)

	actual, _ := b.paramCache.LoadOrStore(t, list)
	return actual.([]string)
}

// collectParamNames walks a struct type and collects the path parameter names
// referenced by param tags. Path parameters are flat, but the walk follows
// nested and embedded structs so tags are found regardless of nesting.
func collectParamNames(t reflect.Type, seen map[reflect.Type]struct{}, names map[string]struct{}) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if _, ok := seen[t]; ok {
		return
	}
	seen[t] = struct{}{}

	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() && !field.Anonymous {
				continue
			}
			if raw, ok := field.Tag.Lookup(SourceParam); ok && raw != "" && raw != "-" {
				name := raw
				if index := strings.IndexByte(name, ','); index >= 0 {
					name = name[:index]
				}
				if name == "" {
					name = field.Name
				}
				if name != "" && name != "-" && !strings.ContainsAny(name, ".[]") {
					names[name] = struct{}{}
				}
			}
			collectParamNames(field.Type, seen, names)
		}
	case reflect.Slice, reflect.Array:
		collectParamNames(t.Elem(), seen, names)
	case reflect.Map:
		collectParamNames(t.Elem(), seen, names)
	}
}

func queryValues(r *http.Request) (map[string][]string, error) {
	if r.URL == nil {
		return nil, nil
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("binding: cannot parse query string: %w", err)
	}
	return values, nil
}

// headerValues canonicalizes header names so tags work with maps built by
// hand as well as headers parsed by net/http.
func headerValues(h http.Header) map[string][]string {
	if len(h) == 0 {
		return nil
	}
	values := make(map[string][]string, len(h))
	for key, entries := range h {
		canonical := textproto.CanonicalMIMEHeaderKey(key)
		values[canonical] = append(values[canonical], entries...)
	}
	return values
}

func cookieValues(r *http.Request) map[string][]string {
	cookies := r.Cookies()
	if len(cookies) == 0 {
		return nil
	}
	values := make(map[string][]string, len(cookies))
	for _, cookie := range cookies {
		values[cookie.Name] = append(values[cookie.Name], cookie.Value)
	}
	return values
}

func mergePassError(dst form.DecodeErrors, err error) error {
	if err == nil {
		return nil
	}
	var decodeErrs form.DecodeErrors
	if !errors.As(err, &decodeErrs) {
		return err
	}
	for path, fieldErr := range decodeErrs {
		if previous, ok := dst[path]; ok {
			dst[path] = errors.Join(previous, fieldErr)
			continue
		}
		dst[path] = fieldErr
	}
	return nil
}
