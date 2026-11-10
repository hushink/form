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

package binding_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/hushink/form/binding"
)

func ExampleBind() {
	type Search struct {
		Keyword string `form:"keyword" query:"q"`
		Page    int    `form:"page" query:"page"`
	}

	req := httptest.NewRequest(http.MethodPost, "/?q=go&page=1", strings.NewReader("keyword=form&page=2"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var search Search
	if err := binding.Bind(req, &search); err != nil {
		panic(err)
	}

	// The form body has higher priority than the query string by default.
	fmt.Println(search.Keyword, search.Page)
	// Output: form 2
}
