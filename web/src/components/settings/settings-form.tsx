/**
 * Settings page configuration sections.
 * Provides form components for editing various system configurations.
 */
import { useQueryClient } from "@tanstack/react-query";
import { LoaderIcon, SaveIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import type {
	AgentConfigUpdate,
	AuditConfigUpdate,
	ClarityConfigUpdate,
	ComplexityConfigUpdate,
	Config,
	GitConfigUpdate,
	UpdateConfig,
} from "#/api/schemas";
import {
	getGetApiV1ConfigQueryKey,
	useGetApiV1Config,
	usePutApiV1Config,
} from "../../api/config/config";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Label } from "../ui/label";
import { Separator } from "../ui/separator";
import { Slider } from "../ui/slider";
import { Switch } from "../ui/switch";

// Type guard for successful config response
function isSuccessResponse(
	response: { data: Config | { message?: string }; status: number } | undefined,
): response is { data: Config; status: 200 } {
	return response?.status === 200;
}

// Helper to check if form has changes
function hasChanges(original: UpdateConfig, current: UpdateConfig): boolean {
	return JSON.stringify(original) !== JSON.stringify(current);
}

// ============ Agent Config Section ============

interface AgentConfigSectionProps {
	config: AgentConfigUpdate;
	onChange: (config: AgentConfigUpdate) => void;
}

export function AgentConfigSection({
	config,
	onChange,
}: AgentConfigSectionProps) {
	return (
		<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
			<h3 className="text-lg font-semibold text-card-foreground">
				Agent Configuration
			</h3>
			<p className="mt-1 text-sm text-muted-foreground">
				Configure agent behavior and retry settings.
			</p>

			<div className="mt-6 space-y-6">
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
							onChange({
								...config,
								taskRetryLimit: e.target.value
									? Number(e.target.value)
									: undefined,
							})
						}
					/>
					<p className="text-xs text-muted-foreground">
						Maximum retry attempts for failed tasks (0-10)
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
							onChange({
								...config,
								maxConcurrency: e.target.value
									? Number(e.target.value)
									: undefined,
							})
						}
					/>
					<p className="text-xs text-muted-foreground">
						Maximum number of concurrent tasks (1-10)
					</p>
				</div>

				{/* Approval Timeout */}
				<div className="space-y-2">
					<Label htmlFor="approvalTimeout">Approval Timeout</Label>
					<Input
						id="approvalTimeout"
						type="text"
						placeholder="e.g., 30m, 1h"
						value={config.approvalTimeout ?? ""}
						onChange={(e) =>
							onChange({
								...config,
								approvalTimeout: e.target.value || undefined,
							})
						}
					/>
					<p className="text-xs text-muted-foreground">
						Timeout duration for approval requests (e.g., 30m, 1h)
					</p>
				</div>
			</div>
		</section>
	);
}

// ============ Audit Config Section ============

interface AuditConfigSectionProps {
	config: AuditConfigUpdate;
	onChange: (config: AuditConfigUpdate) => void;
}

export function AuditConfigSection({
	config,
	onChange,
}: AuditConfigSectionProps) {
	return (
		<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
			<h3 className="text-lg font-semibold text-card-foreground">
				Audit Configuration
			</h3>
			<p className="mt-1 text-sm text-muted-foreground">
				Configure audit logging behavior.
			</p>

			<div className="mt-6 space-y-6">
				{/* Enabled */}
				<div className="flex items-center justify-between">
					<div className="space-y-0.5">
						<Label htmlFor="auditEnabled">Enable Audit Logging</Label>
						<p className="text-xs text-muted-foreground">
							Track all system activities and changes
						</p>
					</div>
					<Switch
						id="auditEnabled"
						checked={config.enabled ?? false}
						onCheckedChange={(checked) =>
							onChange({ ...config, enabled: checked })
						}
					/>
				</div>

				<Separator />

				{/* Retention Days */}
				<div className="space-y-2">
					<Label htmlFor="retentionDays">Retention Days</Label>
					<Input
						id="retentionDays"
						type="number"
						min={1}
						max={365}
						placeholder="1-365"
						value={config.retentionDays ?? ""}
						onChange={(e) =>
							onChange({
								...config,
								retentionDays: e.target.value
									? Number(e.target.value)
									: undefined,
							})
						}
					/>
					<p className="text-xs text-muted-foreground">
						Number of days to retain audit logs (1-365)
					</p>
				</div>

				{/* Max Output Length */}
				<div className="space-y-2">
					<Label htmlFor="maxOutputLength">Max Output Length</Label>
					<Input
						id="maxOutputLength"
						type="number"
						min={1000}
						placeholder="e.g., 10000"
						value={config.maxOutputLength ?? ""}
						onChange={(e) =>
							onChange({
								...config,
								maxOutputLength: e.target.value
									? Number(e.target.value)
									: undefined,
							})
						}
					/>
					<p className="text-xs text-muted-foreground">
						Maximum length of output stored in audit logs
					</p>
				</div>
			</div>
		</section>
	);
}

