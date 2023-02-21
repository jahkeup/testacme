// SPDX-License-Identifier: LGPL-3.0-or-later

package randomports

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandom(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		success := []int{
			1, 2, 16, 64,
		}

		for _, n := range success {
			t.Run(fmt.Sprintf("random_%d", n), func(t *testing.T) {
				ports, err := RandomPorts(n)
				assert.NoError(t, err)
				if assert.NotEmpty(t, ports) {
					for _, p := range ports {
						assert.False(t, Reserve(p), "should not be available for reservation")
					}
				}
			})
		}
	})
}

func TestLoopbackCheck(t *testing.T) {
	t.Log(loopbackAddresses())
	assert.NotEqual(t, ":0", bindAddress)
}
