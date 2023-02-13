package testacme

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/letsencrypt/pebble/v2/ca"
	"github.com/letsencrypt/pebble/v2/db"
	"github.com/letsencrypt/pebble/v2/va"
	"github.com/letsencrypt/pebble/v2/wfe"
)

type testingT interface {
	Cleanup(func())
}

type PebbleServerConfig struct {
	PerPage int `json:"per-page"`

	HTTPVerificationPort int `json:"http-verification-port"`
	TLSVerificationPort int `json:"tls-verification-port"`

	PermitInsecureGET bool `json:"permit-insecure-get"`
	RequireExternalAccountBinding bool `json:"require-external-account-binding"`

	CertificateValidityPeriod time.Duration `json:"certificate-validity-period"`
	CertificateAlternateChains int `json:"certificate-alternate-chains"`
	CertificateChainLength int `json:"certificate-chain-length"`
}

const (
	DefaultPerPage = 3
	DefaultHTTPVerificationPort = 5080
	DefaultTLSVerificationPort = 5443
	DefautCertificateValidityPeriod = 5 * 365 * 24 * time.Hour + 24 * time.Hour // poor approximation of 5 years.
	DefaultCertificateAlternateChains = 3
	MaxCertificateAlternateChains = 15 // arbitrary, want it higher? Please make a PR with some comments regarding your use case :)
	DefaultCertificateChainLength = 2 // root + intermediary
)

type pebbleConfig struct {
	Context context.Context
	net.ListenConfig

	PebbleServerConfig PebbleServerConfig
	PebbleLogger *log.Logger
	PebbleCA *ca.CAImpl
	PebbleVA *va.VAImpl
	PebbleDB *db.MemoryStore
	PebbleWFE *wfe.WebFrontEndImpl
}

func newPebbleConfig(t testingT) (*pebbleConfig, func ()) {
	var defaultPebbleContext context.Context
	if provider, ok := t.(interface{
		Context() context.Context
	}); ok {
		// t had a context, use the value it provides.
		defaultPebbleContext = provider.Context()
		if defaultPebbleContext == nil {
			panic("provider yielded a nil context; context required to start testacme")
		}
	} else {
		defaultPebbleContext = context.Background()
	}

	var target = &pebbleConfig{}

	finalize := func() {
		if target.Context == nil {
			target.Context = defaultPebbleContext
		}

		if target.PebbleLogger == nil {
			// If you want logs, then you're going to have to configure a
			// logger.
			var logger log.Logger
			logger.SetOutput(ioutil.Discard)
			target.PebbleLogger = &logger
		}

		if target.PebbleServerConfig.HTTPVerificationPort == 0 {
			target.PebbleServerConfig.HTTPVerificationPort = DefaultHTTPVerificationPort
		}
		if target.PebbleServerConfig.TLSVerificationPort == 0 {
			target.PebbleServerConfig.TLSVerificationPort = DefaultTLSVerificationPort
		}
		if target.PebbleServerConfig.PerPage == 0 {
			target.PebbleServerConfig.PerPage = DefaultPerPage
		}

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
				"") 
		}

		if target.PebbleWFE == nil {
			frontend := wfe.New(target.PebbleLogger, target.PebbleDB, target.PebbleVA, target.PebbleCA,
				!target.PebbleServerConfig.PermitInsecureGET, // strict
				target.PebbleServerConfig.RequireExternalAccountBinding) // strict EAB
			target.PebbleWFE = &frontend
		}
	}

	return target, finalize
}

func WithContext(ctx context.Context) PebbleOption {
	return func(pc *pebbleConfig) error {
		pc.Context = ctx
		return nil
	}
}

func WithPebbleServerConfig(c PebbleServerConfig) PebbleOption {
	return func(pc *pebbleConfig) error {
		pc.PebbleServerConfig = c
		return nil
	}
}

type PebbleOption = func(*pebbleConfig) error

type Pebble struct {
	shutdown func()

	listener net.Listener
	managementListener net.Listener
}

func NewPebble(t testingT, options ...PebbleOption) Pebble {
	config, finalize := newPebbleConfig(t)

	for _, option := range options {
		err := option(config)
		if err != nil {
			panic(fmt.Sprintf("invalid options: %s", err))
		}
	}
	finalize()

	testacmeCtx, cancel := context.WithCancel(config.Context)

	listener, err := config.Listen(testacmeCtx, "unix", "")
	if err != nil {
		// if you're here because tests are panicking, can you create an issue
		// and share the details of the panic & desired outcome? Thanks! And,
		// Sorry!
		panic(fmt.Sprintf("unable to listen on unix socket: %s", err))
	}

	return Pebble{
		shutdown: cancel,
		listener: listener,
	}
}

func runPebble(ctx context.Context, listener net.Listener, managementListener net.Listener) error {

}

func (p Pebble) Addr() net.Addr {
	return p.listener.Addr()
}

func (p Pebble) ManagementAddr() net.Addr {
	return p.managementListener.Addr()
}

func (p *Pebble) Shutdown() {
	if p.shutdown != nil {
		p.shutdown()
		p.shutdown = nil
	} else {
		// if you're here because tests are panicking, can you create an issue
		// and share the details of the panic & thoughts on why this is
		// unexpected? Thanks! And, Sorry!
		panic(`Shutdown() called more than once (maybe some calls that "ensure" the testacme shuts down?)`)
	}
}
