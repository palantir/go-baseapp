// Copyright 2023 Palantir Technologies, Inc.
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

package baseapp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

func Example_ignoreRequests() {
	ignoreHandler := NewIgnoreHandler()

	printHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			fmt.Printf("%s - logs: %t\n", r.URL.Path, IsIgnored(r, IgnoreRule{Logs: true}))
			fmt.Printf("%s - metrics: %t\n", r.URL.Path, IsIgnored(r, IgnoreRule{Metrics: true}))
		})
	}

	exampleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			Ignore(r, IgnoreRule{Logs: true})
		}
	})

	h := ignoreHandler(printHandler(exampleHandler))

	r1 := httptest.NewRequest("GET", "/api/health", nil)
	w1 := httptest.NewRecorder()
	h.ServeHTTP(w1, r1)

	r2 := httptest.NewRequest("GET", "/api/auth/login", nil)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)

	// Output:
	// /api/health - logs: true
	// /api/health - metrics: false
	// /api/auth/login - logs: false
	// /api/auth/login - metrics: false
}
