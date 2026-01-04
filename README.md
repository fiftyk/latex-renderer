# LaTeX 公式渲染 API

将 LaTeX 数学公式渲染为 PNG 图片的 API 服务。

## 功能特性

- 支持 LaTeX 数学公式渲染
- 支持自定义样式（颜色、背景、字体大小、内边距）
- 支持多种缓存后端（本地文件系统、阿里云 OSS、腾讯云 COS）
- 高性能，首次渲染后缓存命中响应 < 1ms

## API 接口

### 渲染公式

```
GET /api?latex=<LaTeX公式>
```

### 请求参数

| 参数 | 必填 | 默认值 | 说明 | 示例 |
|------|------|--------|------|------|
| `latex` | 是 | - | LaTeX 公式内容 | `latex=E=mc^2` |
| `color` | 否 | black | 字体颜色 (hex) | `color=%23ff0000` |
| `background` | 否 | transparent | 背景颜色 (hex) | `background=%23ffffff` |
| `fontSize` | 否 | 16 | 字体大小 (px, 8-72) | `fontSize=24` |
| `padding` | 否 | 20 | 内边距 (px, 0-200) | `padding=50` |

### 使用示例

```bash
# 基础用法
curl "http://localhost:8080/api?latex=E=mc^2" -o formula.png

# 红色字体
curl "http://localhost:8080/api?latex=E=mc^2&color=%23ff0000" -o red.png

# 蓝色字体 + 24px
curl "http://localhost:8080/api?latex=E=mc^2&color=%230000ff&fontSize=24" -o blue.png

# 白色背景
curl "http://localhost:8080/api?latex=\sum_{i=1}^n&background=%23ffffff" -o white-bg.png

# 大内边距
curl "http://localhost:8080/api?latex=\int_{0}^{\infty}&padding=50" -o padded.png

# 完整定制 (绿色 + 白色背景 + 32px + 30px 内边距)
curl "http://localhost:8080/api?latex=f(x)=\int_{-\infty}^{\infty}\hat{f}(\xi)e^{2\pi i\xi x}d\xi&color=%2300ff00&background=%23ffffff&fontSize=32&padding=30" -o full-custom.png
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

### 安装运行

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

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN go build -o latex-renderer .

FROM chrome:latest
COPY --from=builder /app/latex-renderer /usr/local/bin/
CMD ["latex-renderer"]
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
| 缓存 Key | `md5(latex|format|color|background|fontSize|padding)` |

## 性能数据

| 场景 | 响应时间 |
|------|----------|
| 首次渲染 (冷启动) | ~5s |
| 缓存命中 | < 1ms |

## 项目结构

```
latex-renderer/
├── main.go              # 主入口
├── api/
│   ├── handler.go       # HTTP 处理器
│   └── routes.go        # 路由定义
├── cache/
│   ├── cache.go         # 缓存接口
│   ├── local.go         # 本地文件系统缓存
│   └── oss.go           # OSS 缓存
├── config/
│   └── config.go        # 配置管理
├── renderer/
│   └── chrome.go        # Chrome 渲染器
└── README.md
```

## 许可证

MIT
