package utils_test

import (
    "bytes"
    "crypto/md5"
    "crypto/sha1"
    "crypto/sha256"
    "crypto/sha512"
    "encoding/hex"
    "errors"
	"hash"
    "os"
    "testing"

    "github.com/22827099/DFS_v1/common/utils"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGetHasher(t *testing.T) {
    tests := []struct {
        name     string
        hashType utils.HashType
        expected hash.Hash
    }{
        {"MD5", utils.MD5, md5.New()},
        {"SHA1", utils.SHA1, sha1.New()},
        {"SHA256", utils.SHA256, sha256.New()},
        {"SHA512", utils.SHA512, sha512.New()},
        {"默认值", "unknown", sha256.New()},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            hasher := utils.GetHasher(tt.hashType)
            // 比较哈希函数的类型
            assert.Equal(t, tt.expected.Size(), hasher.Size())
            assert.Equal(t, tt.expected.BlockSize(), hasher.BlockSize())
        })
    }
}

func TestHashBytes(t *testing.T) {
    testData := []byte("test data for hashing")
    
    tests := []struct {
        name     string
        hashType utils.HashType
        expected string
    }{
        {"MD5", utils.MD5, "e48e13207341b6bffb7fb1622282247b"},
        {"SHA1", utils.SHA1, "d85f4d6f4a5eaed71c0eadc5ca658d5f3e61eced"},
        {"SHA256", utils.SHA256, "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"},
        {"SHA512", utils.SHA512, "e39973c1b2411a61436c3f36c3654ebd0e27b19b322f8710a1bb0cf6a79539714cee1a490c1be1f38362aac19e44d93f5e88bde54f41be0d053978c3d6afa34a"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := utils.HashBytes(testData, tt.hashType)
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
            
            // 同时测试 HashString
            strResult, err := utils.HashString(string(testData), tt.hashType)
            require.NoError(t, err)
            assert.Equal(t, tt.expected, strResult)
        })
    }
}

func TestHashFile(t *testing.T) {
    // 创建临时测试文件
    content := []byte("test file content")
    tmpfile, err := os.CreateTemp("", "hashtest")
    require.NoError(t, err)
    defer os.Remove(tmpfile.Name())
    
    _, err = tmpfile.Write(content)
    require.NoError(t, err)
    require.NoError(t, tmpfile.Close())
    
    // 预先计算期望的哈希值
    md5Hash := md5.New()
    md5Hash.Write(content)
    expectedMD5 := hex.EncodeToString(md5Hash.Sum(nil))
    
    sha1Hash := sha1.New()
    sha1Hash.Write(content)
    expectedSHA1 := hex.EncodeToString(sha1Hash.Sum(nil))
    
    // 测试HashFile
    result, err := utils.HashFile(tmpfile.Name(), utils.MD5)
    require.NoError(t, err)
    assert.Equal(t, expectedMD5, result)
    
    result, err = utils.HashFile(tmpfile.Name(), utils.SHA1)
    require.NoError(t, err)
    assert.Equal(t, expectedSHA1, result)
    
    // 测试文件不存在的情况
    _, err = utils.HashFile("non_existent_file.txt", utils.MD5)
    assert.Error(t, err)
}

func TestHashReader(t *testing.T) {
    content := "test reader content"
    reader := bytes.NewBufferString(content)
    
    // 预先计算期望的SHA256哈希
    sha256Hash := sha256.New()
    sha256Hash.Write([]byte(content))
    expected := hex.EncodeToString(sha256Hash.Sum(nil))
    
    // 测试HashReader
    result, err := utils.HashReader(reader, utils.SHA256)
    require.NoError(t, err)
    assert.Equal(t, expected, result)
    
    // 测试读取错误的情况
    errReader := &ErrorReader{Err: errors.New("read error")}
    _, err = utils.HashReader(errReader, utils.SHA256)
    assert.Error(t, err)
}

// ErrorReader 用于测试读取错误
type ErrorReader struct {
    Err error
}

func (r *ErrorReader) Read(p []byte) (n int, err error) {
    return 0, r.Err
}

func TestConvenienceFunctions(t *testing.T) {
    data := []byte("test data")
    
    t.Run("MD5Hash", func(t *testing.T) {
        expected, _ := utils.HashBytes(data, utils.MD5)
        assert.Equal(t, expected, utils.MD5Hash(data))
    })
    
    t.Run("SHA1Hash", func(t *testing.T) {
        expected, _ := utils.HashBytes(data, utils.SHA1)
        assert.Equal(t, expected, utils.SHA1Hash(data))
    })
    
    t.Run("SHA256Hash", func(t *testing.T) {
        expected, _ := utils.HashBytes(data, utils.SHA256)
        assert.Equal(t, expected, utils.SHA256Hash(data))
    })
    
    t.Run("SHA512Hash", func(t *testing.T) {
        expected, _ := utils.HashBytes(data, utils.SHA512)
        assert.Equal(t, expected, utils.SHA512Hash(data))
    })
}

func TestEmptyInputs(t *testing.T) {
    // 测试空数据
    emptyData := []byte{}
    
    // 对空数据的每种哈希类型进行测试
    hashTypes := []utils.HashType{utils.MD5, utils.SHA1, utils.SHA256, utils.SHA512}
    
    for _, hashType := range hashTypes {
        t.Run(string(hashType)+"_empty", func(t *testing.T) {
            result, err := utils.HashBytes(emptyData, hashType)
            require.NoError(t, err)
            assert.NotEmpty(t, result, "空数据的哈希值不应为空")
            
            // 空字符串
            strResult, err := utils.HashString("", hashType)
            require.NoError(t, err)
            assert.Equal(t, result, strResult, "空数组和空字符串的哈希值应相同")
        })
    }
    
    // 测试空Reader
    emptyReader := bytes.NewReader(emptyData)
    result, err := utils.HashReader(emptyReader, utils.SHA256)
    require.NoError(t, err)
    assert.NotEmpty(t, result, "空Reader的哈希值不应为空")
}

func BenchmarkHash(b *testing.B) {
    data := []byte("benchmark test data")
    largeData := make([]byte, 1024*1024) // 1MB 数据
    for i := range largeData {
        largeData[i] = byte(i % 256)
    }
    
    b.Run("MD5-Small", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            utils.MD5Hash(data)
        }
    })
    
    b.Run("SHA1-Small", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            utils.SHA1Hash(data)
        }
    })
    
    b.Run("SHA256-Small", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            utils.SHA256Hash(data)
        }
    })
    
    b.Run("SHA512-Small", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            utils.SHA512Hash(data)
        }
    })
    
    // 大数据量测试
    b.Run("MD5-Large", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            utils.MD5Hash(largeData)
        }
    })
    
    b.Run("SHA1-Large", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            utils.SHA1Hash(largeData)
        }
    })
    
    b.Run("SHA256-Large", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            utils.SHA256Hash(largeData)
        }
    })
    
    b.Run("SHA512-Large", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            utils.SHA512Hash(largeData)
        }
    })
}