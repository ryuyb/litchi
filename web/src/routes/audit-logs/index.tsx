import { createFileRoute, useNavigate } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { DownloadIcon } from "lucide-react";
import { useState } from "react";
import { useGetApiV1Audit } from "#/api/audit/audit";
import type { AuditLog } from "#/api/schemas/auditLog";
import type { PaginatedResponseAuditLog } from "#/api/schemas/paginatedResponseAuditLog";
import { DataTable } from "#/components/data-table";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import {
	Pagination,
	PaginationContent,
	PaginationItem,
	PaginationNext,
	PaginationPrevious,
} from "#/components/ui/pagination";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import {
	formatDuration,
	getOperationDisplay,
	getResultDisplay,
	operationOptions,
	resultOptions,
} from "#/lib/audit-config";
import { formatRelativeTime } from "#/lib/date-utils";

export const Route = createFileRoute("/audit-logs/")({
	component: AuditLogsPage,
});

const columns: ColumnDef<AuditLog>[] = [
	{
		accessorKey: "timestamp",
		header: "Time",
		cell: ({ row }) => (
			<span className="text-sm text-muted-foreground">
				{formatRelativeTime(row.original.timestamp)}
			</span>
		),
	},
	{
		accessorKey: "operation",
		header: "Operation",
		cell: ({ row }) => {
			const display = getOperationDisplay(row.original.operation);
			return (
				<span
					className={`rounded-full px-2 py-0.5 text-xs font-medium ${display.color}`}
				>
					{display.label}
				</span>
			);
		},
	},
	{
		accessorKey: "actor",
		header: "Actor",
		cell: ({ row }) => (
			<div className="flex flex-col gap-0.5">
				<span className="text-sm font-medium">{row.original.actor ?? "-"}</span>
				{row.original.actorRole && (
					<span className="text-xs text-muted-foreground">
						{row.original.actorRole}
					</span>
				)}
			</div>
		),
	},
	{
		accessorKey: "resourceType",
		header: "Resource",
		cell: ({ row }) => (
			<div className="flex flex-col gap-0.5">
				<span className="text-sm font-medium">
					{row.original.resourceType ?? "-"}
				</span>
				{row.original.resourceId && (
					<span className="text-xs text-muted-foreground font-mono">
						{row.original.resourceId}
					</span>
				)}
			</div>
		),
	},
	{
		accessorKey: "repository",
		header: "Repository",
		cell: ({ row }) => (
			<span className="text-sm text-muted-foreground">
				{row.original.repository ?? "-"}
			</span>
		),
	},
	{
		accessorKey: "result",
		header: "Result",
		cell: ({ row }) => {
			const display = getResultDisplay(row.original.result);
			return (
				<span
					className={`rounded-full px-2 py-0.5 text-xs font-medium ${display.color}`}
				>
					{display.label}
				</span>
			);
		},
	},
	{
		accessorKey: "duration",
		header: "Duration",
		cell: ({ row }) => (
			<span className="text-sm text-muted-foreground">
				{formatDuration(row.original.duration)}
			</span>
		),
	},
];

