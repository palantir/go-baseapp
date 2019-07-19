// Copyright 2019 Palantir Technologies, Inc.
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

package saml

import (
	"net/http"
	"time"
)

// IDStore stores the request id for SAML auth flows
type IDStore interface {
	// StoreID stores a request ID in such a way that it can be
	// retreived later using GetIDs
	StoreID(w http.ResponseWriter, r *http.Request, id string) error

	// GetIDs returns the currently valid request ID for SAML authentication
	// If no ID is found an empty string should be returned without an error
	GetID(r *http.Request) (string, error)
}

// cookieIDStore is the default insecure id store useful for testing and development.
// for producion use cases a secure tamper proof implementation of IDStore is strongly recommended.
type cookieIDStore struct{}

func (c cookieIDStore) StoreID(w http.ResponseWriter, _ *http.Request, id string) error {

	http.SetCookie(w, &http.Cookie{
		Name:     "saml_id",
		Value:    id,
		MaxAge:   int(5 * time.Minute.Seconds()),
		HttpOnly: true,
		Path:     "/",
	})

	return nil
}

func (c cookieIDStore) GetID(r *http.Request) (string, error) {
	cookie, err := r.Cookie("saml_id")
	if err != nil {
		if err == http.ErrNoCookie {
			return "", nil
		}

		return "", err
	}

	return cookie.Value, nil
}
