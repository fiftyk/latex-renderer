# LaTeX 公式渲染 API

将 LaTeX 数学公式渲染为 PNG 图片的 API 服务。

## 功能特性

- 支持 LaTeX 数学公式渲染
- 支持自定义样式（颜色、背景、字体大小、内边距）
- 支持多种缓存后端（本地文件系统、阿里云 OSS、腾讯云 COS）
- 高性能 Chrome 复用

## API 接口

### 渲染公式

```
GET /api?latex=<LaTeX公式>
```

### 参数

| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| latex | 是 | - | LaTeX 公式内容 |
| format | 否 | png | 输出格式 (png, jpeg) |
| color | 否 | black | 字体颜色 (hex, 如 #ff0000) |
| background | 否 | transparent | 背景颜色 (hex, 如 #ffffff) |
| fontSize | 否 | 16 | 字体大小 (px) |
| padding | 否 | 20 | 内边距 (px) |

### 示例

```bash
# 基础用法
curl "http://localhost:8080/api?latex=E=mc^2" -o formula.png

# 红色字体
curl "http://localhost:8080/api?latex=E=mc^2&color=%23ff0000" -o red.png

# 白色背景 + 大字体
curl "http://localhost:8080/api?latex=x=\frac{-b\pm\sqrt{b^2-4ac}}{2a}&background=%23ffffff&fontSize=24" -o quadratic.png

# 复杂公式
curl "http://localhost:8080/api?latex=\int_{0}^{\infty} e^{-x^2} dx = \frac{\sqrt{\pi}}{2}" -o integral.png
```

### 其他接口

```bash
# 健康检查
curl http://localhost:8080/health

# 服务信息
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

| 变量 | 说明 | 默认值 |
|------|------|--------|
| SERVER_PORT | 服务端口 | 8080 |
| SERVER_HOST | 服务地址 | 0.0.0.0 |
| CACHE_TYPE | 缓存类型 (local/oss) | local |
| CACHE_LOCAL_DIR | 本地缓存目录 | ./cache |
| CACHE_TTL | 缓存过期时间 | 168h |
| OSS_ENDPOINT | OSS endpoint | - |
| OSS_BUCKET | OSS bucket | - |
| OSS_ACCESS_KEY | OSS access key | - |
| OSS_SECRET_KEY | OSS secret key | - |
| OSS_DOMAIN | OSS 自定义域名 | - |
| CHROME_EXECUTABLE_PATH | Chrome 路径 | 自动查找 |
| CHROME_ARGS | Chrome 额外参数 | - |

### OSS 部署示例

```bash
CACHE_TYPE=oss \
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com \
OSS_BUCKET=your-bucket \
OSS_ACCESS_KEY=your-access-key \
OSS_SECRET_KEY=your-secret-key \
./latex-renderer
```

## 缓存策略

- 首次请求：渲染公式并写入缓存
- 后续请求：直接返回缓存数据
- 缓存 Key：`md5(latex|format|color|background|fontSize|padding)`

## 性能优化

- Chrome 浏览器复用，降低启动开销
- KaTeX CDN 资源浏览器缓存
- 支持本地文件系统和 OSS 两种缓存后端

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
