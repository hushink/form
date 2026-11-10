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

package form_test

import (
	"fmt"

	"github.com/hushink/form"
)

func ExampleMarshal() {
	type Address struct {
		City string `form:"city"`
	}
	type User struct {
		Name    string   `form:"name"`
		Tags    []string `form:"tags"`
		Address Address  `form:"address"`
	}

	values, err := form.Marshal(User{
		Name:    "Alice",
		Tags:    []string{"admin", "user"},
		Address: Address{City: "Shanghai"},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(values["name"])
	fmt.Println(values["tags"])
	fmt.Println(values["address.city"])

	// Output:
	// [Alice]
	// [admin user]
	// [Shanghai]
}

func ExampleUnmarshal() {
	type Address struct {
		City string `form:"city"`
	}
	type User struct {
		Name    string   `form:"name"`
		Tags    []string `form:"tags"`
		Address Address  `form:"address"`
	}

	values := map[string][]string{
		"name":         {"Alice"},
		"tags":         {"admin", "user"},
		"address.city": {"Shanghai"},
	}

	var user User
	if err := form.Unmarshal(values, &user); err != nil {
		panic(err)
	}

	fmt.Println(user.Name)
	fmt.Println(user.Tags)
	fmt.Println(user.Address.City)

	// Output:
	// Alice
	// [admin user]
	// Shanghai
}

func ExampleNewDecoder() {
	type Query struct {
		Page  int `query:"page"`
		Limit int `query:"limit"`
	}

	decoder := form.NewDecoder(form.WithTagName("query"))

	var query Query
	if err := decoder.Decode(map[string][]string{
		"page":  {"2"},
		"limit": {"20"},
	}, &query); err != nil {
		panic(err)
	}

	fmt.Println(query.Page, query.Limit)

	// Output:
	// 2 20
}
