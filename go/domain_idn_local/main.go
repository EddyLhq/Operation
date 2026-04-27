package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	apiToken                = "c18a61a83ef17e3de1f2990cf1837324"
	slackToken              = "xoxb-8093679394885-8330984712113-Rn5mwRFv36w1xRcgTYoWU4h7"
	channelId               = "C0936T2MKPG"
	monitorProcessChannelId = "C0936T2MKPG"
)

// 配置结构体
type Config struct {
	GameDomains          []string `json:"game_domains"`           // 游戏域名列表（仅需Ping检测）
	WebDomains           []string `json:"web_domains"`            // Web域名列表（需要Ping+HTTP检测）
	LogPath              string   `json:"log_path"`               // 日志文件路径
	CheckIntervalMinutes int      `json:"check_interval_minutes"` // 检测间隔分钟数
	// CheckProcessMinutes  int      `json:"check_process_minutes"`  //进程间隔检查分钟数
}

var (
	config           Config
	domainStatus     = make(map[string]int) // 记录每个域名的连续封禁次数
	domainStatusLock sync.Mutex
)

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := validateConfig(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	// 初始化日志
	if err := initLogger(); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	log.Printf("启动域名监控服务，检测间隔: %d 分钟\n", config.CheckIntervalMinutes)

	// 立即执行一次检测
	//checkAllDomains()

	sendTxt := fmt.Sprintf("封禁域名服务进程已启动!")
	if err := sendMessage(slackToken, monitorProcessChannelId, sendTxt); err != nil {
		log.Println("监控进程通知发送Slack通知失败:", err)
	}
	// 使用 WaitGroup 等待所有 goroutine
	// var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// //进程监控goroutine
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	processTicker := time.NewTicker(time.Duration(config.CheckProcessMinutes) * time.Minute)
	// 	defer processTicker.Stop()

	// 	for {
	// 		select {
	// 		case <-processTicker.C:
	// 			if err := sendMessage(slackToken, monitorProcessChannelId, "封禁域名服务进程正常!"); err != nil {
	// 				log.Println("进程状态通知发送失败:", err)
	// 			}
	// 		case <-ctx.Done():
	// 			log.Println("进程监控退出")
	// 			return
	// 		}
	// 	}
	// }()
	//域名检测goroutine
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	domainTicker := time.NewTicker(time.Duration(config.CheckIntervalMinutes) * time.Minute)
	// 	defer domainTicker.Stop()

	// 	// 立即执行一次，然后按间隔执行
	// 	for {
	// 		select {
	// 		case <-domainTicker.C:
	// 			checkAllDomains()
	// 		case <-ctx.Done():
	// 			log.Println("域名检测退出")
	// 			return
	// 		}
	// 	}
	// }()
	go func() {
		Ticker := time.NewTicker(time.Duration(config.CheckIntervalMinutes) * time.Minute)
		defer Ticker.Stop()
		for {
			select {
			case <-Ticker.C:
				checkAllDomains()
			case <-ctx.Done():
				log.Println("域名检测退出")
				return
			}
		}

	}()
	// 等待中断信号
	sigCh := newFunction()
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	log.Println("服务已启动，按 Ctrl+C 退出...")

	// 阻塞等待退出信号
	<-sigCh
	log.Println("收到退出信号，正在关闭服务...")

	// 发送关闭通知
	shutdownTxt := "封禁域名服务进程正在关闭..."
	if err := sendMessage(slackToken, monitorProcessChannelId, shutdownTxt); err != nil {
		log.Println("关闭通知发送失败:", err)
	}
	cancel() // 通知所有 goroutine 退出
	// wg.Wait() // 等待所有 goroutine 完成
}

func newFunction() chan os.Signal {
	sigCh := make(chan os.Signal, 1)
	return sigCh
}

func loadConfig() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	configPath := filepath.Join(filepath.Dir(exePath), "config.json")
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	if err := json.Unmarshal(configFile, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 设置默认值
	if config.CheckIntervalMinutes <= 0 {
		config.CheckIntervalMinutes = 1
	}
	if config.LogPath == "" {
		config.LogPath = "C:\\idn-domain-monitor\\domain_block_monitor.log"
	}

	return nil
}

