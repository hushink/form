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
	"fmt"
	"reflect"
	"strings"
	"time"
)

const (
	defaultTagName         = "form"
	defaultValueSeparator  = "|"
	defaultMaxArraySize    = 10000
	defaultTimeLayout      = time.RFC3339
	defaultMaxPathDepth    = 256
	defaultMaxEncodeDepth  = 1000
	defaultNamespacePrefix = "."
	defaultNamespaceSuffix = ""
)

// Mode controls which struct fields are processed.
type Mode uint8

const (
	// ModeImplicit processes every exported field that does not use the "-" tag.
	ModeImplicit Mode = iota
	// ModeExplicit processes only fields that carry the configured tag.
	ModeExplicit
	// ModeExplicitEmbedded behaves like ModeExplicit, but also traverses
	// anonymous embedded structs that have no tag so their tagged fields can be
	// processed. Other untagged fields are still skipped.
	ModeExplicitEmbedded
)

// AnonymousMode controls how anonymous embedded structs are addressed.
type AnonymousMode uint8

const (
	// AnonymousEmbed flattens embedded fields into the parent namespace.
	AnonymousEmbed AnonymousMode = iota
	// AnonymousSeparate keeps embedded fields under the embedded type's name.
	AnonymousSeparate
)

// DurationFormat controls how time.Duration values are encoded.
type DurationFormat uint8

const (
	// DurationString encodes durations using time.Duration.String ("1h30m").
	DurationString DurationFormat = iota
	// DurationNanoseconds encodes durations as integer nanoseconds.
	DurationNanoseconds
)

// options holds configuration shared by the Encoder and Decoder.
type options struct {
	tagName           string
	naming            NamingStrategy
	maxArraySize      int
	timeLayout        string
	mode              Mode
	anonymousMode     AnonymousMode
	namespacePrefix   string
	namespaceSuffix   string
	strictUnknownKeys bool
	zeroEmpty         bool
	caseInsensitive   bool

	rejectDuplicateTags            bool
	appendSlices                   bool
	rejectValueReceiverUnmarshaler bool
	topLevelMarshaler              bool
	durationFormat                 DurationFormat

	// Internal options used by Binder to compose several decoding passes.
	skipRequired           bool
	skipDefaults           bool
	requiredTargetFallback bool
}

// Option configures an Encoder or Decoder.
type Option func(*options)

// WithTagName sets the struct tag name used to read field options.
// An empty name is ignored.
func WithTagName(name string) Option {
	return func(o *options) {
		if name != "" {
			o.tagName = name
		}
	}
}

// WithNamingStrategy sets the conversion applied to a field name when the
// configured tag does not specify an explicit name, for example an untagged
// field or `form:",required"`. Explicit tag names always win.
//
// The strategy is used by both Encoder and Decoder. The default is nil, which
// keeps the Go field name unchanged. SnakeCase, KebabCase, and LowerCamelCase
// are provided as ready-to-use strategies; a custom NamingStrategy may be used
// as well.
func WithNamingStrategy(strategy NamingStrategy) Option {
	return func(o *options) {
		if strategy != nil {
			o.naming = strategy
		}
	}
}

// WithCaseInsensitive makes decoding match struct-field segments without
// regard to case. An exact key match always wins; otherwise the
// lexicographically smallest case-insensitive match is used so behavior stays
// deterministic. Encoding is unaffected, and keys inside brackets (map keys,
// slice indexes) remain case-sensitive.
//
// The default is false. The option is primarily useful for map-style sources
// such as form and query parameters; header fields are already canonicalized,
// JSON uses encoding/json's own case-insensitive matching, and XML is
// case-sensitive.
func WithCaseInsensitive(caseInsensitive bool) Option {
	return func(o *options) {
		o.caseInsensitive = caseInsensitive
	}
}

// WithMaxArraySize limits the highest slice/array index accepted while
// decoding indexed form keys (for example "Items[1000000]"). The default is
// 10000. Values less than or equal to zero are ignored.
func WithMaxArraySize(size int) Option {
	return func(o *options) {
		if size > 0 {
			o.maxArraySize = size
		}
	}
}

// WithTimeLayout sets the layout used to encode and decode time.Time values.
// The default is time.RFC3339. An empty layout is ignored.
func WithTimeLayout(layout string) Option {
	return func(o *options) {
		if layout != "" {
			o.timeLayout = layout
		}
	}
}

// WithMode sets how struct fields are selected. The default is ModeImplicit.
func WithMode(mode Mode) Option {
	return func(o *options) {
		o.mode = mode
	}
}

// WithAnonymousMode sets how anonymous embedded structs are addressed.
// The default is AnonymousEmbed.
func WithAnonymousMode(mode AnonymousMode) Option {
	return func(o *options) {
		o.anonymousMode = mode
	}
}

// WithNamespacePrefix sets the separator used between nested field names.
// The default is ".". An empty prefix is ignored.
func WithNamespacePrefix(prefix string) Option {
	return func(o *options) {
		if prefix != "" {
			o.namespacePrefix = prefix
		}
	}
}

// WithNamespaceSuffix sets the suffix appended to nested field names.
// The default is empty.
func WithNamespaceSuffix(suffix string) Option {
	return func(o *options) {
		o.namespaceSuffix = suffix
	}
}

