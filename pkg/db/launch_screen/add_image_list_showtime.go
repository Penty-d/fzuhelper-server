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
	"github.com/west2-online/fzuhelper-server/pkg/db/model"
)

func (c *DBLaunchScreen) AddImageListShowTime(ctx context.Context, pictureList *[]model.Picture) error {
	if len(*pictureList) == 0 {
		return nil
	}
	// 内存中自增仅用于调用方响应展示, 数据库计数由下方原子自增完成
	ids := make([]int64, 0, len(*pictureList))
	for i := range *pictureList {
		(*pictureList)[i].ShowTimes++
		ids = append(ids, (*pictureList)[i].ID)
	}
	// 使用 UpdateColumn 原子自增, 避免并发下丢失计数或用旧值覆盖其他字段
	if err := c.client.WithContext(ctx).Table(constants.LaunchScreenTableName).
		Where("id IN ?", ids).
		UpdateColumn("show_times", gorm.Expr("show_times + 1")).Error; err != nil {
		return fmt.Errorf("dal.AddImageListShowTime error: %w", err)
	}
	return nil
}
