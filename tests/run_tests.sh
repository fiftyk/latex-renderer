#!/bin/bash

# 性能测试脚本
echo "开始性能测试..."
echo ""

# 测试配置
concurrencies=(2 4 8 16 32 64 128 256 512 1000)
total_requests=1000

for concurrency in "${concurrencies[@]}"; do
    echo "=== 测试: 并发${concurrency} ==="

    # 计算总请求数：至少1000，或者并发数的100倍
    reqs=$((concurrency * 100))
    if [ $reqs -lt $total_requests ]; then
        reqs=$total_requests
    fi

    # 执行测试
    ab -n $reqs -c $concurrency -q "http://localhost:8080/api?latex=E=mc^2" \
        > /root/latex-renderer/results/test_$(printf "%02d" $concurrency)_concurrency_${concurrency}.txt 2>&1

    # 检查服务是否还活着
    if ! curl -s http://localhost:8080/health > /dev/null; then
        echo "服务已崩溃！停止测试。"
        break
    fi

    echo "完成并发${concurrency}测试"
    echo ""

    # 等待一下让服务恢复
    sleep 2
done

echo "所有测试完成！"
