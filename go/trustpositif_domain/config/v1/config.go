package config

import (
	"crypto/aes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	_key = "123"
)

type Domain struct {
	UseIp  bool     `json:"useIp,omitempty" yaml:"useIp,omitempty"`
	Domain string   `json:"domain,omitempty" yaml:"domain,omitempty"`
	Ip     []string `json:"ip,omitempty" yaml:"ip,omitempty"`
}

type Welcome struct {
	Scheme string `json:"scheme" yaml:"scheme"`
	Domain string `json:"domain" yaml:"domain"`
	Ip     string `json:"ip" yaml:"ip"`
	Nginx  Domain `json:"nginx,omitempty" yaml:"nginx,omitempty"`
	Http   Domain `json:"http,omitempty" yaml:"http,omitempty"`
	Web    Domain `json:"web,omitempty" yaml:"web,omitempty"`
	CDN    Domain `json:"cdn,omitempty" yaml:"cdn,omitempty"`
	Avatar Domain `json:"avatar,omitempty" yaml:"avatar,omitempty"`
	Tcp    Domain `json:"tcp,omitempty" yaml:"tcp,omitempty"`
}

type Config struct {
	Welcome []Welcome `json:"welcome"`
}

func (p *Config) Decode(data []byte) error {
	err := json.Unmarshal(data, &p.Welcome)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (p *Config) Encode() []byte {
	buf, err := json.Marshal(&p.Welcome)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return buf
}

func (p *Config) YamlDecode(data []byte) error {
	err := yaml.Unmarshal(data, p)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (p *Config) YamlEncode() string {
	buf, err := yaml.Marshal(p)
	if err != nil {
		return ""
	}
	return string(buf)
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
	buf, err := json.MarshalIndent(&p.Welcome, "", "\t")
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

func (p *Config) Domains() ([]string, error) {
	domainsSet := map[string]bool{}

	for _, v := range p.Welcome {
		domainsSet[fmt.Sprintf("idn.%s", v.Domain)] = true
		domainsSet[fmt.Sprintf("cdn.%s", v.Domain)] = true
		domainsSet[fmt.Sprintf("hall.%s", v.Domain)] = true

		domainsSet[v.Nginx.Domain] = true
		if v.Nginx.UseIp {
			for _, vv := range v.Nginx.Ip {
				domainsSet[vv] = true
			}
		}
		domainsSet[v.Http.Domain] = true
		if v.Http.UseIp {
			for _, vv := range v.Http.Ip {
				domainsSet[vv] = true
			}
		}
		domainsSet[v.Web.Domain] = true
		if v.Web.UseIp {
			for _, vv := range v.Web.Ip {
				domainsSet[vv] = true
			}
		}
		domainsSet[v.CDN.Domain] = true
		if v.CDN.UseIp {
			for _, vv := range v.CDN.Ip {
				domainsSet[vv] = true
			}
		}
		domainsSet[v.Avatar.Domain] = true
		if v.Avatar.UseIp {
			for _, vv := range v.Avatar.Ip {
				domainsSet[vv] = true
			}
		}
		tcpDomain := strings.Split(v.Tcp.Domain, ":")
		domainsSet[tcpDomain[0]] = true
		if v.Tcp.UseIp {
			for _, vv := range v.Tcp.Ip {
				item := strings.Split(vv, ":")
				domainsSet[item[0]] = true
			}
		}
	}

	domains := []string{}
	for k, _ := range domainsSet {
		domains = append(domains, k)
	}

	return domains, nil
}
