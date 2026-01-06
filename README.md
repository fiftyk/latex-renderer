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
| `fontSize` | 否 | 16 | 字体大小 (px, 8-72) | `fontSize=24` |
| `padding` | 否 | 20 | 内边距 (px, 0-200) | `padding=50` |
| `color` | 否 | black | 字体颜色 | `color=red` |
| `background` | 否 | transparent | 背景颜色 | `background=#ffffff` |

### 使用示例

```bash
# 基础用法
curl "http://localhost:8080/api?latex=E=mc^2" -o formula.png

# 大字体
curl "http://localhost:8080/api?latex=E=mc^2&fontSize=24" -o large.png

# 大内边距
curl "http://localhost:8080/api?latex=\int_{0}^{\infty}&padding=50" -o padded.png

# 自定义颜色
curl "http://localhost:8080/api?latex=x^2&color=blue&background=#f0f0f0" -o styled.png
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

# 运行服务
./latex-renderer
```

服务默认在 `http://localhost:8080` 启动。

### Docker 部署

```bash
# 构建镜像
docker build -t latex-renderer .

# 运行容器
docker run -d --name latex-renderer \
  -p 8080:8080 \
  --security-opt seccomp=unconfined \
  -v /path/to/logs:/app/logs \
  latex-renderer:latest
```

**注意**: `--security-opt seccomp=unconfined` 是因为容器内 Chrome 需要沙箱配置。

### Docker Hub

> 需要先登录 Docker Hub 并推送镜像

```bash
# 登录 Docker Hub
docker login

# 推送镜像（需要对应权限）
docker tag latex-renderer:latest your-username/latex-renderer:latest
docker push your-username/latex-renderer:latest

# 拉取并运行
docker pull your-username/latex-renderer:latest
docker run -d --name latex-renderer \
  -p 8080:8080 \
  --security-opt seccomp=unconfined \
  -v /path/to/logs:/app/logs \
  your-username/latex-renderer:latest
```

## 配置

### 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `SERVER_PORT` | 否 | 8080 | 服务端口 |
| `SERVER_HOST` | 否 | 0.0.0.0 | 服务地址 |
| `CACHE_TYPE` | 否 | local | 缓存类型 (local/oss) |
| `CACHE_LOCAL_DIR` | 否 | ./cache | 本地缓存目录 |
| `CACHE_TTL` | 否 | 168h | 缓存过期时间 |
| `OSS_ENDPOINT` | 当 type=oss 时 | - | OSS endpoint |
| `OSS_BUCKET` | 当 type=oss 时 | - | OSS bucket 名称 |
| `OSS_ACCESS_KEY` | 当 type=oss 时 | - | OSS access key |
| `OSS_SECRET_KEY` | 当 type=oss 时 | - | OSS secret key |
| `OSS_DOMAIN` | 否 | - | OSS 自定义域名 |
| `CHROME_EXECUTABLE_PATH` | 否 | 自动查找 | Chrome 可执行文件路径 |
| `CHROME_ARGS` | 否 | - | Chrome 额外启动参数 |
| `LOG_PATH` | 否 | - | 日志文件路径，留空则输出到 stdout |
| `LOG_LEVEL` | 否 | info | 日志级别 (debug/info/warn/error) |

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
| 缓存 Key | `md5(latex|format|scale|fontSize|padding)` |

## 性能数据

| 场景 | 响应时间 |
|------|----------|
| 首次渲染 (已预热) | ~1-2s |
| 缓存命中 | < 1ms |

服务启动时自动预热 Chrome 浏览器，后续请求复用浏览器实例，避免首次请求慢。

## 许可证

MIT
