// Audit log display configuration

// Operation type configuration
export const operationConfig: Record<string, { label: string; color: string }> =
	{
		"session.start": {
			label: "Session Start",
			color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
		},
		"session.pause": {
			label: "Session Pause",
			color:
				"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
		},
		"session.resume": {
			label: "Session Resume",
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
		},
		"session.rollback": {
			label: "Session Rollback",
			color:
				"bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300",
		},
		"session.terminate": {
			label: "Session Terminate",
			color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
		},
		"task.skip": {
			label: "Task Skip",
			color:
				"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
		},
		"task.retry": {
			label: "Task Retry",
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
		},
		"repo.enable": {
			label: "Repo Enable",
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
		},
		"repo.disable": {
			label: "Repo Disable",
			color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
		},
		"repo.update": {
			label: "Repo Update",
			color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
		},
		"config.update": {
			label: "Config Update",
			color:
				"bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
		},
	};

// Result status configuration
export const resultConfig: Record<string, { label: string; color: string }> = {
	success: {
		label: "Success",
		color: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
	},
	failed: {
		label: "Failed",
		color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
	},
	denied: {
		label: "Denied",
		color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300",
	},
};

// Get operation display config
export function getOperationDisplay(operation: string | undefined): {
	label: string;
	color: string;
} {
	if (!operation) {
		return { label: "Unknown", color: "bg-gray-100 text-gray-800" };
	}
	return (
		operationConfig[operation] ?? {
			label: operation,
			color: "bg-gray-100 text-gray-800",
		}
	);
}

// Get result display config
export function getResultDisplay(result: string | undefined): {
	label: string;
	color: string;
} {
	if (!result) {
		return { label: "Unknown", color: "bg-gray-100 text-gray-800" };
	}
	return (
		resultConfig[result] ?? {
			label: result,
			color: "bg-gray-100 text-gray-800",
		}
	);
}

// Available operation types for filter dropdown
export const operationOptions = Object.keys(operationConfig);

// Available result types for filter dropdown
export const resultOptions = Object.keys(resultConfig);

// Format duration in milliseconds to human readable
export function formatDuration(ms: number | undefined): string {
	if (!ms) return "-";

	if (ms < 1000) {
		return `${ms}ms`;
	}

	const seconds = ms / 1000;
	if (seconds < 60) {
		return `${seconds.toFixed(1)}s`;
	}

	const minutes = Math.floor(seconds / 60);
	const remainingSeconds = seconds % 60;
	return `${minutes}m ${remainingSeconds.toFixed(0)}s`;
}
