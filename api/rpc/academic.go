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

	"github.com/west2-online/fzuhelper-server/kitex_gen/academic"
	"github.com/west2-online/fzuhelper-server/kitex_gen/model"
)

func GetScoresRPC(ctx context.Context, req *academic.GetScoresRequest) (scores []*model.Score, err error) {
	resp, err := call(ctx, "GetScoresRPC", func() (*academic.GetScoresResponse, error) {
		return academicClient.GetScores(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.Scores, nil
}

func GetGPARPC(ctx context.Context, req *academic.GetGPARequest) (gpa *model.GPABean, err error) {
	resp, err := call(ctx, "GetGPARPC", func() (*academic.GetGPAResponse, error) {
		return academicClient.GetGPA(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.Gpa, nil
}

func GetCreditRPC(ctx context.Context, req *academic.GetCreditRequest) (credit []*model.Credit, err error) {
	resp, err := call(ctx, "GetCreditRPC", func() (*academic.GetCreditResponse, error) {
		return academicClient.GetCredit(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.Major, nil
}

func GetUnifiedExamRPC(ctx context.Context, req *academic.GetUnifiedExamRequest) (unifiedExam []*model.UnifiedExam, err error) {
	resp, err := call(ctx, "GetUnifiedExamRPC", func() (*academic.GetUnifiedExamResponse, error) {
		return academicClient.GetUnifiedExam(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return resp.UnifiedExam, nil
}

func GetCultivatePlanRPC(ctx context.Context, req *academic.GetPlanRequest) (string, error) {
	resp, err := call(ctx, "GetCultivatePlanRPC", func() (*academic.GetPlanResponse, error) {
		return academicClient.GetPlan(ctx, req)
	})
	if err != nil {
		return "", err
	}
	return resp.Url, nil
}

func GetCreditV2RPC(ctx context.Context, req *academic.GetCreditV2Request) (*model.CreditResponse, error) {
	resp, err := call(ctx, "GetCreditV2RPC", func() (*academic.GetCreditV2Response, error) {
		return academicClient.GetCreditV2(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return &resp.Credit, nil
}
