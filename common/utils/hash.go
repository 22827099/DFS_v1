// hash.go 提供了多种哈希算法实现的工具包
// 支持MD5、SHA1、SHA256、SHA512四种哈希算法
// 可以对字节数组、字符串、文件和IO流进行哈希计算
package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// HashType 表示支持的哈希类型
// 用于指定使用哪种哈希算法进行计算
type HashType string

const (
	MD5    HashType = "md5"    // MD5哈希算法，速度快但安全性较低
	SHA1   HashType = "sha1"   // SHA1哈希算法，安全性中等
	SHA256 HashType = "sha256" // SHA256哈希算法，安全性较高，默认算法
	SHA512 HashType = "sha512" // SHA512哈希算法，安全性最高但计算较慢
)

// GetHasher 根据哈希类型返回对应的哈希函数
// 输入参数为哈希类型，返回对应的哈希实现
// 如果提供了不支持的类型，默认返回SHA256
func GetHasher(hashType HashType) hash.Hash {
	switch hashType {
	case MD5:
		return md5.New()
	case SHA1:
		return sha1.New()
	case SHA256:
		return sha256.New()
	case SHA512:
		return sha512.New()
	default:
		return sha256.New() // 默认使用SHA256
	}
}

// HashBytes 对字节数组计算哈希值
// 参数: data - 要计算哈希的字节数组，hashType - 使用的哈希算法
// 返回: 十六进制格式的哈希字符串和可能的错误
func HashBytes(data []byte, hashType HashType) (string, error) {
	hasher := GetHasher(hashType)
	_, err := hasher.Write(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HashString 对字符串计算哈希值
// 内部调用HashBytes，将字符串转换为字节数组
func HashString(data string, hashType HashType) (string, error) {
	return HashBytes([]byte(data), hashType)
}

// HashFile 对文件计算哈希值
// 参数: filePath - 文件路径，hashType - 使用的哈希算法
// 返回: 文件内容的哈希值和可能的错误（如文件不存在）
func HashFile(filePath string, hashType HashType) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := GetHasher(hashType)
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HashReader 对io.Reader计算哈希值
// 适用于需要对流数据计算哈希的场景
func HashReader(reader io.Reader, hashType HashType) (string, error) {
	hasher := GetHasher(hashType)
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// 以下是便捷方法，直接使用特定的哈希算法，忽略错误返回

// MD5Hash 返回数据的MD5哈希值(便捷方法)
// 常用于生成文件或数据的校验和
func MD5Hash(data []byte) string {
	hash, _ := HashBytes(data, MD5)
	return hash
}

// SHA1Hash 返回数据的SHA1哈希值(便捷方法)
func SHA1Hash(data []byte) string {
	hash, _ := HashBytes(data, SHA1)
	return hash
}

// SHA256Hash 返回数据的SHA256哈希值(便捷方法)
// 推荐用于需要安全性的场景
func SHA256Hash(data []byte) string {
	hash, _ := HashBytes(data, SHA256)
	return hash
}

// SHA512Hash 返回数据的SHA512哈希值(便捷方法)
// 提供最高级别的安全性
func SHA512Hash(data []byte) string {
	hash, _ := HashBytes(data, SHA512)
	return hash
}
