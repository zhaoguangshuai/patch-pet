// Package dogwalk 代遛狗模块 HTTP Handler
package dogwalk

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/patch-pet/patch-pet/pkg/constants"
	"github.com/patch-pet/patch-pet/pkg/types"
	"github.com/patch-pet/patch-pet/pkg/utils"
)

// Handler 代遛模块 HTTP Handler
type Handler struct {
	repo        *Repository
	riskGateway *RiskGateway
}

// NewHandler 创建代遛 Handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:        repo,
		riskGateway: NewRiskGateway(),
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	dogwalk := rg.Group("/dog-walk")
	{
		dogwalk.POST("/opportunities/detect", h.DetectOpportunity)
		dogwalk.POST("/opportunities/:id/allow-plan", h.AllowPlan)
		dogwalk.GET("/vendors", h.ListVendors)
		dogwalk.POST("/routes/plan", h.PlanRoute)
		dogwalk.POST("/plans/:plan_id/payment-orders", h.CreatePaymentOrder)
		dogwalk.POST("/payment-callback", h.PaymentCallback)
		dogwalk.POST("/orders/:order_id/book", h.BookVendor)
		dogwalk.GET("/orders/:order_id", h.GetOrderStatus)
		dogwalk.POST("/orders/:order_id/reports", h.GenerateReport)
		dogwalk.GET("/orders/list", h.ListOrders)
	}
}

// DetectOpportunityRequest 需求识别请求
type DetectOpportunityRequest struct {
	PetID        string          `json:"pet_id" binding:"required"`
	FamilyID     string          `json:"family_id" binding:"required"`
	TriggerReason json.RawMessage `json:"trigger_reason" binding:"required"`
	Confidence   float64         `json:"confidence" binding:"required,min=0,max=1"`
}

// DetectOpportunity 代遛需求识别
// POST /api/v1/dog-walk/opportunities/detect
// confidence 范围 0~1，由 AI 模型输出
func (h *Handler) DetectOpportunity(c *gin.Context) {
	var req DetectOpportunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 序列化触发原因
	triggerReasonJSON := ""
	if len(req.TriggerReason) > 0 {
		triggerReasonJSON = string(req.TriggerReason)
	}

	// 创建需求实体
	opp := DogWalkOpportunity{
		ID:                utils.GenerateULID(types.IDPrefix("opp")),
		FamilyID:          req.FamilyID,
		PetID:             req.PetID,
		TriggerReasonJSON: triggerReasonJSON,
		Confidence:        req.Confidence,
		Status:            string(OppCandidateDetected),
		CreatedBy:         "agent",
	}

	if err := h.repo.CreateOpportunity(c.Request.Context(), &opp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "创建需求失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    constants.CodeSuccess,
		"message": "需求识别成功",
		"data":    opp,
	})
}

// AllowPlan 用户确认方案
// POST /api/v1/dog-walk/opportunities/:id/allow-plan
func (h *Handler) AllowPlan(c *gin.Context) {
	oppID := c.Param("id")
	if oppID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "id 不能为空",
			"data":    nil,
		})
		return
	}

	opp, err := h.repo.GetOpportunityByID(c.Request.Context(), oppID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "查询需求失败",
			"data":    nil,
		})
		return
	}
	if opp == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "需求不存在",
			"data":    nil,
		})
		return
	}

	if opp.Status != string(OppCandidateDetected) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkInvalidTransition,
			"message": "当前状态不允许确认方案",
			"data":    nil,
		})
		return
	}

	opp.Status = string(OppAwaitingPermission)
	if err := h.repo.UpdateOpportunity(c.Request.Context(), opp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "更新需求状态失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "方案已确认",
		"data":    opp,
	})
}

// ListVendors 服务商查询
// GET /api/v1/dog-walk/vendors
func (h *Handler) ListVendors(c *gin.Context) {
	sortBy := c.DefaultQuery("sort_by", "rating")

	// 风险网关校验排序字段
	decision := h.riskGateway.CheckVendorSort(sortBy)
	if decision.Blocked {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    constants.CodeDogwalkAutoOrderBlocked,
			"message": decision.Reason,
			"data":    nil,
		})
		return
	}

	// TODO: 接入服务商数据源
	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "ok",
		"data":    []interface{}{},
	})
}

// PlanRouteRequest 路线规划请求
type PlanRouteRequest struct {
	OpportunityID string `json:"opportunity_id" binding:"required"`
	Preference    string `json:"preference"`
}

// PlanRoute 路线规划
// POST /api/v1/dog-walk/routes/plan
func (h *Handler) PlanRoute(c *gin.Context) {
	var req PlanRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkRouteFailed,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// TODO: 接入地图服务进行路线规划
	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "路线规划成功",
		"data": gin.H{
			"route_id": utils.GenerateULID(types.IDPrefix("plan")),
		},
	})
}

