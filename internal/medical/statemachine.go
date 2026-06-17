package medical

import "github.com/patch-pet/patch-pet/internal/workflow"

// NewMedicalStateMachine 创建医疗疗程状态机（13 个状态）
// 转换规则严格对齐 requirements.md 3.1 节
func NewMedicalStateMachine() *workflow.StateMachine {
	sm := workflow.NewStateMachine()

	// EpisodeCreated → InstructionDrafted（草稿生成）
	sm.AddTransition(workflow.Transition{
		From:  string(EpisodeCreated),
		To:    string(InstructionDrafted),
		Event: "draft_instructions",
	})

	// InstructionDrafted → AwaitingOwnerConfirm（等待主人确认）
	sm.AddTransition(workflow.Transition{
		From:  string(InstructionDrafted),
		To:    string(AwaitingOwnerConfirm),
		Event: "submit_for_confirm",
	})

	// AwaitingOwnerConfirm → Active（主人确认后激活）
	sm.AddTransition(workflow.Transition{
		From:  string(AwaitingOwnerConfirm),
		To:    string(EpisodeActive),
		Event: "owner_confirmed",
	})

	// Active → TaskDue（任务到期）
	sm.AddTransition(workflow.Transition{
		From:  string(EpisodeActive),
		To:    string(TaskDue),
		Event: "task_due",
	})

	// TaskDue → Done（任务完成）
	sm.AddTransition(workflow.Transition{
		From:  string(TaskDue),
		To:    string(TaskDone),
		Event: "task_completed",
	})

	// TaskDue → Snoozed（任务延期）
	sm.AddTransition(workflow.Transition{
		From:  string(TaskDue),
		To:    string(TaskSnoozed),
		Event: "task_snoozed",
	})

	// TaskDue → Missed（任务错过）
	sm.AddTransition(workflow.Transition{
		From:  string(TaskDue),
		To:    string(TaskMissed),
		Event: "task_missed",
	})

	// Snoozed → TaskDue（延期后重新到期）
	sm.AddTransition(workflow.Transition{
		From:  string(TaskSnoozed),
		To:    string(TaskDue),
		Event: "task_due_again",
	})

	// Active → SummaryReady（生成复诊摘要）
	sm.AddTransition(workflow.Transition{
		From:  string(EpisodeActive),
		To:    string(SummaryReady),
		Event: "summary_generated",
	})

	// SummaryReady → AwaitingAuthorization（等待授权）
	sm.AddTransition(workflow.Transition{
		From:  string(SummaryReady),
		To:    string(AwaitingAuthorization),
		Event: "await_auth",
	})

	// AwaitingAuthorization → SharedToClinic（授权后推送诊所）
	sm.AddTransition(workflow.Transition{
		From:  string(AwaitingAuthorization),
		To:    string(SharedToClinic),
		Event: "shared_to_clinic",
	})

	// Active → HostedCare（托管照护）
	sm.AddTransition(workflow.Transition{
		From:  string(EpisodeActive),
		To:    string(HostedCare),
		Event: "hosted_care",
	})

	// Active → Closed（关闭疗程）
	sm.AddTransition(workflow.Transition{
		From:  string(EpisodeActive),
		To:    string(EpisodeClosed),
		Event: "close",
	})

	// 标记终态：SharedToClinic / HostedCare / Closed 不可再变更
	sm.MarkFinal(string(SharedToClinic))
	sm.MarkFinal(string(HostedCare))
	sm.MarkFinal(string(EpisodeClosed))

	return sm
}
