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

	"gorm.io/gorm/clause"

	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/db/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
)

// CreateNotices 批量创建通知，与 url 唯一索引（init.sql 的 unique_url）冲突的行直接忽略，
// 语义与 CreateNotice 容忍重复一致
func (d *DBNotice) CreateNotices(ctx context.Context, notices []*model.Notice) (err error) {
	if len(notices) == 0 {
		return nil
	}
	for _, notice := range notices {
		notice.Id, err = d.sf.NextVal()
		if err != nil {
			return errno.Errorf(errno.InternalDatabaseErrorCode, "dal.CreateNotices: NextVal error: %s", err)
		}
	}
	if err := d.client.WithContext(ctx).Table(constants.NoticeTableName).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(notices, constants.NoticePageSize).Error; err != nil {
		return errno.Errorf(errno.InternalDatabaseErrorCode, "dal.CreateNotices error: %s", err)
	}
	return nil
}
