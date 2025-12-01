package utils

import (
	"bytes"
	"io"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// convertGBKToUTF8 将GBK编码的字节切片转换为UTF-8字符串
// 该函数用于处理Windows命令行工具（如netsh）的中文输出
func convertGBKToUTF8(gbkData []byte) string {
	reader := transform.NewReader(bytes.NewReader(gbkData), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return string(gbkData)
	}
	return string(d)
}
