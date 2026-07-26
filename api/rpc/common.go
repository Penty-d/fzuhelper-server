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

	"github.com/west2-online/fzuhelper-server/kitex_gen/common"
	"github.com/west2-online/fzuhelper-server/kitex_gen/model"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
)

func GetCSSRPC(ctx context.Context, req *common.GetCSSRequest) (*[]byte, error) {
	resp, err := commonClient.GetCSS(ctx, req)
	if err != nil {
		logger.WithCtx(ctx).Errorf("GetCSSRPC: RPC called failed: %v", err.Error())
		return nil, errno.InternalServiceError.WithMessage(err.Error())
	}
	if len(resp.Css) < 1 {
		return nil, errno.InternalServiceError
	}
	return &resp.Css, nil
}

func GetHtmlRPC(ctx context.Context, req *common.GetHtmlRequest) (*[]byte, error) {
	resp, err := commonClient.GetHtml(ctx, req)
	if err != nil {
		logger.WithCtx(ctx).Errorf("GetHtmlRPC: RPC called failed: %v", err.Error())
		return nil, errno.InternalServiceError.WithMessage(err.Error())
	}
	if len(resp.Html) < 1 {
		return nil, errno.InternalServiceError
	}
	return &resp.Html, nil
}

func GetUserAgreementRPC(ctx context.Context, req *common.GetUserAgreementRequest) (*[]byte, error) {
	resp, err := commonClient.GetUserAgreement(ctx, req)
	if err != nil {
		logger.WithCtx(ctx).Errorf("GetUserAgreementRPC: RPC called failed: %v", err.Error())
		return nil, errno.InternalServiceError.WithMessage(err.Error())
	}
	if len(resp.UserAgreement) < 1 {
		return nil, errno.InternalServiceError
	}
	return &resp.UserAgreement, nil
}

func GetTermsListRPC(ctx context.Context, req *common.TermListRequest) (*model.TermList, error) {
	resp, err := call(ctx, "GetTermsListRPC", func() (*common.TermListResponse, error) {
		return commonClient.GetTermsList(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.TermLists, nil
}

func GetTermRPC(ctx context.Context, req *common.TermRequest) (*model.TermInfo, error) {
	resp, err := call(ctx, "GetTermRPC", func() (*common.TermResponse, error) {
		return commonClient.GetTerm(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.TermInfo, nil
}

func GetNoticesRPC(ctx context.Context, req *common.NoticeRequest) ([]*model.NoticeInfo, int64, error) {
	resp, err := call(ctx, "GetNoticesRPC", func() (*common.NoticeResponse, error) {
		return commonClient.GetNotices(ctx, req)
	})
	if err != nil {
		return nil, 0, err
	}
	return resp.Notices, resp.Total, nil
}

func GetContributorRPC(ctx context.Context, req *common.GetContributorInfoRequest) (*common.GetContributorInfoResponse, error) {
	resp, err := call(ctx, "GetContributorRPC", func() (*common.GetContributorInfoResponse, error) {
		return commonClient.GetContributorInfo(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func GetToolboxConfigRPC(ctx context.Context, req *common.GetToolboxConfigRequest) ([]*model.ToolboxConfig, error) {
	resp, err := call(ctx, "GetToolboxConfigRPC", func() (*common.GetToolboxConfigResponse, error) {
		return commonClient.GetToolboxConfig(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.Config, nil
}

func GetToolboxConfigListRPC(ctx context.Context, req *common.GetToolboxConfigListRequest) ([]*model.ToolboxConfig, int64, error) {
	resp, err := call(ctx, "GetToolboxConfigListRPC", func() (*common.GetToolboxConfigListResponse, error) {
		return commonClient.GetToolboxConfigList(ctx, req)
	})
	if err != nil {
		return nil, 0, err
	}
	return resp.Config, resp.Total, nil
}

func PutToolboxConfigRPC(ctx context.Context, req *common.PutToolboxConfigRequest) (*common.PutToolboxConfigResponse, error) {
	resp, err := call(ctx, "PutToolboxConfigRPC", func() (*common.PutToolboxConfigResponse, error) {
		return commonClient.PutToolboxConfig(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func TracePingRPC(ctx context.Context, req *common.TracePingRequest) (string, error) {
	resp, err := call(ctx, "TracePingRPC", func() (*common.TracePingResponse, error) {
		return commonClient.TracePing(ctx, req)
	})
	if err != nil {
		return "", err
	}
	return resp.Message, nil
}
