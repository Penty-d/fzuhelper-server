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
	"fmt"

	"github.com/west2-online/fzuhelper-server/internal/course/pack"
	loginmodel "github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/pkg/base"
	customContext "github.com/west2-online/fzuhelper-server/pkg/base/context"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/db/model"
	"github.com/west2-online/fzuhelper-server/pkg/taskqueue"
	"github.com/west2-online/fzuhelper-server/pkg/utils"
	"github.com/west2-online/jwch"
	"github.com/west2-online/yjsy"
)

// getTermsList 是 GetTermsList/GetTermsListYjsy 的公共骨架：
// 优先读缓存，未命中时通过 fetchTerms 从教务/研究生系统拉取学期列表，并异步写缓存与落库。
// funcName 用于错误信息前缀，fetchTerms 由调用方注入本科生/研究生的拉取实现。
func (s *CourseService) getTermsList(loginData *loginmodel.LoginData, funcName string, fetchTerms func() ([]string, error)) ([]string, error) {
	stuId := customContext.ExtractIDFromLoginData(loginData)
	key := fmt.Sprintf(constants.TermsKeyFormat, stuId)
	// getter 自带未命中判定, 单次 round-trip 完成读缓存; 过期竞态会自然回源而不是报错
	terms, ok, err := s.cache.Course.GetTermsCache(s.ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%s: Get terms cache fail: %w", funcName, err)
	}
	if ok {
		return terms, nil
	}

	terms, err = fetchTerms()
	if err != nil {
		return nil, fmt.Errorf("%s: Get terms fail: %w", funcName, err)
	}

	// 异步任务在 handler 返回后才执行，提前捕获与请求解耦的 ctx，避免复用已被框架回收的请求级上下文
	taskCtx := context.WithoutCancel(s.ctx)
	s.taskQueue.Add(fmt.Sprintf("setTermsCache:%s", stuId), taskqueue.QueueTask{Execute: func() error {
		return s.cache.Course.SetTermsCache(taskCtx, key, terms)
	}})
	s.taskQueue.Add(fmt.Sprintf("putTerms:%s", stuId), taskqueue.QueueTask{Execute: func() error {
		return s.putTermToDatabase(taskCtx, stuId, pack.BuildTermOnDB(terms))
	}})

	return terms, nil
}

// GetTermsList 会返回当前用户含有课表的学期信息
func (s *CourseService) GetTermsList(loginData *loginmodel.LoginData) ([]string, error) {
	return s.getTermsList(loginData, "service.GetTermList", func() ([]string, error) {
		stu := jwch.NewStudent().WithLoginData(loginData.GetId(), utils.ParseCookies(loginData.GetCookies()))
		terms, err := stu.GetTerms()
		if err = base.HandleJwchError(err); err != nil {
			return nil, err
		}
		return terms.Terms, nil
	})
}

// GetTermsListYjsy 会返回当前研究生用户含有课表的学期信息
func (s *CourseService) GetTermsListYjsy(loginData *loginmodel.LoginData) ([]string, error) {
	return s.getTermsList(loginData, "service.GetTermListYjsy", func() ([]string, error) {
		stu := yjsy.NewStudent().WithLoginData(utils.ParseCookies(loginData.Cookies))
		terms, err := stu.GetTerms()
		if err = base.HandleYjsyError(err); err != nil {
			return nil, err
		}
		return terms.Terms, nil
	})
}

func (s *CourseService) putTermToDatabase(ctx context.Context, stuId string, termList string) error {
	old, err := s.db.Course.GetUserTermByStuId(ctx, stuId)
	if err != nil {
		return err
	}
	if old == nil {
		dbId, err := s.sf.NextVal()
		if err != nil {
			return err
		}
		_, err = s.db.Course.CreateUserTerm(ctx, &model.UserTerm{
			Id:       dbId,
			StuId:    stuId,
			TermTime: termList,
		})
		if err != nil {
			return err
		}
	} else if old.TermTime != termList {
		_, err = s.db.Course.UpdateUserTerm(ctx, &model.UserTerm{
			Id:       old.Id,
			StuId:    stuId,
			TermTime: termList,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