function AuditLogsPage() {
	const navigate = useNavigate();
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(50);
	const [operationFilter, setOperationFilter] = useState<string>("all");
	const [resultFilter, setResultFilter] = useState<string>("all");
	const [repoSearch, setRepoSearch] = useState("");
	const [actorSearch, setActorSearch] = useState("");

	// Build API params
	const params = {
		page,
		pageSize,
		operation: operationFilter !== "all" ? operationFilter : undefined,
		result: resultFilter !== "all" ? resultFilter : undefined,
		repository: repoSearch || undefined,
		actor: actorSearch || undefined,
	};

	// Fetch audit logs using the generated hook
	const {
		data: response,
		isLoading,
		isError,
		error,
	} = useGetApiV1Audit(params);

	// Extract data from response
	const isSuccess = response?.status === 200;
	const responseData: PaginatedResponseAuditLog | undefined = isSuccess
		? response?.data
		: undefined;
	const auditLogs = responseData?.data ?? [];
	const pagination = responseData?.pagination;
	const totalPages = pagination?.totalPages ?? 1;

	// Handle row click - navigate to detail page
	const handleRowClick = (log: AuditLog) => {
		if (log.id) {
			navigate({ to: `/audit-logs/${log.id}` });
		}
	};

	// Export to CSV
	const handleExport = () => {
		if (auditLogs.length === 0) return;

		const headers = [
			"Time",
			"Operation",
			"Actor",
			"Actor Role",
			"Resource Type",
			"Resource ID",
			"Repository",
			"Session ID",
			"Issue Number",
			"Result",
			"Duration (ms)",
			"Error",
		];

		const rows = auditLogs.map((log) => [
			log.timestamp ?? "",
			log.operation ?? "",
			log.actor ?? "",
			log.actorRole ?? "",
			log.resourceType ?? "",
			log.resourceId ?? "",
			log.repository ?? "",
			log.sessionId ?? "",
			log.issueNumber?.toString() ?? "",
			log.result ?? "",
			log.duration?.toString() ?? "",
			log.error ?? "",
		]);

		const csvContent = [
			headers.join(","),
			...rows.map((row) =>
				row.map((cell) => `"${cell.replace(/"/g, '""')}"`).join(","),
			),
		].join("\n");

		const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
		const url = URL.createObjectURL(blob);
		const link = document.createElement("a");
		link.href = url;
		link.download = `audit-logs-${new Date().toISOString().split("T")[0]}.csv`;
		link.click();
		URL.revokeObjectURL(url);
	};

	const canPreviousPage = page > 1;
	const canNextPage = page < totalPages;

	return (
		<div className="space-y-6">
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<div className="flex items-center justify-between">
					<div>
						<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
							Audit Logs
						</h1>
						<p className="mt-2 text-muted-foreground">
							Track all system activities and changes.
						</p>
					</div>
					<Button
						variant="outline"
						size="sm"
						onClick={handleExport}
						disabled={auditLogs.length === 0}
					>
						<DownloadIcon className="size-4" />
						<span>Export Current Page</span>
					</Button>
				</div>
			</section>

			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground">
					Recent Activities
				</h2>

				{/* Filters */}
				<div className="mt-4 flex flex-wrap items-center gap-4">
					{/* Operation filter */}
					<div className="flex items-center gap-2">
						<span className="text-sm text-muted-foreground">Operation:</span>
						<Select value={operationFilter} onValueChange={setOperationFilter}>
							<SelectTrigger className="h-8 w-[160px]">
								<SelectValue placeholder="All" />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="all">All</SelectItem>
								{operationOptions.map((op) => (
									<SelectItem key={op} value={op}>
										{getOperationDisplay(op).label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					{/* Result filter */}
					<div className="flex items-center gap-2">
						<span className="text-sm text-muted-foreground">Result:</span>
						<Select value={resultFilter} onValueChange={setResultFilter}>
							<SelectTrigger className="h-8 w-[120px]">
								<SelectValue placeholder="All" />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="all">All</SelectItem>
								{resultOptions.map((r) => (
									<SelectItem key={r} value={r}>
										{getResultDisplay(r).label}
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

					{/* Actor search */}
					<div className="flex items-center gap-2">
						<span className="text-sm text-muted-foreground">Actor:</span>
						<Input
							type="text"
							placeholder="username"
							value={actorSearch}
							onChange={(e) => setActorSearch(e.target.value)}
							className="h-8 w-[140px]"
						/>
					</div>
				</div>

				{/* Error state */}
				{isError && (
					<div className="mt-4 rounded-lg border border-destructive bg-destructive/10 p-4">
						<p className="text-sm text-destructive">
							Failed to load audit logs: {error?.message ?? "Unknown error"}
						</p>
					</div>
				)}

				{/* Data table */}
				<div className="mt-4 space-y-4">
					<DataTable
						columns={columns}
						data={auditLogs}
						loading={isLoading}
						onRowClick={handleRowClick}
					/>

					{/* Pagination */}
					<div className="flex items-center justify-between">
						<div className="flex items-center space-x-2">
							<span className="text-sm text-muted-foreground">
								Rows per page
							</span>
							<Select
								value={`${pageSize}`}
								onValueChange={(value) => {
									setPageSize(Number(value));
									setPage(1);
								}}
							>
								<SelectTrigger className="h-8 w-[70px]">
									<SelectValue placeholder={pageSize} />
								</SelectTrigger>
								<SelectContent side="top">
									{[10, 20, 30, 40, 50].map((size) => (
										<SelectItem key={size} value={`${size}`}>
											{size}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
						<Pagination>
							<PaginationContent>
								<PaginationItem>
									<PaginationPrevious
										onClick={() => setPage(page - 1)}
										className={
											!canPreviousPage
												? "pointer-events-none opacity-50"
												: "cursor-pointer"
										}
									/>
								</PaginationItem>
								<PaginationItem>
									<span className="flex h-9 w-9 items-center justify-center text-sm">
										{page}
									</span>
								</PaginationItem>
								<PaginationItem>
									<PaginationNext
										onClick={() => setPage(page + 1)}
										className={
											!canNextPage
												? "pointer-events-none opacity-50"
												: "cursor-pointer"
										}
									/>
								</PaginationItem>
							</PaginationContent>
						</Pagination>
					</div>
				</div>
			</section>
		</div>
	);
}
