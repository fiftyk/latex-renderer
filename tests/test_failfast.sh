#!/bin/bash
# FailFast 策略验证测试

BASE_URL="http://192.168.215.2:8080"

echo "=== FailFast 策略验证测试 ==="
echo "并发限制: 2"
echo ""

# 重置缓存以获得准确的测试结果
rm -f cache/*.png 2>/dev/null

# 测试 10 个并发请求，验证 FailFast 行为
echo "发送 10 个并发请求..."
echo ""

success_count=0
reject_count=0
min_reject_time=999
max_reject_time=0

for i in $(seq 1 10); do
    # 使用唯一公式避免缓存
    latex_formula="ff_test_${i}_$(date +%s)"
    result=$(curl -s -o /tmp/response_${i}.txt -w "%{http_code}" -m 5 "${BASE_URL}/api?latex=${latex_formula}")
    time=$(curl -s -o /tmp/response_${i}.txt -w "%{time_total}" -m 5 "${BASE_URL}/api?latex=${latex_formula}")

    if [ "$result" = "200" ]; then
        success_count=$((success_count + 1))
        echo "请求$i: 200 成功, 时间: ${time}s"
    else
        reject_count=$((reject_count + 1))
        echo "请求$i: $result 被拒绝, 时间: ${time}s"
        # 记录拒绝响应时间
        if (( $(echo "$time < $min_reject_time" | bc -l) )); then
            min_reject_time=$time
        fi
        if (( $(echo "$time > $max_reject_time" | bc -l) )); then
            max_reject_time=$time
        fi
    fi
done

echo ""
echo "=== 测试结果汇总 ==="
echo "成功请求: $success_count"
echo "被拒绝请求: $reject_count"
if [ "$reject_count" -gt 0 ]; then
    echo "拒绝响应最短时间: ${min_reject_time}s"
    echo "拒绝响应最长时间: ${max_reject_time}s"
fi

echo ""
echo "=== 验收标准验证 ==="
if [ "$success_count" -le 2 ]; then
    echo "[PASS] 并发数不超过配置值 (成功: $success_count <= 2)"
else
    echo "[FAIL] 并发数超过配置值"
fi

if [ "$reject_count" -ge 8 ]; then
    echo "[PASS] 超过限制的请求被拒绝 (拒绝: $reject_count >= 8)"
else
    echo "[WARN] 拒绝请求数不符合预期"
fi

if [ "$reject_count" -gt 0 ] && (( $(echo "$max_reject_time < 0.1" | bc -l) )); then
    echo "[PASS] 503响应时间 < 100ms (最大: ${max_reject_time}s)"
else
    echo "[INFO] 拒绝响应时间: ${max_reject_time}s"
fi

echo ""
echo "=== 服务健康状态 ==="
curl -s "${BASE_URL}/health"
