// SPDX-License-Identifier: LGPL-3.0-or-later

package randomports

import (
	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	// maxCachedPorts is the number of tracked ports after having been vended to
	// avoid handing out ports that were recently vended (ie: maybe not bound
	// yet, but will be!).
	maxCachedPorts = 64
)

var (
	vendedPorts cacheGate
)

func init() {
	g, err := newCacheGate(maxCachedPorts)
	if err != nil {
		panic("cannot initialize port cache: " + err.Error())
	}
	vendedPorts = g
}

type cacheGate struct {
	ports *lru.Cache[Port, struct{}]
}

func newCacheGate(sz uint8) (cacheGate, error) {
	cache, err := lru.New[Port, struct{}](int(sz))
	if err != nil {
		return cacheGate{}, err
	}

	return cacheGate{
		ports: cache,
	}, nil
}

// InUse returns true when the given port should *not* be vended. If
// false, the given value is tracked and suitable to be vended to callers.
func (gate cacheGate) InUse(p Port) bool {
	if gate.ports == nil {
		// do not vend ports unless the cache is setup
		return true
	}

	exists, _ := gate.ports.ContainsOrAdd(p, struct{}{})
	return exists
}
