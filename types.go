// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"net/http"
	"net/http/httptest"
)

type TestACME interface {
	Porter
	Server() *httptest.Server
	Client() *http.Client
	ACMEDirectoryURL() string
}
