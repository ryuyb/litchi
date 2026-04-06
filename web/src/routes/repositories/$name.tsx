import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeftIcon, LoaderIcon, SaveIcon } from "lucide-react";
import { useEffect, useState } from "react";
import {
	getGetApiV1RepositoriesNameEffectiveConfigQueryKey,
	getGetApiV1RepositoriesNameQueryKey,
	useGetApiV1RepositoriesName,
	useGetApiV1RepositoriesNameEffectiveConfig,
	usePutApiV1RepositoriesName,
} from "#/api/repositories/repositories";
import type { EffectiveConfig } from "#/api/schemas/effectiveConfig";
import type { RepoConfig } from "#/api/schemas/repoConfig";
import type { Repository } from "#/api/schemas/repository";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import { Slider } from "#/components/ui/slider";
import { Switch } from "#/components/ui/switch";

export const Route = createFileRoute("/repositories/$name")({
	component: RepositoryConfigPage,
});

// Helper to check if response is successful (status 200)
function isSuccessfulRepositoryResponse(
	response:
		| { data: Repository | { message?: string; code?: string }; status: number }
		| undefined,
): response is { data: Repository; status: 200 } {
	return response?.status === 200;
}

function isSuccessfulEffectiveConfigResponse(
	response:
		| {
				data: EffectiveConfig | { message?: string; code?: string };
				status: number;
		  }
		| undefined,
): response is { data: EffectiveConfig; status: 200 } {
	return response?.status === 200;
}

