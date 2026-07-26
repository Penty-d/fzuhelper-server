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

package constants

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRedisKeyFormat 锁定线上已有数据依赖的 key 格式，格式一旦变化缓存将整体失效
func TestRedisKeyFormat(t *testing.T) {
	assert.Equal(t, "terms:102300000", fmt.Sprintf(TermsKeyFormat, "102300000"))
	assert.Equal(t, "course:102300000:202401", fmt.Sprintf(CourseListKeyFormat, "102300000", "202401"))
	assert.Equal(t, "user_friends:102300000", fmt.Sprintf(UserFriendsKeyFormat, "102300000"))
	assert.Equal(t, "codes:102300000", fmt.Sprintf(InvitationCodeKeyFormat, "102300000"))
	assert.Equal(t, "code_mapping:abc123", fmt.Sprintf(CodeMappingKeyFormat, "abc123"))
	assert.Equal(t, "scores:102300000", fmt.Sprintf(ScoresKeyFormat, "102300000"))
}
