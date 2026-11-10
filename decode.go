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
	"sort"
	"strconv"
	"strings"
	"time"
)

// Decoder decodes map[string][]string into Go structs.
//
// A Decoder caches struct metadata and is safe for concurrent use after
// construction.
type Decoder struct {
	opts  options
	cache *structCache
}

// NewDecoder creates a new Decoder with the given options.
func NewDecoder(opts ...Option) *Decoder {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &Decoder{opts: o, cache: newStructCache(o.tagName, o.mode, o.naming, o.rejectDuplicateTags)}
}

// Decode decodes map[string][]string into a value.
//
// v must be a non-nil pointer. Structs are decoded by field; other targets use
// the root key (""), with maps using "[key]" and slices/arrays using "[i]" for
// nested elements. Decoding continues after a field error; all collected
// errors are returned as DecodeErrors.
func (d *Decoder) Decode(data map[string][]string, v any) error {
	if err := d.opts.validateNamespace(); err != nil {
		return err
	}
	if err := validateDecodeTarget(v); err != nil {
		return err
	}

	rv := reflect.ValueOf(v).Elem()

	root, errs := buildPathTree(data, d.opts.namespacePrefix, d.opts.namespaceSuffix)
	if errs == nil {
		errs = make(DecodeErrors)
	}

	state := &decodeState{d: d, errs: errs}
	if rv.Kind() == reflect.Struct && rv.Type() != timeType && !implementsUnmarshalerType(rv.Type()) {
		state.decodeStruct(rv, root, "")
	} else {
		state.decodeValue(rv, root, "")
	}

	if d.opts.strictUnknownKeys {
		state.collectUnknown(root)
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Unmarshal decodes map[string][]string into a value using default options.
func Unmarshal(data map[string][]string, v any) error {
	return defaultDecoder.Decode(data, v)
}

var defaultDecoder = NewDecoder()

// validateDecodeTarget checks the destination without building a path tree.
// Binder uses it before parsing a request so invalid targets fail early.
func validateDecodeTarget(v any) error {
	if v == nil {
		return ErrNilPointer
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return ErrNotPointer
	}
	if rv.IsNil() {
		return ErrNilPointer
	}
	return nil
}

type decodeState struct {
	d    *Decoder
	errs DecodeErrors
}

// field resolves a struct field name against a path node. Exact matches win;
// with case-insensitive decoding enabled, the lexicographically smallest
// case-insensitive match is chosen for deterministic behavior.
func (s *decodeState) field(node *pathNode, name string) *pathNode {
	if node == nil {
		return nil
	}
	if child, ok := node.fields[name]; ok {
		return child
	}
	if !s.d.opts.caseInsensitive {
		return nil
	}

	var match *pathNode
	var matchName string
	for candidate, child := range node.fields {
		if !strings.EqualFold(candidate, name) {
			continue
		}
		if match == nil || candidate < matchName {
			match = child
			matchName = candidate
		}
	}
	return match
}

func (s *decodeState) addError(path string, err error) {
	if path == "" {
		path = "<root>"
	}
	if prev, ok := s.errs[path]; ok {
		s.errs[path] = errors.Join(prev, err)
		return
	}
	s.errs[path] = err
}

// decodeStruct walks one struct level. It returns true when at least one field
// was set (including fields populated from defaults).
func (s *decodeState) decodeStruct(v reflect.Value, node *pathNode, prefix string) bool {
	info := s.d.cache.get(v.Type())
	if info.duplicateErr != nil {
		s.addError(prefix, info.duplicateErr)
	}
	set := false

	for i := range info.fields {
		f := &info.fields[i]
		fv := v.Field(f.index)
		fieldPath := s.d.opts.joinPath(prefix, f.name)

		if f.invalidName {
			s.addError(fieldPath, fmt.Errorf("form: invalid tag name %q", f.name))
			continue
		}

		if f.required && !s.d.opts.skipRequired {
			if f.hasDefault {
				s.addError(fieldPath, fmt.Errorf("form: required field %q cannot have a default", fieldPath))
			} else if !s.requiredSatisfied(node, f, fv) {
				s.addError(fieldPath, EmptyFieldError{Key: fieldPath})
			}
		}

		if f.anonymous {
			if s.decodeAnonymous(fv, node, prefix, fieldPath, f) {
				set = true
			}
			continue
		}

		child := s.field(node, f.name)
		if child != nil {
			child.matched = true
		}
		if child == nil || !child.hasContent() {
			if f.hasDefault && !s.d.opts.skipDefaults {
				if s.decodeDefault(fv, fieldPath, f) {
					set = true
				}
				continue
			}
			// Apply defaults nested inside a non-pointer struct even when the
			// parent received no input at all.
			if isPlainStructType(f.typ) && f.typ.Kind() == reflect.Struct {
				if s.decodeStruct(fv, &pathNode{}, fieldPath) {
					set = true
				}
			}
			continue
		}

		if s.decodeValue(fv, child, fieldPath) {
			set = true
		}
	}

	return set
}

func (s *decodeState) decodeAnonymous(v reflect.Value, node *pathNode, prefix, fieldPath string, f *fieldInfo) bool {
	set := false

	if isFlattenableAnonymousType(f.typ) {
		if !v.CanInterface() {
			return s.decodeUnexportedAnonymous(v, node, prefix, fieldPath, f, s.d.opts.anonymousMode == AnonymousSeparate)
		}

		if s.d.opts.anonymousMode == AnonymousSeparate {
			child := s.field(node, f.name)
			if child != nil {
				child.matched = true
			}
			if child != nil && child.hasContent() {
				if s.decodeValue(v, child, fieldPath) {
					set = true
				}
			} else if f.typ.Kind() == reflect.Struct {
				if s.decodeStruct(v, &pathNode{}, fieldPath) {
					set = true
				}
			}
			return set
		}

		// Default embedding mode: promoted fields live next to the parent's
		// fields. Also accept the embedded type's own namespace.
		if s.decodeValue(v, node, prefix) {
			set = true
		}
		if child := s.field(node, f.name); child != nil {
			child.matched = true
			if child.hasContent() && s.decodeValue(v, child, fieldPath) {
				set = true
			}
		}
		if !node.hasContent() && f.typ.Kind() == reflect.Struct {
			if s.decodeStruct(v, &pathNode{}, prefix) {
				set = true
			}
		}
		return set
	}

	// Embedded non-struct (or time.Time / custom Unmarshaler) fields use their
	// own name as the key.
	if child := s.field(node, f.name); child != nil {
		child.matched = true
		if child.hasContent() && s.decodeValue(v, child, fieldPath) {
			set = true
		}
	}
	return set
}

// decodeUnexportedAnonymous handles embedded structs whose field itself cannot
// be interfaced or set. Their exported subfields are still settable. A nil
// unexported embedded pointer cannot be allocated through reflection; when
// input targets it, a clear error is reported instead of silently ignoring it.
func (s *decodeState) decodeUnexportedAnonymous(v reflect.Value, node *pathNode, prefix, fieldPath string, f *fieldInfo, separate bool) bool {
	target := v
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			s.reportNilUnexportedEmbedded(v, node, prefix, fieldPath, f, separate)
			return false
		}
		target = v.Elem()
	}

	set := false
	if separate {
		child := s.field(node, f.name)
		if child != nil {
			child.matched = true
			if child.hasContent() && s.decodeStruct(target, child, fieldPath) {
				set = true
			}
		}
		if v.Kind() == reflect.Struct && (child == nil || !child.hasContent()) {
			if s.decodeStruct(target, &pathNode{}, fieldPath) {
				set = true
			}
		}
		return set
	}

	if s.decodeStruct(target, node, prefix) {
		set = true
	}
	if child := s.field(node, f.name); child != nil {
		child.matched = true
		if child.hasContent() && s.decodeStruct(target, child, fieldPath) {
			set = true
		}
	}
	if v.Kind() == reflect.Struct && !node.hasContent() {
		if s.decodeStruct(target, &pathNode{}, prefix) {
			set = true
		}
	}
	return set
}

