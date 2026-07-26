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

	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/db/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
)

// GetStudentsByIds 按学号批量查询学生信息, 不存在的学号不会出现在结果中(Find 空结果不报错)
func (c *DBUser) GetStudentsByIds(ctx context.Context, stuIds []string) ([]*model.Student, error) {
	students := make([]*model.Student, 0, len(stuIds))
	if len(stuIds) == 0 {
		return students, nil
	}
	if err := c.client.WithContext(ctx).Table(constants.UserTableName).Where("stu_id IN ?", stuIds).Find(&students).Error; err != nil {
		logger.Errorf("dal.GetStudentsByIds error:%v", err)
		return nil, errno.Errorf(errno.InternalDatabaseErrorCode, "dal.GetStudentsByIds error:%v", err)
	}
	return students, nil
}
