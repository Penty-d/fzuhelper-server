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
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/west2-online/fzuhelper-server/config"
	"github.com/west2-online/fzuhelper-server/internal/common"
	"github.com/west2-online/fzuhelper-server/internal/common/pack"
	commonSvc "github.com/west2-online/fzuhelper-server/internal/common/service"
	"github.com/west2-online/fzuhelper-server/kitex_gen/common/commonservice"
	"github.com/west2-online/fzuhelper-server/pkg/base"
	baseserver "github.com/west2-online/fzuhelper-server/pkg/base/server"
	"github.com/west2-online/fzuhelper-server/pkg/cache"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/db"
	"github.com/west2-online/fzuhelper-server/pkg/db/model"
	"github.com/west2-online/fzuhelper-server/pkg/github"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
	"github.com/west2-online/fzuhelper-server/pkg/taskqueue"
	"github.com/west2-online/fzuhelper-server/pkg/umeng"
	"github.com/west2-online/fzuhelper-server/pkg/upyun"
	"github.com/west2-online/jwch"
)

var (
	serviceName = constants.CommonServiceName
	clientSet   *base.ClientSet
	taskQueue   taskqueue.TaskQueue
)

func init() {
	config.Init(serviceName)
	logger.Init(serviceName, config.GetLoggerLevel())
	clientSet = base.NewClientSet(base.WithDBClient(), base.WithRedisClient(constants.RedisDBCommon))
	taskQueue = taskqueue.NewBaseTaskQueue()
}

func loadNotice(db *db.Database) {
	stu := jwch.NewStudent().WithUser(config.DefaultUser.Account, config.DefaultUser.Password)
	_, totalPage, err := stu.GetNoticeInfo(&jwch.NoticeInfoReq{PageNum: 1})
	if err != nil {
		logger.Errorf("syncer init: failed to get notice info: %v", err)
	}
	// 预热总页数缓存，供 GetNotice 直接读取，避免每次请求都实时爬取教务处
	if totalPage > 0 {
		if err := clientSet.CacheClient.Common.SetNoticeTotalPageCache(context.Background(), totalPage); err != nil {
			logger.Warnf("syncer init: failed to cache notice total page: %v", err)
		}
	}
	// 初始化数据库
	for i := 1; i <= totalPage; i++ {
		content, _, err := stu.GetNoticeInfo(&jwch.NoticeInfoReq{PageNum: i})
		if err != nil {
			logger.Errorf("syncer init: failed to get notice info in page %d: %v", i, err)
		}
		ctx := context.Background()

		// 按页批量查询并在内存中 diff，替代逐条 COUNT + INSERT，减少启动期 DB round-trip
		newRows, err := filterNewNotices(ctx, db, content)
		if err != nil {
			logger.Warnf("syncer init: failed to check notice exists in page %d: %v", i, err)
			continue
		}
		if len(newRows) == 0 {
			continue
		}

		infos := make([]*model.Notice, 0, len(newRows))
		for _, row := range newRows {
			infos = append(infos, &model.Notice{
				Title:       row.Title,
				PublishedAt: row.Date,
				URL:         row.URL,
			})
		}
		if err = db.Notice.CreateNotices(ctx, infos); err != nil {
			logger.Warnf("syncer init: failed to create notices in page %d: %v", i, err)
			continue
		}

		// 走 taskQueue 处理调课通知，天然获得并发上限与失败重试，避免瞬时大量 goroutine 轰击上游
		for _, row := range newRows {
			taskQueue.Add("processNotice:"+row.URL, taskqueue.QueueTask{Execute: func() error {
				return commonSvc.NewCommonService(context.Background(), clientSet, taskQueue).ProcessAutoAdjustCourseNotice(row)
			}})
		}
	}
	logger.Infof("syncer init: notice syncer init success")
}

