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
	"strconv"
	"time"

	"github.com/west2-online/fzuhelper-server/pkg/constants"
	db "github.com/west2-online/fzuhelper-server/pkg/db/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
	"github.com/west2-online/fzuhelper-server/pkg/taskqueue"
	"github.com/west2-online/jwch"
	"github.com/west2-online/yjsy"
)

// getUserInfo 是 GetUserInfo 与 GetUserInfoYjsy 的公共骨架, 承载缓存检查、DB 判存、插入/更新与异步写缓存;
// fnName 用于错误包装以保留真实调用方名称, fetch 负责从对应教务系统拉取学生信息
func (s *UserService) getUserInfo(stuId string, fnName string, fetch func() (*db.Student, error)) (*db.Student, error) {
	// 查询cache; getter 自带未命中判定, 单次 round-trip 完成读缓存, 过期竞态会自然回源而不是报错
	stuInfo, ok, err := s.cache.User.GetStuInfoCache(s.ctx, stuId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fnName, err)
	}
	if ok {
		return stuInfo, nil
	}

	// 未命中cache，查询数据库是否存入此学生信息
	exist, stuInfo, err := s.db.User.GetStudentById(s.ctx, stuId)
	isUpdate := false
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fnName, err)
	}
	if exist {
		if stuInfo.UpdatedAt.Add(constants.StuInfoExpireTime).After(time.Now()) {
			s.taskQueue.Add(fmt.Sprintf("setStuInfoCache:%s", stuId), taskqueue.QueueTask{Execute: func() error {
				return s.cache.User.SetStuInfoCache(s.ctx, stuId, stuInfo)
			}})
			return stuInfo, nil
		}
		isUpdate = true
	}

	// 拉取学生信息后插入/更新
	userModel, err := fetch()
	if err != nil {
		return nil, err
	}
	if isUpdate {
		err = s.db.User.UpdateStudent(s.ctx, userModel)
	} else {
		err = s.db.User.CreateStudent(s.ctx, userModel)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fnName, err)
	}

	// 存入cache
	s.taskQueue.Add(fmt.Sprintf("setStuInfoCache:%s", stuId), taskqueue.QueueTask{Execute: func() error {
		return s.cache.User.SetStuInfoCache(s.ctx, stuId, userModel)
	}})

	return userModel, nil
}

func (s *UserService) GetUserInfo(stuId string) (*db.Student, error) {
	return s.getUserInfo(stuId, "service.GetUserInfo", func() (*db.Student, error) {
		stu := jwch.NewStudent().WithLoginData(s.Identifier, s.cookies)
		resp, err := stu.GetInfo()
		if err != nil {
			return nil, errno.Errorf(errno.InternalServiceErrorCode, "service.GetUserInfo: jwch failed: %v", err)
		}
		grade, _ := strconv.Atoi(resp.Grade)
		return &db.Student{
			StuId:    stuId,
			Name:     resp.Name,
			Sex:      resp.Sex,
			Birthday: resp.Birthday,
			College:  resp.College,
			Grade:    int64(grade),
			Major:    resp.Major,
		}, nil
	})
}

func (s *UserService) GetUserInfoYjsy(stuId string) (*db.Student, error) {
	return s.getUserInfo(stuId, "service.GetUserInfoYjsy", func() (*db.Student, error) {
		stu := yjsy.NewStudent().WithLoginData(s.cookies)
		resp, err := stu.GetStudentInfo()
		if err != nil {
			return nil, errno.Errorf(errno.InternalServiceErrorCode, "service.GetUserInfoYjsy: yjsy failed: %v", err)
		}
		grade, _ := strconv.Atoi(resp.Grade)
		return &db.Student{
			StuId:    stuId,
			Name:     resp.Name,
			Sex:      resp.Sex,
			Birthday: resp.Birthday,
			College:  resp.College,
			Grade:    int64(grade),
			Major:    resp.Major,
		}, nil
	})
}
