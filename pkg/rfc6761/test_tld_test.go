// SPDX-License-Identifier: LGPL-3.0-or-later

package rfc6761

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalTest(t *testing.T) {
	testcases := map[string]string{
		"foo":       "foo.test.",
		"foo.":      "foo.test.",
		"foo.bar":   "foo.bar.test.",
		".test.":    ".test.",
		"":          ".test.",
		"foo.test.": "foo.test.",
	}

	for input, expected := range testcases {
		t.Run(input, func(t *testing.T) {
			assert.Equal(t, expected, CanonicalTest(input))
		})
	}
}
