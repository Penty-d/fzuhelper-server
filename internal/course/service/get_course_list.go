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
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/west2-online/fzuhelper-server/internal/course/pack"
	"github.com/west2-online/fzuhelper-server/kitex_gen/course"
	kitexModel "github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/pkg/base"
	customContext "github.com/west2-online/fzuhelper-server/pkg/base/context"
	"github.com/west2-online/fzuhelper-server/pkg/cache"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/db/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
	"github.com/west2-online/fzuhelper-server/pkg/taskqueue"
	"github.com/west2-online/fzuhelper-server/pkg/utils"
	"github.com/west2-online/jwch"
	"github.com/west2-online/yjsy"
)

// getCachedCourses 读取本科生课表缓存。只有最新两个学期的课程会被缓存，
// 学期缓存缺失、请求学期不在缓存范围内或课表缓存未命中时均返回 hit=false，由调用方回源
func (s *CourseService) getCachedCourses(termKey, courseKey, term string) ([]*jwch.Course, bool, error) {
	termsList, termsOk, err := s.cache.Course.GetTermsCache(s.ctx, termKey)
	if err != nil {
		return nil, false, fmt.Errorf("service.GetCourseList: Get term fail: %w", err)
	}
	if !termsOk {
		return nil, false, nil
	}
	if !slices.Contains(pack.GetTop2Terms(&jwch.Term{Terms: termsList}).Terms, term) {
		return nil, false, nil
	}

	courses, coursesOk, err := s.cache.Course.GetCoursesCache(s.ctx, courseKey)
	if err != nil {
		return nil, false, fmt.Errorf("service.GetCourseList: Get courses fail: %w", err)
	}
	return courses, coursesOk, nil
}

// getCachedCoursesYjsy 读取研究生课表缓存，语义同 getCachedCourses
func (s *CourseService) getCachedCoursesYjsy(termKey, courseKey, term string) ([]*yjsy.Course, bool, error) {
	termsList, termsOk, err := s.cache.Course.GetTermsCache(s.ctx, termKey)
	if err != nil {
		return nil, false, fmt.Errorf("service.GetCourseListYjsy: Get terms fail: %w", err)
	}
	if !termsOk {
		return nil, false, nil
	}
	if !slices.Contains(pack.GetTop2TermsYjsy(&yjsy.Term{Terms: termsList}).Terms, term) {
		return nil, false, nil
	}

	courses, coursesOk, err := s.cache.Course.GetCoursesCacheYjsy(s.ctx, courseKey)
	if err != nil {
		return nil, false, fmt.Errorf("service.GetCourseListYjsy: Get courses fail: %w", err)
	}
	return courses, coursesOk, nil
}

