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

package course

import (
	"context"
	"fmt"

	"github.com/west2-online/fzuhelper-server/pkg/cache/internal/codec"
	"github.com/west2-online/jwch"
	"github.com/west2-online/yjsy"
)

// GetCoursesCache 获取本科生课表缓存; key 不存在时返回 found=false 且不视为错误
func (c *CacheCourse) GetCoursesCache(ctx context.Context, key string) ([]*jwch.Course, bool, error) {
	course, found, err := codec.GetJSON[[]*jwch.Course](ctx, c.client, key)
	if err != nil {
		return nil, false, fmt.Errorf("dal.GetCoursesCache: cache failed: %w", err)
	}
	return course, found, nil
}

// GetCoursesCacheYjsy 获取研究生课表缓存; key 不存在时返回 found=false 且不视为错误
func (c *CacheCourse) GetCoursesCacheYjsy(ctx context.Context, key string) ([]*yjsy.Course, bool, error) {
	course, found, err := codec.GetJSON[[]*yjsy.Course](ctx, c.client, key)
	if err != nil {
		return nil, false, fmt.Errorf("dal.GetCoursesCacheYjsy: cache failed: %w", err)
	}
	return course, found, nil
}
