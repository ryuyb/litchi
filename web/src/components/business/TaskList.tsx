import { useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
	CheckCircleIcon,
	CircleIcon,
	LoaderIcon,
	RefreshCcwIcon,
	SkipForwardIcon,
	StopCircleIcon,
	XCircleIcon,
} from "lucide-react";
import { useCallback, useMemo } from "react";
import type { Task } from "#/api/schemas";
import { getGetApiV1SessionsIdDetailQueryKey } from "#/api/sessions/sessions";
import {
	usePostApiV1SessionsSessionIdTasksTaskIdRetry,
	usePostApiV1SessionsSessionIdTasksTaskIdSkip,
} from "#/api/tasks/tasks";
import { DataTable } from "#/components/data-table";
import { Button } from "#/components/ui/button";
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "#/components/ui/tooltip";

export interface TaskListProps {
	sessionId: string;
	tasks: Task[];
	onTaskOperationSuccess?: () => void;
}

// Task status icon configuration
const taskStatusConfig: Record<
	string,
	{ color: string; icon: React.ReactNode }
> = {
	pending: {
		color: "text-muted-foreground",
		icon: <CircleIcon className="size-4" />,
	},
	ready: {
		color: "text-blue-600 dark:text-blue-400",
		icon: <CheckCircleIcon className="size-4" />,
	},
	in_progress: {
		color: "text-orange-600 dark:text-orange-400",
		icon: <LoaderIcon className="size-4 animate-spin" />,
	},
	completed: {
		color: "text-green-600 dark:text-green-400",
		icon: <CheckCircleIcon className="size-4" />,
	},
	failed: {
		color: "text-red-600 dark:text-red-400",
		icon: <XCircleIcon className="size-4" />,
	},
	skipped: {
		color: "text-gray-500 dark:text-gray-400",
		icon: <StopCircleIcon className="size-4" />,
	},
};

// Action buttons component for a single task
function TaskActionButtons({
	sessionId,
	task,
	onSuccess,
}: {
	sessionId: string;
	task: Task;
	onSuccess?: () => void;
}) {
	const queryClient = useQueryClient();
	const status = task.status;

	// Retry mutation - only for failed tasks
	const retryMutation = usePostApiV1SessionsSessionIdTasksTaskIdRetry({
		mutation: {
			onSuccess: () => {
				queryClient.invalidateQueries({
					queryKey: getGetApiV1SessionsIdDetailQueryKey(sessionId),
				});
				onSuccess?.();
			},
		},
	});

	// Skip mutation - for pending/ready/failed tasks
	const skipMutation = usePostApiV1SessionsSessionIdTasksTaskIdSkip({
		mutation: {
			onSuccess: () => {
				queryClient.invalidateQueries({
					queryKey: getGetApiV1SessionsIdDetailQueryKey(sessionId),
				});
				onSuccess?.();
			},
		},
	});

	// Determine which actions are available based on status
	const canRetry = status === "failed";
	const canSkip =
		status === "pending" || status === "ready" || status === "failed";
	const noActions = !canRetry && !canSkip;

	if (noActions) {
		return <span className="text-muted-foreground text-sm">-</span>;
	}

	const handleRetry = () => {
		if (task.id) {
			retryMutation.mutate({
				sessionId,
				taskId: task.id,
				data: {},
			});
		}
	};

	const handleSkip = () => {
		if (task.id) {
			skipMutation.mutate({
				sessionId,
				taskId: task.id,
				data: {},
			});
		}
	};

	return (
		<TooltipProvider>
			<div className="flex items-center gap-1">
				{canRetry && (
					<Tooltip>
						<TooltipTrigger asChild>
							<Button
								variant="ghost"
								size="sm"
								onClick={handleRetry}
								disabled={retryMutation.isPending}
								className="h-8 px-2"
							>
								<RefreshCcwIcon
									className={`size-4 ${retryMutation.isPending ? "animate-spin" : ""}`}
								/>
							</Button>
						</TooltipTrigger>
						<TooltipContent side="top">
							<p>Retry this task</p>
						</TooltipContent>
					</Tooltip>
				)}
				{canSkip && (
					<Tooltip>
						<TooltipTrigger asChild>
							<Button
								variant="ghost"
								size="sm"
								onClick={handleSkip}
								disabled={skipMutation.isPending}
								className="h-8 px-2"
							>
								<SkipForwardIcon className="size-4" />
							</Button>
						</TooltipTrigger>
						<TooltipContent side="top">
							<p>Skip this task</p>
						</TooltipContent>
					</Tooltip>
				)}
			</div>
		</TooltipProvider>
	);
}

// Dependencies display component
function TaskDependencies({
	dependencies,
	tasks,
}: {
	dependencies?: string[];
	tasks: Task[];
}) {
	if (!dependencies || dependencies.length === 0) {
		return <span className="text-muted-foreground text-sm">-</span>;
	}

	// Create a map of task id to order for display
	const taskOrderMap = new Map<string, number>();
	for (const task of tasks) {
		if (task.id) {
			taskOrderMap.set(task.id, task.order ?? 0);
		}
	}

	return (
		<div className="flex flex-wrap gap-1">
			{dependencies.map((depId) => {
				const order = taskOrderMap.get(depId);
				return (
					<span
						key={depId}
						className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs"
					>
						#{order ?? depId.slice(0, 6)}
					</span>
				);
			})}
		</div>
	);
}

export function TaskList({
	sessionId,
	tasks,
	onTaskOperationSuccess,
}: TaskListProps) {
	// Memoize the on success callback
	const handleSuccess = useCallback(() => {
		onTaskOperationSuccess?.();
	}, [onTaskOperationSuccess]);

	// Create columns with access to sessionId and tasks
	const columns: ColumnDef<Task>[] = useMemo(
		() => [
			{
				accessorKey: "order",
				header: "#",
				cell: ({ row }) => (
					<span className="font-mono text-sm">{row.original.order ?? "-"}</span>
				),
			},
			{
				accessorKey: "description",
				header: "Description",
				cell: ({ row }) => (
					<span className="text-sm">{row.original.description ?? "-"}</span>
				),
			},
			{
				accessorKey: "status",
				header: "Status",
				cell: ({ row }) => {
					const status = row.original.status;
					const statusDisplay = row.original.statusDisplay;

					const config = status ? taskStatusConfig[status] : null;

					return (
						<div className="flex items-center gap-2">
							{config ? (
								<>
									<span className={config.color}>{config.icon}</span>
									<span className={`text-sm ${config.color}`}>
										{statusDisplay ?? status}
									</span>
								</>
							) : (
								<span className="text-muted-foreground text-sm">
									{statusDisplay ?? status ?? "-"}
								</span>
							)}
						</div>
					);
				},
			},
			{
				accessorKey: "dependencies",
				header: "Dependencies",
				cell: ({ row }) => (
					<TaskDependencies
						dependencies={row.original.dependencies}
						tasks={tasks}
					/>
				),
			},
			{
				accessorKey: "retryCount",
				header: "Retries",
				cell: ({ row }) => (
					<span className="font-mono text-sm">
						{row.original.retryCount ?? 0}
					</span>
				),
			},
			{
				id: "actions",
				header: "Actions",
				cell: ({ row }) => (
					<TaskActionButtons
						sessionId={sessionId}
						task={row.original}
						onSuccess={handleSuccess}
					/>
				),
			},
		],
		[sessionId, tasks, handleSuccess],
	);

	return <DataTable columns={columns} data={tasks} />;
}
