import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/settings/")({
	component: SettingsPage,
});

function SettingsPage() {
	return (
		<div className="space-y-6">
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
					Settings
				</h1>
				<p className="mt-2 text-muted-foreground">
					Configure system preferences and integrations.
				</p>
			</section>

			{/* Settings Groups */}
			<section className="grid gap-4 lg:grid-cols-2">
				<div className="rounded-xl border border-border bg-card p-6 shadow-sm">
					<h2 className="text-lg font-semibold text-card-foreground">
						GitHub Integration
					</h2>
					<div className="mt-4 space-y-3">
						<div className="flex items-center justify-between">
							<span className="text-sm text-muted-foreground">
								Token Status
							</span>
							<span className="rounded-full bg-sidebar-accent px-2 py-1 text-xs font-medium text-sidebar-accent-foreground">
								Connected
							</span>
						</div>
						<div className="flex items-center justify-between">
							<span className="text-sm text-muted-foreground">Webhook URL</span>
							<span className="text-xs text-foreground">
								https://api.litchi.dev/webhook
							</span>
						</div>
					</div>
				</div>

				<div className="rounded-xl border border-border bg-card p-6 shadow-sm">
					<h2 className="text-lg font-semibold text-card-foreground">
						Agent Configuration
					</h2>
					<div className="mt-4 space-y-3">
						<div className="flex items-center justify-between">
							<span className="text-sm text-muted-foreground">Retry Limit</span>
							<span className="text-xs text-foreground">3 attempts</span>
						</div>
						<div className="flex items-center justify-between">
							<span className="text-sm text-muted-foreground">Timeout</span>
							<span className="text-xs text-foreground">30 minutes</span>
						</div>
					</div>
				</div>
			</section>
		</div>
	);
}