// CreatePaymentOrderRequest 创建支付订单请求
type CreatePaymentOrderRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

// CreatePaymentOrder 创建支付订单
// POST /api/v1/dog-walk/plans/:plan_id/payment-orders
func (h *Handler) CreatePaymentOrder(c *gin.Context) {
	planID := c.Param("plan_id")
	if planID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "plan_id 不能为空",
			"data":    nil,
		})
		return
	}

	var req CreatePaymentOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkInvalidTransition,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 风险网关校验：禁止自动下单
	decision := h.riskGateway.CheckAutoOrder(true)
	if decision.Blocked {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    constants.CodeDogwalkAutoOrderBlocked,
			"message": decision.Reason,
			"data":    nil,
		})
		return
	}

	// TODO: 调用支付服务创建支付订单
	c.JSON(http.StatusCreated, gin.H{
		"code":    constants.CodeSuccess,
		"message": "支付订单创建成功",
		"data": gin.H{
			"payment_order_id": utils.GenerateULID(types.IDPrefix("order")),
			"plan_id":          planID,
		},
	})
}

// PaymentCallbackRequest 支付回调请求
type PaymentCallbackRequest struct {
	PaymentOrderID string `json:"payment_order_id" binding:"required"`
	Status         string `json:"status" binding:"required"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

// PaymentCallback 支付回调
// POST /api/v1/dog-walk/payment-callback
// 30s/轮 × 3，幂等
func (h *Handler) PaymentCallback(c *gin.Context) {
	var req PaymentCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodePaymentFailed,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// TODO: 查询订单并更新状态
	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "支付回调处理成功",
		"data": gin.H{
			"payment_order_id": req.PaymentOrderID,
			"status":           req.Status,
		},
	})
}

// BookVendorRequest 预约服务商请求
type BookVendorRequest struct {
	VendorID       string `json:"vendor_id" binding:"required"`
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

// BookVendor 预约服务商
// POST /api/v1/dog-walk/orders/:order_id/book
func (h *Handler) BookVendor(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "order_id 不能为空",
			"data":    nil,
		})
		return
	}

	var req BookVendorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkInvalidTransition,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// TODO: 查询订单，校验状态，调用服务商预约
	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "服务商预约成功",
		"data": gin.H{
			"order_id":   orderID,
			"vendor_id":  req.VendorID,
			"status":     string(OrderBooked),
		},
	})
}

// GetOrderStatus 获取实时服务状态
// GET /api/v1/dog-walk/orders/:order_id
func (h *Handler) GetOrderStatus(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "order_id 不能为空",
			"data":    nil,
		})
		return
	}

	order, err := h.repo.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "查询订单失败",
			"data":    nil,
		})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "订单不存在",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "ok",
		"data":    order,
	})
}

// GenerateReportRequest 生成服务报告请求
type GenerateReportRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
}

// GenerateReport 生成服务报告
// POST /api/v1/dog-walk/orders/:order_id/reports
func (h *Handler) GenerateReport(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "order_id 不能为空",
			"data":    nil,
		})
		return
	}

	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkInvalidTransition,
			"message": "请求参数无效: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// 查询订单
	order, err := h.repo.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "查询订单失败",
			"data":    nil,
		})
		return
	}
	if order == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "订单不存在",
			"data":    nil,
		})
		return
	}

	// 只有已完成的订单才能生成报告
	if order.Status != OrderCompleted {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkInvalidTransition,
			"message": "当前状态不允许生成报告",
			"data":    nil,
		})
		return
	}

	report := DogWalkReport{
		ID:      utils.GenerateULID(types.IDPrefix("report")),
		OrderID: orderID,
	}

	if err := h.repo.CreateReport(c.Request.Context(), &report); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "创建报告失败",
			"data":    nil,
		})
		return
	}

	// 更新订单状态为 ReportReady
	order.Status = OrderReportReady
	if err := h.repo.UpdateOrder(c.Request.Context(), order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "更新订单状态失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    constants.CodeSuccess,
		"message": "服务报告生成成功",
		"data":    report,
	})
}

// ListOrders 历史订单分页
// GET /api/v1/dog-walk/orders/list
func (h *Handler) ListOrders(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "user_id 不能为空",
			"data":    nil,
		})
		return
	}

	pageNum := 1
	pageSize := 20
	if pn, exists := c.Get("pageNum"); exists {
		pageNum = pn.(int)
	}
	if ps, exists := c.Get("pageSize"); exists {
		pageSize = ps.(int)
	}

	orders, total, err := h.repo.ListOrders(c.Request.Context(), userID, pageNum, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    constants.CodeDogwalkOppNotFound,
			"message": "查询订单列表失败",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    constants.CodeSuccess,
		"message": "ok",
		"data": gin.H{
			"list":     orders,
			"total":    total,
			"pageNum":  pageNum,
			"pageSize": pageSize,
		},
	})
}
