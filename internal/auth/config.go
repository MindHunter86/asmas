package auth

import (
	"bytes"
	"regexp"
	"strings"

	futils "github.com/gofiber/fiber/v2/utils"
	"gopkg.in/yaml.v3"
)

type (
	YamlRoot struct {
		Config *YamlConfig
	}
	YamlConfig struct {
		AuthorizationList []*YamlAuthorization `yaml:"authorization_list"`
	}
	YamlAuthorization struct {
		Name    string
		Domains string `yaml:",omitempty"`

		// !!!!
		// !!!!
		// !!!!
		AltNames int `yaml:"-"`

		Allow  string                  `yaml:",omitempty"`
		Reload map[string]*YamlService `yaml:",omitempty"`

		allowregexp *regexp.Regexp
	}
	YamlService struct {
		Command []string `yaml:"cmd"`
	}
)

func (m *YamlConfig) authorizationByFqdn(fqdn string) *YamlAuthorization {
	for _, authorization := range m.AuthorizationList {
		if authorization.Name == fqdn {
			return authorization
		}
	}

	return nil
}

func (m *YamlAuthorization) isAuthorizationAllowed(hostname string) bool {
	if m.allowregexp == nil {
		allows := strings.Split(m.Allow, ",")

		for _, allow := range allows {
			if allow == hostname {
				return true
			}
		}

		return false
	}

	return m.allowregexp.Match(futils.UnsafeBytes(hostname))
}

//
//
//

func (*AuthService) unmarshalYamlConfig(payload []byte) (_ *YamlConfig, e error) {
	config := &YamlRoot{}

	decoder := yaml.NewDecoder(bytes.NewBuffer(payload))
	if e = decoder.Decode(config); e != nil {
		return
	}

	return config.Config, e
}
