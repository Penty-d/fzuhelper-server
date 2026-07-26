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

package utils

import (
	"slices"

	"github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
)

// passthroughCodes 需要原样透传给客户端的业务错误码。
// 客户端依赖这些码触发具体处理（重新登录、补齐请求头等），
// 压平成通用 BizError 会让客户端无法区分而只能给出笼统提示
var passthroughCodes = []int64{
	errno.BizJwchCookieExceptionCode,
	errno.BizJwchEvaluationNotFoundCode,
	errno.ParamMissingHeaderCode,
}

// IsSuccess 通用的rpc结果处理
func IsSuccess(baseResp *model.BaseResp) bool {
	return baseResp.Code == errno.SuccessCode
}

// HandleBaseRespWithCookie 调用jwch库的接口的结果处理， 将 resp.Base 中包含的错误转换成 errno 类型
func HandleBaseRespWithCookie(baseResp *model.BaseResp) error {
	if baseResp.Code == errno.SuccessCode {
		return nil
	}
	if slices.Contains(passthroughCodes, baseResp.Code) {
		return errno.NewErrNo(baseResp.Code, baseResp.Msg)
	}
	return errno.BizError.WithMessage(baseResp.Msg)
}
