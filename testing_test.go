// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestNamedEmail(t *testing.T) {
	const expected = "testtestnamedemail@testacme.test"
	actual := TestNamedEmail(t)
	assert.Equal(t, expected, actual)

	t.Run("nested", func(t *testing.T) {
		const expected = "testtestnamedemail_nested@testacme.test"
		actual := TestNamedEmail(t)
		assert.Equal(t, expected, actual)

		t.Run("nested", func(t *testing.T) {
			const expected = "testtestnamedemail_nested_nested@testacme.test"
			actual := TestNamedEmail(t)
			assert.Equal(t, expected, actual)
		})
	})
}
