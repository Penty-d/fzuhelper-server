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

package notice

import (
	"context"

	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/db/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
)

// GetNoticesByUrls 根据 url 列表批量查询已存在的通知，仅返回 title 和 url 字段，
// 用于同步任务在内存中按 (title, url) 做 diff，替代逐条 IsNoticeExists 查询
func (d *DBNotice) GetNoticesByUrls(ctx context.Context, urls []string) (list []model.Notice, err error) {
	if len(urls) == 0 {
		return nil, nil
	}
	// Unscoped 与 IsNoticeExists 保持一致：不过滤软删除记录，避免重复插入
	if err := d.client.WithContext(ctx).Table(constants.NoticeTableName).Unscoped().
		Select("title", "url").
		Where("url IN ?", urls).
		Find(&list).Error; err != nil {
		return nil, errno.Errorf(errno.InternalDatabaseErrorCode, "dal.GetNoticesByUrls error: %s", err)
	}
	return list, nil
}
