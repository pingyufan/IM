package models

import (
	"IM/utils"
	"context"
	"time"
)

// 将用户在线信息存储到 Redis 中
func SetUserOnlineInfo(key string, val []byte, timeTTL time.Duration) {
	ctx := context.Background()
	utils.Red.Set(ctx, key, val, timeTTL)
}
