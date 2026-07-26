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
	"flag"
	"fmt"
	"time"

	"github.com/west2-online/fzuhelper-server/config"
	"github.com/west2-online/fzuhelper-server/internal/academic"
	"github.com/west2-online/fzuhelper-server/internal/academic/service"
	"github.com/west2-online/fzuhelper-server/kitex_gen/academic/academicservice"
	"github.com/west2-online/fzuhelper-server/pkg/base"
	baseserver "github.com/west2-online/fzuhelper-server/pkg/base/server"
	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
	"github.com/west2-online/fzuhelper-server/pkg/taskqueue"
	"github.com/west2-online/fzuhelper-server/pkg/utils"
)

var (
	serviceName = constants.AcademicServiceName
	clientSet   *base.ClientSet
	taskQueue   taskqueue.TaskQueue
	runTask     = flag.String("run-task", "", "manually run a specific task and exit")
)

func init() {
	config.Init(serviceName)
	logger.Init(serviceName, config.GetLoggerLevel())
	clientSet = base.NewClientSet(base.WithDBClient(), base.WithRedisClient(constants.RedisDBAcademic))
	taskQueue = taskqueue.NewBaseTaskQueue()
}

func main() {
	flag.Parse()

	if *runTask != "" {
		if err := runManualTask(*runTask); err != nil {
			logger.Fatalf("Academic: manual task %s failed: %v", *runTask, err)
		}
		logger.Infof("Academic: manual task %s completed successfully", *runTask)
		return
	}

	svr := academicservice.NewServer(
		academic.NewAcademicService(clientSet, taskQueue),
		baseserver.MustAssembleServerOptions(serviceName, clientSet.Close)...,
	)

	taskQueue.AddSchedule(constants.CourseTeacherScoresTaskKey, taskqueue.ScheduleQueueTask{
		Execute: updateCourseTeacherScoresTask,
		GetScheduleTime: func() time.Duration {
			// 每天凌晨4点
			return utils.DurationUntilNextDaily(
				constants.CourseTeacherScoresRefreshHour,
				constants.CourseTeacherScoresRefreshMinute,
				constants.ChinaTZ,
			)
		},
	})

	taskQueue.Start()
	if err := svr.Run(); err != nil {
		logger.Fatalf("Academic: server run failed: %v", err)
	}
}

func runManualTask(taskName string) error {
	ctx := context.Background()
	switch taskName {
	case constants.CourseTeacherScoresTaskKey:
		return updateCourseTeacherScoresTask(ctx)
	default:
		return fmt.Errorf("unknown task: %s", taskName)
	}
}

func updateCourseTeacherScoresTask(ctx context.Context) error {
	logger.WithCtx(ctx).Infof("Academic: update course teacher scores task start")
	svc := service.NewAcademicService(ctx, clientSet, nil)
	if err := svc.UpdateCourseTeacherScores(); err != nil {
		logger.WithCtx(ctx).Errorf("Academic: update course teacher scores task failed: %v", err)
		return err
	}
	logger.WithCtx(ctx).Infof("Academic: update course teacher scores task finished")
	return nil
}
