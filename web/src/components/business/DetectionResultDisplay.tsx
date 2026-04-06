import { useQueryClient } from "@tanstack/react-query";
import {
	FlaskConicalIcon,
	PaintbrushIcon,
	RefreshCcwIcon,
	ScanSearchIcon,
} from "lucide-react";
import { memo } from "react";
import {
	useGetApiV1RepositoriesNameDetection,
	usePostApiV1RepositoriesNameDetection,
} from "#/api/repositories/repositories";
import type { DetectedProject, DetectedTool } from "#/api/schemas";
import { Badge } from "#/components/ui/badge";
import { Button } from "#/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "#/components/ui/card";
import { Progress } from "#/components/ui/progress";
import { Skeleton } from "#/components/ui/skeleton";

// Helper to check if response is successful (status 200)
function isSuccessfulDetectionResponse(
	response:
		| {
				data: DetectedProject | { message?: string; code?: string };
				status: number;
		  }
		| undefined,
): response is { data: DetectedProject; status: 200 } {
	return response?.status === 200;
}

export interface DetectionResultDisplayProps {
	/** Repository name in owner/repo format */
	repositoryName: string;
	/** Optional callback when applying recommended config */
	onApplyConfig?: (config: DetectedProject) => void;
}

/**
 * Get color class based on confidence level
 * - >= 80: Green (high confidence)
 * - >= 50: Yellow (medium confidence)
 * - < 50: Red (low confidence)
 */
const getConfidenceColor = (confidence: number): string => {
	if (confidence >= 80) return "text-green-600 dark:text-green-400";
	if (confidence >= 50) return "text-yellow-600 dark:text-yellow-400";
	return "text-red-600 dark:text-red-400";
};

/**
 * Get progress bar indicator color based on confidence level
 */
const getConfidenceProgressColor = (confidence: number): string => {
	if (confidence >= 80) return "bg-green-600 dark:bg-green-400";
	if (confidence >= 50) return "bg-yellow-600 dark:bg-yellow-400";
	return "bg-red-600 dark:bg-red-400";
};

/**
 * Project type badge variant configuration
 */
const projectTypeVariants: Record<string, "default" | "secondary" | "outline"> =
	{
		Go: "default",
		NodeJS: "secondary",
		Python: "default",
		Rust: "default",
		Java: "secondary",
		Mixed: "outline",
		Unknown: "outline",
	};

/**
 * Tool type icons mapping
 */
const toolTypeIcons: Record<string, React.ReactNode> = {
	formatter: <PaintbrushIcon className="size-4" />,
	linter: <ScanSearchIcon className="size-4" />,
	tester: <FlaskConicalIcon className="size-4" />,
};

/**
 * Tool type badge variant configuration
 */
const toolTypeVariants: Record<
	string,
	"default" | "secondary" | "outline" | "info" | "success" | "warning"
> = {
	formatter: "info",
	linter: "warning",
	tester: "success",
};

/**
 * Display a single detected tool
 */
const ToolCard = memo(function ToolCard({ tool }: { tool: DetectedTool }) {
	const type = tool.type ?? "unknown";
	const name = tool.name ?? "Unknown";
	const configFile = tool.configFile;
	const detectionBasis = tool.detectionBasis;
	const recommendedCommand = tool.recommendedCommand;

	const icon = toolTypeIcons[type] ?? <ScanSearchIcon className="size-4" />;
	const variant = toolTypeVariants[type] ?? "outline";

	return (
		<div className="flex flex-col gap-1 rounded-md border p-2">
			<div className="flex items-center gap-2">
				<Badge variant={variant} className="gap-1">
					{icon}
					{type}
				</Badge>
				<span className="font-medium text-sm">{name}</span>
			</div>
			{configFile && (
				<div className="text-muted-foreground text-xs">
					Config: <code className="font-mono">{configFile}</code>
				</div>
			)}
			{detectionBasis && (
				<div className="text-muted-foreground text-xs">
					Basis: {detectionBasis}
				</div>
			)}
			{recommendedCommand && (
				<div className="text-muted-foreground text-xs">
					Command:{" "}
					<code className="font-mono">
						{recommendedCommand.command}
						{recommendedCommand.args &&
							recommendedCommand.args.length > 0 &&
							` ${recommendedCommand.args.join(" ")}`}
					</code>
				</div>
			)}
		</div>
	);
});

