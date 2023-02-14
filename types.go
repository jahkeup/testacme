// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"net/http"
	"net/http/httptest"
)

// TestACME describes a minimal ACME testacme implementation suitable for
// integration & functional testing usages.
type TestACME interface {
	Porter
	// Client provides a configured HTTP client (eg: with TLS configured).
	Client() *http.Client
	// Server returns the running ACME HTTP server.
	Server() *httptest.Server
	// ACMEDirectoryURL is the URL to fetch RFC8555 7.1.1 compliant Directory
	// object for this ACME instance.
	//
	// https://www.rfc-editor.org/rfc/rfc8555.html#section-7.1.1
	ACMEDirectoryURL() string
}

// Porter describes the methods provided to lookup the ports used in
// verification.
type Porter interface {
	// HTTPVerificationPort is the port number used in HTTP challenge
	// verification.
	HTTPVerificationPort() int
	// TLSVerificationPort is the port number used in TLS challenge
	// verification.
	TLSVerificationPort() int
}
