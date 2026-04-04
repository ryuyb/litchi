import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/issues/")({
	component: IssuesPage,
});

function IssuesPage() {
	return (
		<div className="space-y-6">
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
					Issues
				</h1>
				<p className="mt-2 text-muted-foreground">
					Manage and track GitHub Issues across your repositories.
				</p>
			</section>

			{/* Issue List Placeholder */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Active Issues
				</h2>
				<div className="mt-4 space-y-3">
					{[
						{
							id: 123,
							title: "Add user authentication",
							repo: "litchi",
							stage: "Execution",
						},
						{
							id: 121,
							title: "Update API documentation",
							repo: "litchi",
							stage: "Clarification",
						},
						{
							id: 120,
							title: "Fix database connection pool",
							repo: "litchi",
							stage: "Design",
						},
					].map((issue) => (
						<div
							key={issue.id}
							className="flex items-center justify-between rounded-lg border border-border bg-background p-4"
						>
							<div className="flex flex-col gap-1">
								<p className="text-sm font-medium text-foreground">
									#{issue.id}: {issue.title}
								</p>
								<p className="text-xs text-muted-foreground">{issue.repo}</p>
							</div>
							<span className="rounded-full bg-secondary px-3 py-1 text-xs font-medium text-secondary-foreground">
								{issue.stage}
							</span>
						</div>
					))}
				</div>
			</section>
		</div>
	);
}
