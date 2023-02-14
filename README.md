# testacme

`testacme` is a library enabling the use of ephemeral ACME integration & functional testing.

## Usage

See [test case here](https://github.com/jahkeup/testacme/blob/5b2e6ee1a3d2c32b00cbec3de94069ab755f8889/pebble_test.go#L52-L70) for an example ACME library obtaining a certificate using the `tls-alpn-01` challenge.

Test authors can either use a shared instance or a test-scoped ACME instance:


``` go
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
```

## Related projects

- https://github.com/letsencrypt/pebble

  Minimal ACME implementation - intended for testing use *only*.
  Note: environment variable tuning of Pebble is *not* supported though is not prevented.

- https://github.com/letsencrypt/boulder

  Production-ready ACME implementation - intended for permanent installation in production environments.

- https://datatracker.ietf.org/doc/rfc8555

  ACME IETF RFC

- https://datatracker.ietf.org/doc/rfc6761

  Special-Use Domain Names (`.test` TLD)
