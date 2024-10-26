package auth

import (
	"context"
	"crypto"
	"fmt"
	"time"

	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

type AuthService struct {
	token string

	client       *HttpClient
	pullinterval time.Duration

	signers   openpgp.EntityList
	pgpconfig *packet.Config

	authlist *YamlConfig

	log *zerolog.Logger
}

func NewAuthService(c context.Context, cc *cli.Context) *AuthService {
	return &AuthService{
		token: cc.String("auth-sign-token"),

		client:       NewHttpClient(cc, c.Value(utils.CKeyLogger).(*zerolog.Logger)),
		pullinterval: cc.Duration("auth-github-pull-interval"),

		pgpconfig: &packet.Config{
			DefaultHash: crypto.SHA512,
		},

		log: c.Value(utils.CKeyLogger).(*zerolog.Logger),
	}
}

func (m *AuthService) Boostrap() (e error) {
	if m.signers, e = m.loadConfigSigners(); e != nil {
		return
	}

	if m.authlist, e = m.loadAuthorizationList(); e != nil {
		return
	}

	for _, authorization := range m.authlist.AuthorizationList {
		fmt.Println(string(authorization.Domains))
	}

	return m.loop()
}

func (*AuthService) AuthorizeHostname(hostname []byte) bool { return false }

func (*AuthService) CertificateByDomain(domain string) ([]byte, error) {
	return nil, nil
}

//
//
//

func (*AuthService) loop() (e error) {
	return
}

func (m *AuthService) loadAuthorizationList() (_ *YamlConfig, e error) {
	var response *GithubResponse
	if response, e = m.client.fetchConfigFromGithub(); e != nil {
		return
	}

	if e = m.client.validateGithubResponse(response); e != nil {
		return
	}

	var validated []byte
	if validated, e = m.validateConfigSign(response.Content); e != nil {
		return
	}

	return m.unmarshalYamlConfig(validated)
}
