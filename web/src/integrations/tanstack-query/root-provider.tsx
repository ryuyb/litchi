import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

/**
 * Create QueryClient with optimized defaults for Litchi
 */
export function getQueryClient() {
	return new QueryClient({
		defaultOptions: {
			queries: {
				// Stale time: 30 seconds before data is considered stale
				staleTime: 30 * 1000,
				// Cache time: 5 minutes before unused data is garbage collected
				gcTime: 5 * 60 * 1000,
				// Refetch on window focus for real-time data
				refetchOnWindowFocus: true,
				// Retry failed requests up to 3 times
				retry: 3,
				// Retry delay with exponential backoff
				retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),
			},
			mutations: {
				// Retry mutations once
				retry: 1,
			},
		},
	});
}

/**
 * Get context for TanStack Router SSR integration
 */
export function getContext() {
	const queryClient = getQueryClient();

	return {
		queryClient,
	};
}

/**
 * Query Provider component for wrapping app
 */
export function TanstackQueryProvider({ children }: { children: ReactNode }) {
	const queryClient = getQueryClient();

	return (
		<QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
	);
}

// Default export for backward compatibility
export default TanstackQueryProvider;