func (s *decodeState) reportNilUnexportedEmbedded(v reflect.Value, node *pathNode, prefix, fieldPath string, f *fieldInfo, separate bool) {
	err := fmt.Errorf("form: cannot allocate nil unexported embedded pointer %s", v.Type())

	if separate {
		if child := s.field(node, f.name); child != nil && child.hasContent() {
			child.matched = true
			s.addError(fieldPath, err)
		}
		return
	}

	t := f.typ
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	info := s.d.cache.get(t)
	for i := range info.fields {
		child := s.field(node, info.fields[i].name)
		if child != nil && child.hasContent() {
			child.matched = true
			s.addError(s.d.opts.joinPath(prefix, info.fields[i].name), err)
			return
		}
	}
}

func (s *decodeState) decodeDefault(v reflect.Value, path string, f *fieldInfo) bool {
	node := &pathNode{values: append([]string(nil), f.defaultValue...)}
	return s.decodeValue(v, node, path)
}

// decodeValue decodes the parsed tree node into v. It returns true when v was
// modified.
func (s *decodeState) decodeValue(v reflect.Value, node *pathNode, path string) bool {
	if !v.IsValid() || !v.CanSet() || node == nil || !node.hasContent() {
		return false
	}

	if set, handled := s.tryUnmarshal(v, node, path); handled {
		return set
	}

	t := v.Type()
	if t == timeType {
		return s.decodeTime(v, node, path)
	}
	if t == durationType {
		return s.decodeDuration(v, node, path)
	}
	if set, handled := s.tryTextUnmarshal(v, node, path); handled {
		return set
	}
	if isByteSliceType(t) {
		return s.decodeBytes(v, node, path)
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			elem := reflect.New(t.Elem())
			if !s.decodeValue(elem.Elem(), node, path) {
				return false
			}
			v.Set(elem)
			return true
		}
		// Update an existing value in place. This preserves fields that are
		// absent from the current input, which matters when Binder composes
		// several decoding passes.
		return s.decodeValue(v.Elem(), node, path)

	case reflect.Struct:
		return s.decodeStructValue(v, node, path)

	case reflect.Slice:
		return s.decodeSlice(v, node, path)

	case reflect.Array:
		return s.decodeArray(v, node, path)

	case reflect.Map:
		return s.decodeMap(v, node, path)

	case reflect.Interface:
		return s.decodeInterface(v, node, path)

	default:
		return s.decodeBasic(v, node, path)
	}
}

