/*
Copyright 2024 The west2-online Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package codec 提供 pkg/cache 各子包共用的 JSON 编解码辅助,
// 统一 "GET+Unmarshal" 与 "Marshal+SET" 的重复模板
package codec

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"

	"github.com/west2-online/fzuhelper-server/pkg/base/environment"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
)

// nullLiteral JSON 空值字面量; 反序列化到指针/切片时是 no-op, 会留下 nil 值
var nullLiteral = []byte("null")

// GetJSON 从 Redis 读取 key 并将 JSON 反序列化为 T。
// key 不存在(redis.Nil)时返回 found=false 且 err 为 nil, 调用方据此走回源逻辑,
// 避免 EXISTS 与 GET 之间 key 过期时把一次正常的未命中当成错误;
// 其他错误原样返回, 由调用方按各自的错误码/风格包装。
func GetJSON[T any](ctx context.Context, cli *redis.Client, key string) (val T, found bool, err error) {
	data, err := cli.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return val, false, nil
		}
		return val, false, err
	}
	// 缓存内容为 null 时反序列化不会报错但会留下 nil, 直接当作未命中回源,
	// 避免把 nil 指针当作有效命中交给调用方解引用
	if bytes.Equal(bytes.TrimSpace(data), nullLiteral) {
		return val, false, nil
	}
	if err = sonic.Unmarshal(data, &val); err != nil {
		var zero T
		return zero, false, err
	}
	return val, true, nil
}

// SetJSON 将 value 序列化为 JSON 并以 expire 为过期时间写入 Redis。
// 测试环境下直接跳过写入; 失败时以 opName 为前缀记录日志并返回原始错误,
// 是否进一步包装由调用方决定。
func SetJSON[T any](ctx context.Context, cli *redis.Client, key string, value T, expire time.Duration, opName string) error {
	if environment.IsTestEnvironment() {
		return nil
	}
	data, err := sonic.Marshal(value)
	if err != nil {
		logger.Errorf("%s: Marshal failed for key %s: %v", opName, key, err)
		return err
	}
	if err = cli.Set(ctx, key, data, expire).Err(); err != nil {
		logger.Errorf("%s: Set failed for key %s: %v", opName, key, err)
		return err
	}
	return nil
}
