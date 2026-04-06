import { useQueryClient } from "@tanstack/react-query";
import { LoaderIcon, PlusIcon, SaveIcon, Trash2Icon } from "lucide-react";
import { useEffect, useState } from "react";
import {
	getGetApiV1RepositoriesNameValidationConfigQueryKey,
	useGetApiV1RepositoriesNameValidationConfig,
	usePutApiV1RepositoriesNameValidationConfig,
} from "#/api/repositories/repositories";
import type { FormattingConfig } from "#/api/schemas/formattingConfig";
import type { LintingConfig } from "#/api/schemas/lintingConfig";
import type { TestingConfig } from "#/api/schemas/testingConfig";
import type { ToolCommand } from "#/api/schemas/toolCommand";
import type { ValidationConfig } from "#/api/schemas/validationConfig";
import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import { Label } from "#/components/ui/label";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import { Switch } from "#/components/ui/switch";

export interface ValidationConfigFormProps {
	repositoryName: string;
	onSaveSuccess?: () => void;
}

// Failure strategy options
const failureStrategies = [
	{ value: "fail_fast", label: "Fail Fast" },
	{ value: "auto_fix", label: "Auto Fix" },
	{ value: "warn_continue", label: "Warn and Continue" },
	{ value: "skip", label: "Skip" },
];

// No tests strategy options
const noTestsStrategies = [
	{ value: "skip", label: "Skip" },
	{ value: "warn", label: "Warn" },
	{ value: "fail", label: "Fail" },
];

// Default tool command template
const defaultToolCommand: ToolCommand = {
	name: "",
	command: "",
	args: [],
};

// Helper to check if response is successful (status 200)
function isSuccessfulValidationConfigResponse(
	response:
		| {
				data: ValidationConfig | { message?: string; code?: string };
				status: number;
		  }
		| undefined,
): response is { data: ValidationConfig; status: 200 } {
	return response?.status === 200;
}

// Tool command editor component
function ToolCommandEditor({
	tool,
	onChange,
	onRemove,
}: {
	tool: ToolCommand;
	onChange: (tool: ToolCommand) => void;
	onRemove: () => void;
}) {
	return (
		<div className="flex items-start gap-2 rounded-lg border border-border bg-background p-3">
			<div className="flex-1 grid gap-2">
				<div className="grid gap-1">
					<Label className="text-xs">Tool Name</Label>
					<Input
						placeholder="e.g., prettier"
						value={tool.name ?? ""}
						onChange={(e) => onChange({ ...tool, name: e.target.value })}
						className="h-8"
					/>
				</div>
				<div className="grid gap-1">
					<Label className="text-xs">Command</Label>
					<Input
						placeholder="e.g., npx prettier --write"
						value={tool.command ?? ""}
						onChange={(e) => onChange({ ...tool, command: e.target.value })}
						className="h-8"
					/>
				</div>
				<div className="grid gap-1">
					<Label className="text-xs">Args (comma-separated)</Label>
					<Input
						placeholder="e.g., --write, --check"
						value={(tool.args ?? []).join(", ")}
						onChange={(e) =>
							onChange({
								...tool,
								args: e.target.value
									.split(",")
									.map((a) => a.trim())
									.filter(Boolean),
							})
						}
						className="h-8"
					/>
				</div>
			</div>
			<Button
				type="button"
				variant="ghost"
				size="sm"
				onClick={onRemove}
				className="text-destructive hover:text-destructive"
			>
				<Trash2Icon className="size-4" />
			</Button>
		</div>
	);
}

