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
	"fmt"

	"github.com/west2-online/fzuhelper-server/internal/user/pack"
	"github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/taskqueue"
)

func (s *UserService) GetFriendList(stuId string) ([]*model.UserFriendInfo, error) {
	userFriendKey := fmt.Sprintf(constants.UserFriendsKeyFormat, stuId)
	// getter 自带未命中判定, 单次 round-trip 完成读缓存; 过期竞态会自然回源而不是报错
	friendRelation, ok, err := s.cache.User.GetUserFriendCache(s.ctx, userFriendKey)
	if err != nil {
		return nil, fmt.Errorf("service.GetUserFriendCache: %w", err)
	}
	if !ok {
		if friendRelation, err = s.db.User.GetUserFriends(s.ctx, stuId); err != nil {
			return nil, fmt.Errorf("service.GetUserFriendsIdDB: %w", err)
		}
		s.taskQueue.Add(fmt.Sprintf("setFriendListCache:%s", stuId), taskqueue.QueueTask{Execute: func() error {
			return s.cache.User.SetUserFriendListCache(s.ctx, stuId, friendRelation)
		}})
	}
	friendList := make([]*model.UserFriendInfo, 0, len(friendRelation))
	if len(friendRelation) == 0 {
		return friendList, nil
	}
	// 批量获取好友信息: 先一次 MGET 批量查缓存, 未命中的再一次 SQL 批量查库, 避免逐个好友的 N+1 查询
	friendIds := make([]string, 0, len(friendRelation))
	for _, relation := range friendRelation {
		friendIds = append(friendIds, relation.FriendId)
	}
	stuInfoMap, err := s.cache.User.GetStuInfosCache(s.ctx, friendIds)
	if err != nil {
		return nil, fmt.Errorf("service.GetFriendList: %w", err)
	}
	missedIds := make([]string, 0, len(friendIds))
	for _, friendId := range friendIds {
		if _, ok := stuInfoMap[friendId]; !ok {
			missedIds = append(missedIds, friendId)
		}
	}
	if len(missedIds) > 0 {
		students, err := s.db.User.GetStudentsByIds(s.ctx, missedIds)
		if err != nil {
			return nil, fmt.Errorf("service.GetFriendList: %w", err)
		}
		for _, stuInfo := range students {
			stuInfoMap[stuInfo.StuId] = stuInfo
		}
	}
	// 按好友关系原顺序组装返回结果
	for _, relation := range friendRelation {
		if stuInfo, ok := stuInfoMap[relation.FriendId]; ok {
			friendList = append(friendList, pack.BuildFriendInfoResp(stuInfo, relation))
			continue
		}
		// 缓存和数据库都没有该学生信息 则只能模糊返回了
		friendList = append(friendList, &model.UserFriendInfo{
			StuId:     relation.FriendId,
			OrderSeq:  relation.OrderSeq,
			CreatedAt: relation.CreatedAt.Unix(),
		})
	}
	return friendList, nil
}
