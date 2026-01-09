#!/bin/bash

# 测试缓存功能的脚本

echo "=== LaTeX Renderer 缓存测试 ==="
echo ""

# 启动服务器（在后台）
echo "启动服务器..."
./latex-renderer &
SERVER_PID=$!
echo "服务器PID: $SERVER_PID"

# 等待服务器启动
sleep 5

# 测试缓存功能
echo ""
echo "=== 测试 1: 第一次请求（应该缓存未命中） ==="
curl -i "http://localhost:8080/api?latex=E=mc^2" 2>&1 | head -20

echo ""
echo "=== 测试 2: 第二次请求（应该缓存命中） ==="
curl -i "http://localhost:8080/api?latex=E=mc^2" 2>&1 | head -20

echo ""
echo "=== 检查缓存目录 ==="
ls -la ./cache/latex/ 2>/dev/null || echo "缓存目录不存在或为空"

echo ""
echo "=== 检查服务器日志 ==="
echo "服务器仍在运行，PID: $SERVER_PID"
echo "查看日志: tail -f /path/to/log/file"

# 停止服务器
echo ""
echo "=== 停止服务器 ==="
kill $SERVER_PID 2>/dev/null
echo "服务器已停止"
