//go:build never && not_ever

package acmeflow

import (
	"testing"

	"github.com/go-acme/lego/v4/certificate"

	"github.com/jahkeup/testacme"
)

func TestACMEFlow(t *testing.T) {
	pebble := testacme.NewPebble(testacme.NewTestingContext(t))

	user := testacme.ManagedUser(testacme.TestNamedEmail(t)).
		MustRegister(pebble)
	lego := testacme.LegoClient(pebble, user)
	cert, err := lego.Certificate.Obtain(certificate.ObtainRequest{
		Domains: []string{"some.fqdn.test."},
	})
	if err != nil {
		t.Fatalf("lego obtain: %v", err)
	}
	t.Logf("cert domain: %q", cert.Domain)

	// test continues to test renewal or something :)
}
