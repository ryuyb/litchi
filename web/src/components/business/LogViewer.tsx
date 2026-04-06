import {
	CircleIcon,
	EraserIcon,
	LoaderIcon,
	SearchIcon,
	WifiIcon,
	WifiOffIcon,
	XIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Button } from "#/components/ui/button";
import { Input } from "#/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "#/components/ui/select";
import {
	type MessageType,
	useSessionWebSocket,
	type WebSocketMessage,
} from "#/hooks/useSessionWebSocket";
import { cn } from "#/lib/utils";

// Message type color configuration
const messageTypeConfig: Record<MessageType, { color: string; label: string }> =
	{
		// Stage events
		stage_transitioned: {
			color:
				"bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300",
			label: "Stage Transitioned",
		},
		stage_rolled_back: {
			color: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300",
			label: "Stage Rolled Back",
		},
		// Task events
		task_started: {
			color:
				"bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-300",
			label: "Task Started",
		},
		task_completed: {
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
			label: "Task Completed",
		},
		task_failed: {
			color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
			label: "Task Failed",
		},
		task_skipped: {
			color: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300",
			label: "Task Skipped",
		},
		task_retry_started: {
			color:
				"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
			label: "Task Retry",
		},
		// Question events
		question_asked: {
			color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-300",
			label: "Question Asked",
		},
		question_answered: {
			color: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-300",
			label: "Question Answered",
		},
		// Design events
		design_created: {
			color:
				"bg-indigo-100 text-indigo-800 dark:bg-indigo-900 dark:text-indigo-300",
			label: "Design Created",
		},
		design_approved: {
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
			label: "Design Approved",
		},
		design_rejected: {
			color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
			label: "Design Rejected",
		},
		// PR events
		pr_created: {
			color:
				"bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-300",
			label: "PR Created",
		},
		pr_merged: {
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
			label: "PR Merged",
		},
		// Session events
		session_started: {
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
			label: "Session Started",
		},
		session_paused: {
			color:
				"bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
			label: "Session Paused",
		},
		session_resumed: {
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
			label: "Session Resumed",
		},
		session_completed: {
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
			label: "Session Completed",
		},
		session_terminated: {
			color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
			label: "Session Terminated",
		},
		// Control messages
		ping: { color: "bg-gray-100 text-gray-600", label: "Ping" },
		pong: { color: "bg-gray-100 text-gray-600", label: "Pong" },
		error: {
			color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
			label: "Error",
		},
		connected: {
			color:
				"bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
			label: "Connected",
		},
	};

// Event type filter options
const filterOptions = [
	{ value: "all", label: "All Events" },
	{ value: "stage", label: "Stage Events" },
	{ value: "task", label: "Task Events" },
	{ value: "question", label: "Question Events" },
	{ value: "design", label: "Design Events" },
	{ value: "pr", label: "PR Events" },
	{ value: "session", label: "Session Events" },
	{ value: "error", label: "Errors" },
];

// Map filter values to message type prefixes
const filterToPrefixes: Record<string, string[]> = {
	all: [],
	stage: ["stage_"],
	task: ["task_"],
	question: ["question_"],
	design: ["design_"],
	pr: ["pr_"],
	session: ["session_"],
	error: ["error"],
};

// Props for LogViewer component
export interface LogViewerProps {
	sessionId: string;
	maxLogs?: number;
	className?: string;
}

// Format relative time (e.g., "5s ago", "2m ago")
function formatRelativeTime(timestamp: string): string {
	const now = new Date();
	const date = new Date(timestamp);
	const diffMs = now.getTime() - date.getTime();

	if (diffMs < 1000) {
		return "just now";
	}

	const diffSec = Math.floor(diffMs / 1000);
	if (diffSec < 60) {
		return `${diffSec}s ago`;
	}

	const diffMin = Math.floor(diffSec / 60);
	if (diffMin < 60) {
		return `${diffMin}m ago`;
	}

	const diffHour = Math.floor(diffMin / 60);
	if (diffHour < 24) {
		return `${diffHour}h ago`;
	}

	const diffDay = Math.floor(diffHour / 24);
	return `${diffDay}d ago`;
}

// Format timestamp to HH:MM:SS
function formatTime(timestamp: string): string {
	const date = new Date(timestamp);
	return date.toLocaleTimeString("en-US", {
		hour: "2-digit",
		minute: "2-digit",
		second: "2-digit",
		hour12: false,
	});
}

