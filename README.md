# LaTeX 公式渲染 API

将 LaTeX 数学公式渲染为 PNG 图片的 API 服务。

## 功能特性

- 支持 LaTeX 数学公式渲染（KaTeX）
- 支持自定义样式（颜色、背景、字体大小、内边距）
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
  crpi-vrqfzo6fw9cp7rqe.cn-wulanchabu.personal.cr.aliyuncs.com/fiftyk/latex-renderer:latest
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
- 仓库地址: `crpi-vrqfzo6fw9cp7rqe.cn-wulanchabu.personal.cr.aliyuncs.com/fiftyk/latex-renderer`
- 基于 [browserless/chrome](https://github.com/browserless/chrome) 镜像

**注意**: 容器内已配置 `--no-sandbox` 参数，无需额外安全选项。

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
| `CHROME_ARGS` | 否 | `--no-sandbox --disable-setuid-sandbox --disable-dev-shm-usage` | Chrome 启动参数 |
| `LOG_PATH` | 否 | - | 日志文件路径，留空则输出到 stdout |
| `LOG_LEVEL` | 否 | info | 日志级别 (debug/info/warn/error) |
| `LOG_MAX_SIZE` | 否 | 100 | 单个日志文件最大尺寸 (MB) |
| `LOG_MAX_FILES` | 否 | 3 | 保留的日志文件数量 |

### 日志配置示例

```bash
# 输出到文件
LOG_PATH=/app/logs/app.log ./latex-renderer

# 输出到 stdout（Docker 默认）
LOG_PATH= ./latex-renderer

# 调试模式
LOG_LEVEL=debug LOG_PATH=/var/log/latex-renderer/app.log ./latex-renderer
```

### OSS 部署示例

```bash
# 使用阿里云 OSS
CACHE_TYPE=oss \
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com \
OSS_BUCKET=your-bucket \
OSS_ACCESS_KEY=your-access-key \
OSS_SECRET_KEY=your-secret-key \
./latex-renderer
```

## 缓存策略

| 策略 | 说明 |
|------|------|
| 首次请求 | 渲染公式并写入缓存 |
| 后续请求 | 直接返回缓存数据 (响应 < 1ms) |
| 缓存 Key | `md5(latex|format|fontSize|padding)` |

## 性能数据

| 场景 | 响应时间 |
|------|----------|
| 首次渲染 (已预热) | ~1-2s |
| 缓存命中 | < 1ms |

服务启动时自动预热 Chrome 浏览器，后续请求复用浏览器实例，避免首次请求慢。

## 许可证

MIT
