// SPDX-License-Identifier: LGPL-3.0-or-later

package randomports

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheGate(t *testing.T) {
	cg, err := newCacheGate(5)
	require.NoError(t, err)
	require.NotNil(t, cg)

	cg.InUse(1)
	cg.InUse(2)
	cg.InUse(3)
	cg.InUse(4)
	cg.InUse(5)

	for i := Port(1); i <= 5; i++ {
		assert.True(t, cg.InUse(i))
	}

	// Then tracks the latest, 6
	assert.False(t, cg.InUse(6))
	assert.True(t, cg.InUse(6))

	// but lost 1 because cache size is 5
	assert.False(t, cg.InUse(1))
}
