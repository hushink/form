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
	"sync"
)

// structInfo is the cached, tag-parsed representation of a struct type.
type structInfo struct {
	fields       []fieldInfo
	duplicateErr error
}

// structCache caches struct metadata per reflect.Type. Generic instantiations
// such as Box[int] and Box[string] are distinct reflect.Types, so each gets its
// own entry.
type structCache struct {
	tagName          string
	mode             Mode
	naming           NamingStrategy
	rejectDuplicates bool
	m                sync.Map // map[reflect.Type]*structInfo
}

func newStructCache(tagName string, mode Mode, naming NamingStrategy, rejectDuplicates bool) *structCache {
	return &structCache{
		tagName:          tagName,
		mode:             mode,
		naming:           naming,
		rejectDuplicates: rejectDuplicates,
	}
}

func (c *structCache) get(t reflect.Type) *structInfo {
	if cached, ok := c.m.Load(t); ok {
		return cached.(*structInfo)
	}

	info := c.build(t)
	actual, _ := c.m.LoadOrStore(t, info)
	return actual.(*structInfo)
}

func isEmbeddedStructType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || t == timeType {
		return false
	}
	return !implementsMarshalerType(t) && !implementsUnmarshalerType(t)
}

func (c *structCache) build(t reflect.Type) *structInfo {
	info := &structInfo{fields: make([]fieldInfo, 0, t.NumField())}
	seen := make(map[string]string, t.NumField())
	var duplicates []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// Unexported embedded structs (value or pointer) are supported because
		// their exported subfields are settable; other unexported fields are
		// ignored.
		if !field.IsExported() && (!field.Anonymous || !isEmbeddedStructType(field.Type)) {
			continue
		}

		rawTag := field.Tag.Get(c.tagName)
		if c.mode != ModeImplicit && rawTag == "" {
			embedded := c.mode == ModeExplicitEmbedded &&
				field.Anonymous && isEmbeddedStructType(field.Type)
			if !embedded {
				continue
			}
		}

		fi := parseTag(c.tagName, field, c.naming)
		if fi.skip {
			continue
		}
		fi.index = i
		fi.typ = field.Type
		fi.anonymous = field.Anonymous
		fi.invalidName = strings.ContainsAny(fi.name, ".[]")
		info.fields = append(info.fields, fi)

		if c.rejectDuplicates {
			if first, ok := seen[fi.name]; ok {
				duplicates = append(duplicates, fmt.Sprintf("%q used by fields %s and %s", fi.name, first, field.Name))
			} else {
				seen[fi.name] = field.Name
			}
		}
	}

	if len(duplicates) > 0 {
		info.duplicateErr = fmt.Errorf("%w: %s", ErrDuplicateTag, strings.Join(duplicates, "; "))
	}
	return info
}
