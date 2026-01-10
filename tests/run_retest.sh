#!/bin/bash
# 测试脚本

BASE_URL="http://192.168.215.2:8080"

echo "=== 503 响应头检查 ==="
curl -sI "${BASE_URL}/api?latex=test_$(date +%s)" 2>&1 | head -5

echo ""
echo "=== 503 响应体检查 ==="
curl -s "${BASE_URL}/api?latex=test_$(date +%s)" 2>&1 | head -c 100

echo ""
echo ""
echo "=== 持续压力测试: 顺序发送20个请求 ==="
success=0
failed=0
for i in $(seq 1 20); do
    result=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/api?latex=pressure_test_${i}")
    if [ "$result" = "200" ]; then
        success=$((success + 1))
    else
        failed=$((failed + 1))
    fi
done
echo "成功: $success, 失败: $failed"

echo ""
echo "=== 最终健康检查 ==="
curl -s "${BASE_URL}/health"
