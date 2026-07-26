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
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
)

// GetContributorsInfo 通过一次 MGET 批量获取多个 key 的贡献者信息，避免逐 key 多次串行 round-trip
func (c *CacheCommon) GetContributorsInfo(ctx context.Context, keys []string) (map[string][]*model.Contributor, error) {
	// MGET 不接受空参数列表，提前短路
	if len(keys) == 0 {
		return map[string][]*model.Contributor{}, nil
	}
	vals, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, errno.Errorf(errno.InternalDatabaseErrorCode, "dal.GetContributorsInfo: MGet contributor info failed: %v", err)
	}
	contributors := make(map[string][]*model.Contributor, len(keys))
	for i, val := range vals {
		// MGET 对不存在的 key 返回 nil，报错文案与原先 service 层逐 key 校验保持一致
		if val == nil {
			return nil, fmt.Errorf("service.GetContributorInfo: %s not exist", keys[i])
		}
		data, ok := val.(string)
		if !ok {
			return nil, errno.Errorf(errno.InternalJSONErrorCode, "dal.GetContributorsInfo: unexpected value type for key %s", keys[i])
		}
		var info []*model.Contributor
		if err = sonic.Unmarshal([]byte(data), &info); err != nil {
			return nil, errno.Errorf(errno.InternalJSONErrorCode, "dal.GetContributorsInfo: Unmarshal contributor info failed: %v", err)
		}
		contributors[keys[i]] = info
	}
	return contributors, nil
}
