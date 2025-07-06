package db

import (
	"context"
	"dex-ingest-sol/internal/pkg/logger"
	"errors"
	"fmt"
	"time"
)

// RetryWithBackoff retries the given operation with exponential backoff.
// Delay pattern: 100ms, 400ms, 1s, 2s, 5s x N
func RetryWithBackoff(ctx context.Context, maxRetries int, op func() error) error {
	delays := []time.Duration{
		100 * time.Millisecond,
		400 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
	}

	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err() // 主动取消，不再重试
		default:
		}

		err = op()
		if err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}

		if !shouldRetry(err) {
			return err
		}

		// 计算下一次延迟
		delay := delays[min(attempt, len(delays)-1)]

		// 日志放在此处，避免第一次 op() 之前就打印
		logger.Warnf("retrying after error (attempt=%d, delay=%s): %v", attempt+1, delay, err)

		time.Sleep(delay)
	}

	return fmt.Errorf("retry failed after %d attempts: %w", maxRetries, err)
}

// shouldRetry 判断错误是否是临时性网络错误，适合重试。
// 不建议对业务逻辑错误（如语法错误、权限问题）使用该函数。
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	return true
	//errMsg := strings.ToLower(err.Error())
	//
	//// 聚合常见重试关键词
	//timeoutKeywords := []string{
	//	"timeout",
	//	"time out",
	//	"connection refused",
	//	"connection reset",
	//	"broken pipe",
	//	"driver: bad connection",
	//	"i/o timeout",
	//	"connection aborted",
	//	"network is unreachable",
	//	"no such host",
	//	"tls: handshake failure",
	//	"server has gone away",
	//}
	//
	//for _, keyword := range timeoutKeywords {
	//	if strings.Contains(errMsg, keyword) {
	//		return true
	//	}
	//}
	//return false
}
