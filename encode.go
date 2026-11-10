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
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Encoder encodes Go structs into map[string][]string.
//
// An Encoder caches struct metadata and is safe for concurrent use after
// construction.
type Encoder struct {
	opts  options
	cache *structCache
}

// NewEncoder creates a new Encoder with the given options.
func NewEncoder(opts ...Option) *Encoder {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &Encoder{opts: o, cache: newStructCache(o.tagName, o.mode, o.naming, o.rejectDuplicateTags)}
}

// Encode encodes a value into map[string][]string.
//
// Structs are encoded by field; other values use the root key (""), with
// maps using "[key]" and slices/arrays using "[i]" for nested elements.
// Encoding continues after a field error; a partial map and EncodeErrors are
// returned together.
func (e *Encoder) Encode(v any) (map[string][]string, error) {
	if err := e.opts.validateNamespace(); err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrInvalidValue
	}

	rv := reflect.Indirect(reflect.ValueOf(v))
	if !rv.IsValid() {
		return nil, ErrInvalidValue
	}

	state := &encodeState{e: e, values: make(map[string][]string)}
	if rv.Kind() == reflect.Struct && rv.Type() != timeType &&
		(!e.opts.topLevelMarshaler || !implementsMarshalerType(rv.Type())) {
		// Top-level structs are traversed by field by default. This avoids
		// treating a method promoted from an embedded field as a top-level
		// Marshaler unless WithTopLevelMarshaler(true) is used.
		state.encodeStruct(rv, "", false)
	} else {
		state.encodeValue(rv, "", false)
	}

	if len(state.errs) > 0 {
		return state.values, state.errs
	}
	return state.values, nil
}

// Marshal encodes a value into map[string][]string using default options.
func Marshal(v any) (map[string][]string, error) {
	return defaultEncoder.Encode(v)
}

var defaultEncoder = NewEncoder()

type encodeState struct {
	e      *Encoder
	values map[string][]string
	errs   EncodeErrors
	depth  int
}

func (s *encodeState) add(path string, values ...string) {
	if len(values) == 0 {
		return
	}
	s.values[path] = append(s.values[path], values...)
}

func (s *encodeState) addError(path string, err error) {
	if path == "" {
		path = "<root>"
	}
	if prev, ok := s.errs[path]; ok {
		s.errs[path] = errors.Join(prev, err)
		return
	}
	if s.errs == nil {
		s.errs = make(EncodeErrors)
	}
	s.errs[path] = err
}

func (s *encodeState) encodeStruct(v reflect.Value, prefix string, indexContext bool) {
	info := s.e.cache.get(v.Type())
	if info.duplicateErr != nil {
		s.addError(prefix, info.duplicateErr)
	}

	for i := range info.fields {
		f := &info.fields[i]
		fv := v.Field(f.index)

		if f.invalidName {
			s.addError(s.e.opts.joinPath(prefix, f.name), fmt.Errorf("form: invalid tag name %q", f.name))
			continue
		}

		if f.omitempty && !f.hasDefault && isEmptyValue(fv) {
			continue
		}

		if f.anonymous && isFlattenableAnonymousType(f.typ) && s.e.opts.anonymousMode == AnonymousEmbed {
			s.encodeValue(fv, prefix, indexContext)
			continue
		}

		path := s.e.opts.joinPath(prefix, f.name)
		if f.omitempty && f.hasDefault {
			s.encodeWithDefault(fv, path, f, indexContext)
			continue
		}
		s.encodeValue(fv, path, indexContext)
	}
}

func (s *encodeState) encodeWithDefault(v reflect.Value, path string, f *fieldInfo, indexContext bool) {
	tmp := &encodeState{e: s.e, values: make(map[string][]string), depth: s.depth}
	tmp.encodeValue(v, path, indexContext)

	for key, err := range tmp.errs {
		s.addError(key, err)
	}

	if len(tmp.values) == 0 {
		return
	}
	if values, ok := tmp.values[path]; ok && len(tmp.values) == 1 && slices.Equal(values, f.defaultValue) {
		return
	}
	for key, values := range tmp.values {
		s.add(key, values...)
	}
}

