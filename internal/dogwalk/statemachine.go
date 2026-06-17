package dogwalk

import "github.com/patch-pet/patch-pet/internal/workflow"

// NewDogWalkStateMachine 创建代遛订单状态机（14 个状态）
// 转换规则严格对齐 requirements.md 3.2 节
// 禁止跨阶跳变；已完成订单归档禁止二次变更
func NewDogWalkStateMachine() *workflow.StateMachine {
	sm := workflow.NewStateMachine()

	// CandidateDetected → AwaitingPermission（等待用户授权）
	sm.AddTransition(workflow.Transition{
		From:  string(OppCandidateDetected),
		To:    string(OppAwaitingPermission),
		Event: "request_permission",
	})

	// AwaitingPermission → Rejected（用户拒绝）
	sm.AddTransition(workflow.Transition{
		From:  string(OppAwaitingPermission),
		To:    string(OppRejected),
		Event: "rejected",
	})

	// AwaitingPermission → ReminderOnly（仅提醒）
	sm.AddTransition(workflow.Transition{
		From:  string(OppAwaitingPermission),
		To:    string(OppReminderOnly),
		Event: "reminder_only",
	})

	// AwaitingPermission → PlanDrafting（用户确认，开始规划）
	sm.AddTransition(workflow.Transition{
		From:  string(OppAwaitingPermission),
		To:    string(OrderPlanDrafting),
		Event: "allow_plan",
	})

	// PlanDrafting → VendorSelected（选择服务商）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderPlanDrafting),
		To:    string(OrderVendorSelected),
		Event: "vendor_selected",
	})

	// VendorSelected → RoutePreference（路线偏好设置）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderVendorSelected),
		To:    string(OrderRoutePreference),
		Event: "set_route_preference",
	})

	// RoutePreference → RouteReady（路线规划完成）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderRoutePreference),
		To:    string(OrderRouteReady),
		Event: "route_ready",
	})

	// RouteReady → AwaitingPayment（等待支付）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderRouteReady),
		To:    string(OrderAwaitingPayment),
		Event: "create_payment",
	})

	// AwaitingPayment → Paid（支付成功）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderAwaitingPayment),
		To:    string(OrderPaid),
		Event: "payment_success",
	})

	// Paid → Booked（预约服务商成功）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderPaid),
		To:    string(OrderBooked),
		Event: "booked",
	})

	// Booked → InService（服务开始）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderBooked),
		To:    string(OrderInService),
		Event: "service_started",
	})

	// InService → AnomalyDetected（异常检测）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderInService),
		To:    string(OrderAnomalyDetected),
		Event: "anomaly_detected",
	})

	// InService → Completed（服务完成）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderInService),
		To:    string(OrderCompleted),
		Event: "service_completed",
	})

	// Completed → ReportReady（报告生成）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderCompleted),
		To:    string(OrderReportReady),
		Event: "report_generated",
	})

	// ReportReady → LifeStreamWritten（写入生命流）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderReportReady),
		To:    string(OrderLifeStreamWritten),
		Event: "lifestream_written",
	})

	// Saga 补偿路径
	// AwaitingPayment → Refunded（支付超时退款）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderAwaitingPayment),
		To:    string(OrderRefunded),
		Event: "payment_timeout_refund",
	})

	// Booked → Refunded（预约失败退款）
	sm.AddTransition(workflow.Transition{
		From:  string(OrderBooked),
		To:    string(OrderRefunded),
		Event: "booking_failed_refund",
	})

	// 标记终态
	sm.MarkFinal(string(OppRejected))
	sm.MarkFinal(string(OppReminderOnly))
	sm.MarkFinal(string(OrderLifeStreamWritten))
	sm.MarkFinal(string(OrderRefunded))

	return sm
}
