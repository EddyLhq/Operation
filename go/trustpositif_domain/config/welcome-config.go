package config

import (
	v1config "trustpositif_domain/config/v1"
	v2config "trustpositif_domain/config/v2"
)

type WelcomeConfig interface {
	Encode() []byte
	EncodeFormat() string
	EncodeFormatYaml() string
	Decode([]byte) error
	YamlEncode() string
	YamlDecode([]byte) error
	EncryptEncode() string
	DecryptDecode(string) error
	Domains() ([]string, error)
}

func NewWelcomeConfig(isNew bool) WelcomeConfig {
	if isNew {
		return &v2config.Config{}
	}

	return &v1config.Config{}
}
