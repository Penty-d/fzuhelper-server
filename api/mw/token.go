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

package mw

import (
	"crypto"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/west2-online/fzuhelper-server/config"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/errno"
)

type Claims struct {
	StudentID string `json:"student_id"`
	Type      int64  `json:"type"`
	jwt.RegisteredClaims
}

// pubKey 缓存解析后的公钥，公钥是编译期常量，进程内只需解析一次
var pubKey = sync.OnceValues(func() (crypto.PublicKey, error) {
	return jwt.ParseEdPublicKeyFromPEM([]byte(constants.PublicKey))
})

// privKeyEntry 记录私钥 PEM 与其解析结果，用于避免每次签发 token 都重复做 PEM 解析
type privKeyEntry struct {
	pem string
	key crypto.PrivateKey
}

var privKeyCache atomic.Pointer[privKeyEntry]

// parsePrivateKey 解析私钥，私钥可能随配置热更变化，因此按 PEM 内容缓存
func parsePrivateKey(pem string) (crypto.PrivateKey, error) {
	if entry := privKeyCache.Load(); entry != nil && entry.pem == pem {
		return entry.key, nil
	}
	key, err := jwt.ParseEdPrivateKeyFromPEM([]byte(pem))
	if err != nil {
		return nil, err
	}
	privKeyCache.Store(&privKeyEntry{pem: pem, key: key})
	return key, nil
}

// CreateAllToken 创建一对 token，第一个是 access token，第二个是 refresh token
func CreateAllToken() (string, string, error) {
	accessToken, err := CreateToken(constants.TypeAccessToken, "")
	if err != nil {
		return "", "", err
	}
	refreshToken, err := CreateToken(constants.TypeRefreshToken, "")
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// CreateToken 会通过不同 Token 类型创建不同的 Token
func CreateToken(tokenType int64, stuID string) (string, error) {
	if config.Server == nil {
		return "", errno.AuthError.WithMessage("server config not found")
	}

	var expireTime time.Time
	nowTime := time.Now()
	var token string
	var err error

	switch tokenType {
	case constants.TypeAccessToken:
		expireTime = nowTime.Add(constants.AccessTokenTTL)
	case constants.TypeRefreshToken:
		expireTime = nowTime.Add(constants.RefreshTokenTTL)
	case constants.TypeCalendarToken:
		expireTime = nowTime.Add(constants.CalendarTokenTTL)
	}
	claims := Claims{
		StudentID: stuID,
		Type:      tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime), // 过期时间戳
			IssuedAt:  jwt.NewNumericDate(nowTime),    // 当前时间戳
			Issuer:    constants.Issuer,               // 颁发者签名
		},
	}

	// 选择 Ed25519 是出于兼顾性能和安全性的考虑，PS512 安全性太高但性能不好，ES512 速度没有 Ed25519 快
	// 这里不考虑旧版的对称加密
	tokenStruct := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	key, err := parsePrivateKey(config.Server.Secret)
	if err != nil {
		return "", errno.AuthError.WithMessage(fmt.Sprintf("parse private key failed, err: %v", err))
	}

	token, err = tokenStruct.SignedString(key)
	if err != nil {
		return "", errno.AuthError.WithMessage(fmt.Sprintf("sign token failed, err: %v", err))
	}
	return token, nil
}

// CheckToken 会检查 token 是否有效，如果有效则返回 token 类型，否则返回错误(type 会返回 -1)
// Check 成功后返回 token 中的 stu_id
func CheckToken(token string) (int64, string, error) {
	if token == "" {
		return -1, "", errno.AuthMissing
	}

	secret, err := pubKey()
	if err != nil {
		return -1, "", errno.AuthError.WithMessage(fmt.Sprintf("parse public key failed, err: %v", err))
	}

	// 解析并验证 token，验证失败时返回的 token 中同样携带已解析的 claims，无需先做一次 ParseUnverified
	response, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, errno.AuthError.WithMessage(fmt.Sprintf("unexpected signing method: %v", token.Header["alg"]))
		}
		return secret, nil
	})
	// 验证 token 是否有效
	if err != nil {
		// 畸形 token 无法解析出 claims，保持与旧实现（ParseUnverified 失败）一致的错误码
		var ve *jwt.ValidationError
		if response == nil || (errors.As(err, &ve) && ve.Errors&jwt.ValidationErrorMalformed != 0) {
			return -1, "", errno.AuthInvalid.WithError(err)
		}
		claims, ok := response.Claims.(*Claims)
		if !ok {
			return -1, "", errno.AuthError.WithMessage("cannot handle claims")
		}
		return claims.Type, "", checkError(err, claims.Type)
	}

	if claims, ok := response.Claims.(*Claims); ok && response.Valid {
		return claims.Type, claims.StudentID, nil
	}

	return -1, "", errno.AuthInvalid
}

// checkError 会检查错误类型并返回对应的错误(含过期)
func checkError(err error, tokenType int64) error {
	var ve *jwt.ValidationError
	if errors.As(err, &ve) {
		if ve.Errors&jwt.ValidationErrorExpired != 0 {
			if tokenType == constants.TypeAccessToken {
				return errno.AuthAccessExpired
			}
			return errno.AuthRefreshExpired
		}
	}
	return errno.AuthError.WithMessage(err.Error())
}
