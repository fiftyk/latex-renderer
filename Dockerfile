# 构建参数：镜像仓库地址
# 使用示例：
#   docker build --build-arg DOCKER_REGISTRY=registry.cn-hangzhou.aliyuncs.com -t latex-renderer .
ARG DOCKER_REGISTRY=docker.io

# 构建阶段
FROM ${DOCKER_REGISTRY}/golang:1.24-alpine AS builder

WORKDIR /app

# 配置 Go 代理（国内镜像加速）和内存限制
ENV GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct \
    GOSUMDB=off \
    GOGC=20 \
    GOMEMLIMIT=256MiB

# 安装依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用（完全串行编译避免 OOM，限制内存使用）
RUN CGO_ENABLED=0 GOOS=linux go build -p 1 -a -installsuffix cgo -o latex-renderer .

# 运行阶段
FROM ${DOCKER_REGISTRY}/browserless/chrome:latest

# 切换到 root 用户进行安装
USER root

# 安装 wget（用于健康检查）和中文字体支持
RUN apt-get update && apt-get install -y \
    wget \
    fonts-wqy-microhei \
    fonts-wqy-zenhei \
    && rm -rf /var/lib/apt/lists/*

# 复制构建好的应用
COPY --from=builder /app/latex-renderer /usr/local/bin/

# 复制 KaTeX 静态文件
COPY --from=builder /app/static /app/static

# 创建缓存和日志目录
RUN mkdir -p /app/cache /app/logs && chmod 777 /app/cache /app/logs

# 设置环境变量
ENV CHROME_ARGS="--no-sandbox --disable-setuid-sandbox --disable-dev-shm-usage --disable-gpu"

# 默认日志路径
ENV LOG_PATH=/app/logs/app.log

# 切换回非 root 用户（安全最佳实践）
USER blessuser

# 默认命令
CMD ["latex-renderer"]
