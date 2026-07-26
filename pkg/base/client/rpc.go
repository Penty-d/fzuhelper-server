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

package client

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/client/streamclient"
	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/remote/codec/thrift"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/cloudwego/kitex/transport"
	kitextracing "github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"

	"github.com/west2-online/fzuhelper-server/config"
	"github.com/west2-online/fzuhelper-server/kitex_gen/academic/academicservice"
	"github.com/west2-online/fzuhelper-server/kitex_gen/captcha/captchaservice"
	"github.com/west2-online/fzuhelper-server/kitex_gen/classroom/classroomservice"
	"github.com/west2-online/fzuhelper-server/kitex_gen/common/commonservice"
	"github.com/west2-online/fzuhelper-server/kitex_gen/course/courseservice"
	"github.com/west2-online/fzuhelper-server/kitex_gen/launch_screen/launchscreenservice"
	"github.com/west2-online/fzuhelper-server/kitex_gen/oa/oaservice"
	"github.com/west2-online/fzuhelper-server/kitex_gen/paper/paperservice"
	"github.com/west2-online/fzuhelper-server/kitex_gen/user/userservice"
	"github.com/west2-online/fzuhelper-server/kitex_gen/version/versionservice"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
)

// sharedResolver 所有 RPC 客户端共享同一个 etcd resolver。
// resolver 本身无状态可跨 client 复用，避免每个客户端各自建立一条 etcd 连接（连接/watch/goroutine 全部重复）。
// NOTICE: 调用前必须保证 config.Etcd 已初始化（现有调用顺序满足）；OnceValues 会缓存首次错误，但错误路径最终都会 Fatalf 退出进程
var sharedResolver = sync.OnceValues(func() (discovery.Resolver, error) {
	return etcd.NewEtcdResolver([]string{config.Etcd.Addr})
})

// frugalCodec 使用 Frugal 进行解编码，codec 无状态可全局复用
var frugalCodec = thrift.NewThriftCodecWithConfig(thrift.FrugalReadWrite)

// 通用的RPC客户端初始化函数
func initRPCClient[T any](serviceName string, newClientFunc func(string, ...client.Option) (T, error)) (*T, error) {
	if config.Etcd == nil || config.Etcd.Addr == "" {
		return nil, errors.New("config.Etcd.Addr is nil")
	}
	// 获取共享的 Etcd Resolver
	r, err := sharedResolver()
	if err != nil {
		return nil, fmt.Errorf("initRPCClient etcd.NewEtcdResolver failed: %w", err)
	}
	// 初始化具体的RPC客户端
	client, err := newClientFunc(
		serviceName,
		client.WithResolver(r),
		client.WithMuxConnection(constants.MuxConnection),
		client.WithPayloadCodec(frugalCodec),
		client.WithTransportProtocol(transport.TTHeaderFramed),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
		client.WithSuite(kitextracing.NewClientSuite()),
	)
	if err != nil {
		return nil, fmt.Errorf("initRPCClient NewClient failed: %w", err)
	}
	return &client, nil
}

func InitUserRPC() (*userservice.Client, error) {
	return initRPCClient(constants.UserServiceName, userservice.NewClient)
}

func InitClassroomRPC() (*classroomservice.Client, error) {
	return initRPCClient(constants.ClassroomServiceName, classroomservice.NewClient)
}

func InitCourseRPC() (*courseservice.Client, error) {
	return initRPCClient(constants.CourseServiceName, courseservice.NewClient)
}

func InitLaunchScreenRPC() (*launchscreenservice.Client, error) {
	return initRPCClient(constants.LaunchScreenServiceName, launchscreenservice.NewClient)
}

func InitLaunchScreenStreamRPC() (*launchscreenservice.StreamClient, error) {
	if config.Etcd == nil || config.Etcd.Addr == "" {
		return nil, errors.New("config.Etcd.Addr is nil")
	}
	r, err := sharedResolver()
	if err != nil {
		return nil, fmt.Errorf("InitLaunchScreenStreamRPC etcd.NewEtcdResolver failed: %w", err)
	}
	streamClient := launchscreenservice.MustNewStreamClient(
		constants.LaunchScreenServiceName,
		streamclient.Option(client.WithResolver(r)),
	)
	return &streamClient, nil
}

func InitPaperRPC() (*paperservice.Client, error) {
	return initRPCClient(constants.PaperServiceName, paperservice.NewClient)
}

func InitAcademicRPC() (*academicservice.Client, error) {
	return initRPCClient(constants.AcademicServiceName, academicservice.NewClient)
}

func InitVersionRPC() (*versionservice.Client, error) {
	return initRPCClient(constants.VersionServiceName, versionservice.NewClient)
}

func InitCommonRPC() (*commonservice.Client, error) {
	return initRPCClient(constants.CommonServiceName, commonservice.NewClient)
}

func InitOARPC() (*oaservice.Client, error) {
	return initRPCClient(constants.OAServiceName, oaservice.NewClient)
}

func InitCaptchaRPC() (*captchaservice.Client, error) {
	return initRPCClient(constants.CaptchaServiceName, captchaservice.NewClient)
}
