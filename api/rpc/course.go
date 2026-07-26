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

package rpc

import (
	"context"

	"github.com/west2-online/fzuhelper-server/kitex_gen/course"
	"github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
	"github.com/west2-online/fzuhelper-server/pkg/utils"
)

func GetCourseListRPC(ctx context.Context, req *course.CourseListRequest) (courses []*model.Course, err error) {
	resp, err := call(ctx, "GetCourseListRPC", func() (*course.CourseListResponse, error) {
		return courseClient.GetCourseList(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func GetCourseTermsListRPC(ctx context.Context, req *course.TermListRequest) (*course.TermListResponse, error) {
	resp, err := call(ctx, "GetTermListRPC", func() (*course.TermListResponse, error) {
		return courseClient.GetTermList(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func GetCalendarRPC(ctx context.Context, req *course.GetCalendarRequest) ([]byte, error) {
	resp, err := call(ctx, "GetCalendarRPC", func() (*course.GetCalendarResponse, error) {
		return courseClient.GetCalendar(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.Ics, nil
}

func GetLocateDateRPC(ctx context.Context, req *course.GetLocateDateRequest) (*model.LocateDate, error) {
	resp, err := courseClient.GetLocateDate(ctx, req)
	if err != nil {
		logger.WithCtx(ctx).Errorf("GetLocateDateRPC: RPC called failed: %v", err.Error())
		return nil, errno.InternalServiceError.WithMessage(err.Error())
	}
	if !utils.IsSuccess(resp.Base) {
		return nil, errno.NewErrNo(resp.Base.Code, resp.Base.Msg)
	}
	return resp.LocateDate, nil
}

func GetFriendCourseRPC(ctx context.Context, req *course.GetFriendCourseRequest) (courses []*model.Course, err error) {
	resp, err := courseClient.GetFriendCourse(ctx, req)
	if err != nil {
		logger.WithCtx(ctx).Errorf("GetCourseListRPC: RPC called failed: %v", err.Error())
		return nil, errno.InternalServiceError.WithMessage(err.Error())
	}
	// 保留 HandleBaseRespWithCookie 透传的错误码（如 cookie 异常码），仅补充中文语境前缀
	if err = utils.HandleBaseRespWithCookie(resp.Base); err != nil {
		e := errno.ConvertErr(err)
		return nil, errno.NewErrNo(e.ErrorCode, "查看好友课表失败: "+e.ErrorMsg)
	}

	return resp.Data, nil
}

func GetAutoAdjustCourseListRPC(ctx context.Context, req *course.GetAutoAdjustCourseListRequest) (adjustCourses []*model.AdjustCourse, err error) {
	resp, err := courseClient.GetAutoAdjustCourseList(ctx, req)
	if err != nil {
		logger.WithCtx(ctx).Errorf("GetAutoAdjustCourseListRPC: RPC called failed: %v", err.Error())
		return nil, errno.InternalServiceError.WithMessage(err.Error())
	}
	// 保留 HandleBaseRespWithCookie 透传的错误码（如 cookie 异常码），仅补充中文语境前缀
	if err = utils.HandleBaseRespWithCookie(resp.Base); err != nil {
		e := errno.ConvertErr(err)
		return nil, errno.NewErrNo(e.ErrorCode, "获取自动调课列表失败: "+e.ErrorMsg)
	}

	return resp.Data, nil
}

func UpdateAutoAdjustCourseRPC(ctx context.Context, req *course.UpdateAdjustCourseRequest) (err error) {
	resp, err := courseClient.UpdateAdjustCourse(ctx, req)
	if err != nil {
		logger.WithCtx(ctx).Errorf("UpdateAutoAdjustCourseRPC: RPC called failed: %v", err.Error())
		return errno.InternalServiceError.WithMessage(err.Error())
	}
	// 保留 HandleBaseRespWithCookie 透传的错误码（如 cookie 异常码），仅补充中文语境前缀
	if err = utils.HandleBaseRespWithCookie(resp.Base); err != nil {
		e := errno.ConvertErr(err)
		return errno.NewErrNo(e.ErrorCode, "更新自动调课规则失败: "+e.ErrorMsg)
	}
	return nil
}
