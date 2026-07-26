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
	"errors"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"

	"github.com/west2-online/fzuhelper-server/internal/course/pack"
	"github.com/west2-online/fzuhelper-server/kitex_gen/course"
	kitexModel "github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/kitex_gen/user"
	"github.com/west2-online/fzuhelper-server/pkg/base/context"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/db/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
	"github.com/west2-online/fzuhelper-server/pkg/utils"
	"github.com/west2-online/jwch"
)

func (s *CourseService) GetFriendCourse(req *course.GetFriendCourseRequest, loginData *kitexModel.LoginData) ([]*kitexModel.Course, error) {
	var err error
	stuId := context.ExtractIDFromLoginData(loginData)
	// 验证好友
	resp, err := s.userClient.VerifyFriend(s.ctx, &user.VerifyFriendRequest{Id: stuId, FriendId: req.Id})
	if err != nil {
		return nil, fmt.Errorf("service.GetFriendCourse: verify friend failed: %w", err)
	}
	if err = utils.HandleBaseRespWithCookie(resp.Base); err != nil {
		return nil, err
	}
	if !resp.FriendExist {
		return nil, fmt.Errorf("只能查看好友的课表")
	}
	termKey := fmt.Sprintf(constants.TermsKeyFormat, req.Id)
	/* 这里如果terms Cache没命中 无法验证term的合法性 也就会拒绝返回好友课表
	   而term也是会在学生刷新课表时缓存 并且term似乎目前并不在db内存储
	   此外因为jwch与yjsy的区别 term也有两个结构 这边就直接用string来处理了
	*/
	var terms []string
	// getter 自带未命中判定, 单次 round-trip 完成读缓存; 过期竞态会自然回源而不是报错
	termsList, termsOk, err := s.cache.Course.GetTermsCache(s.ctx, termKey)
	if err != nil {
		return nil, fmt.Errorf("service.GetFriendCourse: Get term fail: %w", err)
	}
	if termsOk {
		terms = termsList
	} else {
		dbTerms, err := s.db.Course.GetUserTermByStuId(s.ctx, req.Id)
		if err != nil {
			return nil, fmt.Errorf("service.GetFriendCourse: Get term from database fail: %w", err)
		}
		if dbTerms != nil {
			terms = pack.ParseTerm(dbTerms.TermTime)
		}
	}
	// 查不到 term
	if terms == nil {
		return nil, errno.NewErrNo(errno.InternalServiceErrorCode, "service.GetFriendCourse: Friend termList empty")
	}
	reqTerm, err := resolveFriendTerm(terms, req.Term)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(pack.GetTop2TermLists(terms), reqTerm) {
		return nil, errors.New("只能查看好友最近两个学期的课表")
	}
	/* cache 返回的两个course结构有区别 而目前判别研究生身份的方法需要loginData.Id
	在cache命中的情况下 先后两次尝试获取并返回课表
	*/
	courseKey := fmt.Sprintf(constants.CourseListKeyFormat, req.Id, reqTerm)
	cachedCourses, coursesOk, err := s.cache.Course.GetCoursesCache(s.ctx, courseKey)
	if err != nil {
		return nil, fmt.Errorf("service.GetFriendCourse: Get courses fail: %w", err)
	}
	if coursesOk {
		if cachedCourses != nil {
			return s.removeDuplicateCourses(pack.BuildCourse(cachedCourses)), nil
		}
		// cache 命中却没有course数据 做出查找研究生课表的尝试
		yjsyCourses, yjsyOk, err := s.cache.Course.GetCoursesCacheYjsy(s.ctx, courseKey)
		if err != nil {
			return nil, fmt.Errorf("service.GetFriendCourse: Get yjsy courses fail: %w", err)
		}
		// 研究生课表缓存也未命中时不能返回空课表，继续走下方数据库回源
		if yjsyOk {
			return pack.BuildCourseYjsy(yjsyCourses), nil
		}
	}

	var courses *model.UserCourse
	courses, err = s.db.Course.GetUserTermCourseByStuIdAndTerm(s.ctx, req.Id, reqTerm)
	if err != nil {
		return nil, fmt.Errorf("service.GetFriendCourse: Get courses fail: %w", err)
	}
	if courses == nil {
		return nil, fmt.Errorf("暂无好友课表信息，快通知好友登录App刷新课表吧")
	}
	list := make([]*kitexModel.Course, 0)
	if courses.TermCourses != "" {
		if err = sonic.Unmarshal([]byte(courses.TermCourses), &list); err != nil {
			return nil, fmt.Errorf("service.GetFriendCourse: Unmarshal fail: %w", err)
		}
	}
	// 只处理本科生的调课信息
	if !utils.IsYjsyTerm(reqTerm) {
		adjustCourses, err := s.GetAutoAdjustCourseList(reqTerm)
		if err != nil {
			return nil, fmt.Errorf("service.GetFriendCourse: Get adjust course failed: %w", err)
		}
		for _, c := range list {
			jwchRules := pack.ToJwchScheduleRules(c.ScheduleRules)
			adjustRules := getAdjustRules(jwchRules, adjustCourses)
			c.ScheduleRules = pack.FromJwchScheduleRules(jwch.ApplyAdjustRules(jwchRules, adjustRules))
		}
	}
	return list, nil
}

// resolveFriendTerm 将请求的学期映射为好友学期列表中的合法学期。
// 由于本科生查课表时正确传参是202501、研究生则是2024-2025-1
// 为防止因此导致term无效。下面做了一个映射的操作
func resolveFriendTerm(terms []string, reqTerm string) (string, error) {
	switch {
	case slices.Contains(terms, reqTerm):
		return reqTerm, nil
	case utils.IsYjsyTerm(reqTerm):
		if !slices.Contains(terms, utils.MapYjsyTerm(reqTerm)) {
			return "", errors.New("service.GetFriendCourse: Invalid term")
		}
		return utils.MapYjsyTerm(reqTerm), nil
	case utils.IsJwchTerm(reqTerm):
		if !slices.Contains(terms, utils.MapJwchTerm(reqTerm)) {
			return "", errors.New("service.GetFriendCourse: Invalid term")
		}
		return utils.MapJwchTerm(reqTerm), nil
	default:
		return "", errors.New("service.GetFriendCourse: Invalid term")
	}
}