// decodeInterface decodes into an interface field. A nil interface receives
// the first value as a string (matching go-playground/form); a non-nil
// interface is decoded by creating a fresh value of its dynamic type.
func (s *decodeState) decodeInterface(v reflect.Value, node *pathNode, path string) bool {
	if v.IsNil() {
		if len(node.values) == 0 {
			s.addError(path, fmt.Errorf("form: cannot decode nested fields into interface type %s", v.Type()))
			return false
		}
		value := reflect.ValueOf(node.values[0])
		if !value.Type().AssignableTo(v.Type()) {
			s.addError(path, fmt.Errorf("form: cannot decode value into interface type %s", v.Type()))
			return false
		}
		v.Set(value)
		return true
	}

	dynamic := v.Elem()
	target := reflect.New(dynamic.Type()).Elem()
	target.Set(dynamic)
	if !s.decodeValue(target, node, path) {
		return false
	}
	v.Set(target)
	return true
}

func (s *decodeState) tryUnmarshal(v reflect.Value, node *pathNode, path string) (set, handled bool) {
	if len(node.values) == 0 || v.Kind() == reflect.Interface {
		return false, false
	}

	if s.d.opts.rejectValueReceiverUnmarshaler && valueReceiverMethod(v.Type(), unmarshalerType) {
		s.addError(path, fmt.Errorf("form: value-receiver UnmarshalValues cannot mutate %s", v.Type()))
		return false, true
	}

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			if !v.Type().Implements(unmarshalerType) {
				return false, false
			}
			elem := reflect.New(v.Type().Elem())
			if err := elem.Interface().(Unmarshaler).UnmarshalValues(node.values); err != nil {
				s.addError(path, err)
				return false, true
			}
			v.Set(elem)
			return true, true
		}
		if v.Type().Implements(unmarshalerType) && v.CanInterface() {
			if err := v.Interface().(Unmarshaler).UnmarshalValues(node.values); err != nil {
				s.addError(path, err)
				return false, true
			}
			return true, true
		}
	}

	if v.CanAddr() {
		addr := v.Addr()
		if addr.Type().Implements(unmarshalerType) && addr.CanInterface() {
			if err := addr.Interface().(Unmarshaler).UnmarshalValues(node.values); err != nil {
				s.addError(path, err)
				return false, true
			}
			return true, true
		}
	}

	if v.Type().Implements(unmarshalerType) && v.CanInterface() {
		if err := v.Interface().(Unmarshaler).UnmarshalValues(node.values); err != nil {
			s.addError(path, err)
			return false, true
		}
		return true, true
	}

	return false, false
}

