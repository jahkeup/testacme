// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/miekg/dns"
)

const (
	// TLDRFC6761 is the TLDRFC6761 designated TLD name reserved specifically
	// for testing usages.
	//
	// https://www.rfc-editor.org/rfc/rfc6761#section-6.2
	TLDRFC6761 = "test"
	// TestTLD is a top level domain name suitable for testing contexts. This
	// symbol exists entirely for convenience.
	TestTLD = TLDRFC6761
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
	// TODO: resolve loopback address?
	dnsdb.defaultA = net.ParseIP("127.0.0.1")
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
	msgDB sync.Map

	defaultA net.IP
}

// dbMsgKey yields the string key used to lookup exact reply dns.Msg.
func dbMsgKey(m *dns.Msg) string {
	// one question and one question is defacto*
	//
	// https://github.com/miekg/dns/blob/d8fbd0a7551ae5476df0eb003b247322761c6e82/types.go#L223-L227
	if len(m.Question) != 1 {
		panic("one question and one question only")
	}
	return m.Question[0].String()
}

// DefaultA is the message prototype for a default response.
func (db *NameserverDB) DefaultA(r *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetReply(r)

	m.Answer = []dns.RR{
		&dns.A{
			Hdr: dns.RR_Header{
				Name:   dns.CanonicalName(r.Question[0].Name),
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			A: db.defaultA,
		},
	}

	return m
}

// StoreExact stores the given DNS message for lookup when resolving names.
func (db *NameserverDB) StoreExact(r dns.Msg) {
	db.msgDB.Store(dbMsgKey(&r), &r)
}

// RemoveReplyTo immediately removes the given DNS message (by its question) and
// will no longer be returned in DNS query responses.
func (db *NameserverDB) RemoveReplyTo(r dns.Msg) {
	db.msgDB.Delete(dbMsgKey(&r))
}

// LookupReply retrieves a DNS query response.
func (db *NameserverDB) LookupReply(r *dns.Msg) *dns.Msg {
	val, ok := db.msgDB.Load(dbMsgKey(r))
	if !ok {
		return nil
	}

	m, ok := val.(*dns.Msg)
	if ok {
		return m.Copy()
	} else {
		panic("nameserver db returned non-msg item")
	}
}

// ServeDNS provides the DNS replies for local ACME validation.
func (db *NameserverDB) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	if resp := db.LookupReply(r); resp != nil {
		resp.SetReply(r)
		w.WriteMsg(resp)
		w.Close()
		return
	}

	if len(r.Question) != 1 {
		// one and only one question
		return
	}

	switch r.Question[0].Qtype {
	case dns.TypeA:
		w.WriteMsg(db.DefaultA(r))
		w.Close()
	default:
		dns.DefaultServeMux.ServeDNS(w, r)
	}
}

var _ dns.Handler = (*NameserverDB)(nil)

// MustRR is a helper to create RR values from opaque strings. Panics on invalid
// input. This is intended for use in tests where stable known values are used
// for construction.
func MustRR(rr string) dns.RR {
	ret, err := dns.NewRR(rr)
	if err != nil {
		err = fmt.Errorf("must be valid RR string: %q\nErr: %s", rr, err)
		panic(err)
	}
	return ret
}

// DNSRRMsg expands the given dns.RR record into a dns.Msg with its implied
// Question and embeds the RR in the Answers.
func DNSRRMsg(rr dns.RR) *dns.Msg {
	dn := dns.CanonicalName(rr.Header().Name)

	m := new(dns.Msg)
	m.SetQuestion(dn, rr.Header().Rrtype)
	m.Opcode = dns.OpcodeQuery
	m.Answer = []dns.RR{rr}

	return m
}