// filterNewNotices 批量查询数据库中已存在的通知，按 (title, url) 过滤出尚未入库的新通知。
// 注意：判重键是 (title, url) 而表上的唯一约束只有 url，因此标题被教务处改过的通知每轮都会被判为新增，
// INSERT 会被唯一索引吸收但推送仍会重复触发。此判重口径与重构前的 IsNoticeExists 保持一致，未在本次改动中调整
func filterNewNotices(ctx context.Context, database *db.Database, content []*jwch.NoticeInfo) ([]*jwch.NoticeInfo, error) {
	if len(content) == 0 {
		return nil, nil
	}
	urls := make([]string, 0, len(content))
	for _, row := range content {
		urls = append(urls, row.URL)
	}
	existing, err := database.Notice.GetNoticesByUrls(ctx, urls)
	if err != nil {
		return nil, err
	}
	existSet := make(map[[2]string]struct{}, len(existing))
	for _, row := range existing {
		existSet[[2]string{row.Title, row.URL}] = struct{}{}
	}
	newRows := make([]*jwch.NoticeInfo, 0, len(content))
	for _, row := range content {
		if _, ok := existSet[[2]string{row.Title, row.URL}]; !ok {
			newRows = append(newRows, row)
		}
	}
	return newRows, nil
}

func main() {
	svr := commonservice.NewServer(
		common.NewCommonService(clientSet, taskQueue),
		baseserver.MustAssembleServerOptions(serviceName, clientSet.Close)...,
	)

	taskQueue.AddSchedule(constants.NoticeTaskKey, taskqueue.ScheduleQueueTask{
		Execute: syncNoticeTask,
		GetScheduleTime: func() time.Duration {
			return constants.NoticeUpdateTime
		},
	})
	taskQueue.AddSchedule(constants.ContributorTaskKey, taskqueue.ScheduleQueueTask{
		Execute: syncContributorTask,
		GetScheduleTime: func() time.Duration {
			return constants.ContributorInfoUpdateTime
		},
	})

	// 必须在两个定时任务入队之后再灌入通知：任务队列是 FIFO 且只有 WorkerNumber 个 worker，
	// 冷启动时通知数量可达数千条（其中调课通知还要走 HTTP + LLM），
	// 先入队会把定时任务挤到队尾，导致贡献者缓存长时间不被填充、对应接口持续报错
	loadNotice(clientSet.DBClient)

	taskQueue.Start()

	if err := svr.Run(); err != nil {
		logger.Fatalf("Common: server run failed: %v", err)
	}
}

func syncNoticeTask(ctx context.Context) error {
	logger.WithCtx(ctx).Infof("syncNoticeTask: jwch notice sync task started")
	// 默认爬取第一页的内容（教务处不太可能一次性更新出一页的数据），然后和数据库做 diff 操作
	content, totalPage, err := jwch.NewStudent().WithUser(config.DefaultUser.Account, config.DefaultUser.Password).GetNoticeInfo(&jwch.NoticeInfoReq{PageNum: 1})
	if err != nil {
		logger.WithCtx(ctx).Errorf("notice sync task: failed to get notice info: %v", err)
		return fmt.Errorf("failed to get notice info: %w", err)
	}

	// 回填总页数缓存，供 GetNotice 直接读取，避免每次请求都实时爬取教务处；
	// 解析异常时 totalPage 可能为 0，写入会让 GetNotice 在整个 TTL 内都返回 0 导致客户端分页失效
	if totalPage > 0 {
		if err := clientSet.CacheClient.Common.SetNoticeTotalPageCache(ctx, totalPage); err != nil {
			logger.WithCtx(ctx).Warnf("notice sync task: failed to cache notice total page: %v", err)
		}
	}

	// 批量查询并在内存中按 (title, url) diff 出新增通知
	newRows, err := filterNewNotices(ctx, clientSet.DBClient, content)
	if err != nil {
		return fmt.Errorf("notice sync task: failed to check url exists: %w", err)
	}
	if len(newRows) == 0 {
		return nil
	}

	infos := make([]*model.Notice, 0, len(newRows))
	for _, row := range newRows {
		logger.WithCtx(ctx).Infof("syncNoticeTask: new notice found, title=%s url=%s", row.Title, row.URL)
		infos = append(infos, &model.Notice{
			Title:       row.Title,
			URL:         row.URL,
			PublishedAt: row.Date,
		})
	}
	if err = clientSet.DBClient.Notice.CreateNotices(ctx, infos); err != nil {
		return fmt.Errorf("notice sync task: failed to create notice: %w", err)
	}

	for _, row := range newRows {
		// 走 taskQueue 处理调课通知，统一获得并发上限与失败重试
		taskQueue.Add("processNotice:"+row.URL, taskqueue.QueueTask{Execute: func() error {
			return commonSvc.NewCommonService(context.Background(), clientSet, taskQueue).ProcessAutoAdjustCourseNotice(row)
		}})

		// 进行消息推送
		if ok := umeng.EnqueueAsync(func() error {
			deeplink := constants.UmengJwchNoticeDeeplink + "?url=" + url.QueryEscape(row.URL)
			if err := umeng.SendAndroidGroupcastWithGoApp("教务处通知", row.Title, "", constants.UmengJwchNoticeTag, "教务处", deeplink); err != nil {
				logger.WithCtx(ctx).Errorf("notice sync task: failed to send notice to Android: %v", err)
			}

			if err := umeng.SendIOSGroupcast("教务处通知", "", row.Title, constants.UmengJwchNoticeTag, "教务处", deeplink); err != nil {
				logger.WithCtx(ctx).Errorf("notice sync task: failed to send notice to IOS: %v", err)
			}
			logger.WithCtx(ctx).Infof("notice sync task: notice send success")
			return nil
		}); !ok {
			logger.WithCtx(ctx).Errorf("umeng async queue full, drop notice notification")
		}
	}
	return nil
}

