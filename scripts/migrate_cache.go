package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var (
	version = "1.0.0"
)

// Config 迁移配置
type Config struct {
	LocalDir     string
	OSSEndpoint  string
	OSSBucket    string
	OSSAccessKey string
	OSSSecretKey string
	OSSDomain    string
	DryRun       bool
	Parallel     int
	Help         bool
}

// MigrateResult 迁移结果
type MigrateResult struct {
	Key      string
	Size     int64
	Success  bool
	Error    error
}

func main() {
	startTime := time.Now()

	cfg := parseFlags()
	if cfg.Help {
		printUsage()
		return
	}

	// 验证配置
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("配置错误: %v", err)
	}

	// 获取本地缓存文件
	files, err := getLocalFiles(cfg.LocalDir)
	if err != nil {
		log.Fatalf("获取本地文件失败: %v", err)
	}

	if len(files) == 0 {
		fmt.Println("未找到需要迁移的缓存文件")
		return
	}

	// 计算总大小
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	// 打印预览信息
	fmt.Printf("=== 缓存迁移 %s ===\n", map[bool]string{true: "预览 (dry-run)"}[cfg.DryRun])
	fmt.Printf("本地缓存目录: %s\n", cfg.LocalDir)
	fmt.Printf("发现文件: %d 个\n", len(files))
	fmt.Printf("总大小: %.2f MB\n", float64(totalSize)/1024/1024)

	if cfg.DryRun {
		fmt.Println("\n将迁移以下文件到 OSS:")
		for _, f := range files {
			relPath, _ := filepath.Rel(cfg.LocalDir, f.Path)
			fmt.Printf("  - %s (%.2f KB)\n", relPath, float64(f.Size)/1024)
		}
		fmt.Println("\n提示: 使用 --parallel=N 增加并发数加速迁移")
		fmt.Println("执行迁移: 去掉 --dry-run 参数重新运行")
		return
	}

	// 执行迁移
	fmt.Println("\n开始迁移...")
	results := migrateFiles(cfg, files)

	// 输出统计报告
	printReport(results, time.Since(startTime))
}

// FileInfo 文件信息
type FileInfo struct {
	Path  string
	Key   string
	Size  int64
	ModTime time.Time
}

func parseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.LocalDir, "local", "./cache", "本地缓存目录")
	flag.StringVar(&cfg.OSSEndpoint, "oss-endpoint", "", "OSS endpoint")
	flag.StringVar(&cfg.OSSBucket, "oss-bucket", "", "OSS bucket 名称")
	flag.StringVar(&cfg.OSSAccessKey, "oss-access-key", "", "OSS access key")
	flag.StringVar(&cfg.OSSSecretKey, "oss-secret-key", "", "OSS secret key")
	flag.StringVar(&cfg.OSSDomain, "oss-domain", "", "OSS 自定义域名 (可选)")
	flag.IntVar(&cfg.Parallel, "parallel", 4, "并发上传数")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "预览模式，只显示不执行")
	flag.BoolVar(&cfg.Help, "help", false, "显示帮助信息")
	flag.BoolVar(&cfg.Help, "h", false, "显示帮助信息")

	flag.Usage = printUsage
	flag.Parse()

	// 从环境变量读取默认值
	if cfg.OSSEndpoint == "" {
		cfg.OSSEndpoint = os.Getenv("OSS_ENDPOINT")
	}
	if cfg.OSSBucket == "" {
		cfg.OSSBucket = os.Getenv("OSS_BUCKET")
	}
	if cfg.OSSAccessKey == "" {
		cfg.OSSAccessKey = os.Getenv("OSS_ACCESS_KEY")
	}
	if cfg.OSSSecretKey == "" {
		cfg.OSSSecretKey = os.Getenv("OSS_SECRET_KEY")
	}
	if cfg.OSSDomain == "" {
		cfg.OSSDomain = os.Getenv("OSS_DOMAIN")
	}

	return cfg
}

func validateConfig(cfg *Config) error {
	if cfg.OSSEndpoint == "" {
		return fmt.Errorf("OSS endpoint 不能为空，请使用 --oss-endpoint 或设置 OSS_ENDPOINT 环境变量")
	}
	if cfg.OSSBucket == "" {
		return fmt.Errorf("OSS bucket 不能为空，请使用 --oss-bucket 或设置 OSS_BUCKET 环境变量")
	}
	if cfg.OSSAccessKey == "" {
		return fmt.Errorf("OSS access key 不能为空，请使用 --oss-access-key 或设置 OSS_ACCESS_KEY 环境变量")
	}
	if cfg.OSSSecretKey == "" {
		return fmt.Errorf("OSS secret key 不能为空，请使用 --oss-secret-key 或设置 OSS_SECRET_KEY 环境变量")
	}
	return nil
}

