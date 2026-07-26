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
	"github.com/west2-online/fzuhelper-server/internal/captcha"
	"github.com/west2-online/fzuhelper-server/kitex_gen/captcha/captchaservice"
	"github.com/west2-online/fzuhelper-server/pkg/base"
	baseserver "github.com/west2-online/fzuhelper-server/pkg/base/server"
	captchapkg "github.com/west2-online/fzuhelper-server/pkg/captcha"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
)

var (
	serviceName = constants.CaptchaServiceName
	clientSet   *base.ClientSet
)

func init() {
	config.Init(serviceName)
	logger.Init(serviceName, config.GetLoggerLevel())
	clientSet = base.NewClientSet()
	if err := captchapkg.Init(); err != nil {
		logger.Fatalf("Captcha: init captcha templates failed, err: %v", err)
	}
}

func main() {
	svr := captchaservice.NewServer(
		captcha.NewCaptchaService(clientSet),
		baseserver.MustAssembleServerOptions(serviceName, clientSet.Close)...,
	)

	if err := svr.Run(); err != nil {
		logger.Fatalf("Captcha: server run failed, err: %v", err)
	}
}
