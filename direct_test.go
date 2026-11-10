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
	"reflect"
	"testing"
	"time"
)

type directCustom struct{}

func (*directCustom) UnmarshalValues([]string) error { return nil }

func TestDirectValueSupported(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want bool
	}{
		{"int", reflect.TypeOf(0), true},
		{"*int", reflect.TypeOf((*int)(nil)), true},
		{"time", reflect.TypeOf(time.Time{}), true},
		{"struct", reflect.TypeOf(struct{}{}), false},
		{"slice", reflect.TypeOf([]int{}), false},
		{"array", reflect.TypeOf([2]int{}), false},
		{"map", reflect.TypeOf(map[string]int{}), false},
		{"any", reflect.TypeOf((*any)(nil)).Elem(), true},
		{"custom", reflect.TypeOf(directCustom{}), true},
		{"*custom", reflect.TypeOf((*directCustom)(nil)), true},
		{"*struct", reflect.TypeOf((*struct{ A int })(nil)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := directValueSupported(tt.typ); got != tt.want {
				t.Errorf("directValueSupported(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}
