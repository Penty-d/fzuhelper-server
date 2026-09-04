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

package cos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/west2-online/fzuhelper-server/config"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
)

// JoinFileName 生成文件下载 url(EO 加速域名 + 配置路径 + 文件名)
func JoinFileName(fileName string) string {
	return strings.Join([]string{
		config.Cos.DownloadDomain, config.Cos.Path,
		fileName,
	}, "")
}

// URlUploadFile 上传文件到 COS,url 为 JoinFileName 生成的下载地址,内部解析出对象 key
func URlUploadFile(file []byte, url string) error {
	key, err := keyFromURL(url)
	if err != nil {
		return err
	}
	_, err = GetClient().Object.Put(context.Background(), key, bytes.NewReader(file), nil)
	if err != nil {
		logger.Errorf("URlUploadFile: put %s failed: %v", key, err)
		return errno.UpcloudError
	}
	return nil
}

// URlGetFile 从 COS 下载文件(服务端直连 COS,不走 EO)
func URlGetFile(url string) (*[]byte, error) {
	key, err := keyFromURL(url)
	if err != nil {
		return nil, err
	}
	resp, err := GetClient().Object.Get(context.Background(), key, nil)
	if err != nil {
		logger.Errorf("URlGetFile: get %s failed: %v", key, err)
		return nil, errno.UpcloudError
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warnf("URlGetFile: failed to close response body: %v", err)
		}
	}()

	file, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errno.UpcloudError
	}
	return &file, nil
}

// GetPolicy 生成 COS PostObject 表单上传的 policy(base64)。
// 供 Android 客户端直传使用,返回值语义与原又拍云实现保持一致(仍为 base64 策略串)。
func GetPolicy() string {
	start := time.Now().Unix()
	end := start + config.CosUpload.TokenTimeout
	keyTime := fmt.Sprintf("%d;%d", start, end)

	policy := map[string]interface{}{
		"expiration": time.Unix(end, 0).UTC().Format("2006-01-02T15:04:05.000Z"),
		"conditions": []interface{}{
			map[string]string{"bucket": config.CosUpload.Bucket},
			[]string{"starts-with", "$key", config.CosUpload.Path},
			map[string]string{"q-sign-algorithm": "sha1"},
			map[string]string{"q-ak": config.CosUpload.SecretID},
			map[string]string{"q-sign-time": keyTime},
		},
	}

	policyJSON, _ := json.Marshal(policy)
	return base64.StdEncoding.EncodeToString(policyJSON)
}

// SignStr 根据policy生成 COS PostObject 的 authorization 串,格式为
// "q-sign-algorithm=sha1&q-ak=...&q-key-time=...&q-sign-time=...&q-signature=...",
// 客户端将其按 '&' 拆分后作为表单字段随 policy 一并提交。
// 签名算法:signKey = hex(hmac_sha1(SecretKey, keyTime)),
// stringToSign = "sha1\n<keyTime>\n<sha1hex(policy)>\n",
// signature = hex(hmac_sha1(signKey, stringToSign))。
func SignStr(policy string) string {
	keyTime := keyTimeFromPolicy(policy)
	signKey := hmacSHA1Hex(config.CosUpload.SecretKey, keyTime)
	stringToSign := strings.Join([]string{
		"sha1",
		keyTime,
		sha1Hex(policy),
		"",
	}, "\n")
	signature := hmacSHA1Hex(signKey, stringToSign)

	return strings.Join([]string{
		"q-sign-algorithm=sha1",
		"q-ak=" + config.CosUpload.SecretID,
		"q-key-time=" + keyTime,
		"q-sign-time=" + keyTime,
		"q-signature=" + signature,
	}, "&")
}

// keyTimeFromPolicy 从 base64 编码的 policy 中解析 q-sign-time,
// 保证签名与 policy 中的条件严格一致
func keyTimeFromPolicy(policy string) string {
	policyJSON, err := base64.StdEncoding.DecodeString(policy)
	if err != nil {
		return ""
	}

	var p struct {
		Conditions []map[string]string `json:"conditions"`
	}
	if err := json.Unmarshal(policyJSON, &p); err != nil {
		return ""
	}
	for _, cond := range p.Conditions {
		if v, ok := cond["q-sign-time"]; ok {
			return v
		}
	}
	return ""
}

func hmacSHA1Hex(key, msg string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}

func sha1Hex(msg string) string {
	h := sha1.New()
	h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}
