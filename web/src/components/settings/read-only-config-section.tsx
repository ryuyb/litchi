/**
 * Read-only configuration display for system settings.
 * Shows database, server, and webhook configuration (non-editable).
 */
import type { Config } from "#/api/schemas/config";

interface ReadOnlyConfigSectionProps {
	config: Config | undefined;
}

export function ReadOnlyConfigSection({ config }: ReadOnlyConfigSectionProps) {
	return (
		<div className="space-y-6">
			{/* Database Config */}
			{config?.database && (
				<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
					<h3 className="text-lg font-semibold text-card-foreground">
						Database Configuration
					</h3>
					<p className="mt-1 text-sm text-muted-foreground">
						Database connection settings (read-only)
					</p>

					<div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
						<ConfigItem label="Host" value={config.database.host} />
						<ConfigItem label="Port" value={config.database.port?.toString()} />
						<ConfigItem label="Database" value={config.database.name} />
						<ConfigItem label="User" value={config.database.user} />
						<ConfigItem label="SSL Mode" value={config.database.sslmode} />
						<ConfigItem
							label="Max Open Connections"
							value={config.database.maxOpenConns?.toString()}
						/>
						<ConfigItem
							label="Max Idle Connections"
							value={config.database.maxIdleConns?.toString()}
						/>
						<ConfigItem
							label="Connection Max Lifetime"
							value={config.database.connMaxLifetime}
						/>
					</div>
				</section>
			)}

			{/* Server Config */}
			{config?.server && (
				<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
					<h3 className="text-lg font-semibold text-card-foreground">
						Server Configuration
					</h3>
					<p className="mt-1 text-sm text-muted-foreground">
						Server runtime settings (read-only)
					</p>

					<div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
						<ConfigItem label="Host" value={config.server.host} />
						<ConfigItem label="Port" value={config.server.port?.toString()} />
						<ConfigItem label="Mode" value={config.server.mode} />
						<ConfigItem label="Version" value={config.server.version} />
					</div>
				</section>
			)}

			{/* Webhook Config */}
			{config?.webhook && (
				<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
					<h3 className="text-lg font-semibold text-card-foreground">
						Webhook Configuration
					</h3>
					<p className="mt-1 text-sm text-muted-foreground">
						Webhook processing settings (read-only)
					</p>

					<div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
						<BooleanItem
							label="Idempotency Enabled"
							value={config.webhook.idempotencyEnabled}
						/>
						<ConfigItem
							label="Idempotency TTL"
							value={config.webhook.idempotencyTTL}
						/>
						<BooleanItem
							label="Auto Cleanup"
							value={config.webhook.idempotencyAutoCleanup}
						/>
					</div>
				</section>
			)}

			{/* No config message */}
			{!config?.database && !config?.server && !config?.webhook && (
				<div className="rounded-xl border border-border bg-card p-6 shadow-sm">
					<p className="text-sm text-muted-foreground">
						No system configuration available
					</p>
				</div>
			)}
		</div>
	);
}

// Config item component
function ConfigItem({ label, value }: { label: string; value?: string }) {
	return (
		<div className="rounded-lg border border-border bg-muted/30 p-3">
			<span className="text-xs font-medium text-muted-foreground">{label}</span>
			<p className="mt-1 text-sm font-medium text-foreground">{value ?? "-"}</p>
		</div>
	);
}

// Boolean config item component
function BooleanItem({ label, value }: { label: string; value?: boolean }) {
	return (
		<div className="rounded-lg border border-border bg-muted/30 p-3">
			<span className="text-xs font-medium text-muted-foreground">{label}</span>
			<p className="mt-1 text-sm font-medium text-foreground">
				{value !== undefined ? (value ? "Yes" : "No") : "-"}
			</p>
		</div>
	);
}
