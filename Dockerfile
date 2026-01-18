# 构建参数：镜像仓库地址
# 使用示例：
#   docker build --build-arg DOCKER_REGISTRY=registry.cn-hangzhou.aliyuncs.com -t latex-renderer .
ARG DOCKER_REGISTRY=docker.m.daocloud.io

# 运行阶段 - 使用 browserless/chrome（功能完整的无头 Chrome）
FROM ${DOCKER_REGISTRY}/browserless/chrome:latest

# 切换到 root 用户进行安装
USER root

# 安装 wget（用于健康检查）和中文字体支持
RUN apt-get update && apt-get install -y \
    wget \
    fonts-wqy-microhei \
    fonts-wqy-zenhei \
    && rm -rf /var/lib/apt/lists/*

# 复制预编译的应用
COPY latex-renderer /usr/local/bin/
RUN chmod +x /usr/local/bin/latex-renderer

# 复制 KaTeX 静态文件
COPY static /app/static

# 创建缓存和日志目录，并创建非 root 用户
RUN mkdir -p /app/cache /app/logs && \
    groupadd -r appuser && useradd -r -g appuser appuser && \
    chown -R appuser:appuser /app

# 设置环境变量
ENV CHROME_ARGS="--no-sandbox --disable-setuid-sandbox --disable-dev-shm-usage --disable-gpu --single-process --disable-background-networking --disable-background-timer-throttling --disable-breakpad --disable-sync --metrics-recording-only --mute-audio"
ENV CHROME_EXECUTABLE_PATH=
ENV WORKDIR=/app

# 默认日志路径
ENV LOG_PATH=/app/logs/app.log

# 设置工作目录
WORKDIR /app

# 切换到非 root 用户
USER appuser

# 默认命令
CMD ["/usr/local/bin/latex-renderer"]
