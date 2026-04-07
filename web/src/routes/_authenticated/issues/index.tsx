import { createFileRoute, useNavigate } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import {
	AlertCircleIcon,
	DatabaseIcon,
	FilterIcon,
	SearchIcon,
} from "lucide-react";
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
	getStageColor,
	getStatusColor,
	stageConfig,
	statusConfig,
} from "#/lib/session-config";

export const Route = createFileRoute("/_authenticated/issues/")({
	component: IssuesPage,
});

const columns: ColumnDef<Session>[] = [
	{
		accessorKey: "issueNumber",
		header: "Issue #",
		cell: ({ row }) => (
			<span className="font-mono text-sm bg-secondary/50 px-2 py-0.5 rounded-md text-foreground/80">
				#{row.original.issueNumber}
			</span>
		),
	},
	{
		accessorKey: "issueTitle",
		header: "Title",
		cell: ({ row }) => (
			<span className="font-semibold text-foreground/90 group-hover:text-primary transition-colors">
				{row.original.issueTitle}
			</span>
		),
	},
	{
		accessorKey: "repository",
		header: "Repository",
		cell: ({ row }) => (
			<span className="text-muted-foreground text-sm flex items-center gap-1.5">
				<DatabaseIcon size={14} className="opacity-50" />
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
					className={`rounded-full px-2.5 py-1 text-xs font-semibold tracking-wide ${colorClass} shadow-sm border border-transparent`}
				>
					{stage ? stageConfig[stage].label : "unknown"}
				</span>
			)
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
					className={`rounded-full px-2.5 py-1 text-xs font-semibold tracking-wide ${colorClass} shadow-sm border border-transparent`}
				>
					{status ? statusConfig[status].label : "unknown"}
				</span>
			)
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
	}

	const {
		data: response,
		isLoading,
		isError,
		error,
	} = useGetApiV1Sessions(params);

	const isSuccess = response?.status === 200;
	const responseData: PaginatedResponseSession | undefined = isSuccess
		? response?.data
		: undefined;
	const sessions = responseData?.data ?? [];
	const pagination = responseData?.pagination;
	const totalPages = pagination?.totalPages ?? 1;

	const handleRowClick = (session: Session) => {
		navigate({ to: `/issues/${session.id}` });
	}

	const canPreviousPage = page > 1;
	const canNextPage = page < totalPages;

	return (
		<div className="space-y-8 animate-blur-in w-full pb-10">
			{/* Header Section */}
			<section className="relative overflow-hidden rounded-3xl bg-gradient-to-r from-card to-secondary/30 p-8 shadow-sm border border-border">
				<div className="absolute -right-20 -top-20 opacity-5 blur-3xl">
					<DatabaseIcon size={300} />
				</div>
				<div className="relative z-10">
					<h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-3">
						<span className="w-1.5 h-8 bg-primary rounded-full"></span>
						Issues
					</h1>
					<p className="mt-3 text-muted-foreground max-w-2xl text-lg">
						Manage and track GitHub Issues across your configured repositories.
						Monitor auto-generated PRs and agent progress.
					</p>
				</div>
			</section>

			<section className="glass-card rounded-3xl p-6 shadow-md border border-border/50">
				<div className="flex flex-col lg:flex-row lg:items-center justify-between gap-6 mb-8">
					<h2 className="text-xl font-bold text-foreground flex items-center gap-2">
						<FilterIcon className="text-primary" size={22} />
						Session Explorer
					</h2>

					<div className="flex flex-wrap items-center gap-3 bg-secondary/40 p-2 rounded-2xl border border-border/40">
						<div className="flex items-center gap-2 bg-background px-3 py-1 rounded-xl shadow-sm border border-border/50">
							<span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
								Status
							</span>
							<Select value={statusFilter} onValueChange={setStatusFilter}>
								<SelectTrigger className="h-8 w-[120px] border-0 bg-transparent shadow-none focus:ring-0 px-1 py-0 font-medium">
									<SelectValue placeholder="All" />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="all">All Statuses</SelectItem>
									{Object.entries(statusConfig).map(([key, value]) => (
										<SelectItem key={key} value={key}>
											{value.label}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>

						<div className="flex items-center gap-2 bg-background px-3 py-1 rounded-xl shadow-sm border border-border/50">
							<span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
								Stage
							</span>
							<Select value={stageFilter} onValueChange={setStageFilter}>
								<SelectTrigger className="h-8 w-[140px] border-0 bg-transparent shadow-none focus:ring-0 px-1 py-0 font-medium">
									<SelectValue placeholder="All" />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="all">All Stages</SelectItem>
									{Object.entries(stageConfig).map(([key, value]) => (
										<SelectItem key={key} value={key}>
											{value.label}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>

						<div className="flex items-center gap-2 bg-background px-3 py-1 rounded-xl shadow-sm border border-border/50 relative">
							<SearchIcon
								size={14}
								className="text-muted-foreground absolute left-3"
							/>
							<Input
								type="text"
								placeholder="Search repo..."
								value={repoSearch}
								onChange={(e) => setRepoSearch(e.target.value)}
								className="h-8 w-[180px] border-0 bg-transparent shadow-none focus-visible:ring-0 pl-7 text-sm font-medium"
							/>
						</div>
					</div>
				</div>

				{isError && (
					<div className="mb-6 rounded-2xl border border-destructive/50 bg-destructive/10 p-4 flex items-center gap-3 animate-slide-up-fade">
						<AlertCircleIcon className="text-destructive" size={20} />
						<p className="text-sm font-medium text-destructive">
							Failed to load sessions: {error?.message ?? "Unknown error"}
						</p>
					</div>
				)}

				<div className="overflow-hidden rounded-2xl border border-border/60 bg-card">
					<DataTable
						columns={columns}
						data={sessions}
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
							value={"${pageSize}"}
							onValueChange={(value) => {
								setPageSize(Number(value));
								setPage(1)
							}}
						>
							<SelectTrigger className="h-8 w-[70px] bg-background border-border/50">
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
					<Pagination className="justify-end w-auto mx-0">
						<PaginationContent className="bg-secondary/30 rounded-xl p-1">
							<PaginationItem>
								<PaginationPrevious
									onClick={() => setPage(page - 1)}
									className={
										!canPreviousPage
											? "pointer-events-none opacity-50 text-muted-foreground"
											: "cursor-pointer hover:bg-background rounded-lg"
									}
								/>
							</PaginationItem>
							<PaginationItem>
								<span className="flex h-9 min-w-9 items-center justify-center text-sm font-bold bg-background rounded-lg shadow-sm border border-border/50 px-3">
									Page {page} of {totalPages}
								</span>
							</PaginationItem>
							<PaginationItem>
								<PaginationNext
									onClick={() => setPage(page + 1)}
									className={
										!canNextPage
											? "pointer-events-none opacity-50 text-muted-foreground"
											: "cursor-pointer hover:bg-background rounded-lg"
									}
								/>
							</PaginationItem>
						</PaginationContent>
					</Pagination>
				</div>
			</section>
		</div>
	)
}