func (s *decodeState) tryTextUnmarshal(v reflect.Value, node *pathNode, path string) (bool, bool) {
	if len(node.values) == 0 || v.Kind() == reflect.Interface {
		return false, false
	}

	// time.Time has its own layout-aware handling; in particular, an empty
	// *time.Time must follow ZeroEmpty semantics instead of TextUnmarshaler.
	if v.Kind() == reflect.Pointer && v.Type().Elem() == timeType {
		return false, false
	}

	if s.d.opts.rejectValueReceiverUnmarshaler && valueReceiverMethod(v.Type(), textUnmarshalerType) {
		s.addError(path, fmt.Errorf("form: value-receiver UnmarshalText cannot mutate %s", v.Type()))
		return false, true
	}

	t := v.Type()
	if !implementsTextUnmarshalerType(t) {
		return false, false
	}
	data := []byte(node.values[0])

	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			if !t.Implements(textUnmarshalerType) {
				return false, false
			}
			elem := reflect.New(t.Elem())
			if err := elem.Interface().(encoding.TextUnmarshaler).UnmarshalText(data); err != nil {
				s.addError(path, fmt.Errorf("form: cannot unmarshal text into %s: %w", t, err))
				return false, true
			}
			v.Set(elem)
			return true, true
		}
		if t.Implements(textUnmarshalerType) && v.CanInterface() {
			if err := v.Interface().(encoding.TextUnmarshaler).UnmarshalText(data); err != nil {
				s.addError(path, fmt.Errorf("form: cannot unmarshal text into %s: %w", t, err))
				return false, true
			}
			return true, true
		}
	}

	if v.CanAddr() {
		addr := v.Addr()
		if addr.Type().Implements(textUnmarshalerType) && addr.CanInterface() {
			if err := addr.Interface().(encoding.TextUnmarshaler).UnmarshalText(data); err != nil {
				s.addError(path, fmt.Errorf("form: cannot unmarshal text into %s: %w", addr.Type(), err))
				return false, true
			}
			return true, true
		}
	}

	if t.Implements(textUnmarshalerType) && v.CanInterface() {
		if err := v.Interface().(encoding.TextUnmarshaler).UnmarshalText(data); err != nil {
			s.addError(path, fmt.Errorf("form: cannot unmarshal text into %s: %w", t, err))
			return false, true
		}
		return true, true
	}

	if reflect.PointerTo(t).Implements(textUnmarshalerType) {
		tmp := reflect.New(t)
		tmp.Elem().Set(v)
		if err := tmp.Interface().(encoding.TextUnmarshaler).UnmarshalText(data); err != nil {
			s.addError(path, fmt.Errorf("form: cannot unmarshal text into %s: %w", t, err))
			return false, true
		}
		v.Set(tmp.Elem())
		return true, true
	}

	return false, false
}

func implementsTextUnmarshalerType(t reflect.Type) bool {
	if t.Implements(textUnmarshalerType) {
		return true
	}
	return t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(textUnmarshalerType)
}

// valueReceiverMethod reports whether the (possibly pointer) type implements
// the interface through a value receiver and has no embedded field that could
// make the call mutable. This is a best-effort diagnostic for the common
// value-receiver footgun.
func valueReceiverMethod(t reflect.Type, iface reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() == reflect.Interface || !t.Implements(iface) {
		return false
	}
	return !hasMutableAnonymousInterface(t, iface)
}

func hasMutableAnonymousInterface(t reflect.Type, iface reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.Anonymous {
			continue
		}
		ft := field.Type
		if ft.Implements(iface) {
			return true
		}
		if ft.Kind() != reflect.Pointer && reflect.PointerTo(ft).Implements(iface) {
			return true
		}
	}
	return false
}

