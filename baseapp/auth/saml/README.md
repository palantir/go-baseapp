This package provides basic integration for baseapp with a SAML IDP.  The package handles the auth flow with the IDP (ACS and redirect).  It does not implement any session tracking/memory so users must implement their own.

There are 3 main integration points users should be aware of:

1. `ErrorCallback`: called whenever an error occurs during the auth flow.  The callback is expected to send a response to the request
2. `LoginCallback`: called when a user successfully authenticates.  The callback should create a session based on the passed in assertion.
3. `IDStore`: used to store SAML requestID's to prevent assertion spoofing.

## Example
A simple example of how to integrate the saml package into baseapp

```golang
logger := baseapp.NewLogger(baseapp.LoggingConfig{
    Level:  "debug",
    Pretty: true,
})

p := baseapp.DefaultParams(logger, "")
s, err := baseapp.NewServer(baseapp.HTTPConfig{
    Address: "127.0.0.1",
    Port:    8000,
}, p...)

if err != nil {
    panic(err)
}

spParam := []saml.Param{
    saml.WithCertificateFromFile("./cert.pem"),
    saml.WithKeyFromFile("./key"),
    saml.WithEntityFromURL("http://localhost:8080/simplesaml/saml2/idp/metadata.php"),
    saml.WithACSPath("/saml/acs"),
    saml.WithMetadataPath("/saml/metadata"),
}

sp, err := saml.NewServiceProvider(spParam...)
if err != nil {
    panic(err)
}

s.Mux().Handle(pat.Post("/saml/acs"), sp.ACSHandler())
s.Mux().Handle(pat.Get("/saml/metadata"), sp.MetadataHandler())
s.Mux().HandleFunc(pat.Get("/auth"), sp.DoAuth)

_ = s.Start()
```
