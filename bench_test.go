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
	"testing"
	"time"
)

type benchAddress struct {
	City string `form:"city"`
	Zip  int    `form:"zip"`
}

type benchItem struct {
	Name  string `form:"name"`
	Count int    `form:"count"`
}

type benchFlat struct {
	Name    string    `form:"name"`
	Age     int       `form:"age"`
	Active  bool      `form:"active"`
	Score   float64   `form:"score"`
	Created time.Time `form:"created"`
}

type benchNested struct {
	Name    string            `form:"name"`
	Address benchAddress      `form:"address"`
	Tags    []string          `form:"tags"`
	Items   []benchItem       `form:"items"`
	Meta    map[string]string `form:"meta"`
}

var (
	benchFlatEncoder   = NewEncoder()
	benchFlatDecoder   = NewDecoder()
	benchNestedEncoder = NewEncoder()
	benchNestedDecoder = NewDecoder()

	benchSinkMap map[string][]string
	benchSinkErr error
)

func benchFlatValue() benchFlat {
	return benchFlat{
		Name:    "alice",
		Age:     30,
		Active:  true,
		Score:   98.5,
		Created: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}
}

func benchFlatData() map[string][]string {
	return map[string][]string{
		"name":    {"alice"},
		"age":     {"30"},
		"active":  {"true"},
		"score":   {"98.5"},
		"created": {"2024-01-15T10:00:00Z"},
	}
}

func benchNestedValue() benchNested {
	return benchNested{
		Name:    "alice",
		Address: benchAddress{City: "shanghai", Zip: 200000},
		Tags:    []string{"a", "b", "c"},
		Items:   []benchItem{{Name: "x", Count: 1}, {Name: "y", Count: 2}},
		Meta:    map[string]string{"k1": "v1", "k2": "v2"},
	}
}

func benchNestedData() map[string][]string {
	return map[string][]string{
		"name":           {"alice"},
		"address.city":   {"shanghai"},
		"address.zip":    {"200000"},
		"tags":           {"a", "b", "c"},
		"items[0].name":  {"x"},
		"items[0].count": {"1"},
		"items[1].name":  {"y"},
		"items[1].count": {"2"},
		"meta[k1]":       {"v1"},
		"meta[k2]":       {"v2"},
	}
}

func BenchmarkMarshalFlat(b *testing.B) {
	value := benchFlatValue()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkMap, benchSinkErr = benchFlatEncoder.Encode(value)
	}
}

func BenchmarkUnmarshalFlat(b *testing.B) {
	data := benchFlatData()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var target benchFlat
		benchSinkErr = benchFlatDecoder.Decode(data, &target)
	}
}

func BenchmarkMarshalNested(b *testing.B) {
	value := benchNestedValue()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkMap, benchSinkErr = benchNestedEncoder.Encode(value)
	}
}

func BenchmarkUnmarshalNested(b *testing.B) {
	data := benchNestedData()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var target benchNested
		benchSinkErr = benchNestedDecoder.Decode(data, &target)
	}
}
