#!/bin/bash
# Deployment script for low-memory servers (2GB RAM)
# This script deploys latex-renderer with optimized settings for 2GB memory

set -e

IMAGE_URL="${1:-crpi-vrqfzo6fw9cp7rqe-vpc.cn-wulanchabu.personal.cr.aliyuncs.com/fiftyk/latex-renderer:latest}"

echo "部署 latex-renderer（低内存配置）"
echo "镜像: $IMAGE_URL"
echo ""

# Stop and remove existing container if exists
if docker ps -a --format '{{.Names}}' | grep -q '^latex-renderer$'; then
    echo "停止并删除现有容器..."
    docker stop latex-renderer || true
    docker rm latex-renderer || true
fi

# Pull latest image
echo "拉取最新镜像..."
docker pull "$IMAGE_URL"

# Run container with low-memory configuration
echo "启动容器（低内存配置）..."
docker run -d \
  --name latex-renderer \
  --memory="1536m" \
  --memory-swap="1536m" \
  --restart=unless-stopped \
  -e MAX_CONCURRENT=2 \
  -e RENDERER_MAX_REQUESTS=50 \
  -e RENDERER_MAX_INTERVAL=10m \
  -e RENDERER_OVERLOAD_STRATEGY=queue \
  -e RENDERER_QUEUE_SIZE=4 \
  -e RENDERER_QUEUE_TIMEOUT=10s \
  -e LOG_LEVEL=info \
  -p 8080:8080 \
  "$IMAGE_URL"

echo ""
echo "✓ 部署完成！"
echo ""
echo "配置信息："
echo "  - 内存限制: 1536MB"
echo "  - 最大并发: 2"
echo "  - 容器名称: latex-renderer"
echo "  - 端口: 8080"
echo ""
echo "查看日志: docker logs -f latex-renderer"
echo "查看状态: docker stats latex-renderer"
echo "健康检查: curl http://localhost:8080/health"
