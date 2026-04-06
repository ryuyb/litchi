import { createFileRoute } from "@tanstack/react-router";
import type { Session } from "#/api/schemas";
import { SessionStatus } from "#/api/schemas";
import { useGetApiV1Sessions } from "#/api/sessions/sessions";
import { Skeleton } from "#/components/ui/skeleton";
import { formatRelativeTime } from "#/lib/date-utils";
import {
	getStageColor,
	stageConfig,
	workflowStages,
} from "#/lib/session-config";

export const Route = createFileRoute("/")({
	component: Dashboard,
});

// Skeleton item IDs for loading states
const skeletonIds = ["skeleton-1", "skeleton-2", "skeleton-3"] as const;

// Calculate statistics from sessions
function calculateStats(sessions: Session[] | undefined) {
	if (!sessions) {
		return {
			activeSessions: 0,
			pendingIssues: 0,
			completedPRs: 0,
			successRate: 0,
		};
	}

	const activeCount = sessions.filter(
		(s) => s.status === SessionStatus.active,
	).length;

	// Pending issues: sessions that are not completed or terminated
	const pendingCount = sessions.filter(
		(s) =>
			s.status !== SessionStatus.completed &&
			s.status !== SessionStatus.terminated,
	).length;

	// Completed PRs: sessions with completed status and prNumber
	const completedPRCount = sessions.filter(
		(s) => s.status === SessionStatus.completed && s.prNumber,
	).length;

	// Success rate: completed / (completed + terminated)
	const completedCount = sessions.filter(
		(s) => s.status === SessionStatus.completed,
	).length;
	const terminatedCount = sessions.filter(
		(s) => s.status === SessionStatus.terminated,
	).length;
	const totalFinished = completedCount + terminatedCount;
	const successRate =
		totalFinished > 0 ? (completedCount / totalFinished) * 100 : 0;

	return {
		activeSessions: activeCount,
		pendingIssues: pendingCount,
		completedPRs: completedPRCount,
		successRate,
	};
}

// Skeleton row component for loading state
function SkeletonRow({ id }: { id: string }) {
	return (
		<div
			key={id}
			className="flex items-center justify-between rounded-lg border border-border bg-background p-3"
		>
			<div className="flex flex-col gap-1">
				<Skeleton className="h-4 w-48" />
				<Skeleton className="h-3 w-24" />
			</div>
			<Skeleton className="h-6 w-20 rounded-full" />
		</div>
	);
}

// Stats Card component
function StatsCard({
	label,
	value,
	color,
	isLoading,
}: {
	label: string;
	value: string | number;
	color: string;
	isLoading: boolean;
}) {
	return (
		<div className="rounded-xl border border-border bg-card p-4 shadow-sm">
			<p className="text-sm font-medium text-muted-foreground">{label}</p>
			{isLoading ? (
				<Skeleton className="mt-1 h-8 w-20" />
			) : (
				<p className={`mt-1 text-2xl font-bold ${color}`}>{value}</p>
			)}
		</div>
	);
}

// Activity Item component
function ActivityItem({ session }: { session: Session }) {
	const stage = session.currentStage;
	const stageInfo = stage ? stageConfig[stage] : null;
	const stageColor = getStageColor(stage);

	return (
		<div className="flex items-center justify-between rounded-lg border border-border bg-background p-3">
			<div className="flex flex-col gap-1">
				<p className="text-sm font-medium text-foreground">
					{session.repository} #{session.issueNumber}: {session.issueTitle}
				</p>
				<p className="text-xs text-muted-foreground">
					{formatRelativeTime(session.updatedAt)}
				</p>
			</div>
			<span
				className={`rounded-full px-2 py-1 text-xs font-medium ${stageColor}`}
			>
				{stageInfo?.label ?? "Unknown"}
			</span>
		</div>
	);
}

