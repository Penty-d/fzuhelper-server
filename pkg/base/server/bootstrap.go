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

package server

import (
	"fmt"
	"net"

	"github.com/cloudwego/kitex/pkg/registry"
	"github.com/cloudwego/kitex/server"
	"github.com/cloudwego/netpoll"
	etcd "github.com/kitex-contrib/registry-etcd"

	"github.com/west2-online/fzuhelper-server/config"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
	"github.com/west2-online/fzuhelper-server/pkg/tracing"
	"github.com/west2-online/fzuhelper-server/pkg/utils"
)

// bootstrap.go RPC 服务器启动样板，供各 cmd main 复用

// MustAssembleServerOptions 完成 otel provider、etcd 注册、端口解析与 shutdown hook 注册，
// 并返回组装好的通用服务器配置；任一步骤失败会直接 Fatalf 退出
func MustAssembleServerOptions(serviceName string, shutdownHooks ...func()) []server.Option {
	addr, r := mustPrepareServer(serviceName, shutdownHooks...)
	return AssembleCommonServerConfig(serviceName, addr, r)
}

// MustAssembleServerOptionsWithoutMux 同 MustAssembleServerOptions，但不启用 Mux 传输，
// 供需要流式传输的服务使用
func MustAssembleServerOptionsWithoutMux(serviceName string, shutdownHooks ...func()) []server.Option {
	addr, r := mustPrepareServer(serviceName, shutdownHooks...)
	return AssembleCommonServerConfigWithoutMux(serviceName, addr, r)
}

// mustPrepareServer 初始化 otel provider、etcd 注册中心与监听地址，并注册 shutdown hook
func mustPrepareServer(serviceName string, shutdownHooks ...func()) (net.Addr, registry.Registry) {
	// Open Telemetry provider
	shutdown := tracing.NewOtelProvider(serviceName, config.Otel.Endpoint)

	r, err := etcd.NewEtcdRegistry([]string{config.Etcd.Addr})
	if err != nil {
		logger.Fatalf("%s: new etcd registry failed, err: %v", serviceName, err)
	}
	listenAddr, err := utils.GetAvailablePort()
	if err != nil {
		logger.Fatalf("%s: get available port failed, err: %v", serviceName, err)
	}
	addr, err := netpoll.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		logger.Fatalf("%s: resolve tcp addr failed, err: %v", serviceName, err)
	}

	for _, hook := range shutdownHooks {
		server.RegisterShutdownHook(hook)
	}
	server.RegisterShutdownHook(tracing.ProviderShutdown(shutdown,
		fmt.Sprintf("%s: otel provider shutdown failed: %%v", serviceName))) // otel provider
	return addr, r
}
