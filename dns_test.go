// SPDX-License-Identifier: MIT OR LGPL-3.0-or-later

package testacme

import (
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNS(t *testing.T) {
	ctx := NewTestingContext(t)
	db := new(NameserverDB)

	srv, err := NewDNS(ctx, db)
	require.NoError(t, err)
	t.Logf("starting dns server: %#v (%[1]q)", srv.Addr())

	// Test DNS client
	var resolver dns.Client
	resolver.DialTimeout = 1 * time.Second

	conn, err := resolver.DialContext(ctx, srv.Addr().String())
	require.NoError(t, err, "dns resolver client required")
	err = conn.WriteMsg(&dns.Msg{
		Question: []dns.Question{
			{
				Name:   "can.literally.be.anything.test.",
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			},
		},
	})

	require.NoError(t, err, "failed to send query")
	reply, err := conn.ReadMsg()
	require.NoError(t, err, "failed to read reply")
	t.Logf("%#v", reply)

	// ensure there's actually responses, we'll iterate on the values to check
	// content
	assert.NotEmpty(t, reply.Answer, "should have answers in response")
	for i, ans := range reply.Answer {
		t.Logf("answer[%d]: %#v", i, ans.String())
		switch r := ans.(type) {
		case *dns.A:
			assert.Equal(t, "127.0.0.1", r.A.String(), "should have replied with default A (localhost's address)")
		default:
			assert.Fail(t, "unexpected type in answer: %q", ans.Header())
		}
	}
}

func TestSharedDNS(t *testing.T) {
	srv := SharedDNS()
	t.Logf("starting dns server: %#v (%[1]q)", srv.Addr())

	// Test DNS client
	var resolver dns.Client
	resolver.DialTimeout = 1 * time.Second

	conn, err := resolver.DialContext(NewTestingContext(t), srv.Addr().String())
	require.NoError(t, err, "dns resolver client required")
	err = conn.WriteMsg(&dns.Msg{
		Question: []dns.Question{
			{
				Name:   "can.literally.be.anything.test.",
				Qtype:  dns.TypeA,
				Qclass: dns.ClassINET,
			},
		},
	})

	require.NoError(t, err, "failed to send query")
	reply, err := conn.ReadMsg()
	require.NoError(t, err, "failed to read reply")
	t.Logf("%#v", reply)

	// ensure there's actually responses, we'll iterate on the values to check
	// content
	assert.NotEmpty(t, reply.Answer, "should have answers in response")
	for i, ans := range reply.Answer {
		t.Logf("answer[%d]: %#v", i, ans.String())
		switch r := ans.(type) {
		case *dns.A:
			assert.Equal(t, "127.0.0.1", r.A.String(), "should have replied with default A (localhost's address)")
		default:
			assert.Fail(t, "unexpected type in answer: %q", ans.Header())
		}
	}
}

func TestSharedDNS_NameserverDB(t *testing.T) {
	srv := SharedDNS()
	t.Logf("starting dns server: %#v (%[1]q)", srv.Addr())

	db := SharedNameserverDB()
	db.StoreExact(dns.Msg{
		Question: []dns.Question{
			{
				Name:  "this.exact.name.specifically.test.",
				Qtype: dns.TypeTXT,
			},
		},
		Answer: []dns.RR{
			&dns.TXT{
				Hdr: dns.RR_Header{
					Name:   "this.exact.name.specifically.text.",
					Rrtype: dns.TypeTXT,
					Class:  dns.ClassINET,
					Ttl:    1955,
				},
				Txt: []string{"foo=bar", "baz=qux"},
			},
		},
	})

	// Test DNS client
	var resolver dns.Client
	resolver.DialTimeout = 1 * time.Second

	conn, err := resolver.DialContext(NewTestingContext(t), srv.Addr().String())
	require.NoError(t, err, "dns resolver client required")
	err = conn.WriteMsg(&dns.Msg{
		Question: []dns.Question{
			{
				Name:  "this.exact.name.specifically.test.",
				Qtype: dns.TypeTXT,
			},
		},
	})

	require.NoError(t, err, "failed to send query")
	reply, err := conn.ReadMsg()
	require.NoError(t, err, "failed to read reply")
	t.Logf("%#v", reply)

	// ensure there's actually responses, we'll iterate on the values to check
	// content
	assert.NotEmpty(t, reply.Answer, "should have answers in response")
	for i, ans := range reply.Answer {
		t.Logf("answer[%d]: %#v", i, ans.String())
		switch r := ans.(type) {
		case *dns.TXT:
			const (
				rr = "this.exact.name.specifically.text.\t1955\tIN\tTXT\t\"foo=bar\" \"baz=qux\""
			)
			assert.Equal(t, rr, r.String(), "should have replied with default A (localhost's address)")
		default:
			assert.Fail(t, "unexpected type in answer: %q", ans.Header())
		}
	}
}
