// SPDX-License-Identifier: LGPL-3.0-or-later

package randomports

import "strconv"

// Port holds a port number value. This type provides a convenient handle on
// that port number across a few different use cases.
type Port uint16

// Int returns the port as an int.
func (p Port) Int() int {
	return int(p)
}

// Uint returns the port as an uint.
func (p Port) Uint() uint {
	return uint(p)
}

// Uint16 returns the port as an uint16.
func (p Port) Uint16() uint16 {
	return uint16(p)
}

// String prints the port number as a string.
func (p Port) String() string {
	return strconv.Itoa(int(p))
}

// Must is a builder style conditional that can be used to ensure (by panic)
// that a port number is given (ie: not 0).
func (p *Port) Must() *Port {
	if p == nil {
		panic("no port number value")
	}
	return p
}
