// Copyright 2018 Palantir Technologies, Inc.
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

package oauth2

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
)

var (
	DefaultSessionName = "oauth2"
	sessionStateKey    = "state"
)

type SessionStateStore struct {
	Sessions sessions.Store
}

func (s *SessionStateStore) GenerateState(w http.ResponseWriter, r *http.Request) (string, error) {
	// ignore the error because we always get a session, even if its a new one
	sess, _ := s.Sessions.Get(r, DefaultSessionName)

	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", errors.Wrap(err, "failed to generate state value")
	}

	state := hex.EncodeToString(b)
	sess.Values[sessionStateKey] = state
	return state, sess.Save(r, w)
}

func (s *SessionStateStore) VerifyState(r *http.Request, expected string) (bool, error) {
	sess, err := s.Sessions.Get(r, DefaultSessionName)
	if err != nil {
		return false, err
	}
	st, ok := sess.Values[sessionStateKey]
	if !ok {
		return false, errors.New("no state value found in the session")
	}

	state, ok := st.(string)
	if !ok {
		return false, errors.New("session state value was an incorrect type")
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(state)) == 1, nil
}
