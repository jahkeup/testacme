# testacme

`testacme` is a library enabling the use of ephemeral ACME integration & functional testing.

This library is dual licensed as either MIT OR LGPL-3.0+ - please consider contributing features and fixes to https://github.com/jahkeup/testacme regardless of your licensing preferences.

```yaml
SPDX-License-Identifier: MIT OR LGPL-3.0-or-later
```

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
  Note: environment variable tuning of Pebble is *not* supported and conflicting variables are unset.

- https://github.com/letsencrypt/boulder

  Production-ready ACME implementation - intended for permanent installation in production environments.

- https://datatracker.ietf.org/doc/rfc8555

  ACME IETF RFC

- https://datatracker.ietf.org/doc/rfc6761

  Special-Use Domain Names (`.test` TLD)
