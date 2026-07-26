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
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/west2-online/fzuhelper-server/pkg/utils"
)

func TestDBLaunchScreen_AddPointTime(t *testing.T) {
	type testCase struct {
		name             string
		mockError        error
		mockRowsAffected int64
		inputId          int64
		expectingError   bool
		expectedWrapped  error // 期望被包装的底层错误, 为 nil 时不校验
	}

	testCases := []testCase{
		{
			name:             "AddPointTime_Success",
			mockError:        nil,
			mockRowsAffected: 1,
			inputId:          1,
			expectingError:   false,
		},
		{
			name:             "AddPointTime_RecordNotFound",
			mockError:        nil,
			mockRowsAffected: 0,
			inputId:          2,
			expectingError:   true,
			expectedWrapped:  gorm.ErrRecordNotFound,
		},
		{
			name:             "AddPointTime_DBError",
			mockError:        fmt.Errorf("db error"),
			mockRowsAffected: 0,
			inputId:          3,
			expectingError:   true,
		},
	}
	defer mockey.UnPatchAll()
	for _, tc := range testCases {
		mockey.PatchConvey(tc.name, t, func() {
			mockGormDB := new(gorm.DB)
			mockSnowflake := new(utils.Snowflake)
			mockDBLaunchScreen := NewDBLaunchScreen(mockGormDB, mockSnowflake)

			var gotColumn string
			var gotValue interface{}

			mockey.Mock((*gorm.DB).WithContext).To(func(ctx context.Context) *gorm.DB {
				return mockGormDB
			}).Build()
			mockey.Mock((*gorm.DB).Table).To(func(name string, args ...interface{}) *gorm.DB {
				return mockGormDB
			}).Build()
			mockey.Mock((*gorm.DB).Where).To(func(query interface{}, args ...interface{}) *gorm.DB {
				return mockGormDB
			}).Build()
			mockey.Mock((*gorm.DB).UpdateColumn).To(func(column string, value interface{}) *gorm.DB {
				gotColumn = column
				gotValue = value
				if tc.mockError != nil {
					mockGormDB.Error = tc.mockError
					return mockGormDB
				}
				mockGormDB.RowsAffected = tc.mockRowsAffected
				return mockGormDB
			}).Build()

			err := mockDBLaunchScreen.AddPointTime(context.Background(), tc.inputId)

			if tc.expectingError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "dal.AddPointTime error")
				if tc.expectedWrapped != nil {
					assert.ErrorIs(t, err, tc.expectedWrapped)
				}
			} else {
				assert.NoError(t, err)
				// 校验确实对 point_times 做了原子自增
				assert.Equal(t, "point_times", gotColumn)
				assert.Equal(t, gorm.Expr("point_times + 1"), gotValue)
			}
		})
	}
}
