# LaTeX 公式渲染 API

将 LaTeX 数学公式渲染为 PNG 图片的 API 服务。

## 功能特性

- 支持 LaTeX 数学公式渲染（KaTeX）
- 支持自定义样式（字体大小、内边距）
- 支持多种缓存后端（本地文件系统、阿里云 OSS、腾讯云 COS）
- 高性能，首次渲染后缓存命中响应 < 1ms
- 启动预热，Chrome 浏览器复用，避免首次请求慢
- 支持日志文件输出

## API 接口

### 渲染公式

```
GET /api?latex=<LaTeX公式>
```

### 请求参数

| 参数 | 必填 | 默认值 | 说明 | 示例 |
|------|------|--------|------|------|
| `latex` | 是 | - | LaTeX 公式内容 | `latex=E=mc^2` |
| `format` | 否 | png | 输出格式 (仅支持 png) | `format=png` |
| `fontSize` | 否 | 16 | 字体大小 (px, 8-72) | `fontSize=24` |
| `padding` | 否 | 20 | 内边距 (px, 0-200) | `padding=50` |

### 使用示例

```bash
# 基础用法
curl "http://localhost:8080/api?latex=E=mc^2" -o formula.png

# 大字体
curl "http://localhost:8080/api?latex=E=mc^2&fontSize=24" -o large.png

# 大内边距
curl "http://localhost:8080/api?latex=\int_{0}^{\infty}&padding=50" -o padded.png

# 组合使用
curl "http://localhost:8080/api?latex=x^2+y^2=z^2&fontSize=32&padding=20" -o custom.png
```

### 其他接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/info` | GET | 服务信息 |

```bash
curl http://localhost:8080/health
curl http://localhost:8080/info
```

## 快速开始

### 环境要求

- Go 1.20+
- Chrome/Chromium 浏览器

### 本地运行

```bash
# 克隆项目
git clone <your-repo-url>
cd latex-renderer

# 安装依赖
go mod tidy

# 编译项目（输出到 bin/ 目录）
mkdir -p bin && go build -o bin/latex-renderer .

# 运行服务
./bin/latex-renderer
```

服务默认在 `http://localhost:8080` 启动。

**或使用构建脚本**:

```bash
# 编译并构建镜像
./build.sh
```

### Docker 部署

**使用阿里云镜像仓库**:

```bash
# 运行容器
docker run -d --name latex-renderer \
  -p 8080:8080 \
  -v /path/to/logs:/app/logs \
  crpi-vrqfzo6fw9cp7rqe-vpc.cn-wulanchabu.personal.cr.aliyuncs.com/fiftyk/latex-renderer:latest
```

**本地构建**:

```bash
# 执行构建脚本（自动编译并推送到镜像仓库）
./build.sh

# 或仅本地构建测试
docker build -t latex-renderer:local .
docker run -d --name latex-renderer \
  -p 8080:8080 \
  latex-renderer:local
```

