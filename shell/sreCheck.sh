#!/bin/bash
# SRE 服务器快速诊断脚本
# 用途：定位高负载、内存泄漏、IO瓶颈、网络异常等问题

set -euo pipefail  # 严格模式：出错即停、未定义变量报错、管道错误捕获

# 颜色定义（便于快速定位关键指标）
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# 打印分隔线与标题
print_title() {
    echo -e "\n${GREEN}==== $1 ====${NC}"
}

# 1. 系统负载与运行队列
print_title "1. 系统负载 (1/5/15分钟) & 运行队列"
uptime
echo -e "运行队列 ≤ CPU核心数 为正常，超过则存在拥堵"

# 2. CPU与内存 TOP10（合并显示，避免两次排序）
print_title "2. CPU + 内存 占用 TOP10 (按CPU降序，前10)"
ps -eo pid,ppid,user,%cpu,%mem,rss,comm --sort=-%cpu | head -11

# 3. 系统瓶颈：CPU上下文切换、队列、IO等待
print_title "3. vmstat 1秒3次 (重点关注 r, b, wa, si, so)"
vmstat 1 3 | awk 'NR==1 || NR==2 || NR==7 {print $0}'  # 显示标题+第一次+最后一次结果

# 4. 磁盘IO瓶颈
print_title "4. 磁盘IO iostat (%util ≥ 80% 表示磁盘繁忙)"
if command -v iostat &> /dev/null; then
    iostat -x 1 3 | awk 'NR==1 || NR==2 || NR==3 || /^[a-z]/ && $NF>0 {print $0}'
else
    echo -e "${RED}iostat 未安装，请运行: yum install sysstat 或 apt install sysstat${NC}"
fi

# 5. 内存使用总览
print_title "5. 内存使用 (available 低 + swap 高 => 内存紧张)"
free -h
echo -e "${YELLOW}提示: available 接近0 且 swap used 持续升高，需排查内存泄漏${NC}"

# 6. 磁盘空间 & inode
print_title "6. 磁盘分区使用率 (≥85% 建议关注)"
df -h | awk 'NR==1 || $5+0 >= 80 {print $0}'
print_title "7. Inode 使用率 (大量小文件场景)"
df -i | awk 'NR==1 || $5+0 >= 80 {print $0}'

# 7. 网络连接统计（整体）
print_title "8. 网络连接汇总"
ss -s

# 8. TCP 连接状态详细（突出 TIME_WAIT / CLOSE_WAIT）
print_title "9. TCP 连接状态分布 (大量 CLOSE_WAIT 表示程序未关闭连接)"
tcp_states=$(ss -tan | awk 'NR>1 {print $1}' | sort | uniq -c)
echo "$tcp_states"
closing=$(echo "$tcp_states" | grep -E "CLOSE-WAIT|TIME-WAIT" || true)
if [ -n "$closing" ]; then
    echo -e "${YELLOW}注意: 若 CLOSE-WAIT 持续 >1000，需检查代码未正确关闭socket${NC}"
fi

# 9. OOM 日志（内存溢出）
print_title "10. 最近 OOM (Out of Memory) 事件"
if dmesg -T | grep -qi "out of memory"; then
    dmesg -T | grep -i "out of memory" | tail -10
else
    echo "未发现 OOM 记录"
fi

# 10. 额外：系统平均负载与CPU核心数对比
print_title "11. 负载/核心数 快速诊断"
core_count=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo)
load_avg=$(uptime | awk -F 'load average:' '{print $2}' | awk -F, '{print $1}' | tr -d ' ')
if command -v bc &> /dev/null; then
    ratio=$(echo "scale=2; $load_avg / $core_count" | bc)
    echo "CPU核心数: $core_count, 1分钟负载: $load_avg, 负载/核心 = $ratio"
    if (( $(echo "$ratio > 1" | bc -l) )); then
        echo -e "${RED}>>> 负载过高，每核心负载 >1.0，存在严重拥塞 <<<${NC}"
    elif (( $(echo "$ratio > 0.7" | bc -l) )); then
        echo -e "${YELLOW}>>> 负载偏高，注意观察 <<<${NC}"
    else
        echo -e "${GREEN}负载正常${NC}"
    fi
else
    echo "CPU核心数: $core_count, 1分钟负载: $load_avg (安装 bc 可自动计算阈值)"
fi

# 11. 可选：查看当前占用CPU最高的内核线程/进程（调试）
print_title "12. D状态进程（不可中断睡眠，常见于IO hang）"
ps -eo state,pid,comm | awk '$1=="D" {print $0}' | head -10

echo -e "\n${GREEN}==== 诊断完成 ====${NC}"