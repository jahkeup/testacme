// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/miekg/dns"
)

// DNS is a nameserver offering only limited capabilities. The server is
// intended for use in testacme where most (or all) queries are expected to
// resolve to the local host. The backing NameserverDB is the "authority" and can
type DNS struct {
	server *dns.Server
}

// NewDNS creates an ephemeral nameserver to drive testacme verifications.
// Queries will default to 127.0.0.1 unless otherwise configured in the
// supporting NameserverDB.
func NewDNS(ctx context.Context, dnsdb *NameserverDB) (*DNS, error) {
	dnsdb.defaultA = dns.Msg{
		Answer: []dns.RR{
			&dns.A{
				Hdr: dns.RR_Header{
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    1800,
				},
				// TODO: resolve localhost and use values found at runtime
				A: net.ParseIP("127.0.0.1"),
			},
		},
	}

	lc := net.ListenConfig{}
	lpc, err := lc.ListenPacket(ctx, "udp", ":0")
	if err != nil {
		return nil, fmt.Errorf("new listener: %w", err)
	}

	server := &dns.Server{
		PacketConn: lpc,
		Handler:    dnsdb,
	}
	go server.ActivateAndServe()

	return &DNS{
		server: server,
	}, nil
}

// Addr returns the net.Addr where the nameserver is listening.
func (d DNS) Addr() net.Addr {
	return d.server.PacketConn.LocalAddr()
}

// NameserverDB holds a basic datastore of query questions mapped to query
// responses. This can be used directly as a DNS handler - and is by DNS above.
type NameserverDB struct {
	m sync.Map

	defaultA dns.Msg
}

func dbMsgKey(m *dns.Msg) string {
	if len(m.Question) != 1 {
		panic("cannot store multi-question messages")
	}
	q := m.Question[0]
	return q.String()
}

// DefaultA builds an adjusted response with respect to the query.
func (db *NameserverDB) DefaultA(q dns.Question) *dns.Msg {
	r := db.defaultA.Copy()

	for i := range r.Answer {
		r.Answer[i].Header().Name = q.Name
	}

	return r
}

// AddMsg stores the given DNS message for lookup when resolving names.
func (db *NameserverDB) AddMsg(r dns.Msg) {
	db.m.Store(dbMsgKey(&r), r.Copy())
}

// DeleteMsg immediately removes the given DNS message (by its question) and
// will no longer be returned in DNS query responses.
func (db *NameserverDB) DeleteMsg(r dns.Msg) {
	db.m.Delete(dbMsgKey(&r))
}

// GetMsg looks up a DNS query response based on its question.
func (db *NameserverDB) GetMsg(r *dns.Msg) *dns.Msg {
	val, ok := db.m.Load(dbMsgKey(r))
	if !ok {
		return nil
	}

	mp, ok := val.(*dns.Msg)
	if ok {
		return mp.Copy()
	} else {
		panic("nameserver db returned non-msg item")
	}
}

// ServeDNS provides the DNS replies for local ACME validation.
func (db *NameserverDB) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	dbresponse := db.GetMsg(r)
	if dbresponse != nil {
		w.WriteMsg(dbresponse)
		return
	}

	var response *dns.Msg
	for _, q := range r.Question {
		if q.Qtype == dns.TypeA {
			a := db.DefaultA(q)
			response = r.Copy()
			response.Answer = a.Answer
		}

		// TODO: support when resolving localhost addresses at runtime
		if q.Qtype == dns.TypeAAAA {
			response = r.Copy()
			response.Rcode = dns.RcodeNameError
		}

		if response != nil {
			break
		}
	}

	if response != nil {
		w.WriteMsg(response)
	} else {
		dns.DefaultServeMux.ServeDNS(w, r)
		return
	}
}

var _ dns.Handler = (*NameserverDB)(nil)
