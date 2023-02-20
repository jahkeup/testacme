// SPDX-License-Identifier: LGPL-3.0-or-later

package randomports

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

const (
	// bindMaxAttempts is the maximum number of times to retry creating sockets.
	// Note, finding already vended ports is doesn't count against the number of
	// attempts made.
	bindMaxAttempts uint8 = 3
	// bindSocketOverhead is used to limit the total time spent allocating ports
	// (by creating listeners). The requested number is multiplied by this
	// factor to scale accordingly.
	bindSocketOverhead = 100 * time.Millisecond
)

// Random returns a list of port numbers that are safe-to-assume to be free. An
// internal list is used track vended ports to avoid concurrent users from
// seeing conflicting ports.
func Random(n int) ([]Port, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(n)*bindSocketOverhead)
	ports, err := randomWithContext(ctx, n)
	cancel()
	return ports, err
}

func randomWithContext(ctx context.Context, n int) ([]Port, error) {
	if n == 0 {
		return nil, errors.New("invalid arg: 0 random ports")
	}

	ports := make([]Port, n)

ports:
	for i := 0; i < n; i++ {
		for j := uint8(0); j < bindMaxAttempts; j++ {
		bind:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			p, err := freeListenPort(ctx)
			if err != nil {
				if j+1 == bindMaxAttempts {
					return nil, fmt.Errorf("cannot allocate port: %w", err)
				}
				continue
			}

			if vendedPorts.InUse(p) {
				// don't count this against the attempts, because our cache
				// filtered it out.
				goto bind
			} else {
				ports[i] = p
				continue ports
			}
		}

		return nil, errors.New("unable to allocate ports")
	}

	return ports, nil
}

var listenConfig net.ListenConfig

func freeListenPort(ctx context.Context) (Port, error) {
	const network = "udp" // use UDP for its smaller socket overhead
	l, err := listenConfig.ListenPacket(context.Background(), network, "0:0")
	if err != nil {
		return 0, err
	}
	l.Close()

	_, port, err := net.SplitHostPort(l.LocalAddr().String())
	if err != nil {
		return 0, fmt.Errorf("bad address from net: %w", err)
	}

	p, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("bad port from net: %w", err)
	}

	return Port(p), nil
}
