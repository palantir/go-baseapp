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
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"encoding/xml"
	"io/ioutil"
	"net/http"

	"github.com/crewjam/saml"
	"github.com/pkg/errors"
)

func WithCertificateFromFile(path string) Param {

	return func(sp *ServiceProvider) error {
		certBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "could not read provided certificate file")
		}

		return WithCertificateFromBytes(certBytes)(sp)
	}

}

func WithCertificateFromBytes(certBytes []byte) Param {
	return func(sp *ServiceProvider) error {
		certPem, _ := pem.Decode(certBytes)
		if certPem == nil {
			return errors.New("could not PEM decode the provided certificate")
		}

		cert, err := x509.ParseCertificate(certPem.Bytes)
		sp.sp.Certificate = cert
		return errors.Wrap(err, "failed to parse provided certificate")
	}

}

func WithKeyFromFile(path string) Param {
	return func(sp *ServiceProvider) error {
		keyBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "could not read provided key file")
		}

		return WithKeyFromBytes(keyBytes)(sp)
	}

}

func WithKeyFromBytes(keyBytes []byte) Param {

	return func(sp *ServiceProvider) error {
		keyPem, _ := pem.Decode(keyBytes)
		if keyPem == nil {
			return errors.New("could not PEM decode the provided private key")
		}

		key, err := x509.ParsePKCS8PrivateKey(keyPem.Bytes)
		if err != nil {
			return errors.Wrap(err, "could not parse provided private key")
		}

		rsaKey, ok := key.(*rsa.PrivateKey)
		sp.sp.Key = rsaKey
		if !ok {
			return errors.New("provided private key was not an RSA key")
		}
		return nil
	}

}

func WithEntityFromURL(url string) Param {

	return func(sp *ServiceProvider) error {
		resp, err := http.Get(url)
		if err != nil {
			return errors.Wrap(err, "failed to download IDP metadata")
		}

		defer func() { _ = resp.Body.Close() }()
		descriptor, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to download IDP metadata")
		}

		return WithEntityFromBytes(descriptor)(sp)
	}

}

func WithEntityFromBytes(metadata []byte) Param {

	return func(sp *ServiceProvider) error {
		var entity saml.EntityDescriptor

		if err := xml.Unmarshal(metadata, &entity); err != nil {
			return errors.Wrap(err, "could not parse returned metadata")
		}
		sp.sp.IDPMetadata = &entity
		return nil
	}

}

func WithACSPath(path string) Param {
	return func(sp *ServiceProvider) error {
		sp.acsPath = path
		return nil
	}
}

func WithMetadataPath(path string) Param {
	return func(sp *ServiceProvider) error {
		sp.metadataPath = path
		return nil
	}
}

func WithForceTLS(force bool) Param {
	return func(sp *ServiceProvider) error {
		sp.forceTLS = force
		return nil
	}
}

func WithLoginCallback(lcb LoginCallback) Param {
	return func(sp *ServiceProvider) error {
		sp.onLogin = lcb
		return nil
	}
}

func WithErrorCallback(ecb ErrorCallback) Param {
	return func(sp *ServiceProvider) error {
		sp.onError = ecb
		return nil
	}
}

func WithIDStore(store IDStore) Param {
	return func(sp *ServiceProvider) error {
		sp.idStore = store
		return nil
	}
}

func WithServiceProvider(s *saml.ServiceProvider) Param {
	return func(sp *ServiceProvider) error {
		sp.sp = s
		return nil
	}
}
