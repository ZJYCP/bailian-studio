package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"bailian-studio/server/internal/model"
	"bailian-studio/server/internal/service"
)

func (a *API) createTask(c *gin.Context) {
	var req service.CreateTaskInput
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}
	task, err := a.Svc.CreateTask(req)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	// 附带产物（重试场景为空）
	a.attachAssets([]*model.Task{task})
	c.JSON(http.StatusOK, task)
}

func (a *API) listTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	q := a.Svc.DB.Model(&model.Task{})
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	if cap := c.Query("capability"); cap != "" {
		q = q.Where("capability = ?", cap)
	}
	if active := c.Query("active"); active == "true" {
		q = q.Where("status IN ?", []string{model.TaskQueued, model.TaskSubmitting, model.TaskRunning})
	}
	var total int64
	q.Count(&total)
	var tasks []model.Task
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&tasks).Error; err != nil {
		serverError(c, err)
		return
	}
	ptrs := make([]*model.Task, len(tasks))
	for i := range tasks {
		ptrs[i] = &tasks[i]
	}
	a.attachAssets(ptrs)
	c.JSON(http.StatusOK, gin.H{"total": total, "items": tasks, "page": page, "page_size": size})
}

// attachAssets 批量挂载产物与输入素材，避免前端 N+1
func (a *API) attachAssets(tasks []*model.Task) {
	if len(tasks) == 0 {
		return
	}
	ids := make([]uint, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.ID)
	}
	var assets []model.Asset
	a.Svc.DB.Where("task_id IN ?", ids).Find(&assets)
	byTask := map[uint][]model.Asset{}
	for _, as := range assets {
		byTask[*as.TaskID] = append(byTask[*as.TaskID], as)
	}
	for _, t := range tasks {
		t.Outputs = filterRole(byTask[t.ID], "output")
		t.Inputs = filterRole(byTask[t.ID], "input")
	}
}

func filterRole(in []model.Asset, role string) []model.Asset {
	var out []model.Asset
	for _, a := range in {
		if a.Role == role {
			out = append(out, a)
		}
	}
	return out
}

func (a *API) getTask(c *gin.Context) {
	var t model.Task
	if err := a.Svc.DB.First(&t, c.Param("id")).Error; err != nil {
		notFound(c, "任务不存在")
		return
	}
	a.attachAssets([]*model.Task{&t})
	c.JSON(http.StatusOK, t)
}

func (a *API) retryTask(c *gin.Context) {
	task, err := a.Svc.RetryTask(parseID(c))
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	a.attachAssets([]*model.Task{task})
	c.JSON(http.StatusOK, task)
}

func (a *API) cancelTask(c *gin.Context) {
	var t model.Task
	if err := a.Svc.DB.First(&t, parseID(c)).Error; err != nil {
		notFound(c, "任务不存在")
		return
	}
	if !t.Active() {
		badRequest(c, "任务已结束")
		return
	}
	t.Status = model.TaskCanceled
	a.Svc.DB.Save(&t)
	c.JSON(http.StatusOK, t)
}

func (a *API) deleteTask(c *gin.Context) {
	id := parseID(c)
	var t model.Task
	if err := a.Svc.DB.First(&t, id).Error; err != nil {
		notFound(c, "任务不存在")
		return
	}
	if t.Active() {
		badRequest(c, "任务仍在进行中，请先取消")
		return
	}
	// 删除产物文件与记录
	var assets []model.Asset
	a.Svc.DB.Where("task_id = ?", id).Find(&assets)
	for _, as := range assets {
		a.Svc.RemoveAssetFile(&as)
	}
	a.Svc.DB.Where("task_id = ?", id).Delete(&model.Asset{})
	a.Svc.DB.Delete(&t)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func parseID(c *gin.Context) uint {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id)
}
