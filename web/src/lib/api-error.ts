/**
 * API error handling utilities for unified error display.
 * Provides error extraction and toast notification helpers.
 */
import { toast } from "sonner";
import type { ApiError } from "#/api/schemas/apiError";

/**
 * Extract error message from API error response.
 * Handles various error formats and provides fallback messages.
 */
export function extractErrorMessage(
	error: unknown,
	fallback = "An unexpected error occurred",
): string {
	// Handle ApiError type from generated API
	if (error && typeof error === "object") {
		const apiError = error as ApiError;
		if (apiError.message) {
			return apiError.message;
		}
		if (apiError.code) {
			return `Error: ${apiError.code}`;
		}
	}

	// Handle Error instances
	if (error instanceof Error) {
		return error.message;
	}

	// Handle string errors
	if (typeof error === "string") {
		return error;
	}

	// Handle fetch/network errors
	if (error && typeof error === "object" && "status" in error) {
		const response = error as { status?: number; data?: unknown };
		if (response.status === 401) {
			return "Unauthorized. Please check your credentials.";
		}
		if (response.status === 403) {
			return "Forbidden. You don't have permission to access this resource.";
		}
		if (response.status === 404) {
			return "Resource not found.";
		}
		if (response.status === 500) {
			return "Server error. Please try again later.";
		}
		if (response.data && typeof response.data === "object") {
			const data = response.data as ApiError;
			if (data.message) {
				return data.message;
			}
		}
		return `Request failed with status ${response.status}`;
	}

	return fallback;
}

/**
 * Show error toast notification for API errors.
 * Automatically extracts message and shows toast with error styling.
 */
export function showErrorToast(
	error: unknown,
	title?: string,
	options?: { description?: string; duration?: number },
): void {
	const message = extractErrorMessage(error);
	const description = options?.description;

	toast.error(title ?? message, {
		description: title ? message : description,
		duration: options?.duration ?? 5000,
	});
}

/**
 * Show success toast notification.
 */
export function showSuccessToast(
	message: string,
	options?: { description?: string; duration?: number },
): void {
	toast.success(message, {
		description: options?.description,
		duration: options?.duration ?? 3000,
	});
}

/**
 * Show warning toast notification.
 */
export function showWarningToast(
	message: string,
	options?: { description?: string; duration?: number },
): void {
	toast.warning(message, {
		description: options?.description,
		duration: options?.duration ?? 4000,
	});
}