func (s *CourseService) GetCourseList(req *course.CourseListRequest, loginData *kitexModel.LoginData) ([]*kitexModel.Course, error) {
	var err error
	stuId := customContext.ExtractIDFromLoginData(loginData)
	termKey := fmt.Sprintf(constants.TermsKeyFormat, stuId)
	courseKey := fmt.Sprintf(constants.CourseListKeyFormat, stuId, req.Term)
	terms := new(jwch.Term)
	// 学期缓存存在
	isRefresh := false
	if req.IsRefresh != nil {
		isRefresh = *req.IsRefresh
	}
	// 不刷新时先读缓存; getter 自带未命中判定, 过期竞态会自然回源而不是报错
	if !isRefresh {
		cachedCourses, hit, err := s.getCachedCourses(termKey, courseKey, req.Term)
		if err != nil {
			return nil, err
		}
		if hit {
			return s.removeDuplicateCourses(pack.BuildCourse(cachedCourses)), nil
		}
	}

	stu := jwch.NewStudent().WithLoginData(loginData.GetId(), utils.ParseCookies(loginData.GetCookies()))

	terms, err = stu.GetTerms()
	if err = base.HandleJwchError(err); err != nil {
		return nil, fmt.Errorf("service.GetCourseList: Get terms failed: %w", err)
	}

	// validate term
	if !slices.Contains(terms.Terms, req.Term) {
		return nil, errors.New("service.GetCourseList: Invalid term")
	}

	courses, err := stu.GetSemesterCourses(req.Term, terms.ViewState, terms.EventValidation)
	if err = base.HandleJwchError(err); err != nil {
		return nil, fmt.Errorf("service.GetCourseList: Get semester courses failed: %w", err)
	}

	// 异步任务在 handler 返回后才执行，提前捕获与请求解耦的 ctx（保留 trace 等值、剥离取消传播），
	// 避免闭包继续使用可能已被框架回收的请求级上下文
	taskCtx := context.WithoutCancel(s.ctx)

	// async put course list to db
	// 数据库存储原始的课表信息（不包含调课信息）
	originalCourses := pack.BuildCourse(courses)
	s.taskQueue.Add(fmt.Sprintf("putCourse:%s", stuId), taskqueue.QueueTask{Execute: func() error {
		return s.putCourseToDatabase(taskCtx, stuId, req.Term, originalCourses)
	}})

	adjustCourses, err := s.GetAutoAdjustCourseList(req.Term)
	if err != nil {
		return nil, fmt.Errorf("service.GetCourseList: Get adjust course failed: %w", err)
	}

	for _, c := range courses {
		adjustRules := getAdjustRules(c.ScheduleRules, adjustCourses)
		c.ScheduleRules = jwch.ApplyAdjustRules(
			jwch.ApplyAdjustRules(c.ScheduleRules, c.AdjustRules),
			adjustRules,
		)
	}

	if slices.Contains(pack.GetTop2Terms(terms).Terms, req.Term) {
		// async put course list to cache
		// 缓存存储调课后的课表信息
		s.taskQueue.Add(courseKey, taskqueue.QueueTask{Execute: func() error {
			return cache.SetSliceCache(s.cache, taskCtx, courseKey, courses,
				constants.CourseTermsKeyExpire, "Course.SetCourseCache")
		}})
		s.taskQueue.Add(termKey, taskqueue.QueueTask{Execute: func() error {
			return cache.SetValueSliceCache(s.cache, taskCtx, termKey, terms.Terms, constants.CourseTermsKeyExpire, "Course.SetTermsCache")
		}})
	}

	// 学期列表异步存库
	s.taskQueue.Add(fmt.Sprintf("putTerms:%s", stuId), taskqueue.QueueTask{Execute: func() error {
		return s.putTermToDatabase(taskCtx, stuId, pack.BuildTermOnDB(terms.Terms))
	}})

	return s.removeDuplicateCourses(pack.BuildCourse(courses)), nil
}

