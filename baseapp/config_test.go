// Copyright 2020 Palantir Technologies, Inc.
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
	"os"
	"reflect"
	"testing"
	"time"
)

func TestSetValuesFromEnv(t *testing.T) {
	tests := map[string]struct {
		Input     func(*HTTPConfig)
		Prefix    string
		Variables map[string]string
		Output    func(*HTTPConfig)
	}{
		"noVariables": {
			Input: func(c *HTTPConfig) {
				c.Address = "0.0.0.0"
				c.Port = 8080
			},
			Output: func(c *HTTPConfig) {
				c.Address = "0.0.0.0"
				c.Port = 8080
			},
		},
		"overwriteExisting": {
			Input: func(c *HTTPConfig) {
				c.Address = "127.0.0.1"
			},
			Variables: map[string]string{
				"ADDRESS": "0.0.0.0",
			},
			Output: func(c *HTTPConfig) {
				c.Address = "0.0.0.0"
			},
		},
		"allVariables": {
			Variables: map[string]string{
				"ADDRESS":            "127.0.0.1",
				"PORT":               "8080",
				"PUBLIC_URL":         "https://baseapp.company.domain",
				"TLS_CERT_FILE":      "/path/to/cert.crt",
				"TLS_KEY_FILE":       "/path/to/key.pem",
				"SHUTDOWN_WAIT_TIME": "5m",
			},
			Output: func(c *HTTPConfig) {
				c.Address = "127.0.0.1"
				c.Port = 8080
				c.PublicURL = "https://baseapp.company.domain"
				c.TLSConfig = &TLSConfig{
					CertFile: "/path/to/cert.crt",
					KeyFile:  "/path/to/key.pem",
				}
				d := 5 * time.Minute
				c.ShutdownWaitTime = &d
			},
		},
		"withPrefix": {
			Input: func(c *HTTPConfig) {
				c.PublicURL = "https://baseapp.company.domain"
			},
			Prefix: "TEST_",
			Variables: map[string]string{
				"TEST_PUBLIC_URL": "https://app.company.domain",
			},
			Output: func(c *HTTPConfig) {
				c.PublicURL = "https://app.company.domain"
			},
		},
		"emptyValues": {
			Input: func(c *HTTPConfig) {
				c.PublicURL = "https://baseapp.company.domain"
			},
			Variables: map[string]string{
				"PUBLIC_URL": "",
			},
			Output: func(c *HTTPConfig) {
				c.PublicURL = ""
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			for k, v := range test.Variables {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("failed to set environment variable: %s: %v", k, err)
				}
			}

			defer func() {
				for k := range test.Variables {
					if err := os.Unsetenv(k); err != nil {
						t.Fatalf("failed to clear environment variable: %s: %v", k, err)
					}
				}
			}()

			var in HTTPConfig
			if test.Input != nil {
				test.Input(&in)
			}

			var out HTTPConfig
			if test.Output != nil {
				test.Output(&out)
			}

			in.SetValuesFromEnv(test.Prefix)

			if !reflect.DeepEqual(out, in) {
				t.Errorf("incorrect configuration\nexpected: %+v\n  actual: %+v", out, in)
			}
		})
	}
}
