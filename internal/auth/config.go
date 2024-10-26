package auth

import (
	"bytes"

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
		Domains string                  `yaml:",omitempty"`
		Reload  map[string]*YamlService `yaml:",omitempty"`
	}
	YamlService struct {
		Command []string `yaml:"cmd"`
	}
)

func (m *AuthService) unmarshalYamlConfig(payload []byte) (_ *YamlConfig, e error) {
	config := &YamlRoot{}

	decoder := yaml.NewDecoder(bytes.NewBuffer(payload))
	if e = decoder.Decode(config); e != nil {
		return
	}

	return config.Config, e
}
