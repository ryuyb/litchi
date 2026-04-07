/**
 * Custom fetch wrapper for Orval-generated API clients.
 * Throws errors for non-2xx responses to properly trigger React Query's onError.
 */

import type { ApiError } from "#/api/schemas/apiError";

export type ErrorType<Error> = Error;

/**
 * Custom fetch implementation that throws on non-2xx responses.
 * Includes credentials for session-based authentication.
 */
export const customFetch = async <T>(
	url: string,
	options?: RequestInit,
): Promise<T> => {
	const response = await fetch(url, {
		...options,
		credentials: "include",
	});

	if (!response.ok) {
		// Parse error body for ApiError
		const errorBody = await response.text();
		const errorData: ApiError = errorBody ? JSON.parse(errorBody) : {};

		// Throw an error that React Query can catch
		throw Object.assign(
			new Error(errorData.message ?? `HTTP error! status: ${response.status}`),
			{
				status: response.status,
				data: errorData,
			},
		);
	}

	// Handle empty responses (204, 205, 304)
	if ([204, 205, 304].includes(response.status)) {
		return {} as T;
	}

	const body = await response.text();
	return body ? JSON.parse(body) : ({} as T);
};
