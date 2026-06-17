package medical

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/patch-pet/patch-pet/pkg/constants"
	"github.com/patch-pet/patch-pet/pkg/utils"
	"github.com/patch-pet/patch-pet/pkg/types"
)

// Handler 医疗模块 HTTP Handler
type Handler struct {
	repo *Repository
}

// NewHandler 创建医疗 Handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	pets := rg.Group("/pets/:pet_id")
	{
		pets.GET("/medical-episodes/current", h.GetCurrentEpisode)
		pets.GET("/medical-episodes/list", h.ListEpisodes)
	}

	episodes := rg.Group("/medical-episodes")
	{
		episodes.POST("/:episode_id/care-tasks/draft", h.DraftCareTasks)
		episodes.POST("/:episode_id/summaries", h.GenerateSummary)
		episodes.POST("/:episode_id/authorizations", h.CreateAuthorization)
	}

	summaries := rg.Group("/medical-summaries")
	{
		summaries.POST("/:summary_id/send-to-clinic", h.SendToClinic)
	}

	tasks := rg.Group("/care-tasks")
	{
		tasks.POST("/:task_id/complete", h.CompleteTask)
	}
}

// GetCurrentEpisode 获取宠物当前疗程
// GET /api/v1/pets/:pet_id/medical-episodes/current
func (h *Handler) GetCurrentEpisode(c *gin.Context) {
	petID := c.Param("pet_id")
	if petID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "pet_id 不能为空",
			"data":    nil,
		})
		return
	}

	episode, err := h.repo.GetCurrentEpisode(c.Request.Context(), petID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询当前疗程失败",
			"data":    nil,
		})
		return
	}

	if episode == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "未找到当前疗程",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "ok",
		"data":    episode,
	})
}

// ListEpisodes 分页查询历史疗程
// GET /api/v1/pets/:pet_id/medical-episodes/list
func (h *Handler) ListEpisodes(c *gin.Context) {
	petID := c.Param("pet_id")
	if petID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "pet_id 不能为空",
			"data":    nil,
		})
		return
	}

	// 使用分页中间件注入的分页参数
	pageNum := 1
	pageSize := 20
	if pn, exists := c.Get("pageNum"); exists {
		pageNum = pn.(int)
	}
	if ps, exists := c.Get("pageSize"); exists {
		pageSize = ps.(int)
	}

	episodes, total, err := h.repo.ListEpisodes(c.Request.Context(), petID, pageNum, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询疗程列表失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "ok",
		"data": gin.H{
			"list":     episodes,
			"total":    total,
			"pageNum":  pageNum,
			"pageSize": pageSize,
		},
	})
}

// DraftCareTasksRequest 生成医嘱任务草稿请求
type DraftCareTasksRequest struct {
	Tasks []DraftTaskItem `json:"tasks" binding:"required,min=1"`
}

// DraftTaskItem 草稿任务项
type DraftTaskItem struct {
	TaskType    string   `json:"task_type" binding:"required"`
	Title       string   `json:"title" binding:"required"`
	Instruction string   `json:"instruction"`
	DueAt       string   `json:"due_at"`
	RiskLevel   string   `json:"risk_level"`
	LockFields  []string `json:"lock_fields"` // 需要锁定的字段列表
}