/**
 * Display tools grouped by type
 */
const ToolsSection = memo(function ToolsSection({
	tools,
}: {
	tools: DetectedTool[];
}) {
	// Group tools by type
	const groupedTools: Record<string, DetectedTool[]> = {};
	for (const tool of tools) {
		const type = tool.type ?? "unknown";
		if (!groupedTools[type]) {
			groupedTools[type] = [];
		}
		groupedTools[type].push(tool);
	}

	const toolTypes = ["formatter", "linter", "tester"];

	return (
		<div className="flex flex-col gap-3">
			{toolTypes.map((type) => {
				const typeTools = groupedTools[type];
				if (!typeTools || typeTools.length === 0) return null;

				const icon = toolTypeIcons[type];
				const label =
					type === "formatter"
						? "Formatters"
						: type === "linter"
							? "Linters"
							: "Testers";

				return (
					<div key={type} className="flex flex-col gap-2">
						<div className="flex items-center gap-2 text-sm font-medium">
							{icon}
							<span>{label}</span>
							<Badge variant="outline" className="text-xs">
								{typeTools.length}
							</Badge>
						</div>
						<div className="grid gap-2 md:grid-cols-2">
							{typeTools.map((tool, index) => (
								<ToolCard key={`${type}-${tool.name ?? index}`} tool={tool} />
							))}
						</div>
					</div>
				);
			})}
			{/* Display unknown type tools if any */}
			{groupedTools.unknown && groupedTools.unknown.length > 0 && (
				<div className="flex flex-col gap-2">
					<div className="flex items-center gap-2 text-sm font-medium">
						<ScanSearchIcon className="size-4" />
						<span>Other Tools</span>
						<Badge variant="outline" className="text-xs">
							{groupedTools.unknown.length}
						</Badge>
					</div>
					<div className="grid gap-2 md:grid-cols-2">
						{groupedTools.unknown.map((tool, index) => (
							<ToolCard key={`unknown-${tool.name ?? index}`} tool={tool} />
						))}
					</div>
				</div>
			)}
		</div>
	);
});

/**
 * Loading skeleton for the detection result
 */
const DetectionSkeleton = memo(function DetectionSkeleton() {
	return (
		<Card>
			<CardHeader>
				<Skeleton className="h-6 w-40" />
				<Skeleton className="h-4 w-60" />
			</CardHeader>
			<CardContent className="flex flex-col gap-4">
				<div className="flex items-center gap-2">
					<Skeleton className="h-6 w-20" />
					<Skeleton className="h-4 w-24" />
				</div>
				<div className="flex flex-col gap-2">
					<Skeleton className="h-4 w-16" />
					<Skeleton className="h-2 w-full" />
				</div>
				<div className="flex flex-col gap-2">
					<Skeleton className="h-4 w-20" />
					<div className="flex gap-2">
						<Skeleton className="h-5 w-16" />
						<Skeleton className="h-5 w-16" />
					</div>
				</div>
				<Skeleton className="h-20 w-full" />
			</CardContent>
		</Card>
	);
});

/**
 * Empty state when no detection result exists
 */
const NoDetectionState = memo(function NoDetectionState({
	onRunDetection,
	isLoading,
}: {
	onRunDetection: () => void;
	isLoading: boolean;
}) {
	return (
		<Card>
			<CardHeader>
				<CardTitle>Project Detection</CardTitle>
				<CardDescription>
					No detection result available for this repository
				</CardDescription>
			</CardHeader>
			<CardContent>
				<Button onClick={onRunDetection} disabled={isLoading}>
					{isLoading ? (
						<RefreshCcwIcon className="size-4 animate-spin" />
					) : (
						<ScanSearchIcon className="size-4" />
					)}
					<span>Run Detection</span>
				</Button>
			</CardContent>
		</Card>
	);
});

/**
 * DetectionResultDisplay component shows the project detection results
 * including project type, primary language, confidence, detected tools, etc.
 *
 * Features:
 * - Displays project type with badge
 * - Shows confidence level with color-coded progress bar
 * - Lists detected languages
 * - Groups detected tools by type (formatter, linter, tester)
 * - Supports re-running detection
 * - Optional callback to apply recommended configuration
 */
