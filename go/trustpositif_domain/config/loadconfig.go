package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	WelcomeUrl []string `yaml:"WelcomeUrl"`
	// WelcomeJsonUrl string   `yaml:"WelcomeJsonUrl"`
	Web []string `yaml:"Web"`
}

func (p *Config) Decode(data []byte) error {
	err := yaml.Unmarshal(data, p)
	if err != nil {
		return err
	}
	return nil
}

func LoadConfigByFile(filename string) (*Config, error) {
	cfg := &Config{}
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	if err := cfg.Decode(buf); err != nil {
		return nil, err
	}

	return cfg, nil
}