// DraftCareTasks 生成医嘱任务草稿
// POST /api/v1/medical-episodes/:episode_id/care-tasks/draft
// 生成任务草稿并锁定指定字段（locked_fields_json）
func (h *Handler) DraftCareTasks(c *gin.Context) {
	episodeID := c.Param("episode_id")
	if episodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "episode_id 不能为空",
			"data":    nil,
		})
		return
	}

	// 校验疗程存在且状态允许
	episode, err := h.repo.GetEpisodeByID(c.Request.Context(), episodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询疗程失败",
			"data":    nil,
		})
		return
	}
	if episode == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "疗程不存在",
			"data":    nil,
		})
		return
	}

	// 只有 Created 状态才能生成草稿
	if episode.Status != EpisodeCreated {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalInvalidTransition,
			"message": "当前状态不允许生成任务草稿",
			"data":    nil,
		})
		return
	}

	var req DraftCareTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalInvalidTransition,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 生成任务草稿
	var tasks []CareTask
	for _, item := range req.Tasks {
		// 锁定字段序列化
		lockFieldsJSON := ""
		if len(item.LockFields) > 0 {
			lockBytes, _ := json.Marshal(item.LockFields)
			lockFieldsJSON = string(lockBytes)
		}

		task := CareTask{
			ID:               utils.GenerateULID(types.IDPrefix("task")),
			EpisodeID:        episodeID,
			TaskType:         item.TaskType,
			Title:            item.Title,
			Instruction:      item.Instruction,
			Status:           "Draft",
			RiskLevel:        item.RiskLevel,
			LockedFieldsJSON: lockFieldsJSON,
			CreatedBy:        "agent", // Agent 生成的草稿
		}

		if item.DueAt != "" {
			dueAt := types.NullableCSTTime{}
			if err := dueAt.UnmarshalJSON([]byte(`"` + item.DueAt + `"`)); err == nil {
				task.DueAt = dueAt
			}
		}

		tasks = append(tasks, task)
	}

	// 保存任务
	if err := h.repo.CreateTasks(c.Request.Context(), tasks); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "保存任务草稿失败",
			"data":    nil,
		})
		return
	}

	// 更新疗程状态为 InstructionDrafted
	episode.Status = InstructionDrafted
	if err := h.repo.UpdateEpisode(c.Request.Context(), episode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "更新疗程状态失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    constants.CodeSuccess,
		"message": "任务草稿生成成功",
		"data": gin.H{
			"episode_id": episodeID,
			"tasks":      tasks,
		},
	})
}

// CompleteTaskRequest 标记任务完成请求
type CompleteTaskRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	Note           string `json:"note"`
}

// CompleteTask 标记任务完成
// POST /api/v1/care-tasks/:task_id/complete
// 幂等键校验：相同 idempotency_key 的重复请求返回首次结果
func (h *Handler) CompleteTask(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "task_id 不能为空",
			"data":    nil,
		})
		return
	}

	var req CompleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalInvalidTransition,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 查询任务
	task, err := h.repo.GetTaskByID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询任务失败",
			"data":    nil,
		})
		return
	}
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "任务不存在",
			"data":    nil,
		})
		return
	}

	// 检查幂等键：已完成的任务直接返回成功
	if task.Status == "Done" {
		c.JSON(http.StatusOK, gin.H{
			"code":    constants.CodeSuccess,
			"message": "任务已完成（幂等）",
			"data":    task,
		})
		return
	}

	// 校验状态转换合法性
	if task.Status != "Draft" && task.Status != "Active" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalInvalidTransition,
			"message": "当前状态不允许标记完成",
			"data":    nil,
		})
		return
	}

	// 更新任务状态
	task.Status = "Done"
	if err := h.repo.UpdateTask(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "更新任务状态失败",
			"data":    nil,
		})
		return
	}

	// 创建任务动作记录（幂等键留痕）
	action := CareTaskAction{
		ID:             utils.GenerateULID(types.IDPrefix("act")),
		TaskID:         taskID,
		UserID:         "system", // TODO: 从 context 获取
		ActionType:     "complete",
		Note:           req.Note,
		IdempotencyKey: req.IdempotencyKey,
	}
	if err := h.repo.CreateAction(c.Request.Context(), &action); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "记录任务动作失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "任务已标记完成",
		"data":    task,
	})
}

// GenerateSummaryRequest 生成摘要请求
type GenerateSummaryRequest struct {
	IdempotencyKey string   `json:"idempotency_key" binding:"required"`
	WindowStart    string   `json:"window_start"`
	WindowEnd      string   `json:"window_end"`
	DataScopes     []string `json:"data_scopes"`
}

