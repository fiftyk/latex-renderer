#!/bin/bash
set -e

# 配置
IMAGE_NAME="latex-renderer"
REGISTRY="${REGISTRY:-crpi-vrqfzo6fw9cp7rqe-vpc.cn-wulanchabu.personal.cr.aliyuncs.com}"
FULL_IMAGE_NAME="${REGISTRY}/fiftyk/${IMAGE_NAME}:latest"
BUILD_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== LaTeX Renderer 构建脚本 ==="
echo "镜像名称: $FULL_IMAGE_NAME"
echo "构建目录: $BUILD_DIR"
echo ""

# 编译 Go 应用
echo "=== 编译 Go 应用 ==="
cd "$BUILD_DIR"
mkdir -p bin

# 清理旧的二进制文件
rm -f bin/latex-renderer latex-renderer

# 使用优化参数编译，减少内存使用
echo "开始编译（使用优化参数）..."
GOGC=10 GODEBUG=gctrace=1 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/latex-renderer .
echo "编译完成"

# 复制到构建上下文（buildx 使用独立上下文）
echo ""
echo "=== 准备构建上下文 ==="
cp bin/latex-renderer latex-renderer
echo "二进制文件已复制到构建上下文"

# 创建.dockerignore临时文件以忽略bin目录（可选优化）
# echo "bin/" > .dockerignore.build 2>/dev/null || true

# 构建并推送镜像
echo ""
echo "=== 构建并推送镜像 ==="
cd "$BUILD_DIR"

# 检查Podman登录状态
if ! podman info > /dev/null 2>&1; then
  echo "错误: Podman未运行"
  exit 1
fi

# 构建镜像
echo "构建镜像..."
podman build \
  -t "$FULL_IMAGE_NAME" \
  --build-arg DOCKER_REGISTRY=docker.m.daocloud.io \
  .

BUILD_STATUS=$?

if [ $BUILD_STATUS -eq 0 ]; then
  echo ""
  echo "推送镜像..."
  # 推送镜像
  podman push "$FULL_IMAGE_NAME"
  BUILD_STATUS=$?
fi

BUILD_STATUS=$?

# 清理临时文件
echo ""
echo "=== 清理临时文件 ==="
rm -f latex-renderer

if [ $BUILD_STATUS -eq 0 ]; then
  echo ""
  echo "=== 构建完成 ==="
  echo "✅ 镜像构建并推送成功"
  echo "📦 镜像: $FULL_IMAGE_NAME"
else
  echo ""
  echo "=== 构建失败 ==="
  echo "❌ 镜像构建或推送失败"
  exit 1
fi