// Truncate payload for display
function truncatePayload(payload: unknown, maxLength = 100): string {
	if (payload === null || payload === undefined) {
		return "";
	}

	const str = typeof payload === "string" ? payload : JSON.stringify(payload);
	if (str.length <= maxLength) {
		return str;
	}
	return `${str.slice(0, maxLength)}...`;
}

// Log entry component
interface LogEntryProps {
	message: WebSocketMessage;
	isExpanded: boolean;
	onToggleExpand: () => void;
}

function LogEntry({ message, isExpanded, onToggleExpand }: LogEntryProps) {
	const config = messageTypeConfig[message.type] || {
		color: "bg-gray-100 text-gray-800",
		label: message.type,
	};
	const hasPayload = message.payload !== null && message.payload !== undefined;

	return (
		<div className="group flex gap-3 py-2 px-3 hover:bg-muted/50 transition-colors">
			{/* Timestamp */}
			<span className="shrink-0 font-mono text-xs text-muted-foreground">
				{formatTime(message.timestamp)}
			</span>

			{/* Event type badge */}
			<span
				className={cn(
					"shrink-0 rounded px-1.5 py-0.5 text-xs font-medium",
					config.color,
				)}
			>
				{config.label}
			</span>

			{/* Payload preview */}
			<div className="flex-1 min-w-0">
				{hasPayload ? (
					<button
						type="button"
						onClick={onToggleExpand}
						className="text-left w-full text-sm text-foreground/80 hover:text-foreground transition-colors"
					>
						{isExpanded ? (
							<pre className="whitespace-pre-wrap break-all text-xs">
								{JSON.stringify(message.payload, null, 2)}
							</pre>
						) : (
							<span className="truncate block">
								{truncatePayload(message.payload)}
							</span>
						)}
					</button>
				) : (
					<span className="text-sm text-muted-foreground italic">
						No payload
					</span>
				)}
			</div>

			{/* Relative time */}
			<span className="shrink-0 text-xs text-muted-foreground">
				{formatRelativeTime(message.timestamp)}
			</span>
		</div>
	);
}

