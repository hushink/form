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

package output_test

import (
	"fmt"
	"net/http/httptest"

	"github.com/hushink/form/output"
)

func ExampleQueryString() {
	type Search struct {
		Query string `query:"q"`
		Page  int    `query:"page"`
	}

	query, err := output.QueryString(Search{Query: "go", Page: 2})
	if err != nil {
		panic(err)
	}
	fmt.Println(query)
	// Output: page=2&q=go
}

func ExampleWriteJSON() {
	recorder := httptest.NewRecorder()
	if err := output.WriteJSON(recorder, map[string]string{"name": "alice"}); err != nil {
		panic(err)
	}
	fmt.Println(recorder.Header().Get("Content-Type"))
	fmt.Println(recorder.Body.String())
	// Output:
	// application/json; charset=utf-8
	// {"name":"alice"}
}
