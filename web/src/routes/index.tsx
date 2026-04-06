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
import { CircleDotIcon, ShieldCheckIcon, AlertCircleIcon, GitPullRequestDraftIcon, ActivityIcon, GitBranchIcon, ChevronRightIcon } from "lucide-react";

export const Route = createFileRoute("/")({
	component: Dashboard,
});

const skeletonIds = ["skeleton-1", "skeleton-2", "skeleton-3", "skeleton-4"] as const;

function calculateStats(sessions: Session[] | undefined) {
	if (!sessions) {
		return {
			activeSessions: 0,
			pendingIssues: 0,
			completedPRs: 0,
			successRate: 0,
		};
	}
	const activeCount = sessions.filter((s) => s.status === SessionStatus.active).length;
	const pendingCount = sessions.filter((s) => s.status !== SessionStatus.completed && s.status !== SessionStatus.terminated).length;
	const completedPRCount = sessions.filter((s) => s.status === SessionStatus.completed && s.prNumber).length;
	const completedCount = sessions.filter((s) => s.status === SessionStatus.completed).length;
	const terminatedCount = sessions.filter((s) => s.status === SessionStatus.terminated).length;
	const totalFinished = completedCount + terminatedCount;
	const successRate = totalFinished > 0 ? (completedCount / totalFinished) * 100 : 0;
	return {
		activeSessions: activeCount,
		pendingIssues: pendingCount,
		completedPRs: completedPRCount,
		successRate,
	};
}

function StatsCard({
	label,
	value,
	icon: Icon,
	colorClass,
	isLoading,
	trend
}: {
	label: string;
	value: string | number;
	icon: any;
	colorClass: string;
	isLoading: boolean;
	trend?: string;
}) {
	return (
		<div className="glass-card rounded-2xl p-6 relative overflow-hidden group">
			<div className={`absolute top-0 right-0 p-4 opacity-10 transition-transform duration-500 group-hover:scale-125 ${colorClass}`}>
				<Icon size={80} />
			</div>
			
			<div className="flex items-center gap-4 relative z-10">
				<div className={`p-3 rounded-xl bg-background shadow-sm ${colorClass}`}>
					<Icon size={24} />
				</div>
				<div>
					<p className="text-sm font-medium text-muted-foreground">{label}</p>
					{isLoading ? (
						<Skeleton className="mt-1 h-8 w-20" />
					) : (
						<div className="flex items-baseline gap-2">
							<p className="mt-1 text-3xl font-bold tracking-tight text-foreground">{value}</p>
							{trend && <span className="text-xs font-medium text-primary bg-primary/10 px-2 py-0.5 rounded-full">{trend}</span>}
						</div>
					)}
				</div>
			</div>
		</div>
	);
}

function ActivityItem({ session, index }: { session: Session; index: number }) {
	const stage = session.currentStage;
	const stageInfo = stage ? stageConfig[stage] : null;
	const stageColor = getStageColor(stage);

	return (
		<div 
			className="glass-card glass-card-hover rounded-xl p-4 flex items-center justify-between group animate-slide-up-fade"
			style={{ animationDelay: `${index * 100}ms` }}
		>
			<div className="flex items-start gap-4">
				<div className="mt-1 p-2 bg-primary/10 rounded-lg text-primary">
					<GitBranchIcon size={18} />
				</div>
				<div className="flex flex-col gap-1.5">
					<p className="text-sm font-semibold text-foreground group-hover:text-primary transition-colors line-clamp-1">
						{session.issueTitle}
					</p>
					<div className="flex items-center gap-3 text-xs text-muted-foreground">
						<span className="flex items-center gap-1 font-medium bg-secondary px-2 py-0.5 rounded-md text-secondary-foreground">
							{session.repository} #{session.issueNumber}
						</span>
						<span className="flex items-center gap-1">
							<ActivityIcon size={12} />
							{formatRelativeTime(session.updatedAt)}
						</span>
					</div>
				</div>
			</div>
			<div className="flex items-center gap-4">
				<span className={`px-2.5 py-1 rounded-full text-xs font-semibold tracking-wide ${stageColor} shadow-sm`}>
					{stageInfo?.label ?? "Unknown"}
				</span>
				<ChevronRightIcon size={16} className="text-muted-foreground opacity-0 -ml-2 group-hover:opacity-100 group-hover:ml-0 transition-all" />
			</div>
		</div>
	);
}

