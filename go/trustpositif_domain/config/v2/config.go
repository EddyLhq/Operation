package config

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

type Domain struct {
	Nginx  string `json:"nginx" yaml:"nginx"`
	Web    string `json:"web" yaml:"web"`
	Http   string `json:"http" yaml:"http"`
	CDN    string `json:"cdn" yaml:"cdn"`
	Avatar string `json:"avatar" yaml:"avatar"`
	Tcp    string `json:"tcp" yaml:"tcp"`
}

type Backup struct {
	UseIp          bool     `json:"useIp" yaml:"useIp"`
	PriorityDomain bool     `json:"priorityDomain" yaml:"priorityDomain"`
	Domains        []string `json:"domains" yaml:"domains"`
	IpList         []string `json:"ipList" yaml:"ipList"`
}

type Config struct {
	Review Domain            `json:"review" yaml:"review"`
	Domain Domain            `json:"domain" yaml:"domain"`
	Backup map[string]Backup `json:"backup" yaml:"backup"`
}

func (p *Config) Decode(data []byte) error {
	err := json.Unmarshal(data, p)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (p *Config) Encode() []byte {
	buf, err := json.Marshal(p)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return buf
}

func (p *Config) DecryptDecode(data string) error {
	buf, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		fmt.Println(err)
		return err
	}

	cipher, err := aes.NewCipher([]byte(_key))
	if err != nil {
		fmt.Println(err)
		return err
	}

	decrypted := make([]byte, len(buf))
	for bs, be := 0, cipher.BlockSize(); bs < len(buf); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Decrypt(decrypted[bs:be], buf[bs:be])
	}

	trim := 0
	if len(decrypted) > 0 {
		trim = len(decrypted) - int(decrypted[len(decrypted)-1])
	}

	decrypted = decrypted[:trim]

	return p.Decode(decrypted)
}

func (p *Config) EncryptEncode() string {
	cipher, err := aes.NewCipher([]byte(_key))
	if err != nil {
		fmt.Println(err)
		return ""
	}

	buf := p.Encode()
	if buf == nil {
		return ""
	}

	length := (len(buf) + aes.BlockSize) / aes.BlockSize
	plain := make([]byte, length*aes.BlockSize)
	copy(plain, buf)
	pad := byte(len(plain) - len(buf))
	for i := len(buf); i < len(plain); i++ {
		plain[i] = pad
	}
	encrypted := make([]byte, len(plain))
	for bs, be := 0, cipher.BlockSize(); bs <= len(buf); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Encrypt(encrypted[bs:be], plain[bs:be])
	}

	return base64.StdEncoding.EncodeToString(encrypted)
}

func (p *Config) EncodeFormat() string {
	buf, err := json.MarshalIndent(p, "", "\t")
	if err != nil {
		return ""
	}

	return string(buf)
}

func (p *Config) EncodeFormatYaml() string {
	buf, err := yaml.Marshal(p)
	if err != nil {
		return ""
	}

	return string(buf)
}

func (p *Config) YamlEncode() string {
	buf, err := yaml.Marshal(p)
	if err != nil {
		return ""
	}
	return string(buf)
}

func (p *Config) YamlDecode(data []byte) error {
	err := yaml.Unmarshal(data, p)
	if err != nil {
		return err
	}
	return nil
}

func (p *Config) getDomain(url string) string {
	domain := strings.Replace(url, "http://", "", -1)
	domain = strings.Replace(domain, "https://", "", -1)
	domains := strings.Split(domain, ":")
	return domains[0]
}

func (p *Config) addDomain(domainSet map[string]bool, domain string) {
	tmp := p.getDomain(domain)
	if _, has := domainSet[tmp]; !has {
		domainSet[tmp] = true
	}
}

func (p *Config) Domains() ([]string, error) {
	domainSet := map[string]bool{}

	p.addDomain(domainSet, p.Review.Nginx)
	p.addDomain(domainSet, p.Review.Web)
	p.addDomain(domainSet, p.Review.Http)
	p.addDomain(domainSet, p.Review.CDN)
	p.addDomain(domainSet, p.Review.Avatar)
	p.addDomain(domainSet, p.Review.Tcp)

	p.addDomain(domainSet, p.Domain.Nginx)
	p.addDomain(domainSet, p.Domain.Web)
	p.addDomain(domainSet, p.Domain.Http)
	p.addDomain(domainSet, p.Domain.CDN)
	p.addDomain(domainSet, p.Domain.Avatar)
	p.addDomain(domainSet, p.Domain.Tcp)

	for k, v := range p.Backup {
		p.addDomain(domainSet, k)
		for _, val := range v.Domains {
			p.addDomain(domainSet, val)
		}
		for _, val := range v.IpList {
			p.addDomain(domainSet, val)
		}
	}

	domains := []string{}
	for k := range domainSet {
		domains = append(domains, k)
	}

	return domains, nil
}

var (
	_key = "123"
)