// ============ Clarity Config Section ============

interface ClarityConfigSectionProps {
	config: ClarityConfigUpdate;
	onChange: (config: ClarityConfigUpdate) => void;
}

export function ClarityConfigSection({
	config,
	onChange,
}: ClarityConfigSectionProps) {
	return (
		<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
			<h3 className="text-lg font-semibold text-card-foreground">
				Clarity Configuration
			</h3>
			<p className="mt-1 text-sm text-muted-foreground">
				Configure clarity threshold for issue understanding.
			</p>

			<div className="mt-6 space-y-6">
				{/* Clarity Threshold */}
				<div className="space-y-4">
					<div className="flex items-center justify-between">
						<Label htmlFor="clarityThreshold">Clarity Threshold</Label>
						<span className="text-sm font-medium">
							{config.threshold ?? 60}
						</span>
					</div>
					<Slider
						id="clarityThreshold"
						min={0}
						max={100}
						step={1}
						value={[config.threshold ?? 60]}
						onValueChange={(value) =>
							onChange({ ...config, threshold: value[0] })
						}
					/>
					<p className="text-xs text-muted-foreground">
						Issues above this threshold are considered clear enough to proceed
					</p>
				</div>

				<Separator />

				{/* Auto Proceed Threshold */}
				<div className="space-y-4">
					<div className="flex items-center justify-between">
						<Label htmlFor="autoProceedThreshold">Auto Proceed Threshold</Label>
						<span className="text-sm font-medium">
							{config.autoProceedThreshold ?? 80}
						</span>
					</div>
					<Slider
						id="autoProceedThreshold"
						min={0}
						max={100}
						step={1}
						value={[config.autoProceedThreshold ?? 80]}
						onValueChange={(value) =>
							onChange({ ...config, autoProceedThreshold: value[0] })
						}
					/>
					<p className="text-xs text-muted-foreground">
						Issues above this threshold automatically proceed to the next stage
					</p>
				</div>

				<Separator />

				{/* Force Clarify Threshold */}
				<div className="space-y-4">
					<div className="flex items-center justify-between">
						<Label htmlFor="forceClarifyThreshold">
							Force Clarify Threshold
						</Label>
						<span className="text-sm font-medium">
							{config.forceClarifyThreshold ?? 30}
						</span>
					</div>
					<Slider
						id="forceClarifyThreshold"
						min={0}
						max={100}
						step={1}
						value={[config.forceClarifyThreshold ?? 30]}
						onValueChange={(value) =>
							onChange({ ...config, forceClarifyThreshold: value[0] })
						}
					/>
					<p className="text-xs text-muted-foreground">
						Issues below this threshold require clarification before proceeding
					</p>
				</div>
			</div>
		</section>
	);
}

// ============ Complexity Config Section ============

interface ComplexityConfigSectionProps {
	config: ComplexityConfigUpdate;
	onChange: (config: ComplexityConfigUpdate) => void;
}

