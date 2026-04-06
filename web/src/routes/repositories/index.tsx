import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { useState } from "react";
import {
	useGetApiV1Repositories,
	usePostApiV1RepositoriesNameDisable,
	usePostApiV1RepositoriesNameEnable,
} from "#/api/repositories/repositories";
import type { Repository } from "#/api/schemas";
import { DataTable } from "#/components/data-table";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";

export const Route = createFileRoute("/repositories/")({
	component: RepositoriesPage,
});

interface RepositoryRowActionsProps {
	repository: Repository;
}

function RepositoryRowActions({ repository }: RepositoryRowActionsProps) {
	const name = repository.name ?? "";
	const enabled = repository.enabled ?? false;
	const queryClient = useQueryClient();

	const enableMutation = usePostApiV1RepositoriesNameEnable({
		mutation: {
			onSuccess: () => {
				queryClient.invalidateQueries({
					queryKey: ["getApiV1Repositories"],
				});
			},
		},
	});

	const disableMutation = usePostApiV1RepositoriesNameDisable({
		mutation: {
			onSuccess: () => {
				queryClient.invalidateQueries({
					queryKey: ["getApiV1Repositories"],
				});
			},
		},
	});

	return (
		<div className="flex gap-2">
			<Link
				to="/repositories/$name"
				params={{ name }}
				className="inline-flex shrink-0 items-center justify-center gap-2 rounded-md text-sm font-medium whitespace-nowrap transition-all outline-none border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground dark:border-input dark:bg-input/30 dark:hover:bg-input/50 h-6 gap-1 px-2 text-xs has-[>svg]:px-1.5"
			>
				Configure
			</Link>
			{enabled ? (
				<Button
					size="xs"
					variant="outline"
					onClick={(e) => {
						e.stopPropagation();
						disableMutation.mutate({ name, data: {} });
					}}
					disabled={disableMutation.isPending}
				>
					Disable
				</Button>
			) : (
				<Button
					size="xs"
					variant="default"
					onClick={(e) => {
						e.stopPropagation();
						enableMutation.mutate({ name, data: {} });
					}}
					disabled={enableMutation.isPending}
				>
					Enable
				</Button>
			)}
		</div>
	);
}

const columns: ColumnDef<Repository>[] = [
	{
		accessorKey: "name",
		header: "Name",
		cell: ({ row }) => (
			<span className="font-medium">{row.original.name ?? "N/A"}</span>
		),
	},
	{
		accessorKey: "enabled",
		header: "Status",
		cell: ({ row }) => {
			const enabled = row.original.enabled;
			if (enabled) {
				return (
					<span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-300">
						Enabled
					</span>
				);
			}
			return (
				<span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-800 dark:bg-gray-800 dark:text-gray-300">
					Disabled
				</span>
			);
		},
	},
	{
		id: "actions",
		header: "Actions",
		cell: ({ row }) => <RepositoryRowActions repository={row.original} />,
	},
];

function RepositoriesPage() {
	const [pageIndex, setPageIndex] = useState(0);
	const [pageSize, setPageSize] = useState(10);
	const [search, setSearch] = useState("");

	const { data, isLoading, isError, error } = useGetApiV1Repositories({
		page: pageIndex + 1,
		pageSize,
	});

	// Handle API response type - data contains status and response data
	const isSuccess = data?.status === 200;
	const repositories: Repository[] = isSuccess ? (data.data.data ?? []) : [];
	const pagination = isSuccess ? data.data.pagination : undefined;
	const totalPages = pagination?.totalPages ?? 1;

	const handlePaginationChange = (
		newPageIndex: number,
		newPageSize: number,
	) => {
		setPageIndex(newPageIndex);
		setPageSize(newPageSize);
	};

	// Filter by search on client side since API doesn't support search param
	const filteredRepositories = search
		? repositories.filter((repo: Repository) =>
				repo.name?.toLowerCase().includes(search.toLowerCase()),
			)
		: repositories;

	return (
		<div className="space-y-6">
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
					Repositories
				</h1>
				<p className="mt-2 text-muted-foreground">
					Configure and manage connected GitHub repositories.
				</p>
			</section>

			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<div className="flex items-center justify-between">
					<h2 className="text-lg font-semibold text-card-foreground">
						Connected Repositories
					</h2>
					<div className="flex gap-2">
						<Input
							type="search"
							placeholder="Search repositories..."
							value={search}
							onChange={(e) => setSearch(e.target.value)}
							className="w-[250px]"
						/>
					</div>
				</div>

				{isError && (
					<div className="mt-4 rounded-lg border border-destructive bg-destructive/10 p-4">
						<p className="text-sm text-destructive">
							Error loading repositories: {error?.message ?? "Unknown error"}
						</p>
					</div>
				)}

				{!isSuccess && data?.status !== undefined && (
					<div className="mt-4 rounded-lg border border-destructive bg-destructive/10 p-4">
						<p className="text-sm text-destructive">
							API returned status {data.status}. Please try again.
						</p>
					</div>
				)}

				<div className="mt-4">
					<DataTable
						columns={columns}
						data={filteredRepositories}
						pageCount={totalPages}
						pageIndex={pageIndex}
						pageSize={pageSize}
						onPaginationChange={handlePaginationChange}
						loading={isLoading}
					/>
				</div>
			</section>
		</div>
	);
}
