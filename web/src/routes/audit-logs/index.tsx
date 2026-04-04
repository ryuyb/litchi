import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/audit-logs/")({
	component: AuditLogsPage,
});

function AuditLogsPage() {
	return (
		<div className="space-y-6">
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
					Audit Logs
				</h1>
				<p className="mt-2 text-muted-foreground">
					Track all system activities and changes.
				</p>
			</section>

			{/* Audit Log List Placeholder */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Recent Activities
				</h2>
				<div className="mt-4 space-y-2">
					{[
						{
							action: "Issue #123 started",
							user: "system",
							time: "2024-01-15 10:30:00",
						},
						{
							action: "Repository litchi configured",
							user: "ryuyb",
							time: "2024-01-15 09:15:00",
						},
						{
							action: "PR #156 merged",
							user: "system",
							time: "2024-01-14 18:45:00",
						},
					].map((log) => (
						<div
							key={log.action}
							className="flex items-center justify-between rounded-lg border border-border bg-background p-3 text-sm"
						>
							<div className="flex flex-col gap-1">
								<p className="font-medium text-foreground">{log.action}</p>
								<p className="text-xs text-muted-foreground">by {log.user}</p>
							</div>
							<p className="text-xs text-muted-foreground">{log.time}</p>
						</div>
					))}
				</div>
			</section>
		</div>
	);
}