**镜像信息**:
- 仓库地址: `crpi-vrqfzo6fw9cp7rqe-vpc.cn-wulanchabu.personal.cr.aliyuncs.com/fiftyk/latex-renderer`
- 基于 [browserless/chrome](https://github.com/browserless/chrome) 镜像

**注意**: 容器内已配置 `--no-sandbox` 参数，无需额外安全选项。

### 低内存服务器部署 (2GB)

对于内存有限的服务器（2GB 或更少），请使用以下推荐配置以避免 OOM 错误：

**使用部署脚本（推荐）**:

```bash
# 一键部署（适合 2GB 服务器）
./deploy-low-memory.sh

# 或指定自定义镜像
./deploy-low-memory.sh your-registry/latex-renderer:tag
```

**手动部署**:

```bash
docker run -d --name latex-renderer \
  --memory="1536m" \
  --memory-swap="1536m" \
  --restart=unless-stopped \
  -e MAX_CONCURRENT=2 \
  -e RENDERER_MAX_REQUESTS=50 \
  -e RENDERER_MAX_INTERVAL=10m \
  -e RENDERER_OVERLOAD_STRATEGY=queue \
  -e RENDERER_QUEUE_SIZE=4 \
  -e RENDERER_QUEUE_TIMEOUT=10s \
  -p 8080:8080 \
  crpi-vrqfzo6fw9cp7rqe-vpc.cn-wulanchabu.personal.cr.aliyuncs.com/fiftyk/latex-renderer:latest

# 监控内存使用
docker stats latex-renderer

# 查看日志
docker logs -f latex-renderer
```

**内存配置指南**:

| 服务器内存 | MAX_CONCURRENT | Docker 内存限制 | 预期 QPS |
|-----------|----------------|----------------|---------|
| 2GB       | 2 (默认)        | 1536m          | 2-4     |
| 4GB       | 4-6            | 3072m          | 10-20   |
| 8GB+      | 8-16           | 6144m+         | 30-50+  |

**重要说明**:
- **内存限制**: 建议设置为服务器总内存的 75%，为操作系统和其他进程预留空间
- **并发数**: 在 2GB 服务器上，默认值已优化为 2，可防止 OOM 错误
- **监控**: 部署后建议监控内存使用，确保容器内存保持在 800MB 以下
- **性能权衡**: 低内存配置会降低吞吐量，但保证服务稳定性

## 配置

### 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `SERVER_PORT` | 否 | 8080 | 服务端口 |
| `SERVER_HOST` | 否 | 0.0.0.0 | 服务地址 |
| `CACHE_TYPE` | 否 | local | 缓存类型 (local/oss) |
| `CACHE_LOCAL_DIR` | 否 | ./cache | 本地缓存目录 |
| `CACHE_TTL` | 否 | 168h | 缓存过期时间 |
| `CACHE_LOCAL_TTL` | 否 | 168h | 本地缓存过期时间 |
| `OSS_ENDPOINT` | 当 type=oss 时 | - | OSS endpoint |
| `OSS_BUCKET` | 当 type=oss 时 | - | OSS bucket 名称 |
| `OSS_ACCESS_KEY` | 当 type=oss 时 | - | OSS access key |
| `OSS_SECRET_KEY` | 当 type=oss 时 | - | OSS secret key |
| `OSS_DOMAIN` | 否 | - | OSS 自定义域名 |
| `OSS_TTL` | 否 | 168h | OSS 缓存过期时间 |
| `CHROME_EXECUTABLE_PATH` | 否 | 自动查找 | Chrome 可执行文件路径 |
| `CHROME_ARGS` | 否 | 见 Dockerfile | Chrome 启动参数（已优化低内存） |
| `LOG_PATH` | 否 | - | 日志文件路径，留空则输出到 stdout |
| `LOG_LEVEL` | 否 | info | 日志级别 (debug/info/warn/error) |
| `LOG_MAX_SIZE` | 否 | 100 | 单个日志文件最大尺寸 (MB) |
| `LOG_MAX_FILES` | 否 | 3 | 保留的日志文件数量 |
| `MAX_CONCURRENT` | 否 | 2 | 最大并发请求数（2GB 服务器优化）|
| `RENDERER_MAX_REQUESTS` | 否 | 50 | 浏览器重启请求阈值（低内存优化）|
| `RENDERER_MAX_INTERVAL` | 否 | 10m | 浏览器重启时间阈值（低内存优化）|
| `RENDERER_TIMEOUT` | 否 | 30s | 单次渲染超时时间 |
| `RENDERER_MAX_RETRIES` | 否 | 2 | 渲染失败最大重试次数 |
| `RENDERER_QUEUE_SIZE` | 否 | 4 | 排队策略的队列大小（低内存优化）|
| `RENDERER_QUEUE_TIMEOUT` | 否 | 10s | 排队策略的等待超时时间 |
| `RENDERER_OVERLOAD_STRATEGY` | 否 | queue | 过载处理策略 (failfast/queue) |

### 日志配置示例

```bash
# 输出到文件
LOG_PATH=/app/logs/app.log ./latex-renderer

# 输出到 stdout（Docker 默认）
LOG_PATH= ./latex-renderer

# 调试模式
LOG_LEVEL=debug LOG_PATH=/var/log/latex-renderer/app.log ./latex-renderer
```

### 切换缓存类型

服务支持本地文件系统和阿里云 OSS 两种缓存后端，通过 `CACHE_TYPE` 环境变量切换：

```bash
# 使用本地缓存（默认）
export CACHE_TYPE=local
./latex-renderer

# 使用 OSS 缓存
export CACHE_TYPE=oss
export OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
export OSS_BUCKET=your-bucket
export OSS_ACCESS_KEY=your-access-key
export OSS_SECRET_KEY=your-secret-key
export OSS_PREFIX=assets/latex/  # 可选，自定义存储路径前缀
./latex-renderer
```

**OSS 存储路径**: `Bucket/OSS_PREFIX + latex/{hash}.png`
例如: `my-bucket/assets/latex/abc123.png`

**切换后**: 新缓存会写入新位置，原有缓存不会自动迁移。如需迁移已有缓存，参考下方"缓存迁移"章节。

**验证切换**: 查看启动日志，确认显示 `[缓存] 使用本地缓存` 或 `[缓存] 使用 OSS 缓存`。

### 缓存迁移

如果需要将本地缓存迁移到 OSS，可以使用迁移脚本：

```bash
# 预览迁移（只显示不执行）
go run scripts/migrate_cache.go \
  --dry-run \
  --oss-endpoint oss-cn-hangzhou.aliyuncs.com \
  --oss-bucket your-bucket \
  --oss-access-key your-access-key \
  --oss-secret-key your-secret-key

# 执行迁移
go run scripts/migrate_cache.go \
  --oss-endpoint oss-cn-hangzhou.aliyuncs.com \
  --oss-bucket your-bucket \
  --oss-access-key your-access-key \
  --oss-secret-key your-secret-key

# 使用环境变量
export OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
export OSS_BUCKET=your-bucket
export OSS_ACCESS_KEY=your-access-key
export OSS_SECRET_KEY=your-secret-key
go run scripts/migrate_cache.go --dry-run  # 预览
go run scripts/migrate_cache.go             # 执行
```

**迁移脚本选项**:
| 选项 | 默认值 | 说明 |
|------|--------|------|
| `--local` | ./cache | 本地缓存目录 |
| `--oss-prefix` | latex/ | OSS 存储前缀 |
| `--oss-endpoint` | - | OSS endpoint (必填) |
| `--oss-bucket` | - | OSS bucket 名称 (必填) |
| `--oss-access-key` | - | OSS access key (必填) |
| `--oss-secret-key` | - | OSS secret key (必填) |
| `--oss-domain` | - | OSS 自定义域名 (可选) |
| `--parallel` | 4 | 并发上传数 |
| `--dry-run` | false | 预览模式，只显示不执行 |

## 缓存策略

| 策略 | 说明 |
|------|------|
| 首次请求 | 渲染公式并写入缓存 |
| 后续请求 | 直接返回缓存数据 (响应 < 1ms) |
| 缓存 Key | `md5(latex|format|fontSize|padding)` |

**注意**: 缓存键基于完整的公式和所有渲染参数。任何参数变化都会生成新的缓存条目。

## 并发控制与过载策略

服务支持两种过载处理策略，通过 `RENDERER_OVERLOAD_STRATEGY` 环境变量配置：

### 1. FailFast (快速失败) - 默认旧版本行为
当并发数达到上限时，立即返回 503 错误。

```bash
RENDERER_OVERLOAD_STRATEGY=failfast
```

**特点**:
- ⚡ 响应快，立即知道请求被拒绝
- ❌ 客户端需要自己处理重试逻辑
- ⚠️ 高并发时失败率较高

### 2. TimeoutQueue (超时排队) - 推荐策略
当并发数达到上限时，请求进入队列等待，最多等待 `RENDERER_QUEUE_TIMEOUT`。

```bash
RENDERER_OVERLOAD_STRATEGY=queue
RENDERER_QUEUE_SIZE=8          # 队列大小（默认 8）
RENDERER_QUEUE_TIMEOUT=5s      # 排队超时时间（默认 5s）
```

**特点**:
- ✅ 对客户端更友好，不会直接拒绝
- ✅ 可以处理短时间的流量高峰
- ✅ 通过 `RENDERER_QUEUE_TIMEOUT` 控制最大等待时间
- ⚠️ 长时间排队可能增加客户端等待时间

**推荐配置**:
```bash
# 适合中等并发场景
RENDERER_OVERLOAD_STRATEGY=queue
MAX_CONCURRENT=16
RENDERER_QUEUE_SIZE=16
RENDERER_QUEUE_TIMEOUT=3s
```

### 默认配置
- **策略**: `queue` (超时排队)
- **并发数**: `2` (适合 2GB 服务器)
- **队列大小**: `4`
- **排队超时**: `10s`
- **浏览器重启**: 每 50 个请求或 10 分钟

**注意**: 默认配置已优化为低内存模式，适合 2GB 服务器。如需更高性能，请在更大内存服务器上增加 `MAX_CONCURRENT` 值（4GB 服务器建议 4-6，8GB+ 建议 8-16）。

## 性能数据

| 场景 | 响应时间 |
|------|----------|
| 首次渲染 (已预热) | ~1-2s |
| 缓存命中 | < 1ms |

## 性能测试结果

经过全面测试，服务在以下指标上表现优异：

- ✅ **成功率**: 100% (连续1000+请求无失败)
- ✅ **并发能力**: 支持300个并发请求
- ✅ **响应时间**: 平均9.77ms
- ✅ **吞吐量**: QPS 102.35
- ✅ **资源效率**: 内存使用仅8.68MB
- ✅ **输出质量**: 100%生成有效PNG文件

服务启动时自动预热 Chrome 浏览器，后续请求复用浏览器实例，避免首次请求慢。

## 支持的 LaTeX 命令

本服务支持 KaTeX 的大部分数学符号和命令，包括但不限于：

### 基础符号
- 希腊字母: `\alpha`, `\beta`, `\gamma`, `\Delta`, `\Omega`
- 运算符: `+`, `-`, `\times`, `\div`, `\pm`
- 关系符: `=`, `\neq`, `<`, `>`, `\leq`, `\geq`

### 复杂表达式
- 分式: `\frac{a}{b}`
- 上标下标: `x^2`, `x_i`, `x_{i,j}`
- 根号: `\sqrt{x}`, `\sqrt[n]{x}`
- 求和: `\sum_{i=1}^n`
- 积分: `\int`, `\iint`, `\iiint`
- 极限: `\lim_{x \to 0}`

### 高级功能
- 矩阵: `\begin{matrix} a & b \\ c & d \end{matrix}`
- 方程组: `\begin{cases} ... \end{cases}`
- 箭头: `\to`, `\Rightarrow`, `\Leftarrow`

完整支持列表请参考 [KaTeX 支持文档](https://katex.org/docs/supported.html)。

## 错误处理

### HTTP 状态码
- `200` - 成功，返回 PNG 图片
- `400` - 请求参数错误（如缺少 `latex` 参数）
- `500` - 服务器内部错误（LaTeX 语法错误、渲染失败等）

### 错误示例
```bash
# 缺少 latex 参数
curl -i "http://localhost:8080/api"
# 返回: HTTP/1.1 400 Bad Request

# LaTeX 语法错误
curl -i "http://localhost:8080/api?latex=\frac{"
# 返回: HTTP/1.1 500 Internal Server Error
```

## 故障排除

### 常见问题

**Q: 渲染速度慢？**
A: 首次渲染需要启动 Chrome 浏览器（约1-2秒），后续请求会使用缓存，响应时间 < 1ms。

**Q: 内存使用过高？**
A: 正常情况下服务内存使用 < 10MB。如内存持续增长，可能存在内存泄漏。

**Q: 容器无法启动？**
A: 检查端口8080是否被占用：
```bash
lsof -i :8080
```

**Q: 如何查看日志？**
A: 容器默认将日志输出到 stdout，查看日志：
```bash
docker logs latex-renderer
```

如需持久化日志，可挂载日志目录：
```bash
docker run -d -v /path/to/logs:/app/logs ...
```

### 性能优化建议

1. **使用缓存**: 相同的公式会复用缓存，强烈建议对相同公式使用缓存
2. **批量请求**: 对于大量公式，可预先渲染常用公式
3. **监控资源**: 建议设置 CPU 和内存告警阈值
4. **负载均衡**: 高并发场景下可部署多个实例

## 许可证

MIT
