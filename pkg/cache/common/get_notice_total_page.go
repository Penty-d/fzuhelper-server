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

package common

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"

	"github.com/west2-online/fzuhelper-server/pkg/errno"
)

// GetNoticeTotalPageCache 获取缓存的教务处通知总页数，未命中时 ok 为 false 且不返回错误
func (c *CacheCommon) GetNoticeTotalPageCache(ctx context.Context) (total int, ok bool, err error) {
	total, err = c.client.Get(ctx, noticeTotalPageKey).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, false, nil
		}
		return 0, false, errno.Errorf(errno.InternalDatabaseErrorCode, "dal.GetNoticeTotalPageCache: get notice total page failed: %v", err)
	}
	return total, true, nil
}
