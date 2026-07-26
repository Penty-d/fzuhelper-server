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

	"github.com/west2-online/fzuhelper-server/pkg/cache/internal/codec"
	"github.com/west2-online/fzuhelper-server/pkg/db/model"
)

// GetStuInfoCache 获取学生信息缓存; key 不存在时返回 found=false 且不视为错误
func (c *CacheUser) GetStuInfoCache(ctx context.Context, key string) (*model.Student, bool, error) {
	info, found, err := codec.GetJSON[*model.Student](ctx, c.client, key)
	if err != nil {
		return nil, false, fmt.Errorf("dal.GetStuInfoCache: GetStuInfo cache failed: %w", err)
	}
	return info, found, nil
}
