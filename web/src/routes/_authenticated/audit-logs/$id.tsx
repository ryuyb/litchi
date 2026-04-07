import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeftIcon, ExternalLinkIcon, LoaderIcon } from "lucide-react";
import {
	type getApiV1AuditIdResponse,
	useGetApiV1AuditId,
} from "#/api/audit/audit";
import type { AuditLog } from "#/api/schemas/auditLog";
import { Button } from "#/components/ui/button";
import {
	formatDuration,
	getOperationDisplay,
	getResultDisplay,
} from "#/lib/audit-config";
import { formatDateTime } from "#/lib/date-utils";

export const Route = createFileRoute("/_authenticated/audit-logs/$id")({
	component: AuditLogDetailPage,
});

// Type guard to check if response is successful (status 200 with AuditLog data)
function isSuccessResponse(
	response: getApiV1AuditIdResponse | undefined,
): response is { status: 200; data: AuditLog; headers: Headers } {
	return response?.status === 200;
}

function AuditLogDetailPage() {
	const { id } = Route.useParams();

	// Fetch audit log detail
	const { data: response, isLoading, error } = useGetApiV1AuditId(id);

	// Check if response is successful
	const isSuccess = isSuccessResponse(response);
	const log: AuditLog | undefined = isSuccess ? response.data : undefined;

	// Loading state
	if (isLoading) {
		return (
			<div className="flex items-center justify-center min-h-[400px] gap-2">
				<LoaderIcon className="size-8 animate-spin text-muted-foreground" />
				<span className="text-muted-foreground">Loading audit log...</span>
			</div>
		)
	}

	// Error state
	if (error) {
		return (
			<div className="rounded-xl border border-destructive bg-card p-6">
				<h2 className="text-lg font-semibold text-destructive">
					Error loading audit log
				</h2>
				<p className="mt-2 text-muted-foreground">
					Failed to load audit log details. Please try again.
				</p>
				<Button asChild className="mt-4">
					<Link to="/audit-logs">
						<ArrowLeftIcon className="size-4" />
						Back to Audit Logs
					</Link>
				</Button>
			</div>
		)
	}

	// No log found
	if (!isSuccess || !log) {
		return (
			<div className="rounded-xl border border-border bg-card p-6">
				<h2 className="text-lg font-semibold">Audit log not found</h2>
				<p className="mt-2 text-muted-foreground">
					The requested audit log could not be found.
				</p>
				<Button asChild className="mt-4">
					<Link to="/audit-logs">
						<ArrowLeftIcon className="size-4" />
						Back to Audit Logs
					</Link>
				</Button>
			</div>
		)
	}

	const operationDisplay = getOperationDisplay(log.operation);
	const resultDisplay = getResultDisplay(log.result);

	return (
		<div className="space-y-6">
			{/* Header with back link */}
			<div className="flex items-center gap-2">
				<Button variant="ghost" size="sm" asChild>
					<Link to="/audit-logs">
						<ArrowLeftIcon className="size-4" />
						<span>Back to Audit Logs</span>
					</Link>
				</Button>
			</div>

			{/* Basic info section */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
					Audit Log Details
				</h1>
				<p className="mt-1 text-sm text-muted-foreground font-mono">
					{log.id ?? "-"}
				</p>

				<div className="mt-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
					{/* Timestamp */}
					<div className="space-y-1">
						<span className="text-xs font-medium text-muted-foreground">
							Timestamp
						</span>
						<p className="text-sm">{formatDateTime(log.timestamp)}</p>
					</div>

					{/* Operation */}
					<div className="space-y-1">
						<span className="text-xs font-medium text-muted-foreground">
							Operation
						</span>
						<span
							className={`rounded-full px-2 py-0.5 text-xs font-medium inline-block ${operationDisplay.color}`}
						>
							{operationDisplay.label}
						</span>
					</div>

					{/* Result */}
					<div className="space-y-1">
						<span className="text-xs font-medium text-muted-foreground">
							Result
						</span>
						<span
							className={`rounded-full px-2 py-0.5 text-xs font-medium inline-block ${resultDisplay.color}`}
						>
							{resultDisplay.label}
						</span>
					</div>

					{/* Actor */}
					<div className="space-y-1">
						<span className="text-xs font-medium text-muted-foreground">
							Actor
						</span>
						<div className="flex items-center gap-2">
							<span className="text-sm font-medium">{log.actor ?? "-"}</span>
							{log.actorRole && (
								<span className="text-xs text-muted-foreground">
									({log.actorRole})
								</span>
							)}
						</div>
					</div>

					{/* Duration */}
					<div className="space-y-1">
						<span className="text-xs font-medium text-muted-foreground">
							Duration
						</span>
						<p className="text-sm">{formatDuration(log.duration)}</p>
					</div>

					{/* Resource */}
					<div className="space-y-1">
						<span className="text-xs font-medium text-muted-foreground">
							Resource Type
						</span>
						<p className="text-sm">{log.resourceType ?? "-"}</p>
					</div>
				</div>
			</section>

			{/* Related entities section */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Related Entities
				</h2>

				<div className="mt-4 space-y-3">
					{/* Session */}
					{log.sessionId && (
						<div className="flex items-center justify-between rounded-lg border border-border bg-muted/50 p-3">
							<div className="flex items-center gap-2">
								<span className="text-sm font-medium">Session:</span>
								<span className="text-sm text-muted-foreground font-mono">
									{log.sessionId}
								</span>
							</div>
							<Button variant="outline" size="sm" asChild>
								<Link to="/issues/$id" params={{ id: log.sessionId }}>
									<span>View Session</span>
								</Link>
							</Button>
						</div>
					)}

					{/* Repository */}
					{log.repository && (
						<div className="flex items-center justify-between rounded-lg border border-border bg-muted/50 p-3">
							<div className="flex items-center gap-2">
								<span className="text-sm font-medium">Repository:</span>
								<span className="text-sm text-muted-foreground font-mono">
									{log.repository}
								</span>
							</div>
							<Button variant="outline" size="sm" asChild>
								<Link
									to="/repositories/$name"
									params={{ name: log.repository }}
								>
									<span>View Repository</span>
								</Link>
							</Button>
						</div>
					)}

					{/* Issue */}
					{log.issueNumber && log.repository && (
						<div className="flex items-center justify-between rounded-lg border border-border bg-muted/50 p-3">
							<div className="flex items-center gap-2">
								<span className="text-sm font-medium">Issue:</span>
								<span className="text-sm text-muted-foreground">
									#{log.issueNumber}
								</span>
							</div>
							<Button variant="outline" size="sm" asChild>
								<a
									href={`https://github.com/${log.repository}/issues/${log.issueNumber}`}
									target="_blank"
									rel="noopener noreferrer"
									aria-label="View on GitHub (external link)"
								>
									<ExternalLinkIcon className="size-4" />
									<span>View on GitHub</span>
								</a>
							</Button>
						</div>
					)}

					{/* Resource ID */}
					{log.resourceId && !log.sessionId && !log.repository && (
						<div className="flex items-center justify-between rounded-lg border border-border bg-muted/50 p-3">
							<div className="flex items-center gap-2">
								<span className="text-sm font-medium">Resource ID:</span>
								<span className="text-sm text-muted-foreground font-mono">
									{log.resourceId}
								</span>
							</div>
						</div>
					)}
				</div>
			</section>

			{/* Operation context section */}
			{(log.output || log.error) && (
				<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
					<h2 className="text-lg font-semibold text-card-foreground">
						Operation Context
					</h2>

					<div className="mt-4 space-y-4">
						{/* Output */}
						{log.output && (
							<div className="space-y-2">
								<span className="text-xs font-medium text-muted-foreground">
									Output
								</span>
								<div className="rounded-lg border border-border bg-muted/50 p-3">
									<pre className="text-sm font-mono whitespace-pre-wrap overflow-auto max-h-[200px]">
										{log.output}
									</pre>
								</div>
							</div>
						)}

						{/* Error */}
						{log.error && (
							<div className="space-y-2">
								<span className="text-xs font-medium text-destructive">
									Error
								</span>
								<div className="rounded-lg border border-destructive bg-destructive/10 p-3">
									<pre className="text-sm font-mono text-destructive whitespace-pre-wrap overflow-auto max-h-[200px]">
										{log.error}
									</pre>
								</div>
							</div>
						)}
					</div>
				</section>
			)}
		</div>
	)
}
