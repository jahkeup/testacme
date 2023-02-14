// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"context"
	"fmt"
	"net"
	"os"
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
	key := fmt.Sprintf("%s-%s", q.Name, dns.TypeToString[q.Qtype])
	fmt.Fprintln(os.Stderr, key)
	return key
}

// DefaultA is the message prototype for a default response.
func (db *NameserverDB) DefaultA() *dns.Msg {
	return &db.defaultA
}

// AddMsg stores the given DNS message for lookup when resolving names.
func (db *NameserverDB) AddMsg(r dns.Msg) {
	db.m.Store(dbMsgKey(&r), r)
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

	mp, ok := val.(dns.Msg)
	if ok {
		return &mp
	} else {
		panic("nameserver db returned non-msg item")
	}
}

// ServeDNS provides the DNS replies for local ACME validation.
func (db *NameserverDB) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	match := db.GetMsg(r)
	if match != nil {
		dbresponse := new(dns.Msg)
		dbresponse.SetReply(r)
		dbresponse.Answer = match.Answer

		fmt.Fprintf(os.Stderr, "%#v\n", dbresponse)

		w.WriteMsg(dbresponse)
		return
	}

	var response *dns.Msg
	for _, q := range r.Question {
		if q.Qtype == dns.TypeA {
			response = new(dns.Msg)
			response.Answer = db.DefaultA().Answer
			for i := range response.Answer {
				response.Answer[i].Header().Name = q.Name
			}
		}

		// TODO: support when resolving localhost addresses at runtime
		if q.Qtype == dns.TypeAAAA {
			response = new(dns.Msg)
			response.SetRcode(r, dns.RcodeNameError)
		}

		if response != nil {
			break
		}
	}

	if response != nil {
		w.WriteMsg(response)
	} else {
		dns.DefaultServeMux.ServeDNS(w, r)
	}
}

var _ dns.Handler = (*NameserverDB)(nil)
