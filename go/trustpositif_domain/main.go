package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
	"trustpositif_domain/config"
)

type SendMessage struct {
	ChatId           string `json:"channel"`
	Text             string `json:"text"`
	ReplyToMessageId int32  `json:"reply_to_message_id"`
}

func (r *SendMessage) Encode() []byte {
	buf, err := json.Marshal(r)
	if err != nil {
		return nil
	}
	return buf
}

func (r *SendMessage) NewReader() io.Reader {
	return bytes.NewReader(r.Encode())
}

type DomainStatus struct {
	Domain string `json:"Domain"`
	Status string `json:"Status"`
}

type CheckDomain struct {
	Response int            `json:"response"`
	Values   []DomainStatus `json:"values"`
}

func (p *CheckDomain) Decode(data []byte) error {
	return json.Unmarshal(data, p)
}

// 从HTML中提取CSRF token
func extractCSRFToken(html string) string {
	// 查找: <input type="hidden" name="csrf_token" value="token值">
	re := regexp.MustCompile(`name="csrf_token"\s+value="([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// 获取CSRF token和客户端
func getCSRFTokenAndClient() (string, *http.Client, error) {
	jar, _ := cookiejar.New(nil)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: tr,
		Jar:       jar,
		Timeout:   30 * time.Second,
	}

	// 访问欢迎页面获取token
	req, err := http.NewRequest("GET", "https://trustpositif.komdigi.go.id/welcome", nil)
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	token := extractCSRFToken(string(body))
	if token == "" {
		return "", client, fmt.Errorf("无法提取CSRF token")
	}

	fmt.Printf("获取到CSRF token: %s...\n", token[:8])
	return token, client, nil
}

func checkDomains(names []string) []string {
	// 获取CSRF token和客户端
	token, client, err := getCSRFTokenAndClient()
	if err != nil {
		fmt.Printf("获取CSRF token失败: %v\n", err)
		return []string{}
	}

	// 去重域名
	seen := make(map[string]bool)
	var cleanNames []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		cleanNames = append(cleanNames, name)
	}

	if len(cleanNames) == 0 {
		fmt.Println("域名列表为空")
		return []string{}
	}

	fmt.Printf("检查 %d 个域名: %v\n", len(cleanNames), cleanNames)

	// 准备表单数据 - 根据JavaScript代码
	formData := url.Values{}
	formData.Set("csrf_token", token)
	formData.Set("name", strings.Join(cleanNames, "\n"))

	// 根据JavaScript代码，请求的URL是 /Rest_server/getrecordsname_home
	requestURL := "https://trustpositif.komdigi.go.id/Rest_server/getrecordsname_home"

	fmt.Printf("发送POST请求到: %s\n", requestURL)
	fmt.Printf("表单数据: csrf_token=%s..., name=%d个域名\n", token[:8], len(cleanNames))

	// 创建POST请求
	req, err := http.NewRequest("POST", requestURL, strings.NewReader(formData.Encode()))
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return []string{}
	}

	// 设置请求头 - 模拟AJAX请求
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://trustpositif.komdigi.go.id/welcome")
	req.Header.Set("Origin", "https://trustpositif.komdigi.go.id")

	// 发送请求
	// startTime := time.Now()
	resp, err := client.Do(req)
	// elapsedTime := time.Since(startTime)

	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return []string{}
	}
	defer resp.Body.Close()

	// fmt.Printf("响应状态码: %d, 耗时: %v\n", resp.StatusCode, elapsedTime)

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return []string{}
	}

	// 检查响应
	bodyStr := string(body)

	// 如果是HTML，打印错误信息
	if strings.Contains(bodyStr, "<!DOCTYPE") || strings.Contains(bodyStr, "<html") {
		fmt.Println("错误：服务器返回了HTML页面而不是JSON")

		if len(bodyStr) > 500 {
			fmt.Printf("HTML预览: %.500s...\n", bodyStr)
		}

		return []string{}
	}

	// 尝试解析JSON
	status := &CheckDomain{}
	if err := status.Decode(body); err != nil {
		fmt.Printf("解析JSON失败: %v\n", err)

		// 显示响应内容以便调试
		if len(bodyStr) > 200 {
			fmt.Printf("响应内容: %.200s\n", bodyStr)
		}

		return []string{}
	}

	fmt.Printf("解析成功，找到 %d 个域名状态\n", len(status.Values))

	// 收集被封禁的域名
	blockedDomains := []string{}
	for _, v := range status.Values {
		switch v.Status {
		case "Ada":
			blockedDomains = append(blockedDomains, v.Domain)
			fmt.Printf("✓ %s - 封禁\n", v.Domain)
		case "Tidak Ada":
			fmt.Printf("✓ %s - 正常\n", v.Domain)
		default:
			fmt.Printf("? %s - 未知状态: %s\n", v.Domain, v.Status)
		}
	}

	return blockedDomains
}

func getWelcome(url string) ([]byte, error) {
	client := &http.Client{}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return body, nil
}

func getNewConfig(url string) (config.WelcomeConfig, error) {
	buf, err := getWelcome(url)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	cfg := config.NewWelcomeConfig(true)
	if err := cfg.DecryptDecode(string(buf)); err != nil {
		fmt.Println(err)
		return nil, err
	}

	return cfg, nil
}

func getDomains(cfg *config.Config) ([]string, error) {
	domainSet := map[string]bool{}
	for _, v := range cfg.WelcomeUrl {
		newCfg, err := getNewConfig(v)
		if err != nil {
			continue
		}

		if vals, err := newCfg.Domains(); err == nil {
			for _, v := range vals {
				domainSet[v] = true
			}
		}
	}

	// 添加web官网检查
	for _, v := range cfg.Web {
		domainSet[v] = true
	}

	domains := []string{}
	for k := range domainSet {
		domains = append(domains, k)
	}
	fmt.Printf("请求域名集合:%s",domains)
	return domains, nil
}

func sendMessage(botToken string, chatId string, text string) error {
	url := fmt.Sprintf("https://slack.com/api/chat.postMessage")
	data := map[string]string{
		"channel": chatId,
		"text":    text,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))

	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+botToken)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: tr,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return err
	}

	log.Println(string(body))

	return nil
}

var (
	// welcomeUrl = "https://idn-web.oss-ap-southeast-5.aliyuncs.com/welcome/welcome.json"
	cfgFile    = "config.yaml"
	slackToken = ""
	channelId  = ""
	checkTime  = 1
)

func init() {
	flag.StringVar(&cfgFile, "c", "config.yaml", "配置文件")
	flag.StringVar(&slackToken, "slacktoken", "", "slack机器人token")
	flag.StringVar(&channelId, "channelid", "", "频道id") //副包的id
	flag.IntVar(&checkTime, "checkTime", 1, "检查时间，单位分钟")
}

func main() {
	flag.Parse()

	cfg, err := config.LoadConfigByFile(cfgFile)
	if err != nil {
		panic(err)
	}

	ticker := time.NewTicker(time.Minute * time.Duration(checkTime))
	//ticker := time.NewTicker(time.Second * time.Duration(checkTime))
	defer ticker.Stop()

	// log.SetFlags(log.LstdFlags)

	fmt.Println("=== 域名封禁检查工具 ===")
	for {
		<-ticker.C
		names, err := getDomains(cfg)
		if err != nil {
			fmt.Println(err)
			continue
		}
		// 方法1：自动获取token
		// fmt.Println("\n方法1：自动获取CSRF token...")
		blockedDomains := checkDomains(names)
		if len(blockedDomains) > 0 {
			fmt.Println("被封禁域名:", blockedDomains)
			txt := fmt.Sprintf("被封域名\n%s\n", strings.Join(blockedDomains, "\n"))
			if err := sendMessage(slackToken, channelId, txt); err != nil {
				continue
			}
		} else {
			fmt.Println("域名未发现异常")
		}

	}
}
