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
	"net"

	"github.com/cloudwego/kitex/pkg/limit"
	"github.com/cloudwego/kitex/pkg/registry"
	"github.com/cloudwego/kitex/pkg/remote/codec/thrift"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/cloudwego/kitex/server"
	kitextracing "github.com/kitex-contrib/obs-opentelemetry/tracing"

	"github.com/west2-online/fzuhelper-server/pkg/constants"
)

// config.go RPC 服务器配置，配置应当只在 cmd 包中调用

// AssembleCommonServerConfig 组装通用 RPC 服务器配置（默认启用 Mux 传输）
func AssembleCommonServerConfig(serviceName string, addr net.Addr, r registry.Registry) []server.Option {
	return assembleServerConfig(serviceName, addr, r, true)
}

// AssembleCommonServerConfigWithoutMux 组装不启用 Mux 传输的通用 RPC 服务器配置；
// 使用 Mux 传输会和流式传输冲突，需要流式传输的服务应使用该函数
func AssembleCommonServerConfigWithoutMux(serviceName string, addr net.Addr, r registry.Registry) []server.Option {
	return assembleServerConfig(serviceName, addr, r, false)
}

// assembleServerConfig 组装通用 RPC 服务器配置，withMux 控制是否启用 Mux 传输
func assembleServerConfig(serviceName string, addr net.Addr, r registry.Registry, withMux bool) []server.Option {
	opts := commonServerConfig()
	opts = append(opts, server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: serviceName}))
	if withMux {
		opts = append(opts, server.WithMuxTransport())
	}
	opts = append(opts,
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithSuite(kitextracing.NewServerSuite()),
	)
	return opts
}

// commonServerConfig 返回通用的服务端配置
func commonServerConfig() []server.Option {
	return []server.Option{
		server.WithLimit(&limit.Option{
			MaxConnections: constants.MaxConnections, // 最大连接数
			MaxQPS:         constants.MaxQPS,         // 最大 QPS
		}),
		server.WithPayloadCodec(thrift.NewThriftCodecWithConfig(thrift.FrugalReadWrite)), // 使用 Frugal 进行解编码
		server.WithMetaHandler(transmeta.ServerTTHeaderHandler),                          // 使用 TTHeader 进行元数据传输
	}
}
