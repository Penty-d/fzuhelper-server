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

package main

import (
	"github.com/west2-online/fzuhelper-server/config"
	"github.com/west2-online/fzuhelper-server/internal/paper"
	"github.com/west2-online/fzuhelper-server/kitex_gen/paper/paperservice"
	"github.com/west2-online/fzuhelper-server/pkg/base"
	baseserver "github.com/west2-online/fzuhelper-server/pkg/base/server"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
	"github.com/west2-online/fzuhelper-server/pkg/upyun"
)

var (
	serviceName = constants.PaperServiceName
	clientSet   *base.ClientSet
)

func init() {
	config.Init(serviceName)
	logger.Init(serviceName, config.GetLoggerLevel())
	// eshook.InitLoggerWithHook(serviceName)
	clientSet = base.NewClientSet(base.WithRedisClient(constants.RedisDBPaper))
	upyun.NewUpYun()
}

func main() {
	svr := paperservice.NewServer(
		paper.NewPaperService(clientSet),
		baseserver.MustAssembleServerOptions(serviceName, clientSet.Close)...,
	)

	if err := svr.Run(); err != nil {
		logger.Fatalf("Paper: server run failed: %v", err)
	}
}
