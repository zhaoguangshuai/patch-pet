// Package utils 公共工具函数
package utils

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/patch-pet/patch-pet/pkg/types"
)

var (
	ulidRand *rand.Rand
	ulidMu   sync.Mutex
)

func init() {
	ulidRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// GenerateULID 生成带前缀的 ULID 格式 ID
// 格式：{prefix}_{26字符ULID}，例如 ep_01JZYQ4M8PABCDEFGHJKLMNP
// 保证全局唯一、时间有序、可按前缀识别实体类型
func GenerateULID(prefix types.IDPrefix) string {
	ulidMu.Lock()
	defer ulidMu.Unlock()

	now := time.Now()
	// 时间戳部分：10 字符 Crockford Base32（毫秒精度，可用约 89 年）
	ts := encodeTime(now.UnixMilli())
	// 随机部分：16 字符 Crockford Base32
	random := encodeRandom(ulidRand, 16)

	return fmt.Sprintf("%s_%s%s", prefix, ts, random)
}

const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// encodeTime 将毫秒时间戳编码为 Crockford Base32（10 字符）
func encodeTime(ms int64) string {
	var buf [10]byte
	for i := 9; i >= 0; i-- {
		buf[i] = crockfordBase32[ms&0x1f]
		ms >>= 5
	}
	return string(buf[:])
}

// encodeRandom 生成指定长度的随机 Crockford Base32 字符串
func encodeRandom(r *rand.Rand, length int) string {
	buf := make([]byte, length)
	for i := range buf {
		buf[i] = crockfordBase32[r.Intn(32)]
	}
	return string(buf)
}
