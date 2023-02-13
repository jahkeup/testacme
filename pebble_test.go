package testacme

import (
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPebble_Ping(t *testing.T) {
	pebble := NewPebble(t, WithPebbleLogger(log.Default()))

	t.Logf("acme addr: %#v", pebble.Addr())
	t.Logf("management addr: %#v", pebble.ManagementAddr())

	client := http.Client{
		Transport: pebble.ClientTransport(),
		Timeout: 10 * time.Second,
	}

	time.Sleep(5 * time.Second)

	resp, err := client.Get("http://testacme/nonce-plz")
	assert.NoError(t, err)
	t.Logf("response: %#v", resp)
	assert.NotNil(t, resp, "should have dir response")
}