export const DetectionResultDisplay = memo(function DetectionResultDisplay({
	repositoryName,
	onApplyConfig,
}: DetectionResultDisplayProps) {
	const queryClient = useQueryClient();

	// Query for detection result
	const {
		data: detectionResponse,
		isLoading,
		isError,
	} = useGetApiV1RepositoriesNameDetection(repositoryName);

	// Mutation for triggering detection
	const detectionMutation = usePostApiV1RepositoriesNameDetection({
		mutation: {
			onSuccess: (response) => {
				// Invalidate the detection query to refresh data
				queryClient.invalidateQueries({
					queryKey: [`/api/v1/repositories/${repositoryName}/detection`],
				});
				// Optionally apply config if callback provided
				if (isSuccessfulDetectionResponse(response) && onApplyConfig) {
					onApplyConfig(response.data);
				}
			},
		},
	});

	// Extract detection data from response
	const hasDetection = isSuccessfulDetectionResponse(detectionResponse);
	const detection = hasDetection ? detectionResponse.data : null;

	// Handle run detection action
	const handleRunDetection = () => {
		detectionMutation.mutate({ name: repositoryName });
	};

	// Handle apply config action
	const handleApplyConfig = () => {
		if (detection && onApplyConfig) {
			onApplyConfig(detection);
		}
	};

	// Loading state
	if (isLoading) {
		return <DetectionSkeleton />;
	}

	// Error state (treat as no detection)
	if (isError || !detection) {
		return (
			<NoDetectionState
				onRunDetection={handleRunDetection}
				isLoading={detectionMutation.isPending}
			/>
		);
	}

	// Extract data from detection
	const projectType = detection.type ?? "Unknown";
	const primaryLanguage = detection.primaryLanguage ?? "Unknown";
	const confidence = detection.confidence ?? 0;
	const languages = detection.languages ?? [];
	const detectedTools = detection.detectedTools ?? [];
	const detectedAt = detection.detectedAt;

	const confidenceColor = getConfidenceColor(confidence);
	const progressColor = getConfidenceProgressColor(confidence);
	const badgeVariant = projectTypeVariants[projectType] ?? "outline";

	// Format detection timestamp
	const detectedAtDisplay = detectedAt
		? new Date(detectedAt).toLocaleString()
		: "Unknown";

	return (
		<Card>
			<CardHeader>
				<CardTitle>Project Detection</CardTitle>
				<CardDescription>
					Automatically detected project configuration
				</CardDescription>
			</CardHeader>
			<CardContent className="flex flex-col gap-4">
				{/* Project Type and Primary Language */}
				<div className="flex items-center gap-2">
					<Badge variant={badgeVariant}>{projectType}</Badge>
					<span className="text-muted-foreground text-sm">
						Primary: {primaryLanguage}
					</span>
				</div>

				{/* Confidence Progress */}
				<div className="flex flex-col gap-2">
					<div className="flex items-center justify-between text-sm">
						<span className="font-medium">Confidence</span>
						<span className={confidenceColor}>{confidence}%</span>
					</div>
					<Progress
						value={confidence}
						className="h-2"
						indicatorClassName={progressColor}
					/>
				</div>

				{/* Languages List */}
				{languages.length > 0 && (
					<div className="flex flex-col gap-2">
						<span className="font-medium text-sm">Languages</span>
						<div className="flex flex-wrap gap-2">
							{languages.map((lang) => (
								<Badge key={lang} variant="outline">
									{lang}
								</Badge>
							))}
						</div>
					</div>
				)}

				{/* Detected Tools Section */}
				{detectedTools.length > 0 && (
					<div className="flex flex-col gap-3">
						<span className="font-medium text-sm">Detected Tools</span>
						<ToolsSection tools={detectedTools} />
					</div>
				)}

				{/* Detection Timestamp */}
				<div className="text-muted-foreground text-xs">
					Detected at: {detectedAtDisplay}
				</div>

				{/* Action Buttons */}
				<div className="flex items-center gap-2 pt-2">
					<Button
						variant="outline"
						size="sm"
						onClick={handleRunDetection}
						disabled={detectionMutation.isPending}
					>
						{detectionMutation.isPending ? (
							<RefreshCcwIcon className="size-4 animate-spin" />
						) : (
							<RefreshCcwIcon className="size-4" />
						)}
						<span>Re-detect</span>
					</Button>
					{onApplyConfig && (
						<Button variant="default" size="sm" onClick={handleApplyConfig}>
							<span>Apply Config</span>
						</Button>
					)}
				</div>
			</CardContent>
		</Card>
	);
});
