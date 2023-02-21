// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"

	"github.com/letsencrypt/pebble/v2/ca"
	"github.com/letsencrypt/pebble/v2/db"
	"github.com/letsencrypt/pebble/v2/va"
	"github.com/letsencrypt/pebble/v2/wfe"

	"github.com/jahkeup/testacme/pkg/randomports"
)

func init() {
	// default to *not* sleeping on the VA challenges to keep tests speeding
	// along :)
	//
	// https://github.com/letsencrypt/pebble/blob/d5fa73840ef4a2efa7870648ae174627ef001e9c/va/va.go#L49
	const noSleepEnv = "PEBBLE_VA_NOSLEEP"
	if _, isset := os.LookupEnv(noSleepEnv); !isset {
		os.Setenv(noSleepEnv, "true")
	}
}

func init() { unsetPebbleEnvs() }

func unsetPebbleEnvs() {
	// Unfortunately, Pebble uses the environment variables over the values
	// given to `ca.New`. The following should be explicitly removed from the
	// environment to ensure values passed in are actually used.
	os.Unsetenv("PEBBLE_ALTERNATE_ROOTS")
	os.Unsetenv("PEBBLE_CHAIN_LENGTH")
}

// PebbleServerConfig provides configuration used to stand up the Pebble
// testacme instance. Note that callers *can* make an inconsistent setup by
// explicitly building up their CA, VA, and DB values.. so take care when being
// creative.
type PebbleServerConfig struct {
	// ListPageSize is the number used to split pages when pagingating response.
	ListPageSize int `json:"per-page"`
	// HTTPVerificationPort is the port connected to for HTTP based challenge
	// verification.
	HTTPVerificationPort int `json:"http-verification-port"`
	// TLSVerificationPort is the port connected to for TLS based challenge
	// verification.
	TLSVerificationPort int `json:"tls-verification-port"`
	// VerificationDNSResolver is the DNS resolver to use when performing
	// verification.
	VerificationDNSResolver string `json:"verification-dns-resolver"`
	// PermitInsecureGET permits HTTP GET requests to API endpoints. This mimics
	// LetsEncrypt's behavior however the spec is for POST - which is the
	// default when this is left unset or set to false.
	PermitInsecureGET bool `json:"permit-insecure-get"`
	// RequireExternalAccountBinding requires EAB values provided in API
	// requests. This is not fully plumbed - if you find that it works (or that
	// it doesn't) , please create an issue to improve the implementation.
	RequireExternalAccountBinding bool `json:"require-external-account-binding"`
	// CertificateValidityPeriod is the duration for which vended certificates
	// are valid for.
	CertificateValidityPeriod time.Duration `json:"certificate-validity-period"`
	// CertificateAlternateChains is the number of alternate Root CA chains to
	// build and operate the testacme CA (Pebble) with.
	CertificateAlternateChains int `json:"certificate-alternate-chains"`
	// CertificateChainLength is the total number of certificates in the
	// generated Root CA chains. Setting this to `1` generates *only* Root CA
	// certificate(s) while `2` would include a single Intermediate CA.
	CertificateChainLength int `json:"certificate-chain-length"`
}

const (
	// DefaultListPageSize is the default value used for Pebble.
	DefaultListPageSize = 3
	// DefautCertificateValidityPeriod is the default value used when issuing
	// certificates.
	DefautCertificateValidityPeriod = 5*365*24*time.Hour + 24*time.Hour // poor approximation of 5 years.
	// DefaultCertificateAlternateChains is the default number of Root CA chains generated.
	DefaultCertificateAlternateChains = 3
	// DefaultCertificateChainLength is the default number of certificates
	// generated in each Root CA chain.
	DefaultCertificateChainLength = 2 // root + intermediary
)

type pebbleConfig struct {
	// Context is the shared context for Pebble services.
	Context context.Context
	// PebbleServerConfig provides the shared Pebble testacme service
	// configuration.
	PebbleServerConfig PebbleServerConfig
	// PebbleLogger provides the shared logger used by Pebble components.
	PebbleLogger *log.Logger
	// PebbleCA is the PKI engine for the testacme service.
	PebbleCA *ca.CAImpl
	// PebbleVA is the verification engine for the testacme service.
	PebbleVA *va.VAImpl
	// PebbleDB is the datastore used by the testacme service.
	PebbleDB *db.MemoryStore
	// PebbleWFE provides the HTTP API for the testacme service.
	PebbleWFE *wfe.WebFrontEndImpl
}

