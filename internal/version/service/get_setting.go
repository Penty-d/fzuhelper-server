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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/west2-online/fzuhelper-server/internal/version/pack"
	"github.com/west2-online/fzuhelper-server/kitex_gen/version"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
	"github.com/west2-online/fzuhelper-server/pkg/singleflight"
	"github.com/west2-online/fzuhelper-server/pkg/upyun"
)

func (s *VersionService) GetCloudSetting(req *version.GetSettingRequest) (*[]byte, error) {
	date := time.Now().In(constants.ChinaTZ).Format("2006-01-02")
	err := s.cache.Version.AddVisit(s.ctx, date)
	if err != nil {
		return nil, fmt.Errorf("VersionService.GetCloudSetting AddVisit error:%w", err)
	}

	// 云控配置为全局数据，用 singleflight 合并并发请求的下载与解析；AddVisit 计数保持逐请求执行不受影响
	cloudSettings, err := singleflight.Do(constants.SingleflightCloudSettingKey, func() (*pack.CloudSetting, error) {
		// 获得Json
		settingJson, err := upyun.URlGetFile(upyun.JoinFileName(cloudSettingFileName))
		if err != nil {
			return nil, err
		}
		noCommentSettingJson, err := getJSONWithoutComments(string(*settingJson))
		if err != nil {
			return nil, err
		}

		// 绑定结构体
		settings := new(pack.CloudSetting)
		if err = json.Unmarshal([]byte(noCommentSettingJson), settings); err != nil {
			return nil, err
		}
		return settings, nil
	})
	if err != nil {
		return nil, fmt.Errorf("VersionService.GetCloudSetting error:%w", err)
	}

	criteria := &pack.Plan{
		Account:   req.Account,
		Version:   req.Version,
		Beta:      req.Beta,
		Phone:     req.Phone,
		IsLogin:   req.IsLogin,
		LoginType: req.LoginType,
	}
	plan, err := findMatchingPlan(&cloudSettings.Plans, criteria)
	if err != nil {
		return nil, fmt.Errorf("VersionService.GetCloudSetting error:%w", err)
	}
	returnPlan := []byte(plan.Plan)
	return &returnPlan, nil
}

// findMatchingPlan 查找匹配的计划,若无传递字段则默认该字段为匹配状态，出现未匹配则直接查找下一计划
func findMatchingPlan(planList *[]pack.Plan, criteria *pack.Plan) (*pack.Plan, error) {
	for _, plan := range *planList {
		if !matchRegexField(plan.Name, criteria.Name) {
			continue
		}
		if !matchRegexField(plan.Account, criteria.Account) {
			continue
		}
		if !matchRegexField(plan.Version, criteria.Version) {
			continue
		}
		if !matchRegexField(plan.Phone, criteria.Phone) {
			continue
		}
		if !matchRegexField(plan.LoginType, criteria.LoginType) {
			continue
		}
		if !matchBoolField(plan.Beta, criteria.Beta) {
			continue
		}
		if !matchBoolField(plan.IsLogin, criteria.IsLogin) {
			continue
		}
		return &plan, nil
	}
	return nil, errno.NoMatchingPlanError
}

// matchRegexField 按正则匹配单个字段，任一侧未传递(nil)则视为匹配
func matchRegexField(pattern, value *string) bool {
	if pattern == nil || value == nil {
		return true
	}
	matched, _ := regexp.MatchString(*pattern, *value)
	return matched
}

// matchBoolField 比较单个布尔字段，任一侧未传递(nil)则视为匹配
func matchBoolField(want, got *bool) bool {
	return want == nil || got == nil || *want == *got
}

// getJSONWithoutComments 获得没有注释的jsonStr
func getJSONWithoutComments(input string) (string, error) {
	lines := strings.Split(input, "\n")
	var cleanLines []string

	for _, line := range lines {
		cleanLine := removeComments(line)
		if cleanLine != "" {
			cleanLines = append(cleanLines, cleanLine)
		}
	}

	jsonStr := strings.Join(cleanLines, "\n")
	return jsonStr, nil
}

// removeComments 用于去除JSON文件里的注释("//"且不会去掉url的"//")
func removeComments(line string) string {
	inString := false
	stringChar := ""

	for i := 0; i < len(line); i++ {
		// 处理字符串边界
		if line[i] == '"' || line[i] == '\'' {
			if !inString {
				inString = true
				stringChar = string(line[i])
			} else if stringChar == string(line[i]) {
				inString = false
			}
		}
		// 检查是否是注释
		if i+1 < len(line) && line[i:i+2] == "//" && !inString {
			return line[:i]
		}
	}

	return line
}
