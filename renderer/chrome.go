package renderer

import (
	"os"
	"strings"
)

// FindChrome 查找 Chrome 可执行文件
func FindChrome() string {
	paths := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// parseChromeArgs 解析Chrome启动参数
func parseChromeArgs(args string) []string {
	if args == "" {
		return nil
	}

	var parsed []string
	for _, part := range strings.Fields(args) {
		if strings.HasPrefix(part, "--") {
			// 处理 --flag=value 格式
			if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
				parsed = append(parsed, "--"+kv[0][2:], kv[1])
			} else {
				// 处理 --flag 格式（无值参数如 --no-sandbox）
				parsed = append(parsed, "--"+part[2:])
			}
		}
	}
	return parsed
}