func validateConfig() error {
	if len(config.GameDomains) == 0 && len(config.WebDomains) == 0 {
		return fmt.Errorf("配置错误: 未配置任何域名")
	}
	if config.CheckIntervalMinutes < 1 {
		return fmt.Errorf("配置错误: 检测间隔不能小于1分钟")
	}
	return nil
}

func initLogger() error {

	// 确保日志目录存在
	if err := os.MkdirAll(filepath.Dir(config.LogPath), 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}
	// logFile, err := os.OpenFile(config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	// 	return err
	// }
	logger := &lumberjack.Logger{
		Filename:   config.LogPath,
		MaxSize:    10,
		MaxBackups: 15,
		MaxAge:     20,
		Compress:   true,
		LocalTime:  true,
	}
	log.SetOutput(logger)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return nil
}
func checkAllDomains() {
	log.Println("开始新一轮域名检测...")
	startTime := time.Now()

	// 使用channel并行检测
	resultChan := make(chan string, len(config.GameDomains)+len(config.WebDomains))

	// 检测游戏域名
	for _, domain := range config.GameDomains {
		go func(d string) {
			isBlocked := checkGameDomain(d)
			if isBlocked {
				domainStatusLock.Lock()
				domainStatus[d]++

				// 第一次检测到封禁，立即进行连续检测
				if domainStatus[d] == 1 {
					domainStatusLock.Unlock()

					// 进行连续两次快速检测（加上第一次已经是3次）
					for i := 0; i < 2; i++ {
						time.Sleep(2 * time.Second) // 间隔2秒检测
						if checkGameDomain(d) {
							domainStatusLock.Lock()
							domainStatus[d]++
							domainStatusLock.Unlock()
						} else {
							break // 如果中途检测正常则跳出
						}
					}

					domainStatusLock.Lock() // 重新加锁检查最终状态
				}

				if domainStatus[d] >= 3 {
					resultChan <- fmt.Sprintf("[游戏封禁] %s (连续%d次检测到封禁)", d, domainStatus[d])
					txt := fmt.Sprintf("来自新加坡监控告警:\n[游戏域名] %s 被telkomsel运营商封禁 (连续3次检测确认)", d)
					if err := sendMessage(slackToken, channelId, txt); err != nil {
						log.Println("发送Slack通知失败:", err)
					}
					domainStatus[d] = 0 // 重置计数
				} else {
					resultChan <- fmt.Sprintf("[游戏疑似封禁] %s (连续%d次检测到封禁)", d, domainStatus[d])
				}
				domainStatusLock.Unlock()
			} else {
				domainStatusLock.Lock()
				if domainStatus[d] > 0 {
					resultChan <- fmt.Sprintf("[游戏恢复] %s (之前连续%d次检测到封禁)", d, domainStatus[d])
					domainStatus[d] = 0 // 重置计数
				} else {
					resultChan <- fmt.Sprintf("[游戏正常] %s", d)
				}
				domainStatusLock.Unlock()
			}
		}(domain)
	}
	// 检测Web域名
	for _, domain := range config.WebDomains {
		go func(d string) {
			isBlocked := checkWebDomain(d)
			if isBlocked {
				domainStatusLock.Lock()
				domainStatus[d]++
				// 第一次检测到封禁，立即进行连续检测
				if domainStatus[d] == 1 {
					domainStatusLock.Unlock() // 先解锁，避免连续检测时死锁

					// 进行连续两次快速检测（加上第一次已经是3次）
					for i := 0; i < 2; i++ {
						time.Sleep(2 * time.Second) // 间隔2秒检测
						if checkGameDomain(d) {
							domainStatusLock.Lock()
							domainStatus[d]++
							domainStatusLock.Unlock()
						} else {
							break // 如果中途检测正常则跳出
						}
					}

					domainStatusLock.Lock() // 重新加锁检查最终状态
				}
				if domainStatus[d] >= 3 {
					resultChan <- fmt.Sprintf("[落地页&官网域名封禁] %s (连续%d次检测到封禁)", d, domainStatus[d])
					txt := fmt.Sprintf("来自新加坡监控告警:\n[落地页&官网域名] %s 被telkomsel运营商封禁 (连续3次检测确认)", d)
					if err := sendMessage(slackToken, channelId, txt); err != nil {
						log.Println("发送Slack通知失败:", err)
					}
					domainStatus[d] = 0 // 重置计数
				} else {
					resultChan <- fmt.Sprintf("[落地页&官网域名疑似封禁] %s (连续%d次检测到封禁)", d, domainStatus[d])
				}
				domainStatusLock.Unlock()
			} else {
				domainStatusLock.Lock()
				if domainStatus[d] > 0 {
					resultChan <- fmt.Sprintf("[落地页&官网域名恢复] %s (之前连续%d次检测到封禁)", d, domainStatus[d])
					domainStatus[d] = 0 // 重置计数
				} else {
					resultChan <- fmt.Sprintf("[落地页&官网域名正常] %s", d)
				}
				domainStatusLock.Unlock()
			}
		}(domain)
	}
	// 收集结果
	for i := 0; i < len(config.GameDomains)+len(config.WebDomains); i++ {
		log.Println(<-resultChan)
	}

	log.Printf("检测完成，耗时: %.2f秒\n", time.Since(startTime).Seconds())
}