// newPebbleConfig initializes the pebbleConfig and provides a finalization
// callback that callers MUST run before using the pebbleConfig.
func newPebbleConfig(ctx context.Context) (*pebbleConfig, func()) {
	if ctx == nil {
		panic("nil context provided")
	}

	var target = &pebbleConfig{
		PebbleServerConfig: PebbleServerConfig{
			ListPageSize: DefaultListPageSize,

			CertificateValidityPeriod:  DefautCertificateValidityPeriod,
			CertificateAlternateChains: DefaultCertificateAlternateChains,
			CertificateChainLength:     DefaultCertificateChainLength,
		},
	}

	finalize := func() {
		target.Context = ctx

		// sorry, no low ports. testing, remember?
		if target.PebbleServerConfig.HTTPVerificationPort < 1024 {
			port, err := randomports.One()
			if err != nil {
				panic(fmt.Sprintf("cannot get random port: %v", err))
			}
			target.PebbleServerConfig.HTTPVerificationPort = port.Int()
		}
		// sorry, no low ports. testing, remember?
		if target.PebbleServerConfig.TLSVerificationPort < 1024 {
			port, err := randomports.One()
			if err != nil {
				panic(fmt.Sprintf("cannot get random port: %v", err))
			}
			target.PebbleServerConfig.TLSVerificationPort = port.Int()
		}

		if target.PebbleLogger == nil {
			// If you want logs, then you're going to have to configure a
			// logger.
			var logger log.Logger
			logger.SetOutput(ioutil.Discard)
			target.PebbleLogger = &logger
		}

		if target.PebbleServerConfig.VerificationDNSResolver == "" {
			// NOTE: could also create a new server for each.. meh.
			target.PebbleServerConfig.VerificationDNSResolver = SharedDNS().Addr().String()
		}

		// TODO: audit environment variables used in pebble
		//
		// Might want to panic or overwrite them to prevent unintentional
		// misconfiguration of Pebble servers.

		if target.PebbleDB == nil {
			target.PebbleDB = db.NewMemoryStore()
		}

		if target.PebbleCA == nil {
			target.PebbleCA = ca.New(
				target.PebbleLogger,
				target.PebbleDB,
				"",
				target.PebbleServerConfig.CertificateAlternateChains,
				target.PebbleServerConfig.CertificateChainLength,
				uint(target.PebbleServerConfig.CertificateValidityPeriod.Seconds()))
		}

		if target.PebbleVA == nil {
			target.PebbleVA = va.New(
				target.PebbleLogger,
				target.PebbleServerConfig.HTTPVerificationPort,
				target.PebbleServerConfig.TLSVerificationPort,
				// "strict" doesn't seem used in v2.4.0
				false,
				// if you need custom resolvers, then you probably want to run Pebble as a
				// daemon instead - please create an issue if you think this ought to change!
				target.PebbleServerConfig.VerificationDNSResolver)
		}

		if target.PebbleWFE == nil {
			// NOTE: this re-seeds via `rand.Seed`
			frontend := wfe.New(
				target.PebbleLogger,
				target.PebbleDB,
				target.PebbleVA,
				target.PebbleCA,
				!target.PebbleServerConfig.PermitInsecureGET,            // strict
				target.PebbleServerConfig.RequireExternalAccountBinding) // strict EAB
			target.PebbleWFE = &frontend
		}
	}

	return target, finalize
}

// WithPebbleLogger uses the provided logger in Pebble services to log debug
// and informational messages at runtime.
func WithPebbleLogger(logger *log.Logger) PebbleOption {
	return func(pc *pebbleConfig) error {
		pc.PebbleLogger = logger
		return nil
	}
}

// WithPebbleHTTPVerificationPort uses the provided port number in HTTP
// challenge verifications.
func WithPebbleHTTPVerificationPort(port uint16) PebbleOption {
	return func(pc *pebbleConfig) error {
		pc.PebbleServerConfig.HTTPVerificationPort = int(port)
		return nil
	}
}

