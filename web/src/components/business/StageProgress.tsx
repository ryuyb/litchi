import {
	CheckCircleIcon,
	CircleIcon,
	LoaderIcon,
	PauseIcon,
	XCircleIcon,
} from "lucide-react";
import { memo } from "react";
import { type SessionCurrentStage, SessionStatus } from "#/api/schemas";
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "#/components/ui/tooltip";
import { workflowStages } from "#/lib/session-config";

export interface StageProgressProps {
	/** Current session stage */
	currentStage: SessionCurrentStage | undefined;
	/** Current session status */
	status: SessionStatus | undefined;
	/** Optional callback when a clickable stage is clicked */
	onStageClick?: (stage: SessionCurrentStage) => void;
}

/**
 * StageProgress component displays the workflow progress through 6 stages:
 * Clarification -> Design -> TaskBreakdown -> Execution -> PullRequest -> Completed
 *
 * Features:
 * - Visual indication of completed, active, and pending stages
 * - Tooltip with stage description on hover
 * - Clickable stages (completed or active) with callback support
 */
export const StageProgress = memo(function StageProgress({
	currentStage,
	status,
	onStageClick,
}: StageProgressProps) {
	const currentIndex = workflowStages.findIndex(
		(stage) => stage.key === currentStage,
	);

	const handleStageClick = (
		stage: SessionCurrentStage,
		isClickable: boolean,
	) => {
		if (!isClickable) return;
		onStageClick?.(stage);
	};

	return (
		<TooltipProvider>
			<div className="flex items-center justify-between">
				{workflowStages.map((stage, index) => {
					const isActive = index === currentIndex;
					const isCompleted = index < currentIndex;
					const isPending = index > currentIndex;
					const isTerminated = status === SessionStatus.terminated;

					// A stage is clickable if it's completed or active (not terminated)
					const isClickable = (isCompleted || isActive) && !isTerminated;

					let icon: React.ReactNode;
					let colorClass: string;

					if (isTerminated) {
						icon = <XCircleIcon className="size-5" />;
						colorClass = "text-red-500";
					} else if (isCompleted) {
						icon = <CheckCircleIcon className="size-5" />;
						colorClass = "text-green-500";
					} else if (isActive) {
						icon =
							status === SessionStatus.paused ? (
								<PauseIcon className="size-5" />
							) : (
								<LoaderIcon className="size-5 animate-spin" />
							);
						colorClass = "text-orange-500";
					} else {
						icon = <CircleIcon className="size-5" />;
						colorClass = "text-muted-foreground";
					}

					const stageContent = (
						<>
							<div className={colorClass}>{icon}</div>
							<span
								className={`text-xs font-medium ${
									isActive ? colorClass : "text-muted-foreground"
								}`}
							>
								{stage.label}
							</span>
						</>
					);

					return (
						<div key={stage.key} className="flex flex-1 items-center">
							<Tooltip>
								<TooltipTrigger asChild>
									{isClickable ? (
										<button
											type="button"
											className="flex flex-col items-center gap-1 transition-opacity cursor-pointer hover:opacity-80 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 rounded-sm bg-transparent border-0 p-0"
											onClick={() => handleStageClick(stage.key, true)}
										>
											{stageContent}
										</button>
									) : (
										<div
											className={`flex flex-col items-center gap-1 cursor-default ${isPending ? "opacity-50" : ""}`}
										>
											{stageContent}
										</div>
									)}
								</TooltipTrigger>
								<TooltipContent side="top">
									<p className="font-medium">{stage.label}</p>
									<p className="text-xs text-muted-foreground">
										{stage.description}
									</p>
								</TooltipContent>
							</Tooltip>
							{index < workflowStages.length - 1 && (
								<div
									className={`mx-2 h-0.5 flex-1 ${
										isCompleted || isActive ? "bg-green-500" : "bg-border"
									}`}
								/>
							)}
						</div>
					);
				})}
			</div>
		</TooltipProvider>
	);
});
