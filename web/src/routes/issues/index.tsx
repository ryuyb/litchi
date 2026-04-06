import { createFileRoute } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import type { Session } from "#/api/schemas";
import { SessionCurrentStage, SessionStatus } from "#/api/schemas";
import { DataTable } from "#/components/data-table";

export const Route = createFileRoute("/issues/")({
	component: IssuesPage,
});

// Demo data using generated types and constants
const demoSessions: Session[] = [
	{
		id: "550e8400-e29b-41d4-a716-446655440001",
		currentStage: SessionCurrentStage.execution,
		status: SessionStatus.active,
		repository: "ryuyb/litchi",
		issueNumber: 123,
		issueTitle: "Add user authentication",
		issue: {
			number: 123,
			title: "Add user authentication",
			body: "Implement OAuth2 authentication flow",
			author: "ryuyb",
			url: "https://github.com/ryuyb/litchi/issues/123",
		},
	},
	{
		id: "550e8400-e29b-41d4-a716-446655440002",
		currentStage: SessionCurrentStage.clarification,
		status: SessionStatus.active,
		repository: "ryuyb/litchi",
		issueNumber: 121,
		issueTitle: "Update API documentation",
		issue: {
			number: 121,
			title: "Update API documentation",
			body: "Add examples for new endpoints",
			author: "contributor",
			url: "https://github.com/ryuyb/litchi/issues/121",
		},
	},
	{
		id: "550e8400-e29b-41d4-a716-446655440003",
		currentStage: SessionCurrentStage.design,
		status: SessionStatus.paused,
		repository: "ryuyb/litchi",
		issueNumber: 120,
		issueTitle: "Fix database connection pool",
		issue: {
			number: 120,
			title: "Fix database connection pool",
			body: "Connection pool exhausted under high load",
			author: "ryuyb",
			url: "https://github.com/ryuyb/litchi/issues/120",
		},
	},
];

const columns: ColumnDef<Session>[] = [
	{
		accessorKey: "issueNumber",
		header: "Issue #",
		cell: ({ row }) => (
			<span className="font-mono text-sm">#{row.original.issueNumber}</span>
		),
	},
	{
		accessorKey: "issueTitle",
		header: "Title",
		cell: ({ row }) => (
			<span className="font-medium">{row.original.issueTitle}</span>
		),
	},
	{
		accessorKey: "repository",
		header: "Repository",
		cell: ({ row }) => (
			<span className="text-muted-foreground text-sm">
				{row.original.repository}
			</span>
		),
	},
	{
		accessorKey: "currentStage",
		header: "Stage",
		cell: ({ row }) => {
			const stage = row.original.currentStage;
			const stageColors: Record<SessionCurrentStage, string> = {
				[SessionCurrentStage.clarification]:
					"bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
				[SessionCurrentStage.design]:
					"bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
				[SessionCurrentStage.task_breakdown]:
					"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
				[SessionCurrentStage.execution]:
					"bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300",
				[SessionCurrentStage.pull_request]:
					"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
				[SessionCurrentStage.completed]:
					"bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300",
			};
			const colorClass = stage
				? stageColors[stage]
				: "bg-gray-100 text-gray-800";
			return (
				<span
					className={`rounded-full px-2 py-0.5 text-xs font-medium ${colorClass}`}
				>
					{stage?.replace("_", " ") ?? "unknown"}
				</span>
			);
		},
	},
	{
		accessorKey: "status",
		header: "Status",
		cell: ({ row }) => {
			const status = row.original.status;
			if (status === SessionStatus.terminated) {
				return (
					<span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800 dark:bg-red-900 dark:text-red-300">
						Terminated
					</span>
				);
			}
			if (status === SessionStatus.paused) {
				return (
					<span className="rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300">
						Paused
					</span>
				);
			}
			if (status === SessionStatus.completed) {
				return (
					<span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-800 dark:bg-gray-800 dark:text-gray-300">
						Completed
					</span>
				);
			}
			return (
				<span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-300">
					Active
				</span>
			);
		},
	},
];

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

			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Active Sessions
				</h2>
				<div className="mt-4">
					<DataTable columns={columns} data={demoSessions} />
				</div>
			</section>
		</div>
	);
}
