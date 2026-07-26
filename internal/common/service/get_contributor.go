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

package service

import (
	"github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
)

func (s *CommonService) GetContributorInfo() (map[string][]*model.Contributor, error) {
	contributorKeys := []string{
		constants.ContributorFzuhelperAppKey,
		constants.ContributorFzuhelperServerKey,
		constants.ContributorJwchKey,
		constants.ContributorYJSYKey,
	}

	// 一次 MGET 批量取回全部 key 的数据，替代逐 key EXISTS+GET 的多次串行 round-trip
	contributors, err := s.cache.Common.GetContributorsInfo(s.ctx, contributorKeys)
	if err != nil {
		return nil, err
	}

	return contributors, nil
}