// WithPebbleTLSVerificationPort uses the provided port number in TLS challenge
// verifications.
func WithPebbleTLSVerificationPort(port uint16) PebbleOption {
	return func(pc *pebbleConfig) error {
		pc.PebbleServerConfig.TLSVerificationPort = int(port)
		return nil
	}
}

// WithPebbleDNS uses the provided DNS implementation when querying for
// verification and connection addresses.
func WithPebbleDNS(dns *DNS) PebbleOption {
	return func(pc *pebbleConfig) error {
		pc.PebbleServerConfig.VerificationDNSResolver = dns.Addr().String()
		return nil
	}
}

// PebbleOption are functions that tune configuration of the Pebble services.
type PebbleOption = func(*pebbleConfig) error

// Pebble is a testacme implementation suitable for use in concurrent testing (depending on verification)
type Pebble struct {
	pebbleConfig

	shutdownTestACME func()

	pebbleServerStart     *sync.Once
	pebbleServer          *httptest.Server
	managementServerStart *sync.Once
	managementServer      *httptest.Server
}

// Pebble provides its verification port numbers.
var _ Porter = (*Pebble)(nil)

// NewPebble creates an initialized, un-started, Pebble testacme server. The
// services are automatically shutdown with respect to the given context. Also
// see `SharedPebble()`.
func NewPebble(ctx context.Context, options ...PebbleOption) Pebble {
	config, finalize := newPebbleConfig(ctx)

	for _, option := range options {
		err := option(config)
		if err != nil {
			panic(fmt.Sprintf("invalid option: %s", err))
		}
	}
	finalize()

	testacmeCtx, cancel := context.WithCancel(config.Context)
	server := httptest.NewUnstartedServer(config.PebbleWFE.Handler())
	managementServer := httptest.NewUnstartedServer(config.PebbleWFE.ManagementHandler())

	// Shutdown the servers when the context ends.
	go func() {
		<-testacmeCtx.Done()

		config.PebbleLogger.Println("shutting down pebble testacme")

		// NOTE: callers should not try to reuse shutdown servers.. but that's
		// not a problem this library intends to solve.
		server.Close()
		managementServer.Close()
	}()

	pebble := &Pebble{
		pebbleConfig: *config,

		shutdownTestACME: cancel,

		pebbleServer:          server,
		managementServer:      managementServer,
		pebbleServerStart:     new(sync.Once),
		managementServerStart: new(sync.Once),
	}

	return *pebble
}

// Start will startup all Pebble servers on their listeners.
func (p Pebble) Start() {
	p.Server()
	p.ManagementServer()
}

// Shutdown will stop all Pebble servers. This does *not* block on shutdown and
// instead immediately returns control to callers.
func (p Pebble) Shutdown() {
	p.shutdownTestACME()
}

// Server provides the testacme (Pebble) server to be used ibn tests. The server
// is started before returned, if not already started.
func (p Pebble) Server() *httptest.Server {
	p.pebbleServerStart.Do(p.pebbleServer.StartTLS)
	return p.pebbleServer
}

// Client provides a configured HTTP Client suitable for dialing the testacme
// ACME services. See tests for example of connections made using the returned
// http.Client.
func (p Pebble) Client() *http.Client {
	return p.Server().Client()
}

// ACMEDirectoryURL returns the URL to the testacme's ACME directory API
// endpoint. Pebble does not and will not serve its Directory at `/directory` as
// LetsEncrypt does to encourage design & testing of Directory lookup based
// usage.
//
// https://github.com/letsencrypt/pebble/blob/d5fa73840ef4a2efa7870648ae174627ef001e9c/wfe/wfe.go#L39-L42
func (p Pebble) ACMEDirectoryURL() string {
	return p.Server().URL + "/dir"
}

// ManagementServer provides the testacme (Pebble) management server to be used
// in tests. The server is started before returned, if not already started.
func (p Pebble) ManagementServer() *httptest.Server {
	p.managementServerStart.Do(p.managementServer.StartTLS)
	return p.managementServer
}

// HTTPVerificationPort is the port that this testacme server will connect to
// for HTTP verification challenges.
func (p Pebble) HTTPVerificationPort() int {
	return p.PebbleServerConfig.HTTPVerificationPort
}

// TLSVerificationPort is the port that this testacme server will connect to for
// TLS verification challenges.
func (p Pebble) TLSVerificationPort() int {
	return p.PebbleServerConfig.TLSVerificationPort
}