export function ComplexityConfigSection({
	config,
	onChange,
}: ComplexityConfigSectionProps) {
	return (
		<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
			<h3 className="text-lg font-semibold text-card-foreground">
				Complexity Configuration
			</h3>
			<p className="mt-1 text-sm text-muted-foreground">
				Configure complexity analysis for issues.
			</p>

			<div className="mt-6 space-y-6">
				{/* Complexity Threshold */}
				<div className="space-y-4">
					<div className="flex items-center justify-between">
						<Label htmlFor="complexityThreshold">Complexity Threshold</Label>
						<span className="text-sm font-medium">
							{config.threshold ?? 70}
						</span>
					</div>
					<Slider
						id="complexityThreshold"
						min={0}
						max={100}
						step={1}
						value={[config.threshold ?? 70]}
						onValueChange={(value) =>
							onChange({ ...config, threshold: value[0] })
						}
					/>
					<p className="text-xs text-muted-foreground">
						Issues above this threshold are considered complex
					</p>
				</div>

				<Separator />

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
							onChange({ ...config, forceDesignConfirm: checked })
						}
					/>
				</div>
			</div>
		</section>
	);
}

// ============ Git Config Section ============

interface GitConfigSectionProps {
	config: GitConfigUpdate;
	onChange: (config: GitConfigUpdate) => void;
}

export function GitConfigSection({ config, onChange }: GitConfigSectionProps) {
	return (
		<section className="rounded-xl border border-border bg-card p-6 shadow-sm">
			<h3 className="text-lg font-semibold text-card-foreground">
				Git Configuration
			</h3>
			<p className="mt-1 text-sm text-muted-foreground">
				Configure Git behavior and branch naming.
			</p>

			<div className="mt-6 space-y-6">
				{/* Branch Naming Pattern */}
				<div className="space-y-2">
					<Label htmlFor="branchNamingPattern">Branch Naming Pattern</Label>
					<Input
						id="branchNamingPattern"
						type="text"
						placeholder="e.g., litchi/{issue-number}"
						value={config.branchNamingPattern ?? ""}
						onChange={(e) =>
							onChange({
								...config,
								branchNamingPattern: e.target.value || undefined,
							})
						}
					/>
					<p className="text-xs text-muted-foreground">
						Pattern for naming branches (supports {`{issue-number}`}{" "}
						placeholder)
					</p>
				</div>

				{/* Default Base Branch */}
				<div className="space-y-2">
					<Label htmlFor="defaultBaseBranch">Default Base Branch</Label>
					<Input
						id="defaultBaseBranch"
						type="text"
						placeholder="e.g., main, master"
						value={config.defaultBaseBranch ?? ""}
						onChange={(e) =>
							onChange({
								...config,
								defaultBaseBranch: e.target.value || undefined,
							})
						}
					/>
					<p className="text-xs text-muted-foreground">
						Default branch to use as base for pull requests
					</p>
				</div>

				{/* Command Timeout */}
				<div className="space-y-2">
					<Label htmlFor="commandTimeout">Command Timeout</Label>
					<Input
						id="commandTimeout"
						type="text"
						placeholder="e.g., 5m, 10m"
						value={config.commandTimeout ?? ""}
						onChange={(e) =>
							onChange({
								...config,
								commandTimeout: e.target.value || undefined,
							})
						}
					/>
					<p className="text-xs text-muted-foreground">
						Timeout for Git commands (e.g., 5m, 10m)
					</p>
				</div>

				<Separator />

				{/* Commit Sign Off */}
				<div className="flex items-center justify-between">
					<div className="space-y-0.5">
						<Label htmlFor="commitSignOff">Commit Sign Off</Label>
						<p className="text-xs text-muted-foreground">
							Add Signed-off-by line to commits
						</p>
					</div>
					<Switch
						id="commitSignOff"
						checked={config.commitSignOff ?? false}
						onCheckedChange={(checked) =>
							onChange({ ...config, commitSignOff: checked })
						}
					/>
				</div>

				{/* Worktree Auto Clean */}
				<div className="flex items-center justify-between">
					<div className="space-y-0.5">
						<Label htmlFor="worktreeAutoClean">Worktree Auto Clean</Label>
						<p className="text-xs text-muted-foreground">
							Automatically clean up worktrees after completion
						</p>
					</div>
					<Switch
						id="worktreeAutoClean"
						checked={config.worktreeAutoClean ?? false}
						onCheckedChange={(checked) =>
							onChange({ ...config, worktreeAutoClean: checked })
						}
					/>
				</div>

				{/* Worktree Base Path */}
				<div className="space-y-2">
					<Label htmlFor="worktreeBasePath">Worktree Base Path</Label>
					<Input
						id="worktreeBasePath"
						type="text"
						placeholder="e.g., /tmp/litchi/worktrees"
						value={config.worktreeBasePath ?? ""}
						onChange={(e) =>
							onChange({
								...config,
								worktreeBasePath: e.target.value || undefined,
							})
						}
					/>
					<p className="text-xs text-muted-foreground">
						Base directory for Git worktrees
					</p>
				</div>
			</div>
		</section>
	);
}

