// SPDX-License-Identifier: LGPL-3.0-or-later

package testacme

import (
	"context"
	"crypto"
	"fmt"
	"strings"
	"testing"

	acmeapi "github.com/go-acme/lego/v4/acme/api"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

var GeneratedEmailDomain = "testacme." + TLDRFC6761

// NewTestingContext creates a context that's canceled at the end of the current
// test scope.
func NewTestingContext(t testing.TB) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return ctx
}

var tokenReplacer = strings.NewReplacer("/", "_")

func TestNamedEmail(t testing.TB) string {
	name := tokenReplacer.Replace(t.Name())
	return fmt.Sprintf("%s@%s", strings.ToLower(name), GeneratedEmailDomain)
}

func ManagedUser(email string) *managedUser {
	pk, err := certcrypto.GeneratePrivateKey(certcrypto.RSA2048)
	if err != nil {
		panic(err)
	}

	return &managedUser{
		email:      email,
		privateKey: pk,
	}
}

func LegoClient(testacme TestACME, user registration.User) *lego.Client {
	config := lego.NewConfig(user)

	config.CADirURL = testacme.ACMEDirectoryURL()
	config.HTTPClient = testacme.Client()

	client, err := lego.NewClient(config)
	if err != nil {
		panic(fmt.Sprintf("failed to get lego client: %v", err))
	}

	return client
}

func LegoAPIClient(testacme TestACME, user registration.User) *acmeapi.Core {
	// TODO: support testing with EAB (requires KID)
	accountKID := user.GetRegistration().URI

	apiclient, err := acmeapi.New(testacme.Client(), "testacme/LegoAPIClient", testacme.ACMEDirectoryURL(), accountKID, user.GetPrivateKey())
	if err != nil {
		panic(fmt.Sprintf("failed to get lego acme (api) client: %v", err))
	}

	return apiclient
}

type managedUser struct {
	email        string
	privateKey   crypto.PrivateKey
	registration *registration.Resource
}

// GetEmail implements registration.User
func (u *managedUser) GetEmail() string {
	return u.email
}

// GetPrivateKey implements registration.User
func (u *managedUser) GetPrivateKey() crypto.PrivateKey {
	return u.privateKey
}

// GetRegistration implements registration.User
func (u *managedUser) GetRegistration() *registration.Resource {
	return u.registration
}

func (u *managedUser) SetRegistration(reg *registration.Resource) {
	u.registration = reg
}

func (u *managedUser) Register(testacme TestACME) error {
	config := lego.NewConfig(u)

	config.CADirURL = testacme.ACMEDirectoryURL()
	config.HTTPClient = testacme.Client()

	client, err := lego.NewClient(config)
	if err != nil {
		return fmt.Errorf("lego client: %w", err)
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{
		TermsOfServiceAgreed: true,
	})
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}

	u.SetRegistration(reg)

	return nil
}

func (u *managedUser) MustRegister(testacme TestACME) *managedUser {
	if err := u.Register(testacme); err != nil {
		panic(err)
	}

	return u
}

var _ registration.User = (*managedUser)(nil)