export function LogViewer({
	sessionId,
	maxLogs = 100,
	className,
}: LogViewerProps) {
	// WebSocket connection
	const { messages, isConnected, error, reconnect, clearMessages } =
		useSessionWebSocket(sessionId, {
			maxMessages: maxLogs,
			autoReconnect: true,
			debug: false,
		});

	// UI state
	const [searchQuery, setSearchQuery] = useState("");
	const [filter, setFilter] = useState("all");
	const [expandedMessages, setExpandedMessages] = useState<Set<string>>(
		new Set(),
	);

	// Auto-scroll ref
	const scrollRef = useRef<HTMLDivElement>(null);
	const [autoScroll, setAutoScroll] = useState(true);

	// Filter messages based on search and type filter (memoized for performance)
	const filteredMessages = useMemo(() => {
		return messages.filter((msg) => {
			// Filter by type
			if (filter !== "all") {
				const prefixes = filterToPrefixes[filter];
				if (prefixes && prefixes.length > 0) {
					const matchesType = prefixes.some((prefix) =>
						msg.type.startsWith(prefix),
					);
					if (!matchesType) return false;
				}
			}

			// Filter by search query
			if (searchQuery) {
				const query = searchQuery.toLowerCase();
				const typeMatch = msg.type.toLowerCase().includes(query);
				const payloadMatch =
					msg.payload &&
					JSON.stringify(msg.payload).toLowerCase().includes(query);
				return typeMatch || payloadMatch;
			}

			return true;
		});
	}, [messages, filter, searchQuery]);

	// Auto-scroll to bottom on new messages
	// biome-ignore lint/correctness/useExhaustiveDependencies: we need to scroll when messages change
	useEffect(() => {
		if (autoScroll && scrollRef.current) {
			scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
		}
	}, [messages.length, autoScroll]);

	// Handle scroll to disable auto-scroll if user scrolls up
	const handleScroll = useCallback(() => {
		if (!scrollRef.current) return;
		const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
		const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
		setAutoScroll(isAtBottom);
	}, []);

	// Toggle message expansion
	const toggleExpand = useCallback((timestamp: string) => {
		setExpandedMessages((prev) => {
			const next = new Set(prev);
			if (next.has(timestamp)) {
				next.delete(timestamp);
			} else {
				next.add(timestamp);
			}
			return next;
		});
	}, []);

	// Clear all filters
	const clearFilters = useCallback(() => {
		setSearchQuery("");
		setFilter("all");
	}, []);

	return (
		<div className={cn("flex flex-col h-full relative", className)}>
			{/* Header with controls */}
			<div className="flex flex-col gap-3 border-b border-border p-4">
				{/* Connection status and clear button */}
				<div className="flex items-center justify-between">
					<div className="flex items-center gap-2">
						{isConnected ? (
							<>
								<WifiIcon className="size-4 text-green-500" />
								<span className="text-sm text-muted-foreground">Connected</span>
							</>
						) : (
							<>
								<WifiOffIcon className="size-4 text-red-500" />
								<span className="text-sm text-muted-foreground">
									{error || "Disconnected"}
								</span>
							</>
						)}
					</div>
					<div className="flex items-center gap-2">
						{!isConnected && (
							<Button
								variant="outline"
								size="sm"
								onClick={reconnect}
								className="gap-1"
							>
								<LoaderIcon className="size-3" />
								<span>Reconnect</span>
							</Button>
						)}
						<Button
							variant="outline"
							size="sm"
							onClick={clearMessages}
							className="gap-1"
						>
							<EraserIcon className="size-3" />
							<span>Clear</span>
						</Button>
					</div>
				</div>

				{/* Search and filter */}
				<div className="flex items-center gap-2">
					<div className="relative flex-1">
						<SearchIcon className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
						<Input
							type="text"
							placeholder="Search logs..."
							value={searchQuery}
							onChange={(e) => setSearchQuery(e.target.value)}
							className="pl-9"
						/>
						{searchQuery && (
							<button
								type="button"
								onClick={() => setSearchQuery("")}
								className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
							>
								<XIcon className="size-4" />
							</button>
						)}
					</div>
					<Select value={filter} onValueChange={setFilter}>
						<SelectTrigger className="w-[150px]">
							<SelectValue placeholder="Filter events" />
						</SelectTrigger>
						<SelectContent>
							{filterOptions.map((option) => (
								<SelectItem key={option.value} value={option.value}>
									{option.label}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</div>

				{/* Log count */}
				<div className="text-xs text-muted-foreground">
					Showing {filteredMessages.length} of {messages.length} logs
					{(searchQuery || filter !== "all") && (
						<button
							type="button"
							onClick={clearFilters}
							className="ml-2 text-primary hover:underline"
						>
							Clear filters
						</button>
					)}
				</div>
			</div>

			{/* Log entries */}
			<div
				ref={scrollRef}
				onScroll={handleScroll}
				className="flex-1 overflow-auto"
			>
				{filteredMessages.length === 0 ? (
					<div className="flex flex-col items-center justify-center h-full text-muted-foreground">
						{messages.length === 0 ? (
							<>
								<CircleIcon className="size-8 mb-2 opacity-50" />
								<span className="text-sm">No logs yet</span>
								<span className="text-xs">Waiting for session events...</span>
							</>
						) : (
							<>
								<SearchIcon className="size-8 mb-2 opacity-50" />
								<span className="text-sm">No matching logs</span>
								<button
									type="button"
									onClick={clearFilters}
									className="text-xs text-primary hover:underline mt-1"
								>
									Clear filters
								</button>
							</>
						)}
					</div>
				) : (
					<div className="divide-y divide-border">
						{filteredMessages.map((msg) => (
							<LogEntry
								key={msg.timestamp}
								message={msg}
								isExpanded={expandedMessages.has(msg.timestamp)}
								onToggleExpand={() => toggleExpand(msg.timestamp)}
							/>
						))}
					</div>
				)}
			</div>

			{/* Auto-scroll indicator */}
			{!autoScroll && filteredMessages.length > 0 && (
				<div className="absolute bottom-20 left-1/2 -translate-x-1/2">
					<Button
						variant="secondary"
						size="sm"
						onClick={() => {
							setAutoScroll(true);
							if (scrollRef.current) {
								scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
							}
						}}
						className="shadow-lg"
					>
						Scroll to bottom
					</Button>
				</div>
			)}
		</div>
	);
}
