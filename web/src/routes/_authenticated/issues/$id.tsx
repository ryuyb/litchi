import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
	ArrowLeftIcon,
	ExternalLinkIcon,
	LoaderIcon,
	PauseIcon,
	PlayIcon,
	RefreshCcwIcon,
	RotateCcwIcon,
	StopCircleIcon,
} from "lucide-react";
import { useState } from "react";
import {
	type RollbackSessionTargetStage,
	SessionCurrentStage,
	SessionStatus,
} from "#/api/schemas";
import type { Session } from "#/api/schemas/session";
import {
	type getApiV1SessionsIdDetailResponse,
	getGetApiV1SessionsIdDetailQueryKey,
	useGetApiV1SessionsIdDetail,
	usePostApiV1SessionsIdPause,
	usePostApiV1SessionsIdRestart,
	usePostApiV1SessionsIdResume,
	usePostApiV1SessionsIdRollback,
	usePostApiV1SessionsIdTerminate,
} from "#/api/sessions/sessions";
import { StageProgress, TaskList } from "#/components/business";
import { Button } from "#/components/ui/button";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { rollbackStages, statusConfig } from "#/lib/session-config";

export const Route = createFileRoute("/_authenticated/issues/$id")({
	component: IssueDetailPage,
});

// Type guard to check if response is successful
function isSuccessResponse(
	response: getApiV1SessionsIdDetailResponse | undefined,
): response is getApiV1SessionsIdDetailResponse & {
	status: 200;
	data: Session;
} {
	return response?.status === 200;
}

// Status badge component
function StatusBadge({ status }: { status: SessionStatus | undefined }) {
	const config = status ? statusConfig[status] : null;

	if (!config) {
		return null;
	}

	return (
		<span
			className={`rounded-full px-2 py-0.5 text-xs font-medium ${config.color}`}
		>
			{config.label}
		</span>
	)
}

