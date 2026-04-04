import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
	component: Dashboard,
});

function Dashboard() {
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
				{[
					{ label: "Active Sessions", value: "12", color: "text-primary" },
					{
						label: "Pending Issues",
						value: "28",
						color: "text-secondary-foreground",
					},
					{
						label: "Completed PRs",
						value: "156",
						color: "text-sidebar-primary",
					},
					{
						label: "Success Rate",
						value: "94.2%",
						color: "text-sidebar-accent-foreground",
					},
				].map((stat) => (
					<div
						key={stat.label}
						className="rounded-xl border border-border bg-card p-4 shadow-sm"
					>
						<p className="text-sm font-medium text-muted-foreground">
							{stat.label}
						</p>
						<p className={`mt-1 text-2xl font-bold ${stat.color}`}>
							{stat.value}
						</p>
					</div>
				))}
			</section>

			{/* Workflow Stages */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Workflow Stages
				</h2>
				<div className="mt-4 flex flex-wrap gap-2">
					{[
						{ name: "Clarification", icon: "📋" },
						{ name: "Design", icon: "🎨" },
						{ name: "TaskBreakdown", icon: "📝" },
						{ name: "Execution", icon: "⚡" },
						{ name: "PullRequest", icon: "🔄" },
						{ name: "Completed", icon: "✅" },
					].map((stage) => (
						<div
							key={stage.name}
							className="flex items-center gap-2 rounded-lg border border-border bg-background px-3 py-2"
						>
							<span className="text-lg">{stage.icon}</span>
							<span className="text-sm font-medium text-foreground">
								{stage.name}
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
					{[
						{
							id: 1,
							title: "Issue #123: Add user authentication",
							status: "Execution",
							time: "2 hours ago",
						},
						{
							id: 2,
							title: "Issue #122: Fix database connection",
							status: "Completed",
							time: "5 hours ago",
						},
						{
							id: 3,
							title: "Issue #121: Update API documentation",
							status: "Clarification",
							time: "1 day ago",
						},
					].map((activity) => (
						<div
							key={activity.id}
							className="flex items-center justify-between rounded-lg border border-border bg-background p-3"
						>
							<div className="flex flex-col gap-1">
								<p className="text-sm font-medium text-foreground">
									{activity.title}
								</p>
								<p className="text-xs text-muted-foreground">{activity.time}</p>
							</div>
							<span
								className={`rounded-full px-2 py-1 text-xs font-medium ${
									activity.status === "Completed"
										? "bg-sidebar-accent text-sidebar-accent-foreground"
										: "bg-secondary text-secondary-foreground"
								}`}
							>
								{activity.status}
							</span>
						</div>
					))}
				</div>
			</section>
		</div>
	);
}
