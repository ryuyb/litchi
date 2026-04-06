import { createFileRoute, useNavigate } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { DownloadIcon, FileTextIcon, FilterIcon, SearchIcon, ClockIcon } from "lucide-react";
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
			<div className="flex items-center gap-1.5 text-xs font-semibold text-muted-foreground bg-secondary/30 px-2 py-1 rounded-md w-fit">
				<ClockIcon size={12} className="opacity-70" />
				{formatRelativeTime(row.original.timestamp)}
			</div>
		),
	},
	{
		accessorKey: "operation",
		header: "Operation",
		cell: ({ row }) => {
			const display = getOperationDisplay(row.original.operation);
			return (
				<span
					className={`rounded-full px-2.5 py-1 text-xs font-bold tracking-wide ${display.color} border border-transparent shadow-[inset_0_1px_rgba(255,255,255,0.2)]`}
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
			<div className="flex flex-col">
				<span className="text-sm font-bold text-foreground/90">{row.original.actor ?? "-"}</span>
				{row.original.actorRole && (
					<span className="text-[10px] uppercase tracking-wider font-bold text-primary max-w-fit">
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
				<span className="text-sm font-semibold">
					{row.original.resourceType ?? "-"}
				</span>
				{row.original.resourceId && (
					<span className="text-xs text-muted-foreground font-mono bg-secondary/50 px-1.5 py-0.5 rounded max-w-fit">
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
			<span className="text-sm font-medium text-muted-foreground">
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
					className={`rounded-full px-2.5 py-1 text-xs font-bold tracking-wide ${display.color} shadow-sm border border-transparent`}
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
			<span className="text-xs font-mono font-medium text-foreground tracking-tight">
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

	const params = {
		page,
		pageSize,
		operation: operationFilter !== "all" ? operationFilter : undefined,
		result: resultFilter !== "all" ? resultFilter : undefined,
		repository: repoSearch || undefined,
		actor: actorSearch || undefined,
	};

	const {
		data: response,
		isLoading,
		isError,
		error,
	} = useGetApiV1Audit(params);

	const isSuccess = response?.status === 200;
	const responseData: PaginatedResponseAuditLog | undefined = isSuccess
		? response?.data
		: undefined;
	const auditLogs = responseData?.data ?? [];
	const pagination = responseData?.pagination;
	const totalPages = pagination?.totalPages ?? 1;

	const handleRowClick = (log: AuditLog) => {
		if (log.id) {
			navigate({ to: `/audit-logs/${log.id}` });
		}
	};

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
		<div className="space-y-8 animate-blur-in w-full pb-10">
			{/* Header Section */}
			<section className="relative overflow-hidden rounded-3xl bg-gradient-to-bl from-card via-muted/50 to-secondary/30 p-8 shadow-sm border border-border/80">
				<div className="absolute left-10 -bottom-10 opacity-5 -rotate-12">
					<FileTextIcon size={250} />
				</div>
				<div className="relative z-10 flex flex-col md:flex-row md:items-center justify-between gap-6">
					<div>
						<h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-3">
							<span className="w-1.5 h-8 bg-foreground rounded-full"></span>
							Audit Logs
						</h1>
						<p className="mt-3 text-muted-foreground max-w-2xl text-lg">
							A secure, verifiable record of all system activities, changes, and agent actions.
						</p>
					</div>
					<Button
						variant="outline"
						onClick={handleExport}
						disabled={auditLogs.length === 0}
						className="h-12 px-6 rounded-xl font-bold shadow-sm self-start md:self-auto hover:bg-background/80"
					>
						<DownloadIcon className="size-5 mr-2" />
						Export CSV
					</Button>
				</div>
			</section>

			<section className="glass-card rounded-3xl p-6 shadow-md border border-border/50">
				<div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6 mb-8 border-b border-border/50 pb-6">
					<h2 className="text-xl font-bold text-foreground flex items-center gap-2">
						<FilterIcon className="text-primary" size={22} />
						Filter Logs
					</h2>

					<div className="flex flex-wrap items-center gap-3 w-full lg:w-auto">
						<div className="flex items-center gap-2 bg-secondary/30 px-3 py-1 rounded-xl shadow-sm border border-border/40 w-full sm:w-auto">
							<span className="text-xs font-semibold text-muted-foreground uppercase tracking-widest shrink-0">Op</span>
							<Select value={operationFilter} onValueChange={setOperationFilter}>
								<SelectTrigger className="h-9 w-full sm:w-[150px] border-0 bg-transparent shadow-none focus:ring-0 px-1 py-0 font-medium">
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

						<div className="flex items-center gap-2 bg-secondary/30 px-3 py-1 rounded-xl shadow-sm border border-border/40 w-full sm:w-auto">
							<span className="text-xs font-semibold text-muted-foreground uppercase tracking-widest shrink-0">Result</span>
							<Select value={resultFilter} onValueChange={setResultFilter}>
								<SelectTrigger className="h-9 w-full sm:w-[120px] border-0 bg-transparent shadow-none focus:ring-0 px-1 py-0 font-medium">
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

						<div className="flex items-center gap-2 bg-secondary/30 px-3 py-1 rounded-xl shadow-sm border border-border/40 relative w-full sm:w-auto">
							<SearchIcon size={14} className="text-muted-foreground absolute left-3" />
							<Input
								type="text"
								placeholder="Repo..."
								value={repoSearch}
								onChange={(e) => setRepoSearch(e.target.value)}
								className="h-9 w-full sm:w-[140px] border-0 bg-transparent shadow-none focus-visible:ring-0 pl-7 text-sm font-medium"
							/>
						</div>
						
						<div className="flex items-center gap-2 bg-secondary/30 px-3 py-1 rounded-xl shadow-sm border border-border/40 relative w-full sm:w-auto">
							<SearchIcon size={14} className="text-muted-foreground absolute left-3" />
							<Input
								type="text"
								placeholder="Actor..."
								value={actorSearch}
								onChange={(e) => setActorSearch(e.target.value)}
								className="h-9 w-full sm:w-[120px] border-0 bg-transparent shadow-none focus-visible:ring-0 pl-7 text-sm font-medium"
							/>
						</div>
					</div>
				</div>

				{isError && (
					<div className="mb-6 rounded-2xl border border-destructive/50 bg-destructive/10 p-4 animate-slide-up-fade">
						<p className="text-sm font-bold text-destructive">
							Failed to load audit logs: {error?.message ?? "Unknown error"}
						</p>
					</div>
				)}

				<div className="overflow-hidden rounded-2xl border border-border/60 bg-card">
					<DataTable
						columns={columns}
						data={auditLogs}
						loading={isLoading}
						onRowClick={handleRowClick}
					/>
				</div>

				<div className="mt-6 flex flex-col sm:flex-row items-center justify-between gap-4">
					<div className="flex items-center space-x-3 bg-secondary/30 px-4 py-2 rounded-xl">
						<span className="text-sm font-medium text-muted-foreground">
							Rows per page
						</span>
						<Select
							value={`${pageSize}`}
							onValueChange={(value) => {
								setPageSize(Number(value));
								setPage(1);
							}}
						>
							<SelectTrigger className="h-8 w-[70px] bg-background border-border/50">
								<SelectValue placeholder={pageSize} />
							</SelectTrigger>
							<SelectContent side="top">
								{[10, 20, 50, 100].map((size) => (
									<SelectItem key={size} value={`${size}`}>
										{size}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>
					<Pagination className="justify-end w-auto mx-0">
						<PaginationContent className="bg-secondary/30 rounded-xl p-1 shadow-inner">
							<PaginationItem>
								<PaginationPrevious
									onClick={() => setPage(page - 1)}
									className={
										!canPreviousPage
											? "pointer-events-none opacity-50 text-muted-foreground"
											: "cursor-pointer hover:bg-background rounded-lg font-bold"
									}
								/>
							</PaginationItem>
							<PaginationItem>
								<span className="flex h-9 min-w-9 items-center justify-center text-sm font-bold bg-background rounded-lg shadow-sm border border-border/50 px-3 text-foreground">
									Page {page} of {totalPages}
								</span>
							</PaginationItem>
							<PaginationItem>
								<PaginationNext
									onClick={() => setPage(page + 1)}
									className={
										!canNextPage
											? "pointer-events-none opacity-50 text-muted-foreground"
											: "cursor-pointer hover:bg-background rounded-lg font-bold"
									}
								/>
							</PaginationItem>
						</PaginationContent>
					</Pagination>
				</div>
			</section>
		</div>
	);
}