function Dashboard() {
	// Fetch all sessions with a larger page size for statistics
	const { data, isLoading, isError, error } = useGetApiV1Sessions(
		{}, // body
		{ pageSize: 100 }, // params - get up to 100 sessions for stats
	);

	const sessions = data?.status === 200 ? data.data.data : undefined;
	const stats = calculateStats(sessions);

	// Get active sessions (sorted by updatedAt, most recent first)
	const activeSessions = sessions
		?.filter((s) => s.status === SessionStatus.active)
		?.sort((a, b) => {
			const aTime = a.updatedAt ? new Date(a.updatedAt).getTime() : 0;
			const bTime = b.updatedAt ? new Date(b.updatedAt).getTime() : 0;
			return bTime - aTime;
		})
		?.slice(0, 5);

	// Get recent activity (all sessions sorted by updatedAt)
	const recentActivity = sessions
		?.sort((a, b) => {
			const aTime = a.updatedAt ? new Date(a.updatedAt).getTime() : 0;
			const bTime = b.updatedAt ? new Date(b.updatedAt).getTime() : 0;
			return bTime - aTime;
		})
		?.slice(0, 5);

	return (
		<div className="space-y-6">
			{/* Welcome Section */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
					Welcome to Litchi
				</h1>
				<p className="mt-2 text-muted-foreground">
					Automated development agent system - from GitHub Issue to Pull
					Request.
				</p>
			</section>

			{/* Stats Cards */}
			<section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<StatsCard
					label="Active Sessions"
					value={stats.activeSessions}
					color="text-primary"
					isLoading={isLoading}
				/>
				<StatsCard
					label="Pending Issues"
					value={stats.pendingIssues}
					color="text-secondary-foreground"
					isLoading={isLoading}
				/>
				<StatsCard
					label="Completed PRs"
					value={stats.completedPRs}
					color="text-sidebar-primary"
					isLoading={isLoading}
				/>
				<StatsCard
					label="Success Rate"
					value={`${stats.successRate.toFixed(1)}%`}
					color="text-sidebar-accent-foreground"
					isLoading={isLoading}
				/>
			</section>

			{/* Error State */}
			{isError && (
				<section className="rounded-xl border border-destructive bg-card p-6 shadow-sm">
					<p className="text-destructive">
						Error loading sessions: {error?.message ?? "Unknown error"}
					</p>
				</section>
			)}

			{/* Active Sessions */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Active Sessions
				</h2>
				<div className="mt-4 space-y-3">
					{isLoading ? (
						skeletonIds.map((id) => (
							<SkeletonRow key={`active-${id}`} id={id} />
						))
					) : activeSessions && activeSessions.length > 0 ? (
						activeSessions.map((session) => (
							<ActivityItem key={session.id} session={session} />
						))
					) : (
						<p className="text-muted-foreground text-sm">
							No active sessions at the moment.
						</p>
					)}
				</div>
			</section>

			{/* Workflow Stages */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Workflow Stages
				</h2>
				<p className="mt-2 text-sm text-muted-foreground">
					Each session progresses through these stages:
				</p>
				<div className="mt-4 flex flex-wrap gap-2">
					{workflowStages.map((stage) => (
						<div
							key={stage.key}
							className="flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2"
						>
							<span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary text-xs font-bold text-primary-foreground">
								{stage.icon}
							</span>
							<span className="text-sm font-medium text-foreground">
								{stage.label}
							</span>
						</div>
					))}
				</div>
			</section>

			{/* Recent Activity */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Recent Activity
				</h2>
				<div className="mt-4 space-y-3">
					{isLoading ? (
						skeletonIds.map((id) => (
							<SkeletonRow key={`activity-${id}`} id={id} />
						))
					) : recentActivity && recentActivity.length > 0 ? (
						recentActivity.map((session) => (
							<ActivityItem key={session.id} session={session} />
						))
					) : (
						<p className="text-muted-foreground text-sm">
							No recent activity to display.
						</p>
					)}
				</div>
			</section>
		</div>
	);
}
