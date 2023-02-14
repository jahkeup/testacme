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
	sharedDNSOnce             sync.Once
	sharedDNS                 *DNS
	sharedDNSNameserverDBOnce sync.Once
	sharedDNSNameserverDB     *NameserverDB
)

func SharedNameserverDB() *NameserverDB {
	sharedDNSNameserverDBOnce.Do(func() {
		db := new(NameserverDB)
		sharedDNSNameserverDB = db
	})
	return sharedDNSNameserverDB
}

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
