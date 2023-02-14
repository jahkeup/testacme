// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"context"
	"fmt"
	"sync"
)

var (
	sharedPebbleOnce sync.Once
	sharedPebble     *Pebble
)

// SharedPebble provides a shared instance of the Pebble ACME server, suitable
// for concurrent use. Note that some test cases & implementations may want (and
// should!) create a new instance for each test. However, the SharedPebble
// instance should be preferred to reduce the computational load on the system
// during tests as cryptographic materials are generated on the fly.
func SharedPebble() Pebble {
	sharedPebbleOnce.Do(func() {
		pebble := NewPebble(context.Background())

		// Start up the servers ahead of time - if the Shared instance is used,
		// then its probably wanted among multiple test code paths.
		pebble.Start()

		sharedPebble = &pebble
	})

	s := *sharedPebble
	s.shutdownTestACME = func() { panic("you can't shutdown the shared instance") }

	return s
}

var (
	sharedDNSNameserverDBOnce sync.Once
	sharedDNSNameserverDB     *NameserverDB
)

// SharedNameserverDB provides a shared instance of the NameserverDB used to
// respond to DNS queries. This instance is used when calling SharedDNS -
// callers may add additional responses using the provided methods.
func SharedNameserverDB() *NameserverDB {
	sharedDNSNameserverDBOnce.Do(func() {
		db := new(NameserverDB)
		sharedDNSNameserverDB = db
	})
	return sharedDNSNameserverDB
}

var (
	sharedDNSOnce             sync.Once
	sharedDNS                 *DNS
)

// SharedDNS provides a shared instance of the helper DNS server, suitable for
// concurrent use.
func SharedDNS() *DNS {
	sharedDNSOnce.Do(func() {
		ns, err := NewDNS(context.Background(), SharedNameserverDB())
		if err != nil {
			panic(fmt.Sprintf("cannot start shared DNS server: %v", err))
		}
		sharedDNS = ns
	})

	return sharedDNS
}