// GenerateSummary 生成医疗复诊摘要
// POST /api/v1/medical-episodes/:episode_id/summaries
// 汇总疗程下所有已完成任务，生成面向主人和诊所的双版本摘要
func (h *Handler) GenerateSummary(c *gin.Context) {
	episodeID := c.Param("episode_id")
	if episodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "episode_id 不能为空",
			"data":    nil,
		})
		return
	}

	var req GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalInvalidTransition,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 校验疗程存在
	episode, err := h.repo.GetEpisodeByID(c.Request.Context(), episodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询疗程失败",
			"data":    nil,
		})
		return
	}
	if episode == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "疗程不存在",
			"data":    nil,
		})
		return
	}

	// 查询已完成的任务
	tasks, err := h.repo.GetTasksByEpisodeID(c.Request.Context(), episodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询任务失败",
			"data":    nil,
		})
		return
	}

	// 统计任务完成情况
	var completedTasks []CareTask
	var totalTasks int
	for _, task := range tasks {
		totalTasks++
		if task.Status == "Done" {
			completedTasks = append(completedTasks, task)
		}
	}

	// 生成摘要内容（主人版 + 诊所版）
	ownerText := generateOwnerSummary(episode, completedTasks, totalTasks)
	clinicText := generateClinicSummary(episode, completedTasks, totalTasks)

	// 序列化数据范围
	dataScopesJSON := ""
	if len(req.DataScopes) > 0 {
		scopeBytes, _ := json.Marshal(req.DataScopes)
		dataScopesJSON = string(scopeBytes)
	}

	// 创建摘要记录
	summary := MedicalSummary{
		ID:             utils.GenerateULID(types.IDPrefix("sum")),
		EpisodeID:      episodeID,
		DataScopesJSON: dataScopesJSON,
		OwnerText:      ownerText,
		ClinicText:     clinicText,
		SafetyStatus:   "safe",
		GeneratedBy:    "agent",
		CreatedBy:      "agent",
	}

	if req.WindowStart != "" {
		ws := types.NullableCSTTime{}
		if err := ws.UnmarshalJSON([]byte(`"` + req.WindowStart + `"`)); err == nil {
			summary.WindowStart = ws
		}
	}
	if req.WindowEnd != "" {
		we := types.NullableCSTTime{}
		if err := we.UnmarshalJSON([]byte(`"` + req.WindowEnd + `"`)); err == nil {
			summary.WindowEnd = we
		}
	}

	if err := h.repo.CreateSummary(c.Request.Context(), &summary); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "保存摘要失败",
			"data":    nil,
		})
		return
	}

	// 更新疗程状态为 SummaryReady
	episode.Status = SummaryReady
	if err := h.repo.UpdateEpisode(c.Request.Context(), episode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "更新疗程状态失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    constants.CodeSuccess,
		"message": "摘要生成成功",
		"data":    summary,
	})
}

// generateOwnerSummary 生成面向主人的摘要
func generateOwnerSummary(episode *MedicalEpisode, completed []CareTask, total int) string {
	return "疗程摘要：共 " + string(rune('0'+total)) + " 项任务，已完成 " +
		string(rune('0'+len(completed))) + " 项。" +
		"\n\n" + DisclaimerCN
}

// generateClinicSummary 生成面向诊所的摘要
func generateClinicSummary(episode *MedicalEpisode, completed []CareTask, total int) string {
	return "疗程ID: " + episode.ID + "，宠物ID: " + episode.PetID +
		"，状态: " + string(episode.Status) +
		"，任务完成: " + string(rune('0'+len(completed))) + "/" + string(rune('0'+total))
}

// SendToClinicRequest 推送诊所请求
type SendToClinicRequest struct {
	ClinicID string `json:"clinic_id" binding:"required"`
}

