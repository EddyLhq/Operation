# GoDaddy 域名过期监控工具

这是一个简单的 Python 脚本，用于监控 GoDaddy 账号下的所有域名。如果域名未开启“自动续费”且距离到期时间少于 31 天，脚本将通过 Slack Webhook 发送提醒通知。

## 功能描述

- 获取所有域名的列表。
- 显示到期时间、状态、是否自动续费。
- **查询并显示每个域名下所有的 A 记录子域名及其指向的 IP。**
- 自动计算剩余天数。
- **自动监控告警**：
  - 如果域名状态不是 `ACTIVE`，立即告警。
  - 自动续费域名：到期前 31 天提醒检查支付方式。
  - 非自动续费域名：到期前 31 天提醒手动续费或开启自动续费。
- **DNS 查询**：支持查询 A 记录子域名（可在 `.env` 中关闭以提高运行速度）。

## 快速开始

### 1. 安装依赖

```bash
pip install -r requirements.txt
```

### 2. 配置环境变量

将 `.env.example` 重命名为 `.env` 并填写你的信息：

- `GODADDY_API_KEY`: 从 [GoDaddy Developer Portal](https://developer.godaddy.com/keys) 获取。
- `GODADDY_API_SECRET`: 同上。
- `SLACK_WEBHOOK_URL`: 你的 Slack App Webhook 地址。
- `GODADDY_PRODUCTION`: 默认为 `True`。如果是测试环境请设置为 `False`。

### 3. 运行脚本

```bash
python monitor.py
```

## 定时任务

建议使用 `crontab` (Linux) 或 任务计划程序 (Windows) 每天运行一次：

```bash
# Linux crontab 示例 (每天上午 9 点运行)
0 9 * * * /usr/bin/python3 /path/to/monitor.py
```

## Docker 部署

如果你想在服务器上以容器化方式运行：

### 1. 构建并启动

```bash
docker-compose up -d --build
```

### 2. 持续监控

在 `.env` 中设置 `CHECK_INTERVAL`（例如 `86400`），容器将按照设定的秒数循环执行监控任务，无需配置宿主机的 `crontab`。

> [!NOTE]
> 配置文件已适配旧版 Docker Compose (1.18.0+)，版本号设定为 `3.3`。
