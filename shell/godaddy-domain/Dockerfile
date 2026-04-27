FROM python:3.11-slim

# 设置工作目录
WORKDIR /app

# 设置时区 (可选，方便日志查看本地时间)
ENV TZ=Asia/Jakarta
RUN apt-get update && apt-get install -y tzdata && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 复制依赖文件并安装
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制源码
COPY . .

# 运行脚本
CMD ["python", "monitor.py"]
