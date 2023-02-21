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

var (
	bindAddress = ":0"
)

func init() {
	los, err := loopbackAddresses()
	if err == nil && len(los) > 0 {
		bindAddress = fmt.Sprintf("%s:0", los[0].String())
	}
}

func loopbackAddresses() ([]net.IP, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	addrs := []net.IP{}
	for _, iface := range ifs {
		if iface.Flags&net.FlagLoopback != net.FlagLoopback {
			continue
		}
		ifaddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, ifaddr := range ifaddrs {
			switch v := ifaddr.(type) {
			case *net.IPNet:
				if v.IP.IsLoopback() {
					addrs = append(addrs, v.IP)
				}
			}
		}
	}

	return addrs, nil
}

// One provides a single port. For convenience.
func One() (Port, error) {
	if ps, err := RandomPorts(1); err == nil {
		return ps[0], nil
	} else {
		return 0, err
	}
}

// Two provides a set of ports. For convenience.
func Two() (Port, Port, error) {
	if ps, err := RandomPorts(2); err == nil {
		return ps[0], ps[1], nil
	} else {
		return 0, 0, err
	}
}

// RandomPorts returns a list of port numbers that are safe-to-assume to be free. An
// internal list is used track vended ports to avoid concurrent users from
// seeing conflicting ports.
func RandomPorts(n int) ([]Port, error) {
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

			if Reserve(p) {
				ports[i] = p
				continue ports
			} else {
				// don't count this against the attempts, because our cache
				// filtered it out.
				goto bind
			}
		}

		return nil, errors.New("unable to allocate ports")
	}

	return ports, nil
}

var listenConfig net.ListenConfig

func freeListenPort(ctx context.Context) (Port, error) {
	const network = "udp" // use UDP for its smaller socket overhead
	l, err := listenConfig.ListenPacket(context.Background(), network, bindAddress)
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
