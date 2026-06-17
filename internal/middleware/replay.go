package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/patch-pet/patch-pet/pkg/constants"
)

// replaySecretKey 防重放签名密钥（从环境变量读取，禁止硬编码）
var replaySecretKey string

func init() {
	replaySecretKey = os.Getenv("REPLAY_SECRET_KEY")
}

// ReplayMiddleware 防重放中间件
// 所有对外接口必须携带 X-Timestamp + X-Signature
// 时间戳偏差 > 5 分钟直接拒绝
// 签名算法：HMAC-SHA256(requestBody + timestamp, secret)
func ReplayMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 健康检查接口跳过
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// 读取时间戳
		timestampStr := r.Header.Get("X-Timestamp")
		if timestampStr == "" {
			writeErrorResponse(w, http.StatusBadRequest, constants.CodeAuthReplayDetected, "缺少 X-Timestamp Header")
			return
		}

		// 解析时间戳（Unix 秒）
		reqTime, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, constants.CodeAuthReplayDetected, "X-Timestamp 格式无效")
			return
		}

		// 校验时间戳偏差（±5 分钟）
		now := time.Now().Unix()
		diff := now - reqTime
		if diff < 0 {
			diff = -diff
		}
		if diff > int64(constants.ReplayTimestampToleranceSec) {
			writeErrorResponse(w, http.StatusBadRequest, constants.CodeAuthReplayDetected, "请求时间戳已过期")
			return
		}

		// 读取签名
		signature := r.Header.Get("X-Signature")
		if signature == "" {
			writeErrorResponse(w, http.StatusBadRequest, constants.CodeAuthReplayDetected, "缺少 X-Signature Header")
			return
		}

		// 读取请求体（需要回填给下游）
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				writeErrorResponse(w, http.StatusBadRequest, constants.CodeAuthReplayDetected, "读取请求体失败")
				return
			}
			r.Body.Close()
			// 回填请求体，供下游 handler 读取
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 计算期望签名
		expected := computeSignature(bodyBytes, timestampStr)

		// 签名校验（常量时间比较，防时序攻击）
		if !hmac.Equal([]byte(signature), []byte(expected)) {
			writeErrorResponse(w, http.StatusBadRequest, constants.CodeAuthReplayDetected, "签名校验失败")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// computeSignature 计算 HMAC-SHA256 签名
// 签名内容：requestBody + timestamp
func computeSignature(body []byte, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(replaySecretKey))
	mac.Write(body)
	mac.Write([]byte(timestamp))
	return hex.EncodeToString(mac.Sum(nil))
}
