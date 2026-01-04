package latex

import "os"

// WriteFile 将数据写入文件
func WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// ReadFile 读取文件
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
