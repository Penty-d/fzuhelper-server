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

	"github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
	"github.com/west2-online/fzuhelper-server/pkg/utils"
)

// baseResper 约束携带 BaseResp 的 kitex 响应类型
type baseResper interface {
	GetBase() *model.BaseResp
}

// call 统一处理 RPC 包装函数的通用骨架：调用 -> 失败记日志并返回 InternalServiceError -> HandleBaseRespWithCookie
// 仅供错误处理逻辑与该骨架逐字一致的包装函数使用，其余定制错误码/消息的包装函数保持各自实现
func call[T baseResper](ctx context.Context, name string, invoke func() (T, error)) (T, error) {
	var zero T
	resp, err := invoke()
	if err != nil {
		logger.WithCtx(ctx).Errorf("%s: RPC called failed: %v", name, err.Error())
		return zero, errno.InternalServiceError.WithMessage(err.Error())
	}
	if err = utils.HandleBaseRespWithCookie(resp.GetBase()); err != nil {
		return zero, err
	}
	return resp, nil
}