func (s *decodeState) decodeDuration(v reflect.Value, node *pathNode, path string) bool {
	if len(node.values) == 0 {
		return false
	}
	value := node.values[0]
	if value == "" {
		if s.d.opts.zeroEmpty {
			v.SetZero()
			return true
		}
		return false
	}

	if duration, err := time.ParseDuration(value); err == nil {
		v.SetInt(int64(duration))
		return true
	}

	// Preserve compatibility with the historical numeric-nanoseconds format.
	if nanoseconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		v.SetInt(nanoseconds)
		return true
	}

	parseErr := fmt.Errorf("form: cannot parse %q as duration (use a value like \"1s\" or numeric nanoseconds)", value)
	s.addError(path, parseErr)
	return false
}

func (s *decodeState) decodeTime(v reflect.Value, node *pathNode, path string) bool {
	if len(node.values) == 0 {
		return false
	}
	value := node.values[0]
	if value == "" {
		if s.d.opts.zeroEmpty {
			v.SetZero()
			return true
		}
		return false
	}

	t, err := time.Parse(s.d.opts.timeLayout, value)
	if err != nil {
		s.addError(path, fmt.Errorf("form: cannot parse %q as time: %w", value, err))
		return false
	}
	v.Set(reflect.ValueOf(t))
	return true
}

func (s *decodeState) decodeBytes(v reflect.Value, node *pathNode, path string) bool {
	if len(node.brackets) > 0 {
		s.addError(path, fmt.Errorf("form: byte slices do not support indexed values"))
		return false
	}
	if len(node.values) == 0 {
		return false
	}
	if node.values[0] == "" {
		if s.d.opts.zeroEmpty {
			v.SetZero()
			return true
		}
		return false
	}

	setByteSlice(v, []byte(node.values[0]))
	return true
}

func (s *decodeState) decodeStructValue(v reflect.Value, node *pathNode, path string) bool {
	if len(node.fields) == 0 {
		if len(node.values) > 0 {
			s.addError(path, fmt.Errorf("form: cannot decode values into struct type %s", v.Type()))
			return false
		}
		if len(node.brackets) > 0 {
			s.addError(path, fmt.Errorf("form: invalid indexed path for struct type %s", v.Type()))
			return false
		}
	}
	return s.decodeStruct(v, node, path)
}

func (s *decodeState) decodeBasic(v reflect.Value, node *pathNode, path string) bool {
	if len(node.values) == 0 {
		return false
	}
	value := node.values[0]

	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
		return true

	case reflect.Bool:
		b, err := parseBoolValue(value)
		if err != nil {
			s.addError(path, fmt.Errorf("form: cannot parse %q as bool: %w", value, err))
			return false
		}
		v.SetBool(b)
		return true

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			if s.d.opts.zeroEmpty {
				v.SetZero()
				return true
			}
			return false
		}
		n, err := strconv.ParseInt(value, 10, v.Type().Bits())
		if err != nil {
			s.addError(path, fmt.Errorf("form: cannot parse %q as %s: %w", value, v.Kind(), err))
			return false
		}
		v.SetInt(n)
		return true

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value == "" {
			if s.d.opts.zeroEmpty {
				v.SetZero()
				return true
			}
			return false
		}
		n, err := strconv.ParseUint(value, 10, v.Type().Bits())
		if err != nil {
			s.addError(path, fmt.Errorf("form: cannot parse %q as %s: %w", value, v.Kind(), err))
			return false
		}
		v.SetUint(n)
		return true

	case reflect.Float32, reflect.Float64:
		if value == "" {
			if s.d.opts.zeroEmpty {
				v.SetZero()
				return true
			}
			return false
		}
		f, err := strconv.ParseFloat(value, v.Type().Bits())
		if err != nil {
			s.addError(path, fmt.Errorf("form: cannot parse %q as %s: %w", value, v.Kind(), err))
			return false
		}
		v.SetFloat(f)
		return true

	default:
		s.addError(path, fmt.Errorf("form: unsupported type %s", v.Type()))
		return false
	}
}

