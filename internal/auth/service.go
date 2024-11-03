package auth

import (
	"context"
	"crypto"
	"errors"
	"regexp"
	"sync"
	"time"

	"github.com/MindHunter86/asmas/internal/gclient"
	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

type AuthService struct {
	token string

	client       *gclient.HttpClient
	pullinterval time.Duration
	pullerrdelay time.Duration

	debugskipgithub bool

	signers   openpgp.EntityList
	pgpconfig *packet.Config

	mu       sync.RWMutex
	authlist *YamlConfig

	log   *zerolog.Logger
	done  func() <-chan struct{}
	abort context.CancelFunc
}


func NewAuthClient(c context.Context, cc *cli.Context) *AuthService {
	return &AuthService{
		token: cc.String("auth-sign-token"),
		log:   c.Value(utils.CKeyLogger).(*zerolog.Logger),
	}
}

func NewAuthService(c context.Context, cc *cli.Context) *AuthService {
	return &AuthService{
		token: cc.String("auth-sign-token"),

		client:       gclient.NewHttpClient(cc, c.Value(utils.CKeyLogger).(*zerolog.Logger)),
		pullinterval: cc.Duration("auth-github-pull-interval"),
		pullerrdelay: cc.Duration("auth-github-pull-error-delay"),

		pgpconfig: &packet.Config{
			DefaultHash: crypto.SHA512,
		},

		debugskipgithub: cc.Bool("debug-skip-github-connect"),

		log:   c.Value(utils.CKeyLogger).(*zerolog.Logger),
		done:  c.Done,
		abort: c.Value(utils.CKeyAbortFunc).(context.CancelFunc),
	}
}

func (m *AuthService) Boostrap() {
	if m.client == nil {
		return
	}

	var e error
	if m.signers, e = m.loadConfigSigners(); e != nil {
		m.log.Error().Msg("an error occurred while loading signers - " + e.Error())
		m.abort()
		return
	}

	for _, entity := range m.signers {
		for _, identity := range entity.Identities {
			m.log.Info().Msgf("loaded trusted signer named as %s", identity.Name)
		}
	}

	if m.authlist, e = m.loadAuthorizationList(); e != nil {
		m.log.Error().Msg("an error occurred while loading authlist - " + e.Error())
		m.abort()
		return
	}

	m.loop()
}

func (m *AuthService) AuthorizeHostname(name, hostname string) (ok bool, _ error) {
	if !m.isApiReady() {
		return false, errors.New("auth service api is not ready yet")
	}

	ok = actionReturbableWithRLock[bool](&m.mu, func() bool {
		var auth *YamlAuthorization
		if auth = m.authlist.authorizationByFqdn(name); auth == nil {
			return false
		}

		return auth.isAuthorizationAllowed(hostname)
	})

	return
}

func (m *AuthService) GetAvailableDomains(hostname string) (_ []string, e error) {
	if !m.isApiReady() {
		return nil, errors.New("auth service api is not ready yet")
	}

	return actionReturbableWithRLock[[]string](&m.mu, func() []string {
		var authorizations []string

		for _, authorization := range m.authlist.AuthorizationList {
			if authorization.isAuthorizationAllowed(hostname) {
				authorizations = append(authorizations, authorization.Name)
			}
		}

		return authorizations
	}), e
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
				m.log.Error().Msg("an error occurred in auth update loop, " + e.Error())
				update.Reset(m.pullerrdelay)
				continue
			}

			m.log.Info().Msg("authorization list has been updated for " + time.Since(started).String())
			update.Reset(m.pullinterval)
		}
	}
}

func (m *AuthService) isApiReady() (ok bool) {
	ok = actionReturbableWithRLock[bool](&m.mu, func() bool {
		return m.authlist != nil
	})

	return
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
	if m.debugskipgithub {
		return
	}

	var response *gclient.GithubResponse
	if response, e = m.client.FetchConfigFromGithub(); e != nil {
		return
	}

	if e = m.client.ValidateGithubResponse(response); e != nil {
		return
	}

	var validated []byte
	if validated, e = m.validateConfigSign(response.Content); e != nil {
		return
	}

	var authlist *YamlConfig
	if authlist, e = m.unmarshalYamlConfig(validated); e != nil {
		return
	}

	if ok := actionReturbableWithRLock[bool](&m.mu, func() bool {
		return m.validateAuthorizationList(authlist)
	}); !ok {
		return nil, errors.New("could not validate received config with authorized domains, check logs")
	}

	return authlist, e
}

func (m *AuthService) validateAuthorizationList(authlist *YamlConfig) (ok bool) {
	var entityname string

	defer func() {
		if r := recover(); r != nil {
			m.log.Error().Msgf("panic has been cauth on processing %s regexp, regexp is invalid", entityname)
			m.log.Debug().Msg("panic has been caught, seems regexp compilation was failed")
			m.log.Trace().Msgf("%+v", r)
		}
	}()

	for _, entity := range authlist.AuthorizationList {
		// save entityname for panic errors
		entityname = entity.Name

		if entity.Domains == "" {
			entity.Domains = entity.Name
			continue
		}

		// check for regexp template
		allowlen := len(entity.Allow)
		if entity.Allow[:1] == "/" && entity.Allow[allowlen-1:allowlen] == "/" {
			entity.allowregexp = regexp.MustCompile(entity.Allow[1 : allowlen-1])
		}

		m.log.Info().Msgf("loaded authorized domain with id %s", entity.Name)
	}

	ok = true
	return
}
