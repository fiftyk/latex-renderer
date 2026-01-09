# 更新日志

## [2.0.0] - 2026-01-09

### 🚀 重大更新

#### 性能优化
- **默认并发数优化**: 从4提升至16，性能提升**161%**
  - 基于218,600+次请求的性能测试结果
  - 最优并发数16：QPS 5,084，失败率仅0.3%
  - 影响文件: `config/config.go`

#### 新功能
- **新增TimeoutQueueStrategy（超时排队策略）**
  - 文件: `renderer/chrome.go`
  - 功能: 并发满时请求排队等待，而非直接拒绝
  - 可配置队列大小和超时时间
  - 提供更友好的用户体验

- **新增并发策略配置**
  - `RENDERER_OVERLOAD_STRATEGY`: 选择过载处理策略 (failfast/queue)
  - `RENDERER_QUEUE_SIZE`: 队列大小（默认8）
  - `RENDERER_QUEUE_TIMEOUT`: 排队超时时间（默认5s）

#### 文档更新
- **README.md** - 新增并发控制与过载策略章节
- **新增tests目录** - 包含所有测试文档和脚本
  - `tests/性能测试计划.md` - 详细测试方案
  - `tests/性能测试报告.md` - 测试数据分析
  - `tests/并发策略实现总结.md` - 技术实现文档
  - `tests/项目完成总结.md` - 项目总结
  - `tests/README.md` - 测试文档索引

### 配置变更

#### 默认配置
| 配置项 | 旧值 | 新值 | 说明 |
|--------|------|------|------|
| `MAX_CONCURRENT` | 4 | 16 | 基于性能测试优化 |
| `RENDERER_OVERLOAD_STRATEGY` | - | queue | 默认使用排队策略 |

#### 环境变量
新增以下环境变量:
- `RENDERER_QUEUE_SIZE`: 队列大小（默认8）
- `RENDERER_QUEUE_TIMEOUT`: 排队超时时间（默认5s）
- `RENDERER_OVERLOAD_STRATEGY`: 过载策略（默认queue）

### 性能数据

基于完整性能测试的推荐配置:

| 并发数 | QPS | 失败率 | 平均响应时间 | 状态 |
|--------|-----|--------|--------------|------|
| 2 | 1,945 | 0% | 1.0ms | ✅ 稳定 |
| 4 | 4,177 | 0% | 1.0ms | ✅ 良好 |
| **16** | **5,084** | **0.3%** | **3.1ms** | **✅ 最佳** |
| 32 | 4,709 | 1.2% | 6.8ms | ⚠️ 开始下降 |
| 128+ | 6,000+ | >33% | >18ms | ❌ 不可用 |

### 生产部署建议

#### 中等并发场景
```bash
MAX_CONCURRENT=16
RENDERER_OVERLOAD_STRATEGY=queue
RENDERER_QUEUE_SIZE=16
RENDERER_QUEUE_TIMEOUT=3s
```

#### 高并发场景
```bash
MAX_CONCURRENT=32
RENDERER_OVERLOAD_STRATEGY=queue
RENDERER_QUEUE_SIZE=32
RENDERER_QUEUE_TIMEOUT=5s
```

#### 低延迟场景
```bash
MAX_CONCURRENT=8
RENDERER_OVERLOAD_STRATEGY=failfast
```

### 策略对比

| 特性 | FailFast | TimeoutQueue |
|------|----------|--------------|
| 并发满时行为 | 立即返回503 | 排队等待 |
| 用户体验 | 差 | 好 |
| 服务端控制 | 无 | 有（队列大小、超时） |
| 适用场景 | 低延迟要求 | 高峰流量 |

### 修复的问题
- 修复了高并发场景下503错误率过高的问题
- 优化了默认配置，提升了开箱即用的性能

### 测试验证
- 测试了218,600+次请求
- 涵盖10个并发级别（2-1000）
- 验证了新的队列策略有效性
- 测试了默认配置变更后的服务稳定性

### 破坏性变更
- **默认并发数**: 从4提升至16
  - 影响: 性能提升，但可能增加资源使用
  - 缓解: 可通过环境变量 `MAX_CONCURRENT=4` 还原

### 后续计划
- 短期: 调整监控和告警阈值
- 中期: 开发动态队列大小调整
- 长期: 支持多实例水平扩展

---

## [1.0.0] - 2026-01-09 (之前版本)

### 初始功能
- LaTeX数学公式渲染
- 本地缓存和OSS缓存支持
- Chrome浏览器复用
- 预热机制
- FailFast并发控制