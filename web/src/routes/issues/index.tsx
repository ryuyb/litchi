import { createFileRoute } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import type { Stage, WorkSession } from "#/api/schemas";
import { DataTable } from "#/components/data-table";

export const Route = createFileRoute("/issues/")({
	component: IssuesPage,
});

// Demo data using generated types
const demoSessions: WorkSession[] = [
	{
		id: "550e8400-e29b-41d4-a716-446655440001",
		stage: "execution" as Stage,
		issue: {
			id: "550e8400-e29b-41d4-a716-446655440010",
			number: 123,
			title: "Add user authentication",
			body: "Implement OAuth2 authentication flow",
			repository: "ryuyb/litchi",
			author: "ryuyb",
		},
		paused: false,
		terminated: false,
	},
	{
		id: "550e8400-e29b-41d4-a716-446655440002",
		stage: "clarification" as Stage,
		issue: {
			id: "550e8400-e29b-41d4-a716-446655440011",
			number: 121,
			title: "Update API documentation",
			body: "Add examples for new endpoints",
			repository: "ryuyb/litchi",
			author: "contributor",
		},
		paused: false,
		terminated: false,
	},
	{
		id: "550e8400-e29b-41d4-a716-446655440003",
		stage: "design" as Stage,
		issue: {
			id: "550e8400-e29b-41d4-a716-446655440012",
			number: 120,
			title: "Fix database connection pool",
			body: "Connection pool exhausted under high load",
			repository: "ryuyb/litchi",
			author: "ryuyb",
		},
		paused: true,
		terminated: false,
	},
];

const columns: ColumnDef<WorkSession>[] = [
	{
		accessorKey: "issue.number",
		header: "Issue #",
		cell: ({ row }) => (
			<span className="font-mono text-sm">#{row.original.issue.number}</span>
		),
	},
	{
		accessorKey: "issue.title",
		header: "Title",
		cell: ({ row }) => (
			<span className="font-medium">{row.original.issue.title}</span>
		),
	},
	{
		accessorKey: "issue.repository",
		header: "Repository",
		cell: ({ row }) => (
			<span className="text-muted-foreground text-sm">
				{row.original.issue.repository}
			</span>
		),
	},
	{
		accessorKey: "stage",
		header: "Stage",
		cell: ({ row }) => {
			const stageColors: Record<Stage, string> = {
				clarification:
					"bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
				design:
					"bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
				task_breakdown:
					"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
				execution:
					"bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300",
				pull_request:
					"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
				completed:
					"bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300",
			};
			return (
				<span
					className={`rounded-full px-2 py-0.5 text-xs font-medium ${stageColors[row.original.stage]}`}
				>
					{row.original.stage.replace("_", " ")}
				</span>
			);
		},
	},
	{
		accessorKey: "paused",
		header: "Status",
		cell: ({ row }) => {
			if (row.original.terminated) {
				return (
					<span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-800 dark:bg-red-900 dark:text-red-300">
						Terminated
					</span>
				);
			}
			if (row.original.paused) {
				return (
					<span className="rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300">
						Paused
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
