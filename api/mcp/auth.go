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

package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpgoserver "github.com/mark3labs/mcp-go/server"

	"github.com/west2-online/fzuhelper-server/api/rpc"
	"github.com/west2-online/fzuhelper-server/kitex_gen/user"
	metainfoContext "github.com/west2-online/fzuhelper-server/pkg/base/context"
	"github.com/west2-online/fzuhelper-server/pkg/utils"
	"github.com/west2-online/jwch"
	"github.com/west2-online/yjsy"
)

type IdentifierData struct {
	ID      string `json:"user_id"`
	Cookies string `json:"user_cookies"`
}

func LoginTool() mcpgoserver.ServerTool {
	return mcpgoserver.ServerTool{
		Tool: mcp.NewTool("login",
			mcp.WithDescription("Use this tool when the user wants to log into the educational system. "+
				"Call this when: user mentions logging in, needs to authenticate, "+
				"or when other tools fail due to no active session. "+
				"Returns success message on successful login."),
			mcp.WithString("student_id",
				mcp.Required(),
				mcp.Description("Student ID for authentication"),
			),
			mcp.WithString("password",
				mcp.Required(),
				mcp.Description("Password for authentication"),
			),
			mcp.WithString("student_type",
				mcp.Description("StudentType for authentication. Defaults to \"1\" (Undergraduate student). "+
					"Set \"2\" for Postgraduate"),
			),
		),
		Handler: handleLogin,
	}
}

func CheckSessionTool() mcpgoserver.ServerTool {
	return mcpgoserver.ServerTool{
		Tool: mcp.NewTool("check_session",
			mcp.WithDescription("Use this tool to verify if the current login session is still valid. "+
				"Call this when: user asks about connection status, before performing operations after a long idle "+
				"period, or to troubleshoot authentication issues. Returns session validity status."),
			mcp.WithString("user_id",
				mcp.Required(),
				mcp.Description("user_id data comes from login method response, (user_cookies field)"),
			),
			mcp.WithString("user_cookies",
				mcp.Required(),
				mcp.Description("user_cookies data comes from login method response, (user_cookies field)"),
			),
		),
		Handler: handleCheckSession,
	}
}

func handleLogin(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	studentID := request.GetString("student_id", "")
	password := request.GetString("password", "")
	studentType := request.GetString("student_type", "")
	if studentType == "" {
		studentType = "1" // 默认本科生
	}
	if studentID == "" {
		return mcp.NewToolResultError("student_id is required"), nil
	}
	if password == "" {
		return mcp.NewToolResultError("password is required"), nil
	}

	var id, cookies string
	var err error
	switch studentType {
	case "2":
		id, cookies, err = rpc.GetLoginDataForYJSYRPC(ctx, &user.GetLoginDataForYJSYRequest{
			Id:       studentID,
			Password: password,
		})
	default:
		id, cookies, err = rpc.GetLoginDataRPC(ctx, &user.GetLoginDataRequest{
			Id:       studentID,
			Password: password,
		})
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Internal RPC request failed: %v", err)), nil
	}

	return mcp.NewToolResultJSON(IdentifierData{
		ID:      id,
		Cookies: cookies,
	})
}

func handleCheckSession(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 验证认证参数
	auth, errResult := ValidateAuthParams(request)
	if errResult != nil {
		return errResult, nil
	}

	var err error
	if utils.IsGraduate(auth.UserID) {
		// 研究生：使用 yjsy 库
		id := metainfoContext.ExtractIDFromIdentifier(auth.UserID)
		err = yjsy.NewStudent().WithUser(id, "").WithLoginData(utils.ParseCookies(auth.UserCookies)).CheckSession()
	} else {
		// 本科生：使用 jwch 库
		// 使用带长度校验的 ExtractIDFromIdentifier，避免过短的 user_id 触发越界 panic
		id := metainfoContext.ExtractIDFromIdentifier(auth.UserID)
		err = jwch.NewStudent().WithUser(id, "").WithLoginData(auth.UserID, utils.ParseCookies(auth.UserCookies)).CheckSession()
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("CheckSession failed: %v", err)), nil
	}
	return mcp.NewToolResultText("Session alive"), nil
}