function Dashboard() {
	const { data, isLoading, isError, error } = useGetApiV1Sessions({ pageSize: 100 });

	const sessions = data?.status === 200 ? data.data.data : undefined;
	const stats = calculateStats(sessions);

	const activeSessions = sessions
		?.filter((s) => s.status === SessionStatus.active)
		?.sort((a, b) => {
			const aTime = a.updatedAt ? new Date(a.updatedAt).getTime() : 0;
			const bTime = b.updatedAt ? new Date(b.updatedAt).getTime() : 0;
			return bTime - aTime;
		})
		?.slice(0, 5);

	const recentActivity = sessions
		?.sort((a, b) => {
			const aTime = a.updatedAt ? new Date(a.updatedAt).getTime() : 0;
			const bTime = b.updatedAt ? new Date(b.updatedAt).getTime() : 0;
			return bTime - aTime;
		})
		?.slice(0, 5);

	return (
		<div className="space-y-8 animate-blur-in max-w-7xl mx-auto pb-10">
			{/* Welcome Section */}
			<section className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-primary/90 via-primary to-primary/80 px-8 py-12 shadow-xl shadow-primary/20 text-primary-foreground border border-primary-foreground/10">
				<div className="absolute inset-0 bg-[url('https://transparenttextures.com/patterns/cubes.png')] opacity-10"></div>
				<div className="absolute right-0 top-0 w-1/2 h-full bg-gradient-to-l from-white/10 to-transparent skew-x-12 -mr-16"></div>
				
				<div className="relative z-10 max-w-2xl">
					<h1 className="text-4xl md:text-5xl font-bold tracking-tight mb-4 filter drop-shadow-md">
						Welcome to Litchi
					</h1>
					<p className="text-lg text-primary-foreground/90 font-medium leading-relaxed max-w-xl">
						Your automated AI development agent. Seamlessly connecting GitHub Issues to brilliant Pull Requests with intelligent code generation.
					</p>
					
					<div className="mt-8 flex gap-4">
						<button className="bg-background text-foreground px-6 py-2.5 rounded-full font-semibold text-sm shadow-lg hover:shadow-xl hover:-translate-y-0.5 transition-all">
							New Session
						</button>
						<button className="bg-primary-foreground/20 backdrop-blur-md text-primary-foreground border border-primary-foreground/30 px-6 py-2.5 rounded-full font-semibold text-sm hover:bg-primary-foreground/30 transition-all">
							View Documentation
						</button>
					</div>
				</div>
			</section>

			{/* Stats Grid */}
			<section className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
				<StatsCard
					label="Active Sessions"
					value={stats.activeSessions}
					icon={ActivityIcon}
					colorClass="text-blue-500"
					isLoading={isLoading}
					trend="+2 today"
				/>
				<StatsCard
					label="Pending Issues"
					value={stats.pendingIssues}
					icon={AlertCircleIcon}
					colorClass="text-amber-500"
					isLoading={isLoading}
				/>
				<StatsCard
					label="Completed PRs"
					value={stats.completedPRs}
					icon={GitPullRequestDraftIcon}
					colorClass="text-emerald-500"
					isLoading={isLoading}
					trend="↑ 12%"
				/>
				<StatsCard
					label="Success Rate"
					value={`${stats.successRate.toFixed(1)}%`}
					icon={ShieldCheckIcon}
					colorClass="text-purple-500"
					isLoading={isLoading}
				/>
			</section>

			{isError && (
				<section className="rounded-2xl border border-destructive/50 bg-destructive/10 p-6 shadow-sm flex items-center gap-3 text-destructive animate-slide-up-fade">
					<AlertCircleIcon size={24} />
					<p className="font-medium">Error loading sessions: {error?.message ?? "Unknown error"}</p>
				</section>
			)}

			<div className="grid lg:grid-cols-3 gap-8">
				<div className="lg:col-span-2 space-y-8">
					{/* Active Sessions */}
					<section>
						<div className="flex items-center justify-between mb-6">
							<h2 className="text-2xl font-bold flex items-center gap-2">
								<span className="w-2 h-8 rounded-full bg-primary block"></span>
								Active Sessions
							</h2>
							<button className="text-sm font-medium text-primary hover:underline">View all</button>
						</div>
						
						<div className="space-y-4 relative">
							{isLoading ? (
								<div className="space-y-4">
									{skeletonIds.map((id) => (
										<div key={`active-${id}`} className="glass-card rounded-xl p-4 flex justify-between items-center">
											<div className="flex gap-4 items-center">
												<Skeleton className="h-10 w-10 rounded-full" />
												<div className="space-y-2">
													<Skeleton className="h-5 w-48" />
													<Skeleton className="h-4 w-32" />
												</div>
											</div>
											<Skeleton className="h-6 w-24 rounded-full" />
										</div>
									))}
								</div>
							) : activeSessions && activeSessions.length > 0 ? (
								activeSessions.map((session, i) => (
									<ActivityItem key={session.id} session={session} index={i} />
								))
							) : (
								<div className="glass-card rounded-2xl p-10 flex flex-col items-center justify-center text-center border-dashed">
									<div className="bg-primary/5 p-4 rounded-full mb-4">
										<CircleDotIcon size={32} className="text-muted-foreground" />
									</div>
									<h3 className="text-lg font-semibold mb-2">No Active Sessions</h3>
									<p className="text-muted-foreground max-w-sm">There are currently no active development sessions. Start a new workflow to see activity here.</p>
								</div>
							)}
						</div>
					</section>

					{/* Recent Activity */}
					<section>
						<div className="flex items-center justify-between mb-6">
							<h2 className="text-2xl font-bold flex items-center gap-2">
								<span className="w-2 h-8 rounded-full bg-secondary-foreground block"></span>
								Recent History
							</h2>
						</div>
						<div className="space-y-4">
							{isLoading ? (
								<div className="space-y-4">
									{skeletonIds.map((id) => (
										<div key={`activity-${id}`} className="glass-card rounded-xl p-4 flex justify-between items-center">
											<div className="flex gap-4 items-center">
												<Skeleton className="h-10 w-10 rounded-full" />
												<div className="space-y-2">
													<Skeleton className="h-5 w-48" />
													<Skeleton className="h-4 w-32" />
												</div>
											</div>
											<Skeleton className="h-6 w-24 rounded-full" />
										</div>
									))}
								</div>
							) : recentActivity && recentActivity.length > 0 ? (
								recentActivity.map((session, i) => (
									<ActivityItem key={session.id} session={session} index={i} />
								))
							) : (
								<div className="glass-card rounded-2xl p-8 flex flex-col items-center justify-center text-center">
									<p className="text-muted-foreground font-medium">No recent activity to display.</p>
								</div>
							)}
						</div>
					</section>
				</div>

				{/* Sidebar/Context info */}
				<div className="space-y-8">
					<section className="glass-card rounded-3xl p-6 shadow-sm border-t-4 border-t-primary">
						<h2 className="text-lg font-bold mb-4 flex items-center gap-2">
							<ActivityIcon className="text-primary" size={20} />
							Workflow Stages
						</h2>
						<p className="text-sm text-muted-foreground mb-6">
							Each AI development session progresses systematically through the following stages:
						</p>
						<div className="space-y-4">
							{workflowStages.map((stage, i) => (
								<div key={stage.key} className="flex gap-4 relative animate-slide-up-fade" style={{ animationDelay: `${i * 100}ms` }}>
									<div className="flex flex-col items-center">
										<div className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold text-white shadow-md z-10 ${i === workflowStages.length - 1 ? 'bg-emerald-500' : 'bg-primary'}`}>
											{stage.icon}
										</div>
										{i !== workflowStages.length - 1 && (
											<div className="w-0.5 h-full bg-border absolute top-8 bottom-[-16px]"></div>
										)}
									</div>
									<div className="pb-4 pt-1">
										<p className="text-sm font-bold text-foreground">{stage.label}</p>
										<p className="text-xs text-muted-foreground mt-1 leading-relaxed">
											{"Automated step execution and status tracking."}
										</p>
									</div>
								</div>
							))}
						</div>
					</section>
					
					<section className="relative rounded-3xl bg-card overflow-hidden border border-border p-6 shadow-sm group">
						<div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity"></div>
						<h3 className="text-sm font-semibold text-muted-foreground mb-2 tracking-wider uppercase">System Status</h3>
						<div className="flex items-center gap-3">
							<span className="relative flex h-3 w-3">
							  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
							  <span className="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
							</span>
							<span className="font-bold text-lg text-foreground">All systems operational</span>
						</div>
						<p className="text-xs text-muted-foreground mt-4">Agent workers are connected and ready to accept new issues.</p>
					</section>
				</div>
			</div>
		</div>
	);
}