// 检测游戏域名（仅使用Ping检测）
func checkGameDomain(domain string) bool {
	isBlocked, _ := checkPing(domain)
	return isBlocked
}
func checkWebDomain(domain string) bool {
	isPingBlocked, isPingReachable := checkPing(domain)
	// Ping明确返回封禁特征（如TTL expired）
	if isPingBlocked {
		return true // 直接判定为封禁
	}
	if isPingReachable {
		return false // 网络通畅，判定为未封禁
	}
	isHTTPBlocked, _ := checkHTTP(domain)
	return isHTTPBlocked
}
func checkPing(domain string) (isBlocked bool, isReachable bool) {
	cmd := exec.Command("ping", "-n", "4", "-w", "1000", domain)
	output, _ := cmd.CombinedOutput()
	//log.Printf("ping结果返回内容如下:\n %s", output)
	outputStr := string(output)
	//log.Printf("ping结果返回内容如下(格式化处理):\n %s", outputStr)
	// 情况1：明确封禁特征（包含Reply from但TTL过期）
	if hasBlockingFeatures(outputStr) {
		return true, false
	}

	// 情况2：统计有效响应包
	is100PercentLoss := parsePacketLoss(outputStr)
	if is100PercentLoss {
		return false, false // 100%丢包视为不可达
	}
	// 情况3：完全无响应
	return false, true
}

func hasBlockingFeatures(output string) bool {
	// 特征1：TTL过期封禁
	if strings.Contains(output, "TTL expired in transit") {
		return true
	}

	// 特征2：运营商拦截页面特征
	if strings.Contains(output, "mypage.blocked") {
		return true
	}

	return false
}

// 解析ping值返回的结果
func parsePacketLoss(output string) (is100PercentLoss bool) {
	// 匹配模式：Packets: Sent = 4, Received = 2, Lost = 2 (50% loss)
	re := regexp.MustCompile(`Packets:\s*Sent\s*=\s*(\d+),\s*Received\s*=\s*(\d+),\s*Lost\s*=\s*(\d+)\s*\((\d+)% loss\)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) == 5 {
		sent, _ := strconv.Atoi(matches[1])
		received, _ := strconv.Atoi(matches[2])
		lost, _ := strconv.Atoi(matches[3])
		lossPercent, _ := strconv.Atoi(matches[4])

		// 验证数据一致性
		if lost == sent-received && lossPercent == (lost*100)/sent {
			return lossPercent == 100
		}
	}
	return false // 默认不是100%丢包
}

// http请求判断封禁监控
func checkHTTP(domain string) (isBlocked bool, isReachable bool) {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   20 * time.Second,
	}
	url := fmt.Sprintf("https://" + domain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, false
	}

	// 设置请求头模拟浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		// 输出错误到请求日志
		log.Printf("返回错误信息: %s", err)
		// 检查错误类型
		if isConnectionForciblyClosed(err) {
			return true, false // 连接被强制关闭，可能被封禁
		}
		return false, true
	}
	defer resp.Body.Close()
	return false, true

}

func isConnectionForciblyClosed(err error) bool {
	if err == nil {
		return false
	}

	// 检查错误字符串
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "context deadline exceeded"),
		strings.Contains(errStr, "connection refused"),
		strings.Contains(errStr, "connection reset"):
		return true // 明确封禁

	default:
		return false // 其他错误视为网络问题
	}
	//return strings.Contains(errStr, "An existing connection was forcibly closed by the remote host")
}

// 发送消息
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