// encodeValue encodes one value at the given path. Interfaces and pointers are
// unwrapped; nil pointers and nil interfaces are omitted.
func (s *encodeState) encodeValue(v reflect.Value, path string, indexContext bool) {
	if s.depth > defaultMaxEncodeDepth {
		s.addError(path, fmt.Errorf("form: maximum encoding depth %d exceeded", defaultMaxEncodeDepth))
		return
	}
	s.depth++
	defer func() { s.depth-- }()

	for {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Interface {
			if v.IsNil() {
				return
			}
			v = v.Elem()
			continue
		}
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return
			}
			if values, ok, err := s.tryMarshal(v); ok {
				if err != nil {
					s.addError(path, err)
					return
				}
				s.add(path, values...)
				return
			}
			v = v.Elem()
			continue
		}
		break
	}

	if values, ok, err := s.tryMarshal(v); ok {
		if err != nil {
			s.addError(path, err)
			return
		}
		s.add(path, values...)
		return
	}

	t := v.Type()
	if t == timeType {
		s.add(path, v.Interface().(time.Time).Format(s.e.opts.timeLayout))
		return
	}
	if t == durationType {
		duration := v.Interface().(time.Duration)
		if s.e.opts.durationFormat == DurationNanoseconds {
			s.add(path, strconv.FormatInt(int64(duration), 10))
		} else {
			s.add(path, duration.String())
		}
		return
	}
	if data, ok, err := s.tryTextMarshal(v); ok {
		if err != nil {
			s.addError(path, err)
			return
		}
		s.add(path, string(data))
		return
	}
	if isByteSliceType(t) {
		if v.Len() == 0 {
			return
		}
		s.add(path, string(bytesOf(v)))
		return
	}

	switch v.Kind() {
	case reflect.String:
		s.add(path, v.String())

	case reflect.Bool:
		s.add(path, strconv.FormatBool(v.Bool()))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s.add(path, strconv.FormatInt(v.Int(), 10))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s.add(path, strconv.FormatUint(v.Uint(), 10))

	case reflect.Float32:
		s.add(path, strconv.FormatFloat(v.Float(), 'f', -1, 32))

	case reflect.Float64:
		s.add(path, strconv.FormatFloat(v.Float(), 'f', -1, 64))

	case reflect.Slice, reflect.Array:
		s.encodeSlice(v, path, indexContext)

	case reflect.Map:
		s.encodeMap(v, path)

	case reflect.Struct:
		s.encodeStruct(v, path, indexContext)

	default:
		s.addError(path, fmt.Errorf("form: unsupported type %s", t))
	}
}

func (s *encodeState) tryTextMarshal(v reflect.Value) ([]byte, bool, error) {
	if !v.IsValid() {
		return nil, false, nil
	}
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil, false, nil
	}

	if v.Type().Implements(textMarshalerType) && v.CanInterface() {
		data, err := v.Interface().(encoding.TextMarshaler).MarshalText()
		return data, true, err
	}

	if v.Kind() != reflect.Pointer && reflect.PointerTo(v.Type()).Implements(textMarshalerType) {
		if v.CanAddr() {
			pv := v.Addr()
			if pv.CanInterface() {
				data, err := pv.Interface().(encoding.TextMarshaler).MarshalText()
				return data, true, err
			}
		} else {
			tmp := reflect.New(v.Type())
			tmp.Elem().Set(v)
			data, err := tmp.Interface().(encoding.TextMarshaler).MarshalText()
			return data, true, err
		}
	}

	return nil, false, nil
}

func (s *encodeState) tryMarshal(v reflect.Value) ([]string, bool, error) {
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil, false, nil
	}

	if v.Type().Implements(marshalerType) && v.CanInterface() {
		values, err := v.Interface().(Marshaler).MarshalValues()
		return values, true, err
	}

	if reflect.PointerTo(v.Type()).Implements(marshalerType) {
		if v.CanAddr() {
			pv := v.Addr()
			if pv.CanInterface() {
				values, err := pv.Interface().(Marshaler).MarshalValues()
				return values, true, err
			}
		} else {
			tmp := reflect.New(v.Type())
			tmp.Elem().Set(v)
			values, err := tmp.Interface().(Marshaler).MarshalValues()
			return values, true, err
		}
	}

	return nil, false, nil
}

func (s *encodeState) encodeSlice(v reflect.Value, path string, indexContext bool) {
	if v.Len() == 0 {
		return
	}

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		// A slice reached through another slice/array or a map value always
		// uses indexes. A slice directly under a struct uses indexes only for
		// elements that cannot be represented as repeated values.
		if indexContext || needsIndex(elem) {
			s.encodeValue(elem, indexedPath(path, i), true)
			continue
		}
		s.encodeValue(elem, path, false)
	}
}

func (s *encodeState) encodeMap(v reflect.Value, path string) {
	if v.Len() == 0 {
		return
	}

	type entry struct {
		key   string
		value reflect.Value
	}
	entries := make([]entry, 0, v.Len())

	for _, key := range v.MapKeys() {
		formatted, err := formatMapKey(key)
		if err != nil {
			s.addError(path, err)
			continue
		}
		if formatted == "" || strings.ContainsAny(formatted, "[]") {
			s.addError(path, fmt.Errorf("form: unsupported map key %q", formatted))
			continue
		}
		entries = append(entries, entry{key: formatted, value: v.MapIndex(key)})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	for _, e := range entries {
		// Map values are always in an indexed context: nested slices and
		// structs inside map values use bracket paths.
		s.encodeValue(e.value, bracketPath(path, e.key), true)
	}
}

func formatMapKey(key reflect.Value) (string, error) {
	switch key.Kind() {
	case reflect.String:
		return key.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(key.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(key.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(key.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(key.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(key.Float(), 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("form: unsupported map key type %s", key.Type())
	}
}

// needsIndex reports whether a slice element must be encoded with an explicit
// [i] path segment. Basic values are emitted as repeated values for the same
// key; pointers, structs, slices, arrays, maps and custom marshalers use
// indexes (matching go-playground/form).
func needsIndex(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return false
		}
		return needsIndex(v.Elem())
	}
	if v.Kind() == reflect.Pointer {
		return true
	}
	if implementsMarshalerType(v.Type()) {
		return true
	}
	switch v.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

func isFlattenableAnonymousType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || t == timeType {
		return false
	}
	return !implementsMarshalerType(t) && !implementsUnmarshalerType(t)
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}

func bytesOf(v reflect.Value) []byte {
	if v.Type() == byteSliceType {
		return v.Bytes()
	}
	return v.Convert(byteSliceType).Bytes()
}