function IssueDetailPage() {
	const { id } = Route.useParams();
	const queryClient = useQueryClient();

	// State for rollback dialog
	const [rollbackStage, setRollbackStage] =
		useState<RollbackSessionTargetStage | null>(null);

	// Fetch session details
	const { data: response, isLoading, error } = useGetApiV1SessionsIdDetail(id);

	// Check if response is successful
	const isSuccess = isSuccessResponse(response);
	const session: Session | undefined = isSuccess ? response.data : undefined;

	// Mutations
	const pauseMutation = usePostApiV1SessionsIdPause({
		mutation: {
			onSuccess: () => {
				queryClient.invalidateQueries({
					queryKey: getGetApiV1SessionsIdDetailQueryKey(id),
				})
			},
		},
	})

	const resumeMutation = usePostApiV1SessionsIdResume({
		mutation: {
			onSuccess: () => {
				queryClient.invalidateQueries({
					queryKey: getGetApiV1SessionsIdDetailQueryKey(id),
				})
			},
		},
	})

	const rollbackMutation = usePostApiV1SessionsIdRollback({
		mutation: {
			onSuccess: () => {
				queryClient.invalidateQueries({
					queryKey: getGetApiV1SessionsIdDetailQueryKey(id),
				})
				setRollbackStage(null);
			},
		},
	})

	const terminateMutation = usePostApiV1SessionsIdTerminate({
		mutation: {
			onSuccess: () => {
				queryClient.invalidateQueries({
					queryKey: getGetApiV1SessionsIdDetailQueryKey(id),
				})
			},
		},
	})

	const restartMutation = usePostApiV1SessionsIdRestart({
		mutation: {
			onSuccess: () => {
				queryClient.invalidateQueries({
					queryKey: getGetApiV1SessionsIdDetailQueryKey(id),
				})
			},
		},
	})

	// Loading state
	if (isLoading) {
		return (
			<div className="flex items-center justify-center min-h-[400px] gap-2">
				<LoaderIcon className="size-8 animate-spin text-muted-foreground" />
				<span className="text-muted-foreground">Loading session...</span>
			</div>
		)
	}

	// Error state
	if (error) {
		return (
			<div className="rounded-xl border border-destructive bg-card p-6">
				<h2 className="text-lg font-semibold text-destructive">
					Error loading session
				</h2>
				<p className="mt-2 text-muted-foreground">
					Failed to load session details. Please try again.
				</p>
				<Button asChild className="mt-4">
					<Link to="/issues">
						<ArrowLeftIcon className="size-4" />
						Back to Issues
					</Link>
				</Button>
			</div>
		)
	}

	// No session found or error response
	if (!isSuccess || !session) {
		return (
			<div className="rounded-xl border border-border bg-card p-6">
				<h2 className="text-lg font-semibold">Session not found</h2>
				<p className="mt-2 text-muted-foreground">
					The requested session could not be found.
				</p>
				<Button asChild className="mt-4">
					<Link to="/issues">
						<ArrowLeftIcon className="size-4" />
						Back to Issues
					</Link>
				</Button>
			</div>
		)
	}

	const issue = session.issue;
	const tasks = session.tasks ?? [];
	const canPause =
		session.status === SessionStatus.active ||
		session.status === SessionStatus.paused;
	const canResume = session.status === SessionStatus.paused;
	const canRollback =
		session.status === SessionStatus.active &&
		session.currentStage !== SessionCurrentStage.clarification;
	const canTerminate =
		session.status === SessionStatus.active ||
		session.status === SessionStatus.paused;
	const canRestart = session.status === SessionStatus.terminated;

	return (
		<div className="space-y-6">
			{/* Header with back link */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Button variant="ghost" size="sm" asChild>
						<Link to="/issues">
							<ArrowLeftIcon className="size-4" />
							<span>Back to Issues</span>
						</Link>
					</Button>
				</div>
				<div className="flex items-center gap-2">
					<StatusBadge status={session.status} />
				</div>
			</div>

			{/* Issue info section */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<div className="flex items-start justify-between">
					<div className="space-y-2">
						<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
							#{issue?.number ?? session.issueNumber ?? "-"}{" "}
							{issue?.title ?? session.issueTitle ?? "Untitled Issue"}
						</h1>
						<div className="flex items-center gap-4 text-sm text-muted-foreground">
							<span>Repository: {session.repository ?? "-"}</span>
							<span>Author: {issue?.author ?? "-"}</span>
						</div>
					</div>
					{issue?.url && (
						<Button variant="outline" size="sm" asChild>
							<a href={issue.url} target="_blank" rel="noopener noreferrer">
								<ExternalLinkIcon className="size-4" />
								<span>View on GitHub</span>
							</a>
						</Button>
					)}
				</div>
				{issue?.body && (
					<div className="mt-4 rounded-lg border border-border bg-muted/50 p-4">
						<h3 className="mb-2 font-semibold text-sm text-muted-foreground">
							Issue Description
						</h3>
						<p className="whitespace-pre-wrap text-sm">{issue.body}</p>
					</div>
				)}
			</section>

			{/* Stage progress section */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="mb-4 text-lg font-semibold text-card-foreground">
					Workflow Progress
				</h2>
				<StageProgress
					currentStage={session.currentStage}
					status={session.status}
				/>
			</section>

			{/* Actions section */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="mb-4 text-lg font-semibold text-card-foreground">
					Actions
				</h2>
				<div className="flex flex-wrap items-center gap-3">
					{/* Pause/Resume buttons */}
					{canPause && session.status === SessionStatus.active && (
						<Button
							variant="outline"
							size="sm"
							onClick={() =>
								pauseMutation.mutate({
									id,
									data: { reason: "User paused session" },
								})
							}
							disabled={pauseMutation.isPending}
						>
							<PauseIcon className="size-4" />
							<span>Pause</span>
						</Button>
					)}
					{canResume && (
						<Button
							variant="outline"
							size="sm"
							onClick={() => resumeMutation.mutate({ id, data: {} })}
							disabled={resumeMutation.isPending}
						>
							<PlayIcon className="size-4" />
							<span>Resume</span>
						</Button>
					)}

					{/* Rollback button with stage selector */}
					{canRollback && (
						<div className="flex items-center gap-2">
							<Select
								value={rollbackStage ?? undefined}
								onValueChange={(value) =>
									setRollbackStage(value as RollbackSessionTargetStage)
								}
							>
								<SelectTrigger size="sm">
									<SelectValue placeholder="Rollback to..." />
								</SelectTrigger>
								<SelectContent>
									{rollbackStages.map((stage) => (
										<SelectItem key={stage.key} value={stage.key}>
											{stage.label}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
							<Button
								variant="outline"
								size="sm"
								onClick={() =>
									rollbackStage &&
									rollbackMutation.mutate({
										id,
										data: {
											targetStage: rollbackStage,
											reason: "User rollback",
										},
									})
								}
								disabled={!rollbackStage || rollbackMutation.isPending}
							>
								<RotateCcwIcon className="size-4" />
								<span>Rollback</span>
							</Button>
						</div>
					)}

					{/* Terminate button */}
					{canTerminate && (
						<Button
							variant="destructive"
							size="sm"
							onClick={() =>
								terminateMutation.mutate({
									id,
									data: { reason: "User terminated session" },
								})
							}
							disabled={terminateMutation.isPending}
						>
							<StopCircleIcon className="size-4" />
							<span>Terminate</span>
						</Button>
					)}

					{/* Restart button */}
					{canRestart && (
						<Button
							variant="default"
							size="sm"
							onClick={() => restartMutation.mutate({ id, data: {} })}
							disabled={restartMutation.isPending}
						>
							<RefreshCcwIcon className="size-4" />
							<span>Restart</span>
						</Button>
					)}
				</div>
			</section>

			{/* Tasks section */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="mb-4 text-lg font-semibold text-card-foreground">
					Tasks ({tasks.length})
				</h2>
				<TaskList sessionId={id} tasks={tasks} />
			</section>
		</div>
	)
}