// Formatting section component
function FormattingSection({
	config,
	onChange,
}: {
	config: FormattingConfig;
	onChange: (config: FormattingConfig) => void;
}) {
	const handleToolChange = (index: number, tool: ToolCommand) => {
		const newTools = [...(config.tools ?? [])];
		newTools[index] = tool;
		onChange({ ...config, tools: newTools });
	};

	const handleToolRemove = (index: number) => {
		const newTools = [...(config.tools ?? [])];
		newTools.splice(index, 1);
		onChange({ ...config, tools: newTools });
	};

	const handleAddTool = () => {
		onChange({
			...config,
			tools: [...(config.tools ?? []), { ...defaultToolCommand }],
		});
	};

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<div className="space-y-0.5">
					<Label className="text-base font-medium">Formatting</Label>
					<p className="text-xs text-muted-foreground">
						Configure code formatting tools and strategies
					</p>
				</div>
				<Switch
					checked={config.enabled ?? false}
					onCheckedChange={(checked) =>
						onChange({ ...config, enabled: checked })
					}
				/>
			</div>

			{config.enabled && (
				<div className="space-y-4 pl-4 border-l-2 border-border">
					{/* Tool list */}
					<div className="space-y-2">
						<Label className="text-sm">Formatting Tools</Label>
						<div className="space-y-2">
							{(config.tools ?? []).map((tool, index) => (
								<ToolCommandEditor
									key={`formatter-${tool.name ?? ""}-${index}`}
									tool={tool}
									onChange={(t) => handleToolChange(index, t)}
									onRemove={() => handleToolRemove(index)}
								/>
							))}
						</div>
						<Button
							type="button"
							variant="outline"
							size="sm"
							onClick={handleAddTool}
							className="w-full"
						>
							<PlusIcon className="size-4" />
							Add Tool
						</Button>
					</div>

					{/* Failure strategy */}
					<div className="space-y-1">
						<Label className="text-sm">Failure Strategy</Label>
						<Select
							value={config.failureStrategy ?? "fail_fast"}
							onValueChange={(value) =>
								onChange({ ...config, failureStrategy: value })
							}
						>
							<SelectTrigger className="w-full">
								<SelectValue placeholder="Select strategy" />
							</SelectTrigger>
							<SelectContent>
								{failureStrategies.map((s) => (
									<SelectItem key={s.value} value={s.value}>
										{s.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>
				</div>
			)}
		</div>
	);
}

// Linting section component
function LintingSection({
	config,
	onChange,
}: {
	config: LintingConfig;
	onChange: (config: LintingConfig) => void;
}) {
	const handleToolChange = (index: number, tool: ToolCommand) => {
		const newTools = [...(config.tools ?? [])];
		newTools[index] = tool;
		onChange({ ...config, tools: newTools });
	};

	const handleToolRemove = (index: number) => {
		const newTools = [...(config.tools ?? [])];
		newTools.splice(index, 1);
		onChange({ ...config, tools: newTools });
	};

	const handleAddTool = () => {
		onChange({
			...config,
			tools: [...(config.tools ?? []), { ...defaultToolCommand }],
		});
	};

	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<div className="space-y-0.5">
					<Label className="text-base font-medium">Linting</Label>
					<p className="text-xs text-muted-foreground">
						Configure linting tools and auto-fix options
					</p>
				</div>
				<Switch
					checked={config.enabled ?? false}
					onCheckedChange={(checked) =>
						onChange({ ...config, enabled: checked })
					}
				/>
			</div>

			{config.enabled && (
				<div className="space-y-4 pl-4 border-l-2 border-border">
					{/* Tool list */}
					<div className="space-y-2">
						<Label className="text-sm">Linting Tools</Label>
						<div className="space-y-2">
							{(config.tools ?? []).map((tool, index) => (
								<ToolCommandEditor
									key={`linter-${tool.name ?? ""}-${index}`}
									tool={tool}
									onChange={(t) => handleToolChange(index, t)}
									onRemove={() => handleToolRemove(index)}
								/>
							))}
						</div>
						<Button
							type="button"
							variant="outline"
							size="sm"
							onClick={handleAddTool}
							className="w-full"
						>
							<PlusIcon className="size-4" />
							Add Tool
						</Button>
					</div>

					{/* Failure strategy */}
					<div className="space-y-1">
						<Label className="text-sm">Failure Strategy</Label>
						<Select
							value={config.failureStrategy ?? "fail_fast"}
							onValueChange={(value) =>
								onChange({ ...config, failureStrategy: value })
							}
						>
							<SelectTrigger className="w-full">
								<SelectValue placeholder="Select strategy" />
							</SelectTrigger>
							<SelectContent>
								{failureStrategies.map((s) => (
									<SelectItem key={s.value} value={s.value}>
										{s.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					{/* Auto-fix toggle */}
					<div className="flex items-center justify-between">
						<div className="space-y-0.5">
							<Label className="text-sm">Auto Fix</Label>
							<p className="text-xs text-muted-foreground">
								Automatically fix linting issues when possible
							</p>
						</div>
						<Switch
							checked={config.autoFix ?? false}
							onCheckedChange={(checked) =>
								onChange({ ...config, autoFix: checked })
							}
						/>
					</div>
				</div>
			)}
		</div>
	);
}

// Testing section component
function TestingSection({
	config,
	onChange,
}: {
	config: TestingConfig;
	onChange: (config: TestingConfig) => void;
}) {
	return (
		<div className="space-y-4">
			<div className="flex items-center justify-between">
				<div className="space-y-0.5">
					<Label className="text-base font-medium">Testing</Label>
					<p className="text-xs text-muted-foreground">
						Configure test commands and strategies
					</p>
				</div>
				<Switch
					checked={config.enabled ?? false}
					onCheckedChange={(checked) =>
						onChange({ ...config, enabled: checked })
					}
				/>
			</div>

			{config.enabled && (
				<div className="space-y-4 pl-4 border-l-2 border-border">
					{/* Test command */}
					<div className="space-y-2">
						<Label className="text-sm">Test Command</Label>
						<div className="grid gap-2">
							<div className="grid gap-1">
								<Label className="text-xs">Command Name</Label>
								<Input
									placeholder="e.g., npm test"
									value={config.command?.command ?? ""}
									onChange={(e) =>
										onChange({
											...config,
											command: { ...config.command, command: e.target.value },
										})
									}
									className="h-8"
								/>
							</div>
							<div className="grid gap-1">
								<Label className="text-xs">Arguments (comma-separated)</Label>
								<Input
									placeholder="e.g., --coverage, --watch"
									value={(config.command?.args ?? []).join(", ")}
									onChange={(e) =>
										onChange({
											...config,
											command: {
												...config.command,
												args: e.target.value
													.split(",")
													.map((a) => a.trim())
													.filter(Boolean),
											},
										})
									}
									className="h-8"
								/>
							</div>
						</div>
					</div>

					{/* Failure strategy */}
					<div className="space-y-1">
						<Label className="text-sm">Failure Strategy</Label>
						<Select
							value={config.failureStrategy ?? "fail_fast"}
							onValueChange={(value) =>
								onChange({ ...config, failureStrategy: value })
							}
						>
							<SelectTrigger className="w-full">
								<SelectValue placeholder="Select strategy" />
							</SelectTrigger>
							<SelectContent>
								{failureStrategies.map((s) => (
									<SelectItem key={s.value} value={s.value}>
										{s.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					{/* No tests strategy */}
					<div className="space-y-1">
						<Label className="text-sm">No Tests Strategy</Label>
						<p className="text-xs text-muted-foreground mb-2">
							How to handle when no test files are found
						</p>
						<Select
							value={config.noTestsStrategy ?? "skip"}
							onValueChange={(value) =>
								onChange({ ...config, noTestsStrategy: value })
							}
						>
							<SelectTrigger className="w-full">
								<SelectValue placeholder="Select strategy" />
							</SelectTrigger>
							<SelectContent>
								{noTestsStrategies.map((s) => (
									<SelectItem key={s.value} value={s.value}>
										{s.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>
				</div>
			)}
		</div>
	);
}

export function ValidationConfigForm({
	repositoryName,
	onSaveSuccess,
}: ValidationConfigFormProps) {
	const queryClient = useQueryClient();

	// Fetch validation config
	const {
		data: validationConfigResponse,
		isLoading,
		error,
	} = useGetApiV1RepositoriesNameValidationConfig(repositoryName);

	// Mutation for updating validation config
	const updateMutation = usePutApiV1RepositoriesNameValidationConfig({
		mutation: {
			onSuccess: () => {
				setSaveSuccess(true);
				setTimeout(() => setSaveSuccess(false), 3000);
				queryClient.invalidateQueries({
					queryKey:
						getGetApiV1RepositoriesNameValidationConfigQueryKey(repositoryName),
				});
				onSaveSuccess?.();
			},
		},
	});

	// Form state
	const [config, setConfig] = useState<ValidationConfig>({
		enabled: false,
		formatting: { enabled: false },
		linting: { enabled: false },
		testing: { enabled: false },
	});
	const [saveSuccess, setSaveSuccess] = useState(false);

	// Extract successful data
	const validationConfig = isSuccessfulValidationConfigResponse(
		validationConfigResponse,
	)
		? validationConfigResponse.data
		: null;

	// Initialize form with fetched config when data loads
	useEffect(() => {
		if (validationConfig) {
			setConfig(validationConfig);
		}
	}, [validationConfig]);

	// Handle section changes
	const handleFormattingChange = (formatting: FormattingConfig) => {
		setConfig((prev) => ({ ...prev, formatting }));
	};

	const handleLintingChange = (linting: LintingConfig) => {
		setConfig((prev) => ({ ...prev, linting }));
	};

	const handleTestingChange = (testing: TestingConfig) => {
		setConfig((prev) => ({ ...prev, testing }));
	};

	// Handle main enabled toggle
	const handleEnabledChange = (enabled: boolean) => {
		setConfig((prev) => ({ ...prev, enabled }));
	};

	// Handle save
	const handleSave = () => {
		updateMutation.mutate({
			name: repositoryName,
			data: { config },
		});
	};

	// Handle reset
	const handleReset = () => {
		if (validationConfig) {
			setConfig(validationConfig);
		}
	};

	// Loading state
	if (isLoading) {
		return (
			<div className="flex items-center justify-center min-h-[200px] gap-2">
				<LoaderIcon className="size-6 animate-spin text-muted-foreground" />
				<span className="text-muted-foreground">
					Loading validation config...
				</span>
			</div>
		);
	}

	// Error state
	if (error) {
		return (
			<div className="rounded-lg border border-destructive bg-card p-4">
				<p className="text-destructive">
					Error loading validation configuration: {error.message}
				</p>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="space-y-0.5">
					<h3 className="text-lg font-semibold">Validation Configuration</h3>
					<p className="text-sm text-muted-foreground">
						Configure formatting, linting, and testing settings for this
						repository
					</p>
				</div>
				<Switch
					checked={config.enabled ?? false}
					onCheckedChange={handleEnabledChange}
				/>
			</div>

			{/* Success/Error Messages */}
			{saveSuccess && (
				<div className="rounded-lg border border-green-500/50 bg-green-500/10 p-3">
					<p className="text-green-600 dark:text-green-400 text-sm">
						Validation configuration saved successfully!
					</p>
				</div>
			)}

			{/* Sections */}
			{config.enabled && (
				<div className="space-y-6">
					<FormattingSection
						config={config.formatting ?? { enabled: false }}
						onChange={handleFormattingChange}
					/>

					<LintingSection
						config={config.linting ?? { enabled: false }}
						onChange={handleLintingChange}
					/>

					<TestingSection
						config={config.testing ?? { enabled: false }}
						onChange={handleTestingChange}
					/>
				</div>
			)}

			{/* Save/Reset Buttons */}
			<div className="flex justify-end gap-3 pt-4">
				<Button type="button" variant="outline" onClick={handleReset}>
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
		</div>
	);
}