func (s *CourseService) putCourseToDatabase(ctx context.Context, stuId string, term string, courses []*kitexModel.Course) error {
	old, err := s.db.Course.GetUserTermCourseSha256ByStuIdAndTerm(ctx, stuId, term)
	if err != nil {
		return err
	}

	json, err := utils.JSONEncode(courses)
	if err != nil {
		return err
	}

	newSha256 := utils.SHA256(json)

	if old == nil {
		dbId, err := s.sf.NextVal()
		if err != nil {
			return err
		}

		_, err = s.db.Course.CreateUserTermCourse(ctx, &model.UserCourse{
			Id:                dbId,
			StuId:             stuId,
			Term:              term,
			TermCourses:       json,
			TermCoursesSha256: newSha256,
		})
		if err != nil {
			return err
		}
	} else if old.TermCoursesSha256 != newSha256 {
		_, err = s.db.Course.UpdateUserTermCourse(ctx, &model.UserCourse{
			Id:                old.Id,
			TermCourses:       json,
			TermCoursesSha256: newSha256,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *CourseService) GetCourseListYjsy(req *course.CourseListRequest, loginData *kitexModel.LoginData) ([]*kitexModel.Course, error) {
	var err error

	stuId := customContext.ExtractIDFromLoginData(loginData)
	termKey := fmt.Sprintf(constants.TermsKeyFormat, stuId)
	courseKey := fmt.Sprintf(constants.CourseListKeyFormat, stuId, req.Term)
	terms := new(yjsy.Term)
	// 学期缓存存在
	isRefresh := false
	if req.IsRefresh != nil {
		isRefresh = *req.IsRefresh
	}
	// 不刷新时先读缓存; getter 自带未命中判定, 过期竞态会自然回源而不是报错
	if !isRefresh {
		cachedCourses, hit, err := s.getCachedCoursesYjsy(termKey, courseKey, req.Term)
		if err != nil {
			return nil, err
		}
		if hit {
			return pack.BuildCourseYjsy(cachedCourses), nil
		}
	}

	// 获取学期信息
	stu := yjsy.NewStudent().WithLoginData(utils.ParseCookies(loginData.Cookies))
	terms, err = stu.GetTerms()
	if err = base.HandleYjsyError(err); err != nil {
		return nil, fmt.Errorf("service.GetCourseListYjsy: Get terms failed: %w", err)
	}

	// 验证学期是否有效
	if !slices.Contains(terms.Terms, req.Term) {
		return nil, errors.New("service.GetCourseListYjsy: Invalid term")
	}

	// 获取该学期的课程
	courses, err := stu.GetSemesterCourses(req.Term)
	if err = base.HandleYjsyError(err); err != nil {
		return nil, fmt.Errorf("service.GetCourseListYjsy: Get semester courses failed: %w", err)
	}

	// 异步任务在 handler 返回后才执行，提前捕获与请求解耦的 ctx，避免复用已被框架回收的请求级上下文
	taskCtx := context.WithoutCancel(s.ctx)

	// 如果是前两个学期，异步缓存课程列表
	if slices.Contains(pack.GetTop2TermsYjsy(terms).Terms, req.Term) {
		s.taskQueue.Add(courseKey, taskqueue.QueueTask{Execute: func() error {
			return cache.SetSliceCache(s.cache, taskCtx, courseKey, courses,
				constants.CourseTermsKeyExpire, "Course.SetCourseCache")
		}})
		s.taskQueue.Add(termKey, taskqueue.QueueTask{Execute: func() error {
			return cache.SetValueSliceCache(s.cache, taskCtx, termKey, terms.Terms, constants.CourseTermsKeyExpire, "Course.SetTermsCache")
		}})
	}

	// 异步将课程列表存入数据库
	s.taskQueue.Add(fmt.Sprintf("putCourse:%s", stuId), taskqueue.QueueTask{Execute: func() error {
		return s.putCourseToDatabase(taskCtx, stuId, req.Term, pack.BuildCourseYjsy(courses))
	}})
	// 学期列表异步存库
	s.taskQueue.Add(fmt.Sprintf("putTerms:%s", stuId), taskqueue.QueueTask{Execute: func() error {
		return s.putTermToDatabase(taskCtx, stuId, pack.BuildTermOnDB(terms.Terms))
	}})

	return pack.BuildCourseYjsy(courses), nil
}

// removeDuplicateCourses 移除重复课程，只保留第一个出现的。
func (s *CourseService) removeDuplicateCourses(courses []*kitexModel.Course) []*kitexModel.Course {
	seen := make(map[string]struct{})
	var result []*kitexModel.Course

	for _, c := range courses {
		srIDs := make([]string, 0, len(c.ScheduleRules))
		for _, rule := range c.ScheduleRules {
			part := fmt.Sprintf("%d-%d-%d-%d",
				rule.StartClass, rule.EndClass,
				rule.StartWeek, rule.EndWeek)
			srIDs = append(srIDs, part)
		}
		sort.Strings(srIDs)

		// 把“课程名 + 教师 + 排课信息”拼成一个全局唯一的 key
		identifier := fmt.Sprintf("%s-%s-%s", c.Name, c.Teacher, strings.Join(srIDs, "|"))

		// 如果 map 里还没出现过这个标识，那就是新课程
		if _, exists := seen[identifier]; !exists {
			seen[identifier] = struct{}{}
			result = append(result, c)
		}
	}

	return result
}

func (s *CourseService) getSemesterCourses(stuID string, term string, isGraduate bool) (course []*kitexModel.Course, err error) {
	courseKey := fmt.Sprintf(constants.CourseListKeyFormat, stuID, term)
	// getter 自带未命中判定, 过期竞态会自然回源而不是报错
	cachedCourses, ok, err := s.cache.Course.GetCoursesCache(s.ctx, courseKey)
	if err != nil {
		return nil, fmt.Errorf("service.GetSemesterCourses: Get courses fail: %w", err)
	}
	if ok {
		return s.removeDuplicateCourses(pack.BuildCourse(cachedCourses)), nil
	}
	// 从数据中获取课程表
	var courses *model.UserCourse
	courses, err = s.db.Course.GetUserTermCourseByStuIdAndTerm(s.ctx, stuID, term)
	if err != nil {
		return nil, fmt.Errorf("service.GetSemesterCourses: Get courses fail: %w", err)
	}
	if courses == nil {
		return nil, errno.NewErrNo(errno.InternalServiceErrorCode, "service.GetSemesterCourses: there is no course in database, please login app and retry")
	}
	// 将数据库中的课程表进行解析转化
	list := make([]*kitexModel.Course, 0)

	if courses.TermCourses != "" {
		if err = sonic.Unmarshal([]byte(courses.TermCourses), &list); err != nil {
			return nil, fmt.Errorf("service.GetSemesterCourses: Unmarshal fail: %w", err)
		}
	}

	// 只处理本科生的调课信息
	if !isGraduate {
		adjustCourses, err := s.GetAutoAdjustCourseList(term)
		if err != nil {
			return nil, fmt.Errorf("service.getSemesterCourses: Get adjust course failed: %w", err)
		}

		for _, c := range list {
			jwchRules := pack.ToJwchScheduleRules(c.ScheduleRules)
			adjustRules := getAdjustRules(jwchRules, adjustCourses)
			c.ScheduleRules = pack.FromJwchScheduleRules(jwch.ApplyAdjustRules(jwchRules, adjustRules))
		}
	}

	// 写入 cache；异步任务在 handler 返回后才执行，使用与请求解耦的 ctx
	taskCtx := context.WithoutCancel(s.ctx)
	s.taskQueue.Add(courseKey, taskqueue.QueueTask{Execute: func() error {
		return cache.SetSliceCache(s.cache, taskCtx, courseKey, list,
			constants.CourseTermsKeyExpire, "Course.SetCourseCache")
	}})
	return list, nil
}

func getAdjustRules(scheduleRules []jwch.CourseScheduleRule, adjustCourses []*model.AutoAdjustCourse) (adjustRules []jwch.CourseAdjustRule) {
	for _, c := range adjustCourses {
		if !c.Enabled {
			continue
		}

		fromWeek := int(c.FromWeek)
		fromWeekday := int(c.FromWeekday)

		canceled := c.ToDate == nil

		for _, r := range scheduleRules {
			if r.StartWeek <= fromWeek && r.EndWeek >= fromWeek && r.Weekday == fromWeekday {
				if canceled {
					adjustRules = append(adjustRules, jwch.CourseAdjustRule{
						OldWeek:       fromWeek,
						OldWeekday:    r.Weekday,
						OldStartClass: r.StartClass,
						OldEndClass:   r.EndClass,
						Canceled:      true,
					})
					continue
				}

				toWeek := int(*c.ToWeek)
				toWeekday := int(*c.ToWeekday)

				adjustRules = append(adjustRules, jwch.CourseAdjustRule{
					OldWeek:       fromWeek,
					OldWeekday:    r.Weekday,
					OldStartClass: r.StartClass,
					OldEndClass:   r.EndClass,
					Canceled:      false,
					NewWeek:       toWeek,
					NewWeekday:    toWeekday,
					NewStartClass: r.StartClass,
					NewEndClass:   r.EndClass,
					NewLocation:   r.Location,
				})
			}
		}
	}

	return adjustRules
}
