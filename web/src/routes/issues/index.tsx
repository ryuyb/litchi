import { createFileRoute, useNavigate } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { useState } from "react";
import type {
	GetApiV1SessionsStage,
	GetApiV1SessionsStatus,
	Session,
} from "#/api/schemas";
import type { PaginatedResponseSession } from "#/api/schemas/paginatedResponseSession";
import { useGetApiV1Sessions } from "#/api/sessions/sessions";
import { DataTable } from "#/components/data-table";
import { Input } from "#/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import {
	getStageColor,
	getStatusColor,
	stageConfig,
	statusConfig,
} from "#/lib/session-config";

export const Route = createFileRoute("/issues/")({
	component: IssuesPage,
});

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
			const colorClass = getStageColor(stage);
			return (
				<span
					className={`rounded-full px-2 py-0.5 text-xs font-medium ${colorClass}`}
				>
					{stage ? stageConfig[stage].label : "unknown"}
				</span>
			);
		},
	},
	{
		accessorKey: "status",
		header: "Status",
		cell: ({ row }) => {
			const status = row.original.status;
			const colorClass = getStatusColor(status);
			return (
				<span
					className={`rounded-full px-2 py-0.5 text-xs font-medium ${colorClass}`}
				>
					{status ? statusConfig[status].label : "unknown"}
				</span>
			);
		},
	},
];

function IssuesPage() {
	const navigate = useNavigate();
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(10);
	const [statusFilter, setStatusFilter] = useState<string>("all");
	const [stageFilter, setStageFilter] = useState<string>("all");
	const [repoSearch, setRepoSearch] = useState("");

	// Build API params
	const params = {
		page,
		pageSize,
		status:
			statusFilter !== "all"
				? (statusFilter as GetApiV1SessionsStatus)
				: undefined,
		stage:
			stageFilter !== "all"
				? (stageFilter as GetApiV1SessionsStage)
				: undefined,
		repo: repoSearch || undefined,
	};

	// Fetch sessions using the generated hook
	const {
		data: response,
		isLoading,
		isError,
		error,
	} = useGetApiV1Sessions(params);

	// Extract data from response - check if success (status 200)
	const isSuccess = response?.status === 200;
	const responseData: PaginatedResponseSession | undefined = isSuccess
		? response?.data
		: undefined;
	const sessions = responseData?.data ?? [];
	const pagination = responseData?.pagination;
	const totalPages = pagination?.totalPages ?? 1;

	// Handle pagination change
	const handlePaginationChange = (
		newPageIndex: number,
		newPageSize: number,
	) => {
		setPage(newPageIndex + 1); // API uses 1-based pagination
		setPageSize(newPageSize);
	};

	// Handle row click - navigate to detail page
	const handleRowClick = (session: Session) => {
		navigate({ to: `/issues/${session.id}` });
	};

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

				{/* Filters */}
				<div className="mt-4 flex flex-wrap items-center gap-4">
					{/* Status filter */}
					<div className="flex items-center gap-2">
						<span className="text-sm text-muted-foreground">Status:</span>
						<Select value={statusFilter} onValueChange={setStatusFilter}>
							<SelectTrigger className="h-8 w-[120px]">
								<SelectValue placeholder="All" />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="all">All</SelectItem>
								{Object.entries(statusConfig).map(([key, value]) => (
									<SelectItem key={key} value={key}>
										{value.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					{/* Stage filter */}
					<div className="flex items-center gap-2">
						<span className="text-sm text-muted-foreground">Stage:</span>
						<Select value={stageFilter} onValueChange={setStageFilter}>
							<SelectTrigger className="h-8 w-[140px]">
								<SelectValue placeholder="All" />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="all">All</SelectItem>
								{Object.entries(stageConfig).map(([key, value]) => (
									<SelectItem key={key} value={key}>
										{value.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					{/* Repository search */}
					<div className="flex items-center gap-2">
						<span className="text-sm text-muted-foreground">Repository:</span>
						<Input
							type="text"
							placeholder="owner/repo"
							value={repoSearch}
							onChange={(e) => setRepoSearch(e.target.value)}
							className="h-8 w-[180px]"
						/>
					</div>
				</div>

				{/* Error state */}
				{isError && (
					<div className="mt-4 rounded-lg border border-destructive bg-destructive/10 p-4">
						<p className="text-sm text-destructive">
							Failed to load sessions: {error?.message ?? "Unknown error"}
						</p>
					</div>
				)}

				{/* Data table */}
				<div className="mt-4">
					<DataTable
						columns={columns}
						data={sessions}
						pageCount={totalPages}
						pageIndex={page - 1} // DataTable uses 0-based, API uses 1-based
						pageSize={pageSize}
						onPaginationChange={handlePaginationChange}
						loading={isLoading}
						onRowClick={handleRowClick}
					/>
				</div>
			</section>
		</div>
	);
}
