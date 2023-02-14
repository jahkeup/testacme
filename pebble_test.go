// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPebble_DirectoryCheck(t *testing.T) {
	pebble := NewPebble(
		NewTestingContext(t),
		WithPebbleLogger(log.Default()))
	t.Logf("pebble server url: %q", pebble.Server().URL)

	resp, err := pebble.Client().Get(pebble.ACMEDirectoryURL())
	assert.NoError(t, err)
	t.Logf("response: %#v", resp)
	require.NotNil(t, resp, "should have dir response")

	directory := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&directory)
	require.NoError(t, err, "should have directory json")
	s, _ := json.MarshalIndent(directory, "dir > ", "  ")
	t.Logf("directory: %s", s)
}

func TestSharedPebble_DirectoryCheck(t *testing.T) {
	// Use the shared pebble
	pebble := SharedPebble()
	t.Logf("pebble server url: %q", pebble.Server().URL)

	resp, err := pebble.Client().Get(pebble.ACMEDirectoryURL())
	assert.NoError(t, err)
	t.Logf("response: %#v", resp)
	require.NotNil(t, resp, "should have dir response")

	directory := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&directory)
	require.NoError(t, err, "should have directory json")
	s, _ := json.MarshalIndent(directory, "dir > ", "  ")
	t.Logf("directory: %s", s)
}

func TestSharedPebble_TLSALPN01(t *testing.T) {
	pebble := SharedPebble()

	user := ManagedUser(TestNamedEmail(t)).MustRegister(pebble)
	t.Logf("registration: %#v", user.GetRegistration())
	client := LegoClient(pebble, user)

	port := pebble.TLSVerificationPort()
	t.Logf("tls verification port: %d", port)
	provider := tlsalpn01.NewProviderServer("", fmt.Sprintf("%d", port))
	t.Logf("provider: %#v", provider)
	client.Challenge.SetTLSALPN01Provider(provider)

	cert, err := client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: []string{"pretend.hostname.com"},
	})
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}