func syncContributorTask(ctx context.Context) error {
	logger.WithCtx(ctx).Info("syncContributorTask: contributor info sync task started")
	urls := []string{
		constants.ContributorFzuhelperApp,
		constants.ContributorFzuhelperServer,
		constants.ContributorJwch,
		constants.ContributorYJSY,
	}
	contributorKeys := []string{
		constants.ContributorFzuhelperAppKey,
		constants.ContributorFzuhelperServerKey,
		constants.ContributorJwchKey,
		constants.ContributorYJSYKey,
	}

	for i, url := range urls {
		rawContributors, err := github.FetchContributorsFromURL(url)
		if err != nil {
			return fmt.Errorf("contributor info sync: failed to fetch contributors from %s: %w", url, err)
		}
		contributors := pack.BuildContributors(rawContributors)
		for i, contributor := range contributors {
			newAvatarUrl, err := uploadAvatar(contributor.AvatarUrl, contributor.Name)
			if err != nil {
				return fmt.Errorf("contributor info sync: failed to upload avatar for contributor %s: %w", contributor.Name, err)
			}
			// 替换头像 url
			contributors[i].AvatarUrl = newAvatarUrl
		}
		if err := cache.SetSliceCache(clientSet.CacheClient, ctx,
			contributorKeys[i], contributors,
			constants.KeyNeverExpire, "Common.SyncContributorInfo"); err != nil {
			return fmt.Errorf("contributor info sync: failed to cache contributors: %w", err)
		}
	}

	return nil
}

const (
	baseUrl    = "https://avatars.githubusercontent.com/u/"
	uploadBase = "http://v0.api.upyun.com/fzuhelper-filedown"
	readBase   = "https://download.w2fzu.com"
)

func uploadAvatar(avatarUrl string, name string) (string, error) {
	if strings.HasPrefix(avatarUrl, baseUrl) {
		// 1.将原始 URL 替换成反代 URL
		parsedUrl, err := url.Parse(avatarUrl)
		if err != nil {
			return "", err
		}
		// parsedUrl.Path[3:]会去掉 `/u/`
		newAvatarUrl := fmt.Sprintf(constants.AvatarProxy, parsedUrl.Path[3:])

		// 2.下载图片并上传又拍云
		resp, err := http.Get(newAvatarUrl)
		if err != nil {
			return "", fmt.Errorf("failed to download avatar from %s: %w", avatarUrl, err)
		}
		imgData, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read avatar image: %w", err)
		}
		// 生成上传用Url
		newAvatarUrl = upyun.GenerateContributorAvatarUrl(name)
		err = upyun.URlUploadFile(imgData, newAvatarUrl)
		if err != nil {
			return "", fmt.Errorf("failed to upload avatar to image host: %w", err)
		}
		_ = resp.Body.Close()

		// 3.最终换成加速域名
		return strings.Replace(newAvatarUrl, uploadBase, readBase, 1), nil
	}

	return "", nil
}
