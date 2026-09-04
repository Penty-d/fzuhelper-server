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

package upyun

import (
	cos "github.com/west2-online/fzuhelper-server/pkg/cos"
)

// SignStr 根据 policy 生成 COS PostObject 的 authorization 串
func SignStr(policy string) string {
	return cos.SignStr(policy)
}

// GetPolicy 生成 COS PostObject 表单上传的 policy(base64)
func GetPolicy() string {
	return cos.GetPolicy()
}

// URlUploadFile 上传文件到 COS
func URlUploadFile(file []byte, url string) error {
	return cos.URlUploadFile(file, url)
}

// URlGetFile 从 COS 下载文件
func URlGetFile(url string) (*[]byte, error) {
	return cos.URlGetFile(url)
}

// JoinFileName 生成文件下载 url
func JoinFileName(fileName string) string {
	return cos.JoinFileName(fileName)
}
