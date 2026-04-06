import { useCallback, useEffect, useRef, useState } from "react";

// WebSocket message types (mirrors backend message.go)
export type MessageType =
	| "stage_transitioned"
	| "stage_rolled_back"
	| "task_started"
	| "task_completed"
	| "task_failed"
	| "task_skipped"
	| "task_retry_started"
	| "question_asked"
	| "question_answered"
	| "design_created"
	| "design_approved"
	| "design_rejected"
	| "pr_created"
	| "pr_merged"
	| "session_started"
	| "session_paused"
	| "session_resumed"
	| "session_completed"
	| "session_terminated"
	| "ping"
	| "pong"
	| "error"
	| "connected";

// WebSocket message structure
export interface WebSocketMessage {
	type: MessageType;
	payload: unknown;
	timestamp: string;
}

// Error payload structure
export interface ErrorPayload {
	code: string;
	message: string;
}

// Connected payload structure
export interface ConnectedPayload {
	sessionId: string;
}

// Hook options
export interface UseSessionWebSocketOptions {
	// Maximum number of messages to keep in memory
	maxMessages?: number;
	// Enable automatic reconnection on disconnect
	autoReconnect?: boolean;
	// Reconnection delay in milliseconds
	reconnectDelay?: number;
	// Enable console logging for debugging
	debug?: boolean;
}

// Hook return type
export interface UseSessionWebSocketReturn {
	messages: WebSocketMessage[];
	isConnected: boolean;
	error: string | null;
	reconnect: () => void;
	clearMessages: () => void;
}

// Event message types (exclude control messages)
export const eventMessageTypes: MessageType[] = [
	"stage_transitioned",
	"stage_rolled_back",
	"task_started",
	"task_completed",
	"task_failed",
	"task_skipped",
	"task_retry_started",
	"question_asked",
	"question_answered",
	"design_created",
	"design_approved",
	"design_rejected",
	"pr_created",
	"pr_merged",
	"session_started",
	"session_paused",
	"session_resumed",
	"session_completed",
	"session_terminated",
];

// Control message types
export const controlMessageTypes: MessageType[] = [
	"ping",
	"pong",
	"error",
	"connected",
];

/**
 * Hook for WebSocket connection to session progress updates
 * @param sessionId - The session ID to connect to
 * @param options - Configuration options
 */
export function useSessionWebSocket(
	sessionId: string,
	options: UseSessionWebSocketOptions = {},
): UseSessionWebSocketReturn {
	const {
		maxMessages = 100,
		autoReconnect = true,
		reconnectDelay = 3000,
		debug = false,
	} = options;

	const [messages, setMessages] = useState<WebSocketMessage[]>([]);
	const [isConnected, setIsConnected] = useState(false);
	const [error, setError] = useState<string | null>(null);

	// Use refs for values that shouldn't trigger re-renders
	const wsRef = useRef<WebSocket | null>(null);
	const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(
		null,
	);
	const manualReconnectRef = useRef(false);

	// Log helper
	const log = useCallback(
		(...args: unknown[]) => {
			if (debug) {
				console.log("[useSessionWebSocket]", ...args);
			}
		},
		[debug],
	);

	// Add message to state with max limit
	const addMessage = useCallback(
		(message: WebSocketMessage) => {
			setMessages((prev) => {
				const newMessages = [...prev, message];
				// Trim to maxMessages
				if (newMessages.length > maxMessages) {
					return newMessages.slice(-maxMessages);
				}
				return newMessages;
			});
		},
		[maxMessages],
	);

	// Clear messages
	const clearMessages = useCallback(() => {
		setMessages([]);
	}, []);

	// Send pong response to ping
	const sendPong = useCallback(() => {
		if (wsRef.current?.readyState === WebSocket.OPEN) {
			const pongMessage: WebSocketMessage = {
				type: "pong",
				payload: null,
				timestamp: new Date().toISOString(),
			};
			wsRef.current.send(JSON.stringify(pongMessage));
			log("Sent pong");
		}
	}, [log]);

	// Handle incoming message
	const handleMessage = useCallback(
		(event: MessageEvent) => {
			try {
				const message: WebSocketMessage = JSON.parse(event.data);
				log("Received:", message.type);

				// Handle control messages
				if (message.type === "ping") {
					sendPong();
					return; // Don't add ping to messages
				}

				if (message.type === "connected") {
					const payload = message.payload as ConnectedPayload;
					log("Connected with sessionId:", payload.sessionId);
					setIsConnected(true);
					setError(null);
				}

				if (message.type === "error") {
					const payload = message.payload as ErrorPayload;
					setError(payload.message);
					log("Error:", payload.message);
				}

				// Add all messages except ping/pong to the list
				if (message.type !== "pong") {
					addMessage(message);
				}
			} catch (err) {
				log("Failed to parse message:", err);
				// Record parse error as an error message
				addMessage({
					type: "error",
					payload: {
						code: "parse_error",
						message:
							err instanceof Error
								? err.message
								: "Failed to parse WebSocket message",
					},
					timestamp: new Date().toISOString(),
				});
			}
		},
		[log, sendPong, addMessage],
	);

	// Connect to WebSocket
	const connect = useCallback(() => {
		// Don't connect if no sessionId
		if (!sessionId) {
			return;
		}

		// Close existing connection if any
		if (wsRef.current) {
			wsRef.current.close();
			wsRef.current = null;
		}

		// Clear any pending reconnect
		if (reconnectTimeoutRef.current) {
			clearTimeout(reconnectTimeoutRef.current);
			reconnectTimeoutRef.current = null;
		}

		// Build WebSocket URL (relative to current origin)
		const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
		const wsUrl = `${protocol}//${window.location.host}/ws/sessions/${sessionId}`;

		log("Connecting to:", wsUrl);

		try {
			const ws = new WebSocket(wsUrl);
			wsRef.current = ws;

			ws.onopen = () => {
				log("Connection opened");
				// Note: isConnected is set to true when we receive 'connected' message
			};

			ws.onmessage = handleMessage;

			ws.onclose = (event) => {
				log("Connection closed:", event.code, event.reason);
				setIsConnected(false);
				wsRef.current = null;

				// Auto-reconnect if enabled and not manual reconnect
				if (autoReconnect && !manualReconnectRef.current) {
					log(`Reconnecting in ${reconnectDelay}ms...`);
					reconnectTimeoutRef.current = setTimeout(() => {
						connect();
					}, reconnectDelay);
				}
			};

			ws.onerror = (event) => {
				log("WebSocket error:", event);
				setError("WebSocket connection error");
				setIsConnected(false);
			};
		} catch (err) {
			log("Failed to create WebSocket:", err);
			setError(
				err instanceof Error ? err.message : "Failed to create WebSocket",
			);
		}
	}, [sessionId, autoReconnect, reconnectDelay, log, handleMessage]);

	// Manual reconnect function
	const reconnect = useCallback(() => {
		manualReconnectRef.current = true;
		connect();
		// Reset after a short delay to allow auto-reconnect to work again
		setTimeout(() => {
			manualReconnectRef.current = false;
		}, 1000);
	}, [connect]);

	// Connect on mount, disconnect on unmount
	useEffect(() => {
		connect();

		return () => {
			manualReconnectRef.current = true; // Prevent auto-reconnect on cleanup
			if (reconnectTimeoutRef.current) {
				clearTimeout(reconnectTimeoutRef.current);
			}
			if (wsRef.current) {
				wsRef.current.close();
				wsRef.current = null;
			}
		};
	}, [connect]);

	return {
		messages,
		isConnected,
		error,
		reconnect,
		clearMessages,
	};
}