func (s *decodeState) decodeSlice(v reflect.Value, node *pathNode, path string) bool {
	elemType := v.Type().Elem()
	set := false

	base := 0
	if s.d.opts.appendSlices && !v.IsNil() {
		base = v.Len()
	}

	if len(node.values) > 0 && directValueSupported(elemType) {
		slice := reflect.MakeSlice(v.Type(), base+len(node.values), base+len(node.values))
		if base > 0 {
			reflect.Copy(slice, v)
		}
		for i, value := range node.values {
			elemNode := &pathNode{values: []string{value}}
			s.decodeValue(slice.Index(base+i), elemNode, path)
		}
		v.Set(slice)
		set = true
	}

	var numericEdges []*pathEdge
	maxIndex := -1
	for _, edge := range node.brackets {
		edge.consumed = true
		edge.node.matched = true
		if !edge.isNum {
			s.addError(bracketPath(path, edge.raw), fmt.Errorf("form: invalid slice index %q", edge.raw))
			continue
		}
		if edge.num >= s.d.opts.maxArraySize {
			s.addError(bracketPath(path, edge.raw), fmt.Errorf("form: index %d exceeds maximum array size %d", edge.num, s.d.opts.maxArraySize))
			continue
		}
		numericEdges = append(numericEdges, edge)
		if edge.num > maxIndex {
			maxIndex = edge.num
		}
	}

	if maxIndex < 0 {
		return set
	}

	var target reflect.Value
	switch {
	case s.d.opts.appendSlices:
		target = v
		if target.Len() < maxIndex+1 {
			grown := reflect.MakeSlice(v.Type(), maxIndex+1, maxIndex+1)
			reflect.Copy(grown, target)
			target = grown
		}
	case set:
		if v.Len() < maxIndex+1 {
			grown := reflect.MakeSlice(v.Type(), maxIndex+1, maxIndex+1)
			reflect.Copy(grown, v)
			v.Set(grown)
		}
		target = v
	default:
		target = reflect.MakeSlice(v.Type(), maxIndex+1, maxIndex+1)
	}

	indexedSet := false
	for _, edge := range numericEdges {
		elemPath := indexedPath(path, edge.num)
		if s.decodeValue(target.Index(edge.num), edge.node, elemPath) {
			indexedSet = true
		}
	}
	if indexedSet {
		v.Set(target)
		set = true
	}
	return set
}

func (s *decodeState) decodeArray(v reflect.Value, node *pathNode, path string) bool {
	elemType := v.Type().Elem()
	set := false

	if len(node.values) > 0 && directValueSupported(elemType) {
		n := len(node.values)
		if n > v.Len() {
			s.addError(path, fmt.Errorf("form: %d values provided for array of length %d", n, v.Len()))
			n = v.Len()
		}
		for i := 0; i < n; i++ {
			elemNode := &pathNode{values: []string{node.values[i]}}
			if s.decodeValue(v.Index(i), elemNode, path) {
				set = true
			}
		}
	}

	for _, edge := range node.brackets {
		edge.consumed = true
		edge.node.matched = true
		if !edge.isNum {
			s.addError(bracketPath(path, edge.raw), fmt.Errorf("form: invalid array index %q", edge.raw))
			continue
		}
		if edge.num >= v.Len() {
			s.addError(bracketPath(path, edge.raw), fmt.Errorf("form: array index %d out of range (length %d)", edge.num, v.Len()))
			continue
		}
		if s.decodeValue(v.Index(edge.num), edge.node, indexedPath(path, edge.num)) {
			set = true
		}
	}

	return set
}

func (s *decodeState) decodeMap(v reflect.Value, node *pathNode, path string) bool {
	if len(node.brackets) == 0 {
		return false
	}

	mp := v
	if v.IsNil() {
		mp = reflect.MakeMap(v.Type())
	}

	set := false
	keyType := v.Type().Key()
	elemType := v.Type().Elem()

	for _, edge := range node.brackets {
		edge.consumed = true
		edge.node.matched = true
		key := reflect.New(keyType).Elem()
		if err := decodeMapKey(key, edge.raw); err != nil {
			s.addError(bracketPath(path, edge.raw), err)
			continue
		}

		elem := reflect.New(elemType).Elem()
		if s.decodeValue(elem, edge.node, bracketPath(path, edge.raw)) {
			mp.SetMapIndex(key, elem)
			set = true
		}
	}

	if set && v.IsNil() {
		v.Set(mp)
	}
	return set
}