function RepositoryConfigPage() {
	const { name } = Route.useParams();
	const queryClient = useQueryClient();

	// Fetch repository data
	const {
		data: repositoryResponse,
		isLoading: isLoadingRepo,
		error: repoError,
	} = useGetApiV1RepositoriesName(name);

	// Fetch effective config
	const {
		data: effectiveConfigResponse,
		isLoading: isLoadingEffective,
		error: effectiveError,
	} = useGetApiV1RepositoriesNameEffectiveConfig(name);

	// Mutation for updating repository
	const updateMutation = usePutApiV1RepositoriesName({
		mutation: {
			onSuccess: () => {
				setSaveSuccess(true);
				setTimeout(() => setSaveSuccess(false), 3000);
				// Invalidate queries to refresh data
				queryClient.invalidateQueries({
					queryKey: getGetApiV1RepositoriesNameQueryKey(name),
				});
				queryClient.invalidateQueries({
					queryKey: getGetApiV1RepositoriesNameEffectiveConfigQueryKey(name),
				});
			},
		},
	});

	// Form state
	const [config, setConfig] = useState<RepoConfig>({});
	const [saveSuccess, setSaveSuccess] = useState(false);

	// Extract successful data
	const repository = isSuccessfulRepositoryResponse(repositoryResponse)
		? repositoryResponse.data
		: null;
	const effectiveConfig = isSuccessfulEffectiveConfigResponse(
		effectiveConfigResponse,
	)
		? effectiveConfigResponse.data
		: null;

	// Initialize form with repository config when data loads
	useEffect(() => {
		if (repository?.config) {
			setConfig(repository.config);
		}
	}, [repository?.config]);

	// Handle form field changes
	const handleNumberChange = (field: keyof RepoConfig, value: string) => {
		const numValue = value === "" ? undefined : Number(value);
		setConfig((prev) => ({
			...prev,
			[field]: numValue,
		}));
	};

	const handleStringChange = (field: keyof RepoConfig, value: string) => {
		setConfig((prev) => ({
			...prev,
			[field]: value === "" ? undefined : value,
		}));
	};

	const handleBooleanChange = (field: keyof RepoConfig, value: boolean) => {
		setConfig((prev) => ({
			...prev,
			[field]: value,
		}));
	};

	// Handle save
	const handleSave = () => {
		updateMutation.mutate({
			name,
			data: { config },
		});
	};

	// Loading state
	if (isLoadingRepo || isLoadingEffective) {
		return (
			<div className="flex items-center justify-center min-h-[400px] gap-2">
				<LoaderIcon className="size-8 animate-spin text-muted-foreground" />
				<span className="text-muted-foreground">Loading configuration...</span>
			</div>
		);
	}

	// Error state
	if (repoError || effectiveError) {
		return (
			<div className="space-y-6">
				<section className="rounded-xl border border-destructive bg-card p-6">
					<h2 className="text-lg font-semibold text-destructive">
						Error loading repository
					</h2>
					<p className="mt-2 text-muted-foreground">
						{repoError?.message ||
							effectiveError?.message ||
							"Failed to load repository configuration. Please try again."}
					</p>
					<Button asChild className="mt-4">
						<Link to="/repositories">
							<ArrowLeftIcon className="size-4" />
							Back to Repositories
						</Link>
					</Button>
				</section>
			</div>
		);
	}

	// No repository found
	if (!repository) {
		return (
			<div className="rounded-xl border border-border bg-card p-6">
				<h2 className="text-lg font-semibold">Repository not found</h2>
				<p className="mt-2 text-muted-foreground">
					The requested repository could not be found.
				</p>
				<Button asChild className="mt-4">
					<Link to="/repositories">
						<ArrowLeftIcon className="size-4" />
						Back to Repositories
					</Link>
				</Button>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			{/* Header with back link */}
			<div className="flex items-center gap-2">
				<Button variant="ghost" size="sm" asChild>
					<Link to="/repositories">
						<ArrowLeftIcon className="size-4" />
						Back to Repositories
					</Link>
				</Button>
			</div>

			{/* Title Section */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<div className="flex items-center justify-between">
					<div>
						<h1 className="text-2xl font-bold tracking-tight text-card-foreground">
							{repository.name}
						</h1>
						<p className="mt-2 text-muted-foreground">
							Configure processing settings for this repository
						</p>
					</div>
					{repository.enabled ? (
						<span className="rounded-full bg-green-100 px-3 py-1 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-300">
							Active
						</span>
					) : (
						<span className="rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-800 dark:bg-gray-800 dark:text-gray-300">
							Disabled
						</span>
					)}
				</div>
			</section>

			{/* Success/Error Messages */}
			{saveSuccess && (
				<div className="rounded-lg border border-green-500/50 bg-green-500/10 p-4">
					<p className="text-green-600 dark:text-green-400">
						Configuration saved successfully!
					</p>
				</div>
			)}

			{/* Repository Config Form */}
			<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
				<h2 className="text-lg font-semibold text-card-foreground mb-6">
					Repository Settings
				</h2>
				<div className="space-y-6">
					{/* Default Model */}
					<div className="space-y-2">
						<Label htmlFor="defaultModel">Default Model</Label>
						<Input
							id="defaultModel"
							placeholder="e.g., claude-3-opus-20240229"
							value={config.defaultModel ?? ""}
							onChange={(e) =>
								handleStringChange("defaultModel", e.target.value)
							}
						/>
						<p className="text-xs text-muted-foreground">
							The default AI model to use for this repository
						</p>
					</div>

					{/* Max Concurrency */}
					<div className="space-y-2">
						<Label htmlFor="maxConcurrency">Max Concurrency</Label>
						<Input
							id="maxConcurrency"
							type="number"
							min={1}
							max={10}
							placeholder="1-10"
							value={config.maxConcurrency ?? ""}
							onChange={(e) =>
								handleNumberChange("maxConcurrency", e.target.value)
							}
						/>
						<p className="text-xs text-muted-foreground">
							Maximum number of concurrent tasks (1-10)
						</p>
					</div>

					{/* Task Retry Limit */}
					<div className="space-y-2">
						<Label htmlFor="taskRetryLimit">Task Retry Limit</Label>
						<Input
							id="taskRetryLimit"
							type="number"
							min={0}
							max={10}
							placeholder="0-10"
							value={config.taskRetryLimit ?? ""}
							onChange={(e) =>
								handleNumberChange("taskRetryLimit", e.target.value)
							}
						/>
						<p className="text-xs text-muted-foreground">
							Maximum retry attempts for failed tasks (0-10)
						</p>
					</div>

					{/* Complexity Threshold */}
					<div className="space-y-2">
						<Label htmlFor="complexityThreshold">
							Complexity Threshold: {config.complexityThreshold ?? 70}
						</Label>
						<Slider
							id="complexityThreshold"
							min={0}
							max={100}
							step={1}
							value={[config.complexityThreshold ?? 70]}
							onValueChange={(value) =>
								handleNumberChange("complexityThreshold", String(value[0]))
							}
						/>
						<p className="text-xs text-muted-foreground">
							Threshold for determining issue complexity (0-100). Higher values
							mean more issues are considered complex.
						</p>
					</div>

					{/* Force Design Confirm */}
					<div className="flex items-center justify-between">
						<div className="space-y-0.5">
							<Label htmlFor="forceDesignConfirm">Force Design Confirm</Label>
							<p className="text-xs text-muted-foreground">
								Require manual confirmation before proceeding from design stage
							</p>
						</div>
						<Switch
							id="forceDesignConfirm"
							checked={config.forceDesignConfirm ?? false}
							onCheckedChange={(checked) =>
								handleBooleanChange("forceDesignConfirm", checked)
							}
						/>
					</div>
				</div>

				{/* Save Button */}
				<div className="mt-8 flex justify-end gap-3">
					<Button
						variant="outline"
						onClick={() => {
							if (repository?.config) {
								setConfig(repository.config);
							}
						}}
					>
						Reset
					</Button>
					<Button onClick={handleSave} disabled={updateMutation.isPending}>
						{updateMutation.isPending ? (
							<LoaderIcon className="size-4 animate-spin" />
						) : (
							<SaveIcon className="size-4" />
						)}
						{updateMutation.isPending ? "Saving..." : "Save Configuration"}
					</Button>
				</div>
			</section>

			{/* Effective Config Display */}
			{effectiveConfig && (
				<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
					<h2 className="text-lg font-semibold text-card-foreground mb-4">
						Effective Configuration
					</h2>
					<p className="text-sm text-muted-foreground mb-4">
						This shows the merged configuration (global defaults + repository
						overrides) that will be used for processing.
					</p>

					<div className="grid gap-4 lg:grid-cols-2">
						{/* Config Source */}
						<div className="rounded-lg border border-border bg-background p-4">
							<h3 className="text-sm font-medium text-foreground mb-2">
								Configuration Source
							</h3>
							<div className="space-y-2 text-xs">
								<div className="flex justify-between">
									<span className="text-muted-foreground">
										Has Repo Config:
									</span>
									<span className="text-foreground">
										{effectiveConfig.hasRepoConfig ? "Yes" : "No"}
									</span>
								</div>
								<div className="flex justify-between">
									<span className="text-muted-foreground">Repository ID:</span>
									<span className="text-foreground">
										{effectiveConfig.repositoryId || "N/A"}
									</span>
								</div>
								<div className="flex justify-between">
									<span className="text-muted-foreground">Enabled:</span>
									<span className="text-foreground">
										{effectiveConfig.enabled ? "Yes" : "No"}
									</span>
								</div>
							</div>
						</div>

						{/* Effective Values */}
						<div className="rounded-lg border border-border bg-background p-4">
							<h3 className="text-sm font-medium text-foreground mb-2">
								Effective Values
							</h3>
							<div className="space-y-2 text-xs">
								{effectiveConfig.effective && (
									<>
										<div className="flex justify-between">
											<span className="text-muted-foreground">
												Default Model:
											</span>
											<span className="text-foreground">
												{effectiveConfig.effective.defaultModel ||
													"Global Default"}
											</span>
										</div>
										<div className="flex justify-between">
											<span className="text-muted-foreground">
												Max Concurrency:
											</span>
											<span className="text-foreground">
												{effectiveConfig.effective.maxConcurrency ??
													"Global Default"}
											</span>
										</div>
										<div className="flex justify-between">
											<span className="text-muted-foreground">
												Task Retry Limit:
											</span>
											<span className="text-foreground">
												{effectiveConfig.effective.taskRetryLimit ??
													"Global Default"}
											</span>
										</div>
										<div className="flex justify-between">
											<span className="text-muted-foreground">
												Complexity Threshold:
											</span>
											<span className="text-foreground">
												{effectiveConfig.effective.complexityThreshold ??
													"Global Default"}
											</span>
										</div>
										<div className="flex justify-between">
											<span className="text-muted-foreground">
												Force Design Confirm:
											</span>
											<span className="text-foreground">
												{effectiveConfig.effective.forceDesignConfirm
													? "Yes"
													: "No"}
											</span>
										</div>
									</>
								)}
							</div>
						</div>

						{/* Global Config */}
						{effectiveConfig.globalConfig && (
							<div className="rounded-lg border border-border bg-background p-4 lg:col-span-2">
								<h3 className="text-sm font-medium text-foreground mb-2">
									Global Default Configuration
								</h3>
								<pre className="text-xs text-muted-foreground overflow-auto">
									{JSON.stringify(effectiveConfig.globalConfig, null, 2)}
								</pre>
							</div>
						)}
					</div>
				</section>
			)}
		</div>
	);
}
