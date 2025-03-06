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
type HashType string

const (
	MD5    HashType = "md5"
	SHA1   HashType = "sha1"
	SHA256 HashType = "sha256"
	SHA512 HashType = "sha512"
)

// GetHasher 根据哈希类型返回对应的哈希函数
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
func HashBytes(data []byte, hashType HashType) (string, error) {
	hasher := GetHasher(hashType)
	_, err := hasher.Write(data)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HashString 对字符串计算哈希值
func HashString(data string, hashType HashType) (string, error) {
	return HashBytes([]byte(data), hashType)
}

// HashFile 对文件计算哈希值
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
func HashReader(reader io.Reader, hashType HashType) (string, error) {
	hasher := GetHasher(hashType)
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// MD5Hash 返回数据的MD5哈希值(便捷方法)
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
func SHA256Hash(data []byte) string {
	hash, _ := HashBytes(data, SHA256)
	return hash
}

// SHA512Hash 返回数据的SHA512哈希值(便捷方法)
func SHA512Hash(data []byte) string {
	hash, _ := HashBytes(data, SHA512)
	return hash
}
