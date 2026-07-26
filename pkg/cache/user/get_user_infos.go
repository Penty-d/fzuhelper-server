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

package user

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"

	"github.com/west2-online/fzuhelper-server/pkg/db/model"
)

// GetStuInfosCache 通过 MGET 一次性批量获取学生信息缓存, 返回按 key(学号)索引的命中结果,
// 未命中的 key 不会出现在返回的 map 中
func (c *CacheUser) GetStuInfosCache(ctx context.Context, keys []string) (map[string]*model.Student, error) {
	infos := make(map[string]*model.Student, len(keys))
	if len(keys) == 0 {
		return infos, nil
	}
	values, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("dal.GetStuInfosCache: MGet stu info cache failed: %w", err)
	}
	for i, value := range values {
		if value == nil { // 未命中缓存, 跳过
			continue
		}
		data, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("dal.GetStuInfosCache: unexpected value type %T for key %s", value, keys[i])
		}
		info := new(model.Student)
		if err = sonic.Unmarshal([]byte(data), info); err != nil {
			return nil, fmt.Errorf("dal.GetStuInfosCache: Unmarshal failed: %w", err)
		}
		infos[keys[i]] = info
	}
	return infos, nil
}
