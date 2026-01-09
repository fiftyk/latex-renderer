# 性能测试与优化文档

本目录包含了 LaTeX 渲染服务的所有性能测试文档、测试脚本和测试结果。

## 文档列表

### 📋 测试计划
- **性能测试计划.md** - 详细的性能测试方案设计
  - 包含10个并发级别的测试设计
  - 测试场景：基准测试、并发递增测试、复杂公式测试
  - 测试工具：Apache Bench (ab)

### 📊 测试报告
- **性能测试报告.md** - 全面的性能数据分析报告
  - 基于218,600+次请求的测试数据
  - 关键发现：最优并发数为16
  - 性能提升：161%（并发2→16）

### 🔧 策略实现
- **并发策略实现总结.md** - TimeoutQueueStrategy实现文档
  - 新增超时排队策略技术细节
  - 配置选项和推荐设置
  - 策略对比和适用场景

### 📈 项目总结
- **项目完成总结.md** - 整个优化项目的总结报告
  - 完成的工作清单
  - 量化收益分析
  - 生产部署建议

### 🔬 测试脚本
- **run_tests.sh** - 自动性能测试脚本
  - 可配置并发级别
  - 自动保存测试结果
  - 支持服务状态检查

- **test_queue_strategy.sh** - 队列策略验证脚本
  - 测试超时排队机制
  - 验证队列大小和超时配置
  - 展示策略行为

### 📁 测试结果
- **results/** - 详细的测试输出文件
  - `test_XX_concurrency_X.txt` - 每个并发级别的详细测试结果
  - 包含QPS、响应时间、失败率等指标

## 关键发现摘要

### 性能数据
| 并发数 | QPS | 失败率 | 平均响应时间 | 评价 |
|--------|-----|--------|--------------|------|
| 2 | 1,945 | 0% | 1.0ms | 稳定但性能偏低 |
| 4 | 4,177 | 0% | 1.0ms | 良好 |
| **16** | **5,084** | **0.3%** | **3.1ms** | **最佳性能** |
| 32 | 4,709 | 1.2% | 6.8ms | 开始下降 |
| 128+ | 6,000+ | >33% | >18ms | 不可用 |

### 默认配置变更
- **旧配置**: MAX_CONCURRENT=4
- **新配置**: MAX_CONCURRENT=16
- **性能提升**: 161%

### 新增并发策略
1. **FailFast** - 快速失败（原有）
   - 立即返回503错误
   - 适用于低延迟要求场景

2. **TimeoutQueue** - 超时排队（新增）
   - 请求排队等待，最多5秒
   - 更友好的用户体验
   - 适用于高峰流量场景

## 推荐生产配置

```bash
# 基本配置
MAX_CONCURRENT=16
RENDERER_OVERLOAD_STRATEGY=queue
RENDERER_QUEUE_SIZE=16
RENDERER_QUEUE_TIMEOUT=3s

# 高并发配置
MAX_CONCURRENT=32
RENDERER_OVERLOAD_STRATEGY=queue
RENDERER_QUEUE_SIZE=32
RENDERER_QUEUE_TIMEOUT=5s
```

## 测试环境
- **操作系统**: Linux 5.10.134
- **容器**: Docker/Podman
- **测试工具**: Apache Bench (ab)
- **总请求数**: 218,600+
- **测试时间**: 约2小时

## 如何复现测试

### 1. 启动服务
```bash
docker run -d --name latex-renderer \
  -p 8080:8080 \
  -v $(pwd)/cache:/app/cache \
  -v $(pwd)/logs:/app/logs \
  latex-renderer:local
```

### 2. 运行性能测试
```bash
cd tests/
./run_tests.sh
```

### 3. 运行队列策略测试
```bash
cd tests/
./test_queue_strategy.sh
```

## 相关文件

- **源代码变更**:
  - `config/config.go` - 新增队列配置
  - `renderer/chrome.go` - 实现TimeoutQueueStrategy
  - `main.go` - 集成策略选择逻辑

- **用户文档**:
  - `README.md` - 更新的用户文档和配置说明
  - 环境变量和并发策略说明

## 联系信息

如有疑问或需要更多信息，请参考项目主文档或联系开发团队。

---
**最后更新**: 2026-01-09
**基于**: 性能测试和策略实现结果