package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
)

func main() {
	testData := []byte("test data for hashing")

	// 计算MD5
	md5Hash := md5.New()
	md5Hash.Write(testData)
	fmt.Println("MD5:", hex.EncodeToString(md5Hash.Sum(nil)))

	// 计算SHA1
	sha1Hash := sha1.New()
	sha1Hash.Write(testData)
	fmt.Println("SHA1:", hex.EncodeToString(sha1Hash.Sum(nil)))

	// 计算SHA256
	sha256Hash := sha256.New()
	sha256Hash.Write(testData)
	fmt.Println("SHA256:", hex.EncodeToString(sha256Hash.Sum(nil)))

	// 计算SHA512
	sha512Hash := sha512.New()
	sha512Hash.Write(testData)
	fmt.Println("SHA512:", hex.EncodeToString(sha512Hash.Sum(nil)))
}
