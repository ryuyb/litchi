package valueobject

// Stage represents the workflow stage enum.
type Stage string

const (
	StageClarification  Stage = "clarification"
	StageDesign         Stage = "design"
	StageTaskBreakdown  Stage = "task_breakdown"
	StageExecution      Stage = "execution"
	StagePullRequest    Stage = "pull_request"
	StageCompleted      Stage = "completed"
)

// TODO: Implement full value objects in T2.1.1-T2.1.4