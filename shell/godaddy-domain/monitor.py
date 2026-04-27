import os
import requests
import json
from datetime import datetime, timezone
from dotenv import load_dotenv

# 加载环境变量
load_dotenv()

GODADDY_API_KEY = os.getenv('GODADDY_API_KEY')
GODADDY_API_SECRET = os.getenv('GODADDY_API_SECRET')
SLACK_WEBHOOK_URL = os.getenv('SLACK_WEBHOOK_URL')
IS_PRODUCTION = os.getenv('GODADDY_PRODUCTION', 'True').lower() == 'true'
ENABLE_A_RECORD_QUERY = os.getenv('ENABLE_A_RECORD_QUERY', 'True').lower() == 'true'
CHECK_INTERVAL = os.getenv('CHECK_INTERVAL') # 如果设置了此变量，则进入循环模式

BASE_URL = 'https://api.godaddy.com' if IS_PRODUCTION else 'https://api.ote-godaddy.com'

def get_godaddy_headers():
    return {
        'Authorization': f'sso-key {GODADDY_API_KEY}:{GODADDY_API_SECRET}',
        'Accept': 'application/json'
    }

def get_all_domains():
    """获取所有域名信息"""
    url = f"{BASE_URL}/v1/domains"
    headers = get_godaddy_headers()
    
    try:
        response = requests.get(url, headers=headers)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"获取域名列表失败: {e}")
        if response.text:
            print(f"详情: {response.text}")
        return []

def get_domain_a_records(domain):
    """获取指定域名下所有 A 记录的子域名"""
    url = f"{BASE_URL}/v1/domains/{domain}/records/A"
    headers = get_godaddy_headers()
    
    try:
        response = requests.get(url, headers=headers)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        # 某些域名可能没有 A 记录，返回 404 或空
        if hasattr(e, 'response') and e.response.status_code == 404:
            return []
        print(f"获取域名 {domain} 的 A 记录失败: {e}")
        return []

def send_slack_notification(message):
    """发送 Slack 通知"""
    if not SLACK_WEBHOOK_URL:
        print("未配置 SLACK_WEBHOOK_URL，跳过通知")
        return

    payload = {"text": message}
    try:
        response = requests.post(SLACK_WEBHOOK_URL, json=payload)
        response.raise_for_status()
        print("Slack 通知发送成功")
    except requests.exceptions.RequestException as e:
        print(f"发送 Slack 通知失败: {e}")

def monitor_domains():
    domains = get_all_domains()
    if not domains:
        print("未找到任何域名或获取失败。")
        return

    alerts = []
    now = datetime.now(timezone.utc)

    print(f"{'域名':<30} | {'到期时间':<25} | {'状态':<10} | {'自动续费':<8}")
    print("-" * 80)

    for domain_info in domains:
        domain_name = domain_info.get('domain')
        status = domain_info.get('status')
        expires_str = domain_info.get('expires')
        renew_auto = domain_info.get('renewAuto', False)

        # 解析到期时间 (GoDaddy 返回格式通常为 2024-05-20T12:34:56.000Z)
        # 去掉结尾的 Z 并处理微秒
        try:
            expires_dt = datetime.fromisoformat(expires_str.replace('Z', '+00:00'))
        except ValueError:
            print(f"解析日期失败: {expires_str}")
            continue

        days_remaining = (expires_dt - now).days

        print(f"{domain_name:<30} | {expires_str:<25} | {status:<10} | {renew_auto}")

        # 1. 状态异常告警 (非 ACTIVE 视为不可用)
        if status != 'ACTIVE':
            alert_msg = f"❗ *域名状态异常通知*\n• *域名*: `{domain_name}`\n• *当前状态*: `{status}`\n请检查该域名是否已被暂停或锁定。"
            alerts.append(alert_msg)

        # 2 & 3. 到期时间少于 31 天告警
        if days_remaining < 31:
            if renew_auto:
                alert_msg = (
                    f"💡 *域名自动续费提醒*\n"
                    f"虽然 `{domain_name}` 域名是自动续费的，但是确保支付方式正确!\n"
                    f"• *到期时间*: {expires_str}\n"
                    f"• *剩余天数*: {days_remaining} 天"
                )
            else:
                alert_msg = (
                    f"⚠️ *域名到期手动续费提醒*\n"
                    f"域名 `{domain_name}` 不是自动续费，如继续使用，请手动续费或者设置自动续费!\n"
                    f"• *到期时间*: {expires_str}\n"
                    f"• *剩余天数*: {days_remaining} 天"
                )
            alerts.append(alert_msg)

        # 4. A 记录查询功能 (根据配置决定是否开启)
        if ENABLE_A_RECORD_QUERY:
            a_records = get_domain_a_records(domain_name)
            if a_records:
                print(f"  └─ A 记录子域名:")
                for record in a_records:
                    name = record.get('name')
                    data = record.get('data')
                    subdomain = f"{name}.{domain_name}" if name != "@" else domain_name
                    print(f"     • {subdomain} -> {data}")
            else:
                print(f"  └─ 无 A 记录")
        
        print() # 换行美化

    if alerts:
        full_alert = "\n\n".join(alerts)
        send_slack_notification(full_alert)
    else:
        print("没有需要提醒的域名。")

if __name__ == "__main__":
    import time
    if not GODADDY_API_KEY or not GODADDY_API_SECRET:
        print("错误: 请在 .env 文件中配置 GODADDY_API_KEY 和 GODADDY_API_SECRET")
        exit(1)
    
    print(f"配置检查: CHECK_INTERVAL={CHECK_INTERVAL}")
    
    if CHECK_INTERVAL and CHECK_INTERVAL.strip():
        try:
            interval = int(CHECK_INTERVAL)
        except ValueError:
            print(f"错误: CHECK_INTERVAL 必须是数字，当前值: '{CHECK_INTERVAL}'")
            monitor_domains()
            exit(0)

        print(f"进入循环监控模式，间隔时间: {interval} 秒")
        while True:
            try:
                monitor_domains()
            except Exception as e:
                print(f"执行监控时发生未捕获的异常: {e}")
            
            print(f"等待下一次检查 ({interval} 秒后)...")
            time.sleep(interval)
    else:
        print("未设置 CHECK_INTERVAL，将只运行一次。")
        monitor_domains()