// SendToClinic 摘要推送诊所
// POST /api/v1/medical-summaries/:summary_id/send-to-clinic
// 必须先完成授权状态机校验：只有 AwaitingAuthorization 状态的疗程才能推送
func (h *Handler) SendToClinic(c *gin.Context) {
	summaryID := c.Param("summary_id")
	if summaryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "summary_id 不能为空",
			"data":    nil,
		})
		return
	}

	var req SendToClinicRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalInvalidTransition,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 查询摘要
	summary, err := h.repo.GetSummaryByID(c.Request.Context(), summaryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询摘要失败",
			"data":    nil,
		})
		return
	}
	if summary == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "摘要不存在",
			"data":    nil,
		})
		return
	}

	// 查询关联的疗程
	episode, err := h.repo.GetEpisodeByID(c.Request.Context(), summary.EpisodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询疗程失败",
			"data":    nil,
		})
		return
	}
	if episode == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "关联疗程不存在",
			"data":    nil,
		})
		return
	}

	// 授权状态机校验：只有 AwaitingAuthorization 状态才能推送诊所
	if episode.Status != AwaitingAuthorization {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalNoAuthorization,
			"message": "当前状态不允许推送诊所，需要先完成授权（当前状态: " + string(episode.Status) + "）",
			"data":    nil,
		})
		return
	}

	// 校验授权是否存在且有效
	auth, err := h.repo.GetActiveAuthorization(c.Request.Context(), summary.EpisodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询授权失败",
			"data":    nil,
		})
		return
	}
	if auth == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    constants.CodeMedicalNoAuthorization,
			"message": "无有效授权，无法推送诊所",
			"data":    nil,
		})
		return
	}

	// 更新疗程状态为 SharedToClinic
	episode.Status = SharedToClinic
	episode.ClinicID = req.ClinicID
	if err := h.repo.UpdateEpisode(c.Request.Context(), episode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "更新疗程状态失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "摘要已推送诊所",
		"data": gin.H{
			"summary_id": summaryID,
			"clinic_id":  req.ClinicID,
			"episode_id": episode.ID,
		},
	})
}

// CreateAuthorizationRequest 创建授权请求
type CreateAuthorizationRequest struct {
	RecipientType  string   `json:"recipient_type" binding:"required"`
	RecipientID    string   `json:"recipient_id"`
	DataScopes     []string `json:"data_scopes" binding:"required,min=1"`
	ExpiresAt      string   `json:"expires_at" binding:"required"`
}

// CreateAuthorization 创建医疗数据授权
// POST /api/v1/medical-episodes/:episode_id/authorizations
// 授权流程：创建授权 → 更新疗程状态为 AwaitingAuthorization
func (h *Handler) CreateAuthorization(c *gin.Context) {
	episodeID := c.Param("episode_id")
	if episodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "episode_id 不能为空",
			"data":    nil,
		})
		return
	}

	var req CreateAuthorizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalInvalidTransition,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 校验疗程存在
	episode, err := h.repo.GetEpisodeByID(c.Request.Context(), episodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "查询疗程失败",
			"data":    nil,
		})
		return
	}
	if episode == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "疗程不存在",
			"data":    nil,
		})
		return
	}

	// 序列化数据范围
	dataScopesJSON := ""
	if len(req.DataScopes) > 0 {
		scopeBytes, _ := json.Marshal(req.DataScopes)
		dataScopesJSON = string(scopeBytes)
	}

	// 解析过期时间
	expiresAt := types.NullableCSTTime{}
	if err := expiresAt.UnmarshalJSON([]byte(`"` + req.ExpiresAt + `"`)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeMedicalInvalidTransition,
			"message": "过期时间格式无效",
			"data":    nil,
		})
		return
	}

	// 创建授权记录
	auth := MedicalAuthorization{
		ID:             utils.GenerateULID(types.IDPrefix("auth")),
		EpisodeID:      episodeID,
		FamilyID:       episode.FamilyID,
		PetID:          episode.PetID,
		AuthorizedBy:   "user", // TODO: 从 context 获取
		RecipientType:  req.RecipientType,
		RecipientID:    req.RecipientID,
		DataScopesJSON: dataScopesJSON,
		ExpiresAt:      expiresAt,
		Status:         "active",
		CreatedBy:      "user",
	}

	if err := h.repo.CreateAuthorization(c.Request.Context(), &auth); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "创建授权失败",
			"data":    nil,
		})
		return
	}

	// 更新疗程状态为 AwaitingAuthorization
	episode.Status = AwaitingAuthorization
	if err := h.repo.UpdateEpisode(c.Request.Context(), episode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeMedicalEpisodeNotFound,
			"message": "更新疗程状态失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    constants.CodeSuccess,
		"message": "授权创建成功",
		"data":    auth,
	})
}
