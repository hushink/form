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
	"reflect"
	"time"
)

// Unmarshaler is the interface implemented by types that can unmarshal
// a slice of string values into themselves.
// Custom types can implement this interface to control their own field decoding.
type Unmarshaler interface {
	UnmarshalValues(values []string) error
}

// Marshaler is the interface implemented by types that can marshal
// themselves into a slice of string values.
// Custom types can implement this interface to control their own field encoding.
type Marshaler interface {
	MarshalValues() ([]string, error)
}

var (
	unmarshalerType     = reflect.TypeFor[Unmarshaler]()
	marshalerType       = reflect.TypeFor[Marshaler]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
	textMarshalerType   = reflect.TypeFor[encoding.TextMarshaler]()
	timeType            = reflect.TypeFor[time.Time]()
	durationType        = reflect.TypeFor[time.Duration]()
	byteType            = reflect.TypeFor[byte]()
	byteSliceType       = reflect.TypeFor[[]byte]()
)
