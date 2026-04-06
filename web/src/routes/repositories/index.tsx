import { useForm } from "@tanstack/react-form";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import {
	CheckCircleIcon,
	GitBranchIcon,
	GitForkIcon,
	LoaderIcon,
	PlusIcon,
	SearchIcon,
	SettingsIcon,
	XCircleIcon,
} from "lucide-react";
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
		<div className="flex gap-2 items-center">
			<Link
				to="/repositories/$name"
				params={{ name }}
				className="inline-flex shrink-0 items-center justify-center gap-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-all border border-border bg-background hover:bg-secondary hover:text-secondary-foreground shadow-sm h-7 px-3"
			>
				<SettingsIcon size={12} />
				Configure
			</Link>
			{enabled ? (
				<Button
					size="sm"
					variant="secondary"
					className="h-7 px-3 text-xs bg-red-100 text-red-700 hover:bg-red-200 border-red-200 dark:bg-red-900/40 dark:text-red-400 dark:hover:bg-red-900/60 dark:border-red-900/50 transition-all font-semibold rounded-lg"
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
					size="sm"
					className="h-7 px-3 text-xs bg-green-100 text-green-700 hover:bg-green-200 border-green-200 dark:bg-green-900/40 dark:text-green-400 dark:hover:bg-green-900/60 dark:border-green-900/50 transition-all font-semibold shadow-none rounded-lg"
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
			<div className="flex items-center gap-2">
				<div className="bg-primary/10 p-1.5 rounded-lg text-primary">
					<GitBranchIcon size={16} />
				</div>
				<span className="font-semibold text-foreground/90">
					{row.original.name ?? "N/A"}
				</span>
			</div>
		),
	},
	{
		accessorKey: "enabled",
		header: "Status",
		cell: ({ row }) => {
			const enabled = row.original.enabled;
			if (enabled) {
				return (
					<span className="relative inline-flex items-center gap-1.5 rounded-full bg-emerald-100 px-2.5 py-0.5 text-xs font-semibold text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400 border border-emerald-200/50 dark:border-emerald-800/50">
						<span className="w-1.5 h-1.5 rounded-full bg-emerald-500 shadow-[0_0_5px_rgba(16,185,129,0.8)]"></span>
						Active
					</span>
				);
			}
			return (
				<span className="relative inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-700 dark:bg-slate-800 dark:text-slate-400 border border-slate-200 dark:border-slate-700">
					<span className="w-1.5 h-1.5 rounded-full bg-slate-400"></span>
					Inactive
				</span>
			);
		},
	},
	{
		accessorKey: "hasInstallation",
		header: "App",
		cell: ({ row }) => {
			const hasInstallation = row.original.hasInstallation;
			if (hasInstallation) {
				return (
					<span className="inline-flex items-center gap-1.5 rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-semibold text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 border border-blue-200/50 dark:border-blue-800/50">
						<CheckCircleIcon size={12} />
						Installed
					</span>
				);
			}
			return (
				<span className="inline-flex items-center gap-1.5 rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-600 dark:bg-slate-800 dark:text-slate-400 border border-slate-200 dark:border-slate-700">
					<XCircleIcon size={12} />
					Not Installed
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
			<SheetContent className="border-l-border/30 bg-background/90 backdrop-blur-3xl sm:max-w-md p-0 overflow-hidden flex flex-col shadow-2xl">
				{/* Decorative Header Block */}
				<div className="relative h-40 bg-gradient-to-br from-primary/80 via-primary to-emerald-500/80 px-8 py-8 overflow-hidden">
					<div className="absolute right-0 top-0 bottom-0 w-32 bg-[url('https://transparenttextures.com/patterns/cubes.png')] opacity-10"></div>
					<div className="absolute -right-6 -bottom-6 opacity-20 rotate-12">
						<GitForkIcon size={120} className="text-white" />
					</div>
					<div className="relative z-10 h-full flex flex-col justify-end">
						<div className="bg-white/20 backdrop-blur-sm w-12 h-12 rounded-xl flex items-center justify-center text-white mb-3 shadow-sm border border-white/20">
							<GitBranchIcon size={24} />
						</div>
					</div>
				</div>

				<SheetHeader className="px-8 pt-6 pb-2 text-left">
					<SheetTitle className="text-3xl font-bold tracking-tight text-foreground">
						Connect Repository
					</SheetTitle>
					<SheetDescription className="text-base text-muted-foreground mt-2 leading-relaxed">
						Litchi will bind to this repository to automatically triage issues
						and generate code.
					</SheetDescription>
				</SheetHeader>

				<form
					onSubmit={(e) => {
						e.preventDefault();
						e.stopPropagation();
						form.handleSubmit();
					}}
					className="flex flex-col flex-1 px-8 py-4"
				>
					<div className="flex-1 space-y-6">
						<form.Field
							name="name"
							// biome-ignore lint/correctness/noChildrenProp: TanStack Form requires children as prop for render prop pattern
							children={(field) => (
								<div className="space-y-3">
									<Label
										htmlFor="name"
										className="text-xs font-bold uppercase tracking-widest text-muted-foreground"
									>
										GitHub Path
									</Label>
									<div className="relative group">
										<Input
											id="name"
											placeholder="owner/repo"
											value={field.state.value}
											onChange={(e) => field.handleChange(e.target.value)}
											onBlur={field.handleBlur}
											className={`h-14 bg-secondary/30 border-border/50 rounded-2xl pl-12 pr-4 text-lg transition-all focus-visible:ring-primary/40 focus-visible:bg-background shadow-sm ${field.state.meta.errors.length > 0 ? "border-destructive/50 focus-visible:ring-destructive/30" : ""}`}
										/>
										<div className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors group-focus-within:text-primary">
											<svg
												width="20"
												height="20"
												viewBox="0 0 24 24"
												fill="none"
												stroke="currentColor"
												strokeWidth="2.5"
												strokeLinecap="round"
												strokeLinejoin="round"
												aria-label="GitHub icon"
											>
												<path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
												<path d="M9 18c-4.51 2-5-2-7-2" />
											</svg>
										</div>
									</div>
									{field.state.meta.errors.length > 0 && (
										<p className="text-destructive text-sm font-semibold flex items-center gap-1.5 animate-slide-up-fade">
											<span className="bg-destructive/20 text-destructive rounded-full w-4 h-4 flex items-center justify-center text-[10px] shrink-0">
												!
											</span>
											{field.state.meta.errors[0]?.message}
										</p>
									)}

									<div className="mt-4 p-4 rounded-xl border border-primary/20 bg-primary/5 flex items-start gap-3">
										<div className="mt-0.5 w-2 h-2 rounded-full bg-primary/60 shrink-0"></div>
										<p className="text-xs font-medium text-foreground/70 leading-relaxed">
											Ensure the Litchi GitHub App is installed on this
											repository before connecting it here.
										</p>
									</div>
								</div>
							)}
						/>
					</div>

					<SheetFooter className="pt-6 border-t border-border/40 mt-auto pb-4">
						<form.Subscribe
							selector={(state) => [state.canSubmit, state.isSubmitting]}
							// biome-ignore lint/correctness/noChildrenProp: TanStack Form Subscribe requires children as prop for render prop pattern
							children={([canSubmit, isSubmitting]) => (
								<Button
									type="submit"
									disabled={!canSubmit || createMutation.isPending}
									className="w-full h-14 rounded-2xl text-base font-bold shadow-lg shadow-primary/20 transition-all hover:-translate-y-1 bg-gradient-to-r from-primary to-primary/90 text-primary-foreground"
								>
									{isSubmitting || createMutation.isPending ? (
										<LoaderIcon className="size-5 animate-spin mr-2" />
									) : (
										<GitBranchIcon className="size-5 mr-3" />
									)}
									{isSubmitting || createMutation.isPending
										? "Provisioning Agent..."
										: "Connect to Workspace"}
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

	const isSuccess = data?.status === 200;
	const repositories: Repository[] = isSuccess ? (data.data.data ?? []) : [];
	const pagination = isSuccess ? data.data.pagination : undefined;
	const totalPages = pagination?.totalPages ?? 1;

	const filteredRepositories = search
		? repositories.filter((repo: Repository) =>
				repo.name?.toLowerCase().includes(search.toLowerCase()),
			)
		: repositories;

	const canPreviousPage = page > 1;
	const canNextPage = page < totalPages;

	return (
		<div className="space-y-8 animate-blur-in w-full pb-10">
			<section className="relative overflow-hidden rounded-3xl bg-gradient-to-tr from-accent/50 to-background p-8 shadow-sm border border-border border-b-accent/20">
				<div className="absolute top-0 right-0 p-8 opacity-5 transform rotate-12">
					<GitForkIcon size={200} />
				</div>
				<div className="relative z-10">
					<h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-3">
						<span className="w-1.5 h-8 bg-accent rounded-full"></span>
						Repositories
					</h1>
					<p className="mt-3 text-muted-foreground max-w-2xl text-lg">
						Configure and manage connected GitHub repositories. Grant access to
						Litchi agent for automated work.
					</p>
				</div>
			</section>

			<section className="glass-card rounded-3xl p-6 shadow-md border border-border/50">
				<div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-8">
					<h2 className="text-xl font-bold text-foreground flex items-center gap-2">
						Connected Repositories
					</h2>
					<div className="flex w-full sm:w-auto items-center gap-4 bg-secondary/30 p-2 rounded-2xl">
						<div className="relative flex-1 sm:w-64">
							<SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground size-4" />
							<Input
								type="search"
								placeholder="Search repos..."
								value={search}
								onChange={(e) => setSearch(e.target.value)}
								className="pl-9 h-10 bg-background border-border/50 rounded-xl focus-visible:ring-primary/30 w-full"
							/>
						</div>
						<Button
							onClick={() => setIsAddSheetOpen(true)}
							className="h-10 rounded-xl px-4 font-bold shadow-md shadow-primary/10 hover:shadow-lg hover:-translate-y-px transition-all"
						>
							<PlusIcon className="size-4 mr-2" />
							Add
						</Button>
					</div>
				</div>

				{isError && (
					<div className="mb-6 rounded-2xl border border-destructive/50 bg-destructive/10 p-4 animate-slide-up-fade">
						<p className="text-sm font-medium text-destructive flex items-center gap-2">
							<span className="bg-destructive/20 p-1 rounded">⚠️</span>
							Error loading repositories: {error?.message ?? "Unknown error"}
						</p>
					</div>
				)}

				{!isSuccess && data?.status !== undefined && (
					<div className="mb-6 rounded-2xl border border-destructive/50 bg-destructive/10 p-4 animate-slide-up-fade">
						<p className="text-sm font-medium text-destructive">
							API returned status {data.status}. Please try again.
						</p>
					</div>
				)}

				<div className="overflow-hidden rounded-2xl border border-border/60 bg-card">
					<DataTable
						columns={columns}
						data={filteredRepositories}
						loading={isLoading}
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

			<AddRepositorySheet
				open={isAddSheetOpen}
				onOpenChange={setIsAddSheetOpen}
			/>
		</div>
	);
}
