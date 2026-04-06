/**
 * System health status display component.
 * Shows health checks for database, GitHub, Git, etc.
 */

import {
	CheckCircleIcon,
	LoaderIcon,
	MinusCircleIcon,
	XCircleIcon,
} from "lucide-react";
import type { HealthCheckItem } from "#/api/schemas/healthCheckItem";
import { useGetApiV1HealthDetail } from "#/api/system/system";

// Status icon and color mapping
function getStatusDisplay(status: string | undefined) {
	switch (status) {
		case "pass":
			return {
				icon: CheckCircleIcon,
				color: "text-green-500",
				bgColor: "bg-green-500/10",
				label: "Healthy",
			};
		case "fail":
			return {
				icon: XCircleIcon,
				color: "text-red-500",
				bgColor: "bg-red-500/10",
				label: "Failed",
			};
		case "warn":
			return {
				icon: MinusCircleIcon,
				color: "text-yellow-500",
				bgColor: "bg-yellow-500/10",
				label: "Warning",
			};
		default:
			return {
				icon: MinusCircleIcon,
				color: "text-muted-foreground",
				bgColor: "bg-muted/50",
				label: "Unknown",
			};
	}
}

// Individual health check card
function HealthCheckCard({ check }: { check: HealthCheckItem }) {
	const display = getStatusDisplay(check.status);
	const Icon = display.icon;

	return (
		<div className={`rounded-lg border border-border p-4 ${display.bgColor}`}>
			<div className="flex items-center justify-between">
				<span className="font-medium text-card-foreground capitalize">
					{check.name ?? "Unknown"}
				</span>
				<div className={`flex items-center gap-1.5 ${display.color}`}>
					<Icon className="size-4" />
					<span className="text-xs font-medium">{display.label}</span>
				</div>
			</div>
			{check.latencyMs !== undefined && (
				<p className="mt-2 text-xs text-muted-foreground">
					Latency: {check.latencyMs}ms
				</p>
			)}
			{check.message && (
				<p className="mt-1 text-xs text-muted-foreground">{check.message}</p>
			)}
			{check.error && (
				<p className="mt-1 text-xs text-destructive">{check.error}</p>
			)}
		</div>
	);
}

// Overall status badge
function OverallStatusBadge({ status }: { status: string | undefined }) {
	const statusConfig = {
		healthy: {
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
			label: "All Systems Operational",
		},
		degraded: {
			color:
				"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
			label: "Degraded",
		},
		unhealthy: {
			color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
			label: "System Issues",
		},
	};

	const config =
		statusConfig[status as keyof typeof statusConfig] ?? statusConfig.unhealthy;

	return (
		<span
			className={`rounded-full px-3 py-1 text-xs font-medium ${config.color}`}
		>
			{config.label}
		</span>
	);
}

export function HealthStatusSection() {
	const {
		data: response,
		isLoading,
		error,
	} = useGetApiV1HealthDetail({
		query: {
			refetchInterval: 30000, // Refresh every 30 seconds
		},
	});

	// Loading state
	if (isLoading) {
		return (
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h3 className="text-lg font-semibold text-card-foreground">
					System Health
				</h3>
				<div className="mt-4 flex items-center gap-2">
					<LoaderIcon className="size-5 animate-spin text-muted-foreground" />
					<span className="text-muted-foreground">
						Checking system health...
					</span>
				</div>
			</section>
		);
	}

	// Error state
	if (error || response?.status !== 200) {
		return (
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h3 className="text-lg font-semibold text-card-foreground">
					System Health
				</h3>
				<div className="mt-4 rounded-lg border border-destructive bg-destructive/10 p-4">
					<p className="text-sm text-destructive">
						Failed to load health status
					</p>
				</div>
			</section>
		);
	}

	const healthDetail = response.data;
	const checks = healthDetail?.checks ?? [];

	return (
		<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
			<div className="flex items-center justify-between">
				<div>
					<h3 className="text-lg font-semibold text-card-foreground">
						System Health
					</h3>
					<p className="mt-1 text-sm text-muted-foreground">
						Real-time status of system components
					</p>
				</div>
				<OverallStatusBadge status={healthDetail?.status} />
			</div>

			{healthDetail?.version && (
				<p className="mt-2 text-xs text-muted-foreground">
					Version: {healthDetail.version}
				</p>
			)}

			<div className="mt-4 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
				{checks.length > 0 ? (
					checks.map((check, index) => (
						<HealthCheckCard key={check.name ?? index} check={check} />
					))
				) : (
					<div className="col-span-full rounded-lg border border-border bg-muted/50 p-4 text-center">
						<p className="text-sm text-muted-foreground">
							No health checks available
						</p>
					</div>
				)}
			</div>
		</section>
	);
}
