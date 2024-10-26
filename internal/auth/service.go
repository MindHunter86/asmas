package auth

import (
	"context"
	"crypto"
	"sync"
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
	pullerrdelay time.Duration

	signers   openpgp.EntityList
	pgpconfig *packet.Config

	mu       sync.RWMutex
	authlist *YamlConfig

	log   *zerolog.Logger
	done  func() <-chan struct{}
	abort context.CancelFunc
}

func NewAuthService(c context.Context, cc *cli.Context) *AuthService {
	return &AuthService{
		token: cc.String("auth-sign-token"),

		client:       NewHttpClient(cc, c.Value(utils.CKeyLogger).(*zerolog.Logger)),
		pullinterval: cc.Duration("auth-github-pull-interval"),
		pullerrdelay: cc.Duration("auth-github-pull-error-delay"),

		pgpconfig: &packet.Config{
			DefaultHash: crypto.SHA512,
		},

		log:   c.Value(utils.CKeyLogger).(*zerolog.Logger),
		done:  c.Done,
		abort: c.Value(utils.CKeyAbortFunc).(context.CancelFunc),
	}
}

func (m *AuthService) Boostrap() {
	var e error
	if m.signers, e = m.loadConfigSigners(); e != nil {
		m.log.Error().Msg("an error occurred while loading signers - " + e.Error())
		m.abort()
		return
	}

	if m.authlist, e = m.loadAuthorizationList(); e != nil {
		m.log.Error().Msg("an error occurred while loading authlist - " + e.Error())
		m.abort()
		return
	}

	// !! LOAD DOMAINS
	// for _, authorization := range m.authlist.AuthorizationList {
	// 	fmt.Println(string(authorization.Domains))
	// }

	m.loop()
}

func (*AuthService) AuthorizeHostname(hostname []byte) bool { return false }

func (*AuthService) CertificateByDomain(domain string) ([]byte, error) {
	return nil, nil
}

//
//
//

func (m *AuthService) loop() {
	m.log.Debug().Msg("initiate auth service update loop")
	defer m.log.Debug().Msg("auth service update loop has been closed")

	update := time.NewTimer(m.pullinterval)

LOOP:
	for {
		select {
		case <-m.done():
			m.log.Info().Msg("internal abort() has been caught; initiate application closing...")
			break LOOP
		case <-update.C:
			update.Stop()

			var e error
			started := time.Now()

			if e = m.updateAuthorizationList(); e != nil {
				m.log.Error().Msg("an error occurred in auth update loop, %s" + e.Error())
				update.Reset(m.pullerrdelay)
				continue
			}

			m.log.Info().Msgf("authorization list has been updated for %s", time.Since(started).String())
			update.Reset(m.pullinterval)
		}
	}
}

func (m *AuthService) updateAuthorizationList() (e error) {
	var newlist *YamlConfig
	if newlist, e = m.loadAuthorizationList(); e != nil {
		return
	}

	actionWithLock(&m.mu, func() {
		m.authlist = newlist
	})
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
