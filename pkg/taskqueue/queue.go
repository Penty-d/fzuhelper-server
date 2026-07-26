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

package taskqueue

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"k8s.io/client-go/util/workqueue"

	"github.com/west2-online/fzuhelper-server/pkg/constants"
	"github.com/west2-online/fzuhelper-server/pkg/logger"
)

// TaskQueueMaxRetries QueueTask 失败重试次数上限，达到后放弃任务，
// 避免永久性失败的任务无限重试且 taskMap/限流器条目永不释放
const TaskQueueMaxRetries = 15

type TaskQueue interface {
	Start()
	Add(key string, task QueueTask)
	AddSchedule(key string, task ScheduleQueueTask)
	worker()
}

// QueueTask 队列任务，使用指数退避和令牌桶限流
type QueueTask struct {
	Execute func() error
}

// ScheduleQueueTask 定时任务
type ScheduleQueueTask struct {
	Execute         func(context.Context) error
	GetScheduleTime func() time.Duration
}

type BaseTaskQueue struct {
	workQueue workqueue.TypedRateLimitingInterface[string]
	taskMap   sync.Map
}

func NewBaseTaskQueue() *BaseTaskQueue {
	return &BaseTaskQueue{
		// 默认限流器
		// - 单任务重试采用指数退避策略：初始延迟为 5ms，最大延迟为 1000 秒。
		// - 整体速率限制：每秒最多 10 次请求，桶大小为 100 个令牌。
		workQueue: workqueue.NewTypedRateLimitingQueue(
			workqueue.DefaultTypedControllerRateLimiter[string](),
		),
	}
}

// Add 想task queue 中添加 task
// ScheduleQueueTask 也实现了 QueueTask 的接口，不需要显示声明
func (btq *BaseTaskQueue) Add(key string, task QueueTask) {
	btq.taskMap.Store(key, task)
	btq.workQueue.Add(key)
}

func (btq *BaseTaskQueue) AddSchedule(key string, task ScheduleQueueTask) {
	btq.taskMap.Store(key, task)
	btq.workQueue.Add(key)
}

func (btq *BaseTaskQueue) Start() {
	for i := 0; i < constants.WorkerNumber; i++ {
		go btq.worker()
	}
}

func (btq *BaseTaskQueue) worker() {
	for {
		key, shutdown := btq.workQueue.Get()
		if shutdown {
			logger.Info("BaseTaskQueue:worker shutdown")
			return
		}

		task, exists := btq.taskMap.Load(key)
		if !exists {
			btq.workQueue.Done(key)
			logger.Warnf("Task evaporated: %s", key)
			continue
		}
		switch task := task.(type) {
		case ScheduleQueueTask:
			if err := btq.executeTask(context.Background(), key, task.Execute, "ScheduleQueueTask"); err != nil {
				btq.workQueue.AddRateLimited(key)
			} else {
				btq.workQueue.AddAfter(key, task.GetScheduleTime())
				btq.workQueue.Forget(key)
			}
			btq.workQueue.Done(key)
		case QueueTask:
			if err := task.Execute(); err != nil {
				if btq.workQueue.NumRequeues(key) >= TaskQueueMaxRetries {
					// 达到重试上限后放弃任务并清理限流器与 taskMap 条目，避免无限重试和内存泄漏
					logger.Errorf("QueueTask %s dropped after %d retries: %v", key, btq.workQueue.NumRequeues(key), err)
					btq.workQueue.Forget(key)
					btq.taskMap.Delete(key)
				} else {
					logger.Errorf("QueueTask execute failed: %v", err)
					btq.workQueue.AddRateLimited(key)
				}
			} else {
				// 成功后必须 Forget，否则限流器中按 key 记录的失败计数会永久残留
				btq.workQueue.Forget(key)
				btq.taskMap.Delete(key)
			}
			btq.workQueue.Done(key)
		default:
			logger.Errorf("BaseTaskQueue:Unknown task type，key: %v", key)
			// 防御性清理：不调用 Done 会导致该 key 永远滞留在 processing 集合，之后同 key 的 Add 无法再被调度
			btq.taskMap.Delete(key)
			btq.workQueue.Done(key)
		}
	}
}

// executeTask 封装一层调用 task.Execute()
func (btq *BaseTaskQueue) executeTask(ctx context.Context, key string, task func(context.Context) error, taskType string) error {
	tracer := otel.Tracer(constants.TaskQueueTracerName)
	ctx, span := tracer.Start(ctx, taskType)
	defer span.End()
	span.SetAttributes(
		attribute.String(constants.AttributeTaskQueueKey, key),
		attribute.String(constants.AttributeTaskQueueType, taskType),
		attribute.Int(constants.AttributeTaskQueueRequeues, btq.workQueue.NumRequeues(key)),
	)

	// execute core
	err := task(ctx)
	if err != nil {
		logger.WithCtx(ctx).Errorf("%s execute failed: %v", taskType, err)
	}
	return err
}
