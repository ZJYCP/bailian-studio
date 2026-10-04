package service

import (
	"context"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"time"

	"bailian-studio/server/internal/dashscope"
	"bailian-studio/server/internal/model"
)

// StartPoller 启动异步任务轮询器（5s 一轮，跟踪 running 状态的远端任务）
func (s *Service) StartPoller(ctx context.Context) {
	// 启动恢复：上次运行中被打断的 submitting 任务标记失败
	s.DB.Model(&model.Task{}).
		Where("status = ? AND (external_task_id IS NULL OR external_task_id = '')", model.TaskSubmitting).
		Updates(map[string]any{"status": model.TaskFailed, "error_code": "interrupted", "error_message": "服务重启，任务提交中断，可点击重试"})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.pollOnce(ctx)
			}
		}
	}()
}

func (s *Service) pollOnce(ctx context.Context) {
	var tasks []model.Task
	if err := s.DB.Where("status = ? AND external_task_id <> ''", model.TaskRunning).Limit(50).Find(&tasks).Error; err != nil {
		return
	}
	providerCache := map[uint]*providerClient{}
	for i := range tasks {
		select {
		case <-ctx.Done():
			return
		default:
		}
		t := &tasks[i]
		pc, ok := providerCache[t.ProviderID]
		if !ok {
			p, err := s.ProviderByID(t.ProviderID)
			if err != nil {
				s.failTask(t, "provider_missing", err.Error())
				continue
			}
			pc = &providerClient{p, s.ClientForProvider(p)}
			providerCache[t.ProviderID] = pc
		}
		s.pollTask(ctx, pc, t)
		time.Sleep(200 * time.Millisecond) // 查询接口默认限 20 RPS
	}
}

type providerClient struct {
	provider *model.Provider
	client   *dashscope.Client
}

func (s *Service) pollTask(ctx context.Context, pc *providerClient, task *model.Task) {
	qCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	st, err := pc.client.GetTask(qCtx, task.ExternalTaskID)
	if err != nil {
		log.Printf("[task %d] 查询失败: %v", task.ID, err)
		return // 网络抖动等下轮再查
	}
	switch st.Output.TaskStatus {
	case "SUCCEEDED":
		var urls []string
		for _, r := range st.Output.Results {
			if r.URL != "" {
				urls = append(urls, r.URL)
			}
		}
		if st.Output.VideoURL != "" {
			urls = append(urls, st.Output.VideoURL)
		}
		if st.Output.AudioURL != "" {
			urls = append(urls, st.Output.AudioURL)
		}
		task.ResultURLs = toJSON(urls)
		s.SaveUsageRecord(task, st.Usage)
		if len(urls) == 0 {
			s.failTask(task, "empty_result", "任务成功但未返回结果 URL")
			return
		}
		kind := "image"
		if task.Capability == "video" {
			kind = "video"
		}
		ok := true
		for i, u := range urls {
			if _, err := s.downloadOutput(ctx, task, u, kind, i); err != nil {
				s.failTask(task, "download_failed", err.Error())
				ok = false
				break
			}
		}
		if ok {
			now := time.Now()
			task.Status = model.TaskSucceeded
			task.FinishedAt = &now
			s.DB.Save(task)
			log.Printf("[task %d] 完成，产物 %d 个", task.ID, len(urls))
		}
	case "FAILED", "CANCELED", "UNKNOWN":
		code := st.Output.Code
		if code == "" {
			code = st.Output.TaskStatus
		}
		msg := st.Output.Message
		if msg == "" {
			msg = "远端任务 " + st.Output.TaskStatus
		}
		s.failTask(task, code, msg)
	default:
		// PENDING / RUNNING：更新 usage 计量（若有）
		if len(st.Usage) > 0 {
			task.Usage = toJSON(st.Usage)
			s.DB.Model(task).Update("usage", task.Usage)
		}
	}
}

// probeImage 读取图片宽高
func probeImage(path string, a *model.Asset) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err == nil {
		a.Width = cfg.Width
		a.Height = cfg.Height
	}
}