// WithStrictUnknownKeys makes decoding report unknown input keys as
// DecodeErrors. The default is false (unknown keys are ignored).
func WithStrictUnknownKeys(strict bool) Option {
	return func(o *options) {
		o.strictUnknownKeys = strict
	}
}

// WithZeroEmpty makes an empty input value reset numeric, time, and []byte
// fields to their zero value instead of leaving them untouched.
func WithZeroEmpty(zero bool) Option {
	return func(o *options) {
		o.zeroEmpty = zero
	}
}

// WithRejectDuplicateTags reports duplicate tag names within a struct as
// ErrDuplicateTag instead of allowing both fields to share the same key.
func WithRejectDuplicateTags(reject bool) Option {
	return func(o *options) {
		o.rejectDuplicateTags = reject
	}
}

// WithAppendSlices makes decoding append to existing slice values instead of
// replacing them. The default is false (replace).
func WithAppendSlices(appendSlices bool) Option {
	return func(o *options) {
		o.appendSlices = appendSlices
	}
}

// WithRejectValueReceiverUnmarshalers reports value-receiver UnmarshalValues or
// UnmarshalText implementations as errors, because they cannot mutate the
// destination field.
func WithRejectValueReceiverUnmarshalers(reject bool) Option {
	return func(o *options) {
		o.rejectValueReceiverUnmarshaler = reject
	}
}

// WithTopLevelMarshaler makes Encode honor a Marshaler implemented by the
// top-level struct instead of always traversing its fields. The default is
// false.
func WithTopLevelMarshaler(use bool) Option {
	return func(o *options) {
		o.topLevelMarshaler = use
	}
}

// WithDurationFormat sets how time.Duration values are encoded.
// The default is DurationString ("1s", "1m30s"); decoding accepts both
// duration strings and numeric nanoseconds.
func WithDurationFormat(format DurationFormat) Option {
	return func(o *options) {
		o.durationFormat = format
	}
}

// WithoutDefaults disables applying `default=...` tag values during decoding.
//
// It is intended for multi-pass decoders that apply defaults in a separate
// pass, such as package binding. Encoding is unaffected.
func WithoutDefaults() Option {
	return func(o *options) {
		o.skipDefaults = true
	}
}

// WithoutRequiredChecks disables `required` validation during decoding,
// including the check that rejects a combination of `required` and `default`.
// Encoding is unaffected.
//
// It is intended for multi-pass decoders that validate required fields in a
// separate pass, such as package binding.
func WithoutRequiredChecks() Option {
	return func(o *options) {
		o.skipRequired = true
	}
}

// WithRequiredFallback allows a non-zero destination value to satisfy a
// `required` field when the input key is absent. An empty input value still
// fails the check.
//
// It is intended for multi-pass decoders, where a value may already have been
// bound from another source.
func WithRequiredFallback() Option {
	return func(o *options) {
		o.requiredTargetFallback = true
	}
}

func defaultOptions() options {
	return options{
		tagName:         defaultTagName,
		maxArraySize:    defaultMaxArraySize,
		timeLayout:      defaultTimeLayout,
		mode:            ModeImplicit,
		anonymousMode:   AnonymousEmbed,
		namespacePrefix: defaultNamespacePrefix,
		namespaceSuffix: defaultNamespaceSuffix,
		durationFormat:  DurationString,
	}
}

// joinPath builds a nested form key using the configured namespace prefix and
// suffix. Root-level names are returned unchanged.
func (o options) joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + o.namespacePrefix + name + o.namespaceSuffix
}

// validateNamespace rejects separators that cannot be parsed unambiguously.
func (o options) validateNamespace() error {
	if o.namespaceSuffix != "" && strings.Contains(o.namespaceSuffix, o.namespacePrefix) {
		return fmt.Errorf("form: namespace suffix %q must not contain the namespace prefix %q", o.namespaceSuffix, o.namespacePrefix)
	}
	return nil
}

// fieldInfo holds parsed struct tag information.
type fieldInfo struct {
	index        int
	name         string
	typ          reflect.Type
	omitempty    bool
	skip         bool
	anonymous    bool
	invalidName  bool
	required     bool
	hasDefault   bool
	defaultValue []string // parsed default values
}

// parseTag parses a struct field's tag with the given tag name. The naming
// strategy applies only when the tag does not provide an explicit name.
func parseTag(tagName string, field reflect.StructField, naming NamingStrategy) fieldInfo {
	tag := field.Tag.Get(tagName)
	if tag == "-" {
		return fieldInfo{skip: true}
	}

	info := fieldInfo{}

	if tag == "" {
		info.name = applyNaming(naming, field.Name)
		return info
	}

	parts := strings.Split(tag, ",")
	if parts[0] == "" {
		info.name = applyNaming(naming, field.Name)
	} else {
		info.name = parts[0]
	}

	for _, opt := range parts[1:] {
		if opt == "omitempty" {
			info.omitempty = true
		} else if opt == "required" {
			info.required = true
		} else if strings.HasPrefix(opt, "default=") {
			defaultStr := strings.TrimPrefix(opt, "default=")
			info.hasDefault = true
			info.defaultValue = strings.Split(defaultStr, defaultValueSeparator)
		}
	}

	return info
}
