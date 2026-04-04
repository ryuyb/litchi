import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/repositories/")({
	component: RepositoriesPage,
});

function RepositoriesPage() {
	return (
		<div className="space-y-6">
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
					Repositories
				</h1>
				<p className="mt-2 text-muted-foreground">
					Configure and manage connected GitHub repositories.
				</p>
			</section>

			{/* Repository List Placeholder */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Connected Repositories
				</h2>
				<div className="mt-4 space-y-3">
					{[
						{ name: "litchi", owner: "ryuyb", issues: 12, prs: 156 },
						{ name: "demo-project", owner: "ryuyb", issues: 5, prs: 23 },
					].map((repo) => (
						<div
							key={repo.name}
							className="flex items-center justify-between rounded-lg border border-border bg-background p-4"
						>
							<div className="flex flex-col gap-1">
								<p className="text-sm font-medium text-foreground">
									{repo.owner}/{repo.name}
								</p>
								<p className="text-xs text-muted-foreground">
									{repo.issues} issues · {repo.prs} PRs
								</p>
							</div>
							<span className="rounded-full bg-sidebar-accent px-3 py-1 text-xs font-medium text-sidebar-accent-foreground">
								Active
							</span>
						</div>
					))}
				</div>
			</section>
		</div>
	);
}