func decodeMapKey(key reflect.Value, raw string) error {
	switch key.Kind() {
	case reflect.String:
		key.SetString(raw)
	case reflect.Bool:
		b, err := parseBoolValue(raw)
		if err != nil {
			return fmt.Errorf("form: cannot parse %q as bool map key: %w", raw, err)
		}
		key.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, key.Type().Bits())
		if err != nil {
			return fmt.Errorf("form: cannot parse %q as %s map key: %w", raw, key.Kind(), err)
		}
		key.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, key.Type().Bits())
		if err != nil {
			return fmt.Errorf("form: cannot parse %q as %s map key: %w", raw, key.Kind(), err)
		}
		key.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, key.Type().Bits())
		if err != nil {
			return fmt.Errorf("form: cannot parse %q as %s map key: %w", raw, key.Kind(), err)
		}
		key.SetFloat(f)
	default:
		return fmt.Errorf("form: unsupported map key type %s", key.Type())
	}
	return nil
}

func parseBoolValue(value string) (bool, error) {
	switch value {
	case "1", "t", "T", "true", "TRUE", "True", "on", "yes", "ok":
		return true, nil
	case "", "0", "f", "F", "false", "FALSE", "False", "off", "no":
		return false, nil
	}
	return false, &strconv.NumError{Func: "ParseBool", Num: value, Err: strconv.ErrSyntax}
}

func setByteSlice(v reflect.Value, b []byte) {
	if v.Type() == byteSliceType {
		v.SetBytes(b)
		return
	}
	v.Set(reflect.ValueOf(b).Convert(v.Type()))
}

func isByteSliceType(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem() == byteType
}

func directValueSupported(t reflect.Type) bool {
	if implementsUnmarshalerType(t) {
		return true
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == timeType {
		return true
	}
	switch t.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return false
	}
	return true
}

func implementsUnmarshalerType(t reflect.Type) bool {
	if t.Implements(unmarshalerType) {
		return true
	}
	return t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(unmarshalerType)
}

func implementsMarshalerType(t reflect.Type) bool {
	if t.Implements(marshalerType) {
		return true
	}
	return t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(marshalerType)
}

func isPlainStructType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || t == timeType {
		return false
	}
	return !implementsUnmarshalerType(t)
}

// collectUnknown reports input nodes that the decoder never reached. It runs
// only when strict unknown-key checking is enabled.
func (s *decodeState) collectUnknown(node *pathNode) {
	if node == nil {
		return
	}

	if !node.matched {
		if node.hasContent() {
			key := node.path
			if key == "" {
				key = "<root>"
			}
			if _, exists := s.errs[key]; !exists {
				s.errs[key] = UnknownKeyError{Key: key}
			}
		}
		return
	}

	if len(node.fields) > 0 {
		names := make([]string, 0, len(node.fields))
		for name := range node.fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			s.collectUnknown(node.fields[name])
		}
	}

	for _, edge := range node.brackets {
		if !edge.consumed {
			key := edge.path
			if _, exists := s.errs[key]; !exists {
				s.errs[key] = UnknownKeyError{Key: key}
			}
			continue
		}
		s.collectUnknown(edge.node)
	}
}

func (s *decodeState) requiredSatisfied(node *pathNode, f *fieldInfo, fv reflect.Value) bool {
	if nodeHasNonEmpty(s.field(node, f.name)) {
		return true
	}

	// AnonymousEmbed: required is satisfied by a promoted (flattened) field.
	if f.anonymous && isFlattenableAnonymousType(f.typ) && s.d.opts.anonymousMode == AnonymousEmbed {
		t := f.typ
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		info := s.d.cache.get(t)
		for i := range info.fields {
			if nodeHasNonEmpty(s.field(node, info.fields[i].name)) {
				return true
			}
		}
	}

	// Binder combines multiple sources. A value already bound from a
	// lower-priority source (or a default) satisfies a required marker on a
	// higher-priority tag.
	return s.d.opts.requiredTargetFallback && !boundValueEmpty(fv)
}

// boundValueEmpty reports whether a destination field should be treated as
// empty when Binder checks required fields across several decoding passes.
func boundValueEmpty(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	case reflect.String, reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

func nodeHasNonEmpty(node *pathNode) bool {
	if node == nil {
		return false
	}
	for _, value := range node.values {
		if value != "" {
			return true
		}
	}
	for _, child := range node.fields {
		if nodeHasNonEmpty(child) {
			return true
		}
	}
	for _, edge := range node.brackets {
		if nodeHasNonEmpty(edge.node) {
			return true
		}
	}
	return false
}
