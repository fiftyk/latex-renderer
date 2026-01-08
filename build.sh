#!/bin/bash
set -e

# 配置
IMAGE_NAME="latex-renderer"
REGISTRY="${REGISTRY:-crpi-vrqfzo6fw9cp7rqe.cn-wulanchabu.personal.cr.aliyuncs.com}"
FULL_IMAGE_NAME="${REGISTRY}/fiftyk/${IMAGE_NAME}:latest"
BUILD_DIR="$(cd "$(dirname "$0")" && pwd)"

# 编译 Go 应用
echo "=== 编译 Go 应用 ==="
cd "$BUILD_DIR"
mkdir -p bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/latex-renderer .

# 复制到构建上下文（buildx 使用独立上下文）
cp bin/latex-renderer latex-renderer

# 构建并推送镜像
echo "=== 构建并推送镜像 ==="
cd "$BUILD_DIR"
docker buildx build --platform linux/amd64 \
  -t "$FULL_IMAGE_NAME" \
  --push \
  .

# 清理临时文件
rm -f latex-renderer

echo "=== 完成 ==="
echo "镜像: $FULL_IMAGE_NAME"
