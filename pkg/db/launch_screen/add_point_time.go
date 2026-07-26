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

package launch_screen

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/west2-online/fzuhelper-server/pkg/constants"
)

func (c *DBLaunchScreen) AddPointTime(ctx context.Context, id int64) error {
	// 使用 UpdateColumn 原子自增, 避免并发下丢失计数或用旧值覆盖其他字段
	res := c.client.WithContext(ctx).Table(constants.LaunchScreenTableName).
		Where("id = ?", id).
		UpdateColumn("point_times", gorm.Expr("point_times + 1"))
	if res.Error != nil {
		return fmt.Errorf("dal.AddPointTime error: %w", res.Error)
	}
	// 保持原有语义: id 不存在时返回 record not found
	if res.RowsAffected == 0 {
		return fmt.Errorf("dal.AddPointTime error: %w", gorm.ErrRecordNotFound)
	}
	return nil
}
