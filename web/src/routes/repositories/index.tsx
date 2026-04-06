import { useForm } from "@tanstack/react-form";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { LoaderIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { z } from "zod";
import {
	useGetApiV1Repositories,
	usePostApiV1Repositories,
	usePostApiV1RepositoriesNameDisable,
	usePostApiV1RepositoriesNameEnable,
} from "#/api/repositories/repositories";
import type { Repository } from "#/api/schemas";
import { DataTable } from "#/components/data-table";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
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
	Sheet,
	SheetContent,
	SheetDescription,
	SheetFooter,
	SheetHeader,
	SheetTitle,
} from "#/components/ui/sheet";

// Zod schema for repository name validation
const repositorySchema = z.object({
	name: z
		.string()
		.min(1, "Repository name is required")
		.refine((val) => val.includes("/"), {
			message: "Repository name must be in owner/repo format",
		}),
});

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

// Add Repository Sheet component
interface AddRepositorySheetProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
}

function AddRepositorySheet({ open, onOpenChange }: AddRepositorySheetProps) {
	const queryClient = useQueryClient();

	const createMutation = usePostApiV1Repositories({
		mutation: {
			onSuccess: () => {
				onOpenChange(false);
				form.reset();
				queryClient.invalidateQueries({
					queryKey: ["getApiV1Repositories"],
				});
			},
		},
	});

	const form = useForm({
		defaultValues: {
			name: "",
		},
		validators: {
			onChange: repositorySchema,
		},
		onSubmit: async ({ value }) => {
			createMutation.mutate({ data: { name: value.name } });
		},
	});

	return (
		<Sheet open={open} onOpenChange={onOpenChange}>
			<SheetContent>
				<SheetHeader>
					<SheetTitle>Add Repository</SheetTitle>
					<SheetDescription>
						Enter the GitHub repository name in owner/repo format to enable
						processing for this repository.
					</SheetDescription>
				</SheetHeader>

				<form
					onSubmit={(e) => {
						e.preventDefault();
						e.stopPropagation();
						form.handleSubmit();
					}}
					className="space-y-4 px-4 py-4"
				>
					<form.Field
						name="name"
						// biome-ignore lint/correctness/noChildrenProp: TanStack Form uses children as render prop
						children={(field) => (
							<div className="space-y-2">
								<Label htmlFor="name">Repository Name</Label>
								<Input
									id="name"
									placeholder="owner/repo"
									value={field.state.value}
									onChange={(e) => field.handleChange(e.target.value)}
									onBlur={field.handleBlur}
								/>
								{field.state.meta.errors.length > 0 && (
									<p className="text-destructive text-sm">
										{field.state.meta.errors[0]?.message}
									</p>
								)}
							</div>
						)}
					/>

					<SheetFooter className="px-0">
						<form.Subscribe
							selector={(state) => [state.canSubmit, state.isSubmitting]}
							// biome-ignore lint/correctness/noChildrenProp: TanStack Form uses children as render prop
							children={([canSubmit, isSubmitting]) => (
								<Button
									type="submit"
									disabled={!canSubmit || createMutation.isPending}
								>
									{isSubmitting || createMutation.isPending ? (
										<LoaderIcon className="size-4 animate-spin" />
									) : (
										<PlusIcon className="size-4" />
									)}
									{isSubmitting || createMutation.isPending
										? "Adding..."
										: "Add Repository"}
								</Button>
							)}
						/>
					</SheetFooter>
				</form>
			</SheetContent>
		</Sheet>
	);
}

function RepositoriesPage() {
	const [page, setPage] = useState(1);
	const [pageSize, setPageSize] = useState(10);
	const [search, setSearch] = useState("");
	const [isAddSheetOpen, setIsAddSheetOpen] = useState(false);

	const { data, isLoading, isError, error } = useGetApiV1Repositories({
		page,
		pageSize,
	});

	// Handle API response type - data contains status and response data
	const isSuccess = data?.status === 200;
	const repositories: Repository[] = isSuccess ? (data.data.data ?? []) : [];
	const pagination = isSuccess ? data.data.pagination : undefined;
	const totalPages = pagination?.totalPages ?? 1;

	// Filter by search on client side since API doesn't support search param
	const filteredRepositories = search
		? repositories.filter((repo: Repository) =>
				repo.name?.toLowerCase().includes(search.toLowerCase()),
			)
		: repositories;

	const canPreviousPage = page > 1;
	const canNextPage = page < totalPages;

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
						<Button onClick={() => setIsAddSheetOpen(true)}>
							<PlusIcon className="size-4" />
							Add Repository
						</Button>
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

				<div className="mt-4 space-y-4">
					<DataTable
						columns={columns}
						data={filteredRepositories}
						loading={isLoading}
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

			<AddRepositorySheet
				open={isAddSheetOpen}
				onOpenChange={setIsAddSheetOpen}
			/>
		</div>
	);
}