func getLocalFiles(dir string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理 .png 文件
		if !info.IsDir() && filepath.Ext(path) == ".png" {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files = append(files, FileInfo{
				Path:   path,
				Key:    relPath,
				Size:   info.Size(),
				ModTime: info.ModTime(),
			})
		}
		return nil
	})

	return files, err
}

func migrateFiles(cfg *Config, files []FileInfo) []MigrateResult {
	// 创建 OSS 客户端
	client, err := oss.New(cfg.OSSEndpoint, cfg.OSSAccessKey, cfg.OSSSecretKey)
	if err != nil {
		log.Fatalf("创建 OSS 客户端失败: %v", err)
	}

	bucket, err := client.Bucket(cfg.OSSBucket)
	if err != nil {
		log.Fatalf("获取 bucket 失败: %v", err)
	}

	// 初始化结果数组
	results := make([]MigrateResult, len(files))

	// 使用 worker pool 进行并发上传
	semaphore := make(chan struct{}, cfg.Parallel)
	var wg sync.WaitGroup

	for i, file := range files {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(idx int, f FileInfo) {
			defer wg.Done()
			defer func() { <-semaphore }()

			result := MigrateResult{
				Key:  f.Key,
				Size: f.Size,
			}

			// 读取文件
			data, err := os.ReadFile(f.Path)
			if err != nil {
				result.Error = fmt.Errorf("读取文件失败: %w", err)
				results[idx] = result
				return
			}

			// 上传到 OSS
			err = bucket.PutObject(f.Key, bytes.NewReader(data), oss.ContentType("image/png"))
			if err != nil {
				result.Error = fmt.Errorf("上传失败: %w", err)
			} else {
				result.Success = true
			}

			results[idx] = result

			// 打印进度
			completed := countCompleted(results)
			if completed%10 == 0 || completed == len(files) {
				fmt.Printf("\r进度: %d/%d (%.1f%%)", completed, len(files), float64(completed)*100/float64(len(files)))
			}
		}(i, file)
	}

	wg.Wait()
	fmt.Println()

	return results
}

func countCompleted(results []MigrateResult) int {
	count := 0
	for _, r := range results {
		if r.Success || r.Error != nil {
			count++
		}
	}
	return count
}

func printReport(results []MigrateResult, duration time.Duration) {
	var success, failed int
	var totalSize int64

	for _, r := range results {
		if r.Success {
			success++
			totalSize += r.Size
		} else {
			failed++
		}
	}

	fmt.Printf("\n=== 缓存迁移完成 ===\n")
	fmt.Printf("发现文件: %d 个\n", len(results))
	fmt.Printf("成功上传: %d 个\n", success)
	fmt.Printf("失败: %d 个\n", failed)
	fmt.Printf("总大小: %.2f MB\n", float64(totalSize)/1024/1024)
	fmt.Printf("总计耗时: %v\n", duration.Round(time.Second))

	if success > 0 {
		speed := float64(success) / duration.Seconds()
		fmt.Printf("上传速度: %.1f files/s\n", speed)
	}

	if failed > 0 {
		fmt.Println("\n失败详情:")
		for _, r := range results {
			if !r.Success && r.Error != nil {
				fmt.Printf("  - %s: %v\n", r.Key, r.Error)
			}
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `缓存迁移工具 v%s

将本地缓存目录中的 PNG 文件迁移到阿里云 OSS

使用方式:
  %s [选项]

选项:
  --local         本地缓存目录 (默认: ./cache)
  --oss-endpoint  OSS endpoint (必填)
  --oss-bucket    OSS bucket 名称 (必填)
  --oss-access-key  OSS access key (必填)
  --oss-secret-key  OSS secret key (必填)
  --oss-domain    OSS 自定义域名 (可选)
  --parallel      并发上传数 (默认: %d)
  --dry-run       预览模式，只显示不执行
  --help, -h      显示帮助信息

环境变量:
  OSS_ENDPOINT, OSS_BUCKET, OSS_ACCESS_KEY, OSS_SECRET_KEY, OSS_DOMAIN

示例:
  # 预览迁移
  %s --dry-run

  # 执行迁移
  %s --oss-endpoint oss-cn-hangzhou.aliyuncs.com --oss-bucket my-bucket --oss-access-key xxx --oss-secret-key xxx

  # 使用环境变量
  export OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
  export OSS_BUCKET=my-bucket
  export OSS_ACCESS_KEY=xxx
  export OSS_SECRET_KEY=xxx
  %s

`, version, os.Args[0], runtime.NumCPU(), os.Args[0], os.Args[0], os.Args[0])
}
