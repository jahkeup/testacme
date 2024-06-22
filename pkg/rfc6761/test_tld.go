// SPDX-License-Identifier: MIT OR LGPL-3.0-or-later

package rfc6761

import (
	"strings"

	"github.com/miekg/dns"
)

const (
	// TestTLD is the rfc6761 designated TLD name reserved specifically
	// for testing usages.
	//
	// https://www.rfc-editor.org/rfc/rfc6761#section-6.2
	TestTLD = "test"
)

// CanonicalTest transforms the provided dn into a TestTLD rooted, fully
// qualified, canonicalized domain name.
func CanonicalTest(dn string) string {
	n := dns.CanonicalName(dn)
	if strings.HasSuffix(n, "test.") {
		return n
	} else {
		return n + "test."
	}
}
