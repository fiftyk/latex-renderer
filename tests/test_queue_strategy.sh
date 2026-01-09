#!/bin/bash

# 测试排队策略脚本

echo "=== 测试排队策略 ==="
echo "并发限制: 2"
echo "队列大小: 2"
echo "队列超时: 2s"
echo ""

# 测试1: 正常请求（应该成功）
echo "测试1: 发送单个请求"
time curl -s "http://localhost:8080/api?latex=E=mc^2" -w "\n响应时间: %{time_total}s\n" -o /dev/null

# 测试2: 达到并发限制的请求
echo ""
echo "测试2: 发送3个并发请求（超过并发限制2）"
for i in {1..3}; do
    curl -s "http://localhost:8080/api?latex=test$i" -w "请求$i: HTTP %{http_code}, 响应时间: %{time_total}s\n" -o /dev/null &
done
wait

echo ""
echo "测试完成"