// ============ Main Settings Form ============

interface SettingsFormProps {
	onSaveSuccess?: () => void;
}

export function SettingsForm({ onSaveSuccess }: SettingsFormProps) {
	const queryClient = useQueryClient();

	// Fetch config
	const { data: response, isLoading, error } = useGetApiV1Config();

	// Form state
	const [formData, setFormData] = useState<UpdateConfig>({});
	const [originalData, setOriginalData] = useState<UpdateConfig>({});

	// Update mutation
	const updateMutation = usePutApiV1Config({
		mutation: {
			onSuccess: () => {
				toast.success("Configuration saved successfully");
				queryClient.invalidateQueries({
					queryKey: getGetApiV1ConfigQueryKey(),
				});
				// Update original data to reset dirty state
				setOriginalData({ ...formData });
				onSaveSuccess?.();
			},
			onError: (err) => {
				toast.error("Failed to save configuration", {
					description: err.message,
				});
			},
		},
	});

	// Initialize form when data loads
	useEffect(() => {
		if (isSuccessResponse(response) && response.data) {
			const config = response.data;
			const initialData: UpdateConfig = {
				agent: config.agent,
				audit: config.audit,
				clarity: config.clarity,
				complexity: config.complexity,
				git: config.git,
			};
			setFormData(initialData);
			setOriginalData(initialData);
		}
	}, [response]);

	// Handle save
	const handleSave = () => {
		updateMutation.mutate({ data: formData });
	};

	// Handle reset
	const handleReset = () => {
		setFormData({ ...originalData });
	};

	// Loading state
	if (isLoading) {
		return (
			<div className="flex items-center justify-center min-h-[400px] gap-2">
				<LoaderIcon className="size-8 animate-spin text-muted-foreground" />
				<span className="text-muted-foreground">Loading configuration...</span>
			</div>
		);
	}

	// Error state
	if (error) {
		return (
			<div className="rounded-xl border border-destructive bg-card p-6">
				<h2 className="text-lg font-semibold text-destructive">
					Error loading configuration
				</h2>
				<p className="mt-2 text-muted-foreground">
					{error.message || "Failed to load configuration. Please try again."}
				</p>
			</div>
		);
	}

	const isDirty = hasChanges(originalData, formData);
	const isSaving = updateMutation.isPending;

	return (
		<div className="space-y-6">
			{/* Agent Config */}
			<AgentConfigSection
				config={formData.agent ?? {}}
				onChange={(agent) => setFormData((prev) => ({ ...prev, agent }))}
			/>

			{/* Audit Config */}
			<AuditConfigSection
				config={formData.audit ?? {}}
				onChange={(audit) => setFormData((prev) => ({ ...prev, audit }))}
			/>

			{/* Clarity Config */}
			<ClarityConfigSection
				config={formData.clarity ?? {}}
				onChange={(clarity) => setFormData((prev) => ({ ...prev, clarity }))}
			/>

			{/* Complexity Config */}
			<ComplexityConfigSection
				config={formData.complexity ?? {}}
				onChange={(complexity) =>
					setFormData((prev) => ({ ...prev, complexity }))
				}
			/>

			{/* Git Config */}
			<GitConfigSection
				config={formData.git ?? {}}
				onChange={(git) => setFormData((prev) => ({ ...prev, git }))}
			/>

			{/* Action Buttons */}
			<div className="flex justify-end gap-3">
				<Button
					variant="outline"
					onClick={handleReset}
					disabled={!isDirty || isSaving}
				>
					Reset
				</Button>
				<Button onClick={handleSave} disabled={!isDirty || isSaving}>
					{isSaving ? (
						<LoaderIcon className="size-4 animate-spin" />
					) : (
						<SaveIcon className="size-4" />
					)}
					{isSaving ? "Saving..." : "Save Configuration"}
				</Button>
			</div>
		</div>
	);
}
