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
	"encoding/xml"
	"net/http"
	"net/url"

	"github.com/crewjam/saml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/hlog"
)

type Error struct {
	Err error

	// The suggested HTTP response code for this error
	ResponseCode int
}

func (s Error) Error() string {
	return s.Err.Error()
}

func newError(err error, status int) Error {
	return Error{
		Err:          err,
		ResponseCode: status,
	}
}

// ErrorCallback is called whenever an error occurs in the saml package.
// The callback is expected to send a response to the request. The http.ResponseWriter
// will not have been written to, allowing the callback to send headers if desired.
type ErrorCallback func(http.ResponseWriter, *http.Request, Error)

// LoginCallback is called whenever an auth flow is successfully completed.
// The callback is responsible preserving the login state.
type LoginCallback func(http.ResponseWriter, *http.Request, *saml.Assertion)

// ServiceProvider is capable of handling a SAML login. It provides
// an http.Handler (via ACSHandler) which can process the http POST from the SAML IDP. It accepts callbacks for both error and
// success conditions so that clients can take action after the auth flow is complete. It also provides a handler
// for serving the service provider metadata XML.
type ServiceProvider struct {
	sp *saml.ServiceProvider

	acsPath      string
	metadataPath string
	logoutPath   string

	forceTLS          bool
	disableEncryption bool

	onError ErrorCallback
	onLogin LoginCallback
	idStore IDStore
}

type Param func(sp *ServiceProvider) error

// NewServiceProvider returns a ServiceProvider. The configuration of the ServiceProvider
// is a result of combinging settings provided to this method and values parsed from the IDP's metadata.
func NewServiceProvider(params ...Param) (*ServiceProvider, error) {

	sp := &ServiceProvider{
		sp: &saml.ServiceProvider{},
	}

	for _, p := range params {
		if err := p(sp); err != nil {
			return nil, err
		}
	}

	if sp.sp.Certificate == nil || sp.sp.Key == nil {
		return nil, errors.New("a certificate and key must be provided")
	}

	if sp.sp.IDPMetadata == nil {
		return nil, errors.New("the IDP Metadata must be provided")
	}

	if sp.acsPath == "" || sp.metadataPath == "" {
		return nil, errors.New("ACS Path and Metadatda path must be provided")
	}

	if sp.onError == nil {
		sp.onError = DefaultErrorCallback
	}

	if sp.onLogin == nil {
		sp.onLogin = DefaultLoginCallback
	}

	if sp.idStore == nil {
		sp.idStore = cookieIDStore{}
	}

	return sp, nil
}

func DefaultErrorCallback(w http.ResponseWriter, r *http.Request, err Error) {
	hlog.FromRequest(r).Error().Err(err.Err).Msg("saml error")
	http.Error(w, http.StatusText(err.ResponseCode), err.ResponseCode)
}

func DefaultLoginCallback(w http.ResponseWriter, r *http.Request, resp *saml.Assertion) {
	w.WriteHeader(http.StatusOK)
}

func (s *ServiceProvider) getSAMLSettingsForRequest(r *http.Request) *saml.ServiceProvider {
	// make a copy in case different requests have different host headers
	newSP := *s.sp

	u := url.URL{
		Host:   r.Host,
		Scheme: "http",
	}

	if s.forceTLS || r.TLS != nil {
		u.Scheme = "https"
	}

	u.Path = s.metadataPath
	newSP.MetadataURL = u

	u.Path = s.acsPath
	newSP.AcsURL = u

	u.Path = s.logoutPath
	newSP.SloURL = u

	return &newSP
}

// DoAuth takes an http.ResponseWriter that has not been written to yet, and conducts and SP initiated login
// If the flow proceeds correctly the user should be redirected to the handler provided by ACSHandler().
func (s *ServiceProvider) DoAuth(w http.ResponseWriter, r *http.Request) {
	sp := s.getSAMLSettingsForRequest(r)

	request, err := sp.MakeAuthenticationRequest(sp.GetSSOBindingLocation(saml.HTTPRedirectBinding))
	if err != nil {
		s.onError(w, r, newError(errors.Wrap(err, "failed to create authentication request"), http.StatusInternalServerError))
		return
	}

	if err := s.idStore.StoreID(w, r, request.ID); err != nil {
		s.onError(w, r, newError(errors.Wrap(err, "failed to store SAML request id"), http.StatusInternalServerError))
		return
	}

	target := request.Redirect("")

	http.Redirect(w, r, target.String(), http.StatusFound)
}

// ACSHandler returns an http.Handler which is capable of validating and processing SAML Responses.
func (s *ServiceProvider) ACSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sp := s.getSAMLSettingsForRequest(r)
		if err := r.ParseForm(); err != nil {
			s.onError(w, r, newError(errors.Wrap(err, "could not parse ACS form"), http.StatusForbidden))
			return
		}
		id, err := s.idStore.GetID(r)
		if err != nil {
			s.onError(w, r, newError(errors.Wrap(err, "could not retrieve id"), http.StatusForbidden))
			return
		}
		assertion, err := sp.ParseResponse(r, []string{id})

		if err != nil {
			if parseErr, ok := err.(*saml.InvalidResponseError); ok {
				err = parseErr.PrivateErr
			}
			s.onError(w, r, newError(errors.Wrap(err, "failed to validate SAML assertion"), http.StatusForbidden))
			return
		}

		s.onLogin(w, r, assertion)
	})

}

// MetadataHandler returns an http.Handler which sends the generated metadata XML in response to a request
func (s *ServiceProvider) MetadataHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metadata := s.getSAMLSettingsForRequest(r).Metadata()

		// post-process the metadata to account for issues in crewjam/saml
		// struct navigation is hardcoded for the return value at implementation time

		if s.logoutPath == "" {
			// remove SingleLogoutService elements if the logout path is not set
			metadata.SPSSODescriptors[0].SSODescriptor.SingleLogoutServices = nil
		}
		if s.disableEncryption {
			// remove encryption keys from metadata
			role := &(metadata.SPSSODescriptors[0].SSODescriptor.RoleDescriptor)
			for i, k := range role.KeyDescriptors {
				if k.Use == "encryption" {
					role.KeyDescriptors = append(role.KeyDescriptors[:i], role.KeyDescriptors[i+1:]...)
				}
			}
		}

		md, err := xml.Marshal(metadata)
		if err != nil {
			s.onError(w, r, newError(errors.Wrap(err, "failed to generate service provider metadata"), http.StatusInternalServerError))
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		// The error isn't handlable or recoverable so don't handle it
		// assign to _ to placate errcheck
		_, _ = w.Write(md)
	})
}
