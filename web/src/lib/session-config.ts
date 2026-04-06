import {
	RollbackSessionTargetStage,
	SessionCurrentStage,
	SessionStatus,
} from "#/api/schemas";

// Stage display configuration
export const stageConfig: Record<
	SessionCurrentStage,
	{ label: string; color: string; icon: string; description: string }
> = {
	[SessionCurrentStage.clarification]: {
		label: "Clarification",
		color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
		icon: "CL",
		description: "Gathering information about the issue",
	},
	[SessionCurrentStage.design]: {
		label: "Design",
		color:
			"bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
		icon: "DE",
		description: "Creating the implementation design",
	},
	[SessionCurrentStage.task_breakdown]: {
		label: "Task Breakdown",
		color:
			"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
		icon: "TB",
		description: "Breaking down into actionable tasks",
	},
	[SessionCurrentStage.execution]: {
		label: "Execution",
		color:
			"bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300",
		icon: "EX",
		description: "Executing the tasks",
	},
	[SessionCurrentStage.pull_request]: {
		label: "Pull Request",
		color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
		icon: "PR",
		description: "Creating the pull request",
	},
	[SessionCurrentStage.completed]: {
		label: "Completed",
		color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300",
		icon: "CO",
		description: "Session completed successfully",
	},
};

// Status display configuration
export const statusConfig: Record<
	SessionStatus,
	{ label: string; color: string }
> = {
	[SessionStatus.active]: {
		label: "Active",
		color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
	},
	[SessionStatus.paused]: {
		label: "Paused",
		color:
			"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
	},
	[SessionStatus.completed]: {
		label: "Completed",
		color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300",
	},
	[SessionStatus.terminated]: {
		label: "Terminated",
		color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
	},
};

// Workflow stages array for iteration
export const workflowStages = Object.entries(stageConfig).map(
	([key, value]) => ({
		key: key as SessionCurrentStage,
		...value,
	}),
);

// Rollback stages (only these can be rolled back to)
export const rollbackStages = [
	{
		key: RollbackSessionTargetStage.clarification,
		label: "Clarification",
	},
	{
		key: RollbackSessionTargetStage.design,
		label: "Design",
	},
	{
		key: RollbackSessionTargetStage.task_breakdown,
		label: "Task Breakdown",
	},
] as const;

// Get stage badge color class
export function getStageColor(stage: SessionCurrentStage | undefined): string {
	return stage ? stageConfig[stage].color : "bg-gray-100 text-gray-800";
}

// Get status badge color class
export function getStatusColor(status: SessionStatus | undefined): string {
	return status ? statusConfig[status].color : "bg-gray-100 text-gray-800";
}
