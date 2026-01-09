# 缓存系统修复和改进报告

## 问题诊断

经过代码审查，发现以下问题可能导致缓存不生效：

1. **缺少调试日志**：缓存操作过程中没有足够的日志输出，无法跟踪缓存写入和读取的状态
2. **缓存写入错误处理不足**：虽然有错误检查，但没有详细日志记录失败原因
3. **缺少标准HTTP缓存头**：没有使用ETag、Cache-Control等标准缓存机制

## 修复内容

### 1. 添加详细调试日志

#### API Handler (`api/handler.go`)
- 添加缓存操作的完整日志链：尝试获取 → 命中/未命中 → 写入/读取结果
- 记录缓存key、大小、ETag等详细信息
- 区分不同状态：hit、miss、read-error、write-error、generated

#### 本地缓存 (`cache/local.go`)
- 初始化时验证目录权限
- Get方法：记录文件存在性、大小、修改时间、TTL检查
- Set方法：记录写入过程、验证写入结果
- Delete/Exists方法：添加相应日志

#### 主程序 (`main.go`)
- 添加缓存初始化进度日志
- 显示缓存类型、目录、TTL等配置信息

### 2. 实现标准HTTP缓存机制

#### ETag支持
- 基于内容MD5生成ETag：`"md5-hash"`
- 检查客户端`If-None-Match`头

#### 304 Not Modified
- 当ETag匹配时返回304状态码，不返回实体内容
- 节省带宽，提高性能

#### 标准响应头
- `ETag`: 实体标签，用于缓存验证
- `Cache-Control`: `public, max-age=31536000` (1年)
- `Last-Modified`: 最后修改时间
- `X-Cache-Status`: 自定义缓存状态
  - `hit`: 缓存命中
  - `miss`: 缓存未命中（刚生成）
  - `generated`: 新生成的内容
  - `read-error`: 读取缓存失败
  - `write-error`: 写入缓存失败

### 3. 响应头设置优化

#### 缓存命中流程
1. 检查缓存是否存在
2. 生成ETag
3. 检查If-None-Match头
4. 如果匹配：返回304
5. 如果不匹配：返回完整内容 + ETag

#### 缓存未命中流程
1. 渲染图片
2. 写入缓存
3. 返回内容 + ETag

## 使用方法

### 检查缓存状态

使用curl测试并查看响应头：

```bash
# 第一次请求（缓存未命中）
curl -i "http://localhost:8080/api?latex=E=mc^2"

# 第二次请求（缓存命中）
curl -i "http://localhost:8080/api?latex=E=mc^2"

# 检查ETag支持
curl -i -H "If-None-Match: \"etag-value\"" "http://localhost:8080/api?latex=E=mc^2"
```

### 查看日志

日志会显示详细的缓存操作信息：

```
[缓存] 尝试获取缓存: key=latex/abc123.png
[缓存-本地] 获取缓存: key=latex/abc123.png, path=./cache/latex/abc123.png
[缓存-本地] 缓存不存在: key=latex/abc123.png
[缓存] 缓存未命中，开始渲染: key=latex/abc123.png
[渲染] 渲染成功: key=latex/abc123.png, size=1234 bytes
[缓存] 写入缓存: key=latex/abc123.png
[缓存-本地] 设置缓存: key=latex/abc123.png, path=./cache/latex/abc123.png, size=1234 bytes
[缓存-本地] 写入缓存成功: key=latex/abc123.png
[缓存] 写入缓存成功: key=latex/abc123.png
```

### 检查缓存文件

```bash
# 查看缓存目录
ls -la ./cache/latex/

# 查看特定缓存文件
ls -lh ./cache/latex/<hash>.png
```

## 潜在问题排查

如果缓存仍然不生效，请检查：

1. **目录权限**：确保进程有权限在缓存目录读写
   ```bash
   ls -la ./cache/
   ```

2. **磁盘空间**：确保有足够空间存储缓存文件
   ```bash
   df -h
   ```

3. **日志输出**：查看服务器日志中的错误信息
   ```bash
   tail -f /path/to/log/file
   ```

4. **缓存目录**：检查默认缓存目录 `./cache` 是否存在
   ```bash
   pwd  # 查看当前工作目录
   ls -la | grep cache
   ```

5. **环境变量**：检查 `CACHE_LOCAL_DIR` 环境变量
   ```bash
   echo $CACHE_LOCAL_DIR
   ```

## 性能优化建议

1. **使用ETag**：客户端支持If-None-Match时，命中缓存返回304，节省带宽
2. **调整TTL**：根据需要调整缓存过期时间
   ```bash
   export CACHE_TTL=168h  # 7天
   ```
3. **监控缓存命中率**：通过X-Cache-Status响应头统计

## 测试用例

详见 `test_cache.sh` 脚本

```bash
chmod +x test_cache.sh
./test_cache.sh
```

## 相关文件

- `api/handler.go`: HTTP处理器，缓存逻辑
- `cache/local.go`: 本地文件系统缓存实现
- `main.go`: 主程序，缓存初始化
- `config/config.go`: 缓存配置
