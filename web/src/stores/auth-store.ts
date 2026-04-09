import { Store } from "@tanstack/store";
import { toast } from "sonner";
import {
	getApiV1AuthMe,
	postApiV1AuthLogin,
	postApiV1AuthLogout,
} from "#/api/auth/auth";
import type { AuthUserResponse } from "#/api/schemas";
import { extractErrorMessage } from "#/lib/api-error";

const AUTH_STORAGE_KEY = "litchi-auth-user";

export interface AuthState {
	/** Current authenticated user */
	user: AuthUserResponse | null;
	/** Loading state for auth operations */
	isLoading: boolean;
	/** Whether user is authenticated */
	isAuthenticated: boolean;
	/** Error from last auth operation */
	error: string | null;
}

const defaultAuthState: AuthState = {
	user: null,
	isLoading: false,
	isAuthenticated: false,
	error: null,
};

/**
 * Load user from localStorage (optimistic hydration)
 */
function loadPersistedUser(): AuthUserResponse | null {
	if (typeof window === "undefined") {
		return null;
	}
	try {
		const stored = localStorage.getItem(AUTH_STORAGE_KEY);
		if (stored) {
			return JSON.parse(stored) as AuthUserResponse;
		}
	} catch (e) {
		console.warn("Failed to load persisted auth user:", e);
	}
	return null;
}

/**
 * Persist user to localStorage
 */
function persistUser(user: AuthUserResponse | null): void {
	if (typeof window === "undefined") {
		return;
	}
	try {
		if (user) {
			localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(user));
		} else {
			localStorage.removeItem(AUTH_STORAGE_KEY);
		}
	} catch (e) {
		console.warn("Failed to persist auth user:", e);
	}
}

/**
 * Auth Store - manages authentication state
 */
export const authStore = new Store<AuthState>({
	...defaultAuthState,
	// Optimistic hydration from localStorage
	user: loadPersistedUser(),
	isAuthenticated: loadPersistedUser() !== null,
});

/**
 * Auth actions
 */
export const authActions = {
	/**
	 * Check authentication status with the server
	 * Call this on app init or route guard
	 */
	checkAuth: async (): Promise<boolean> => {
		authStore.setState((prev) => ({ ...prev, isLoading: true, error: null }));

		try {
			const response = await getApiV1AuthMe();

			if (response.status === 200) {
				authStore.setState((prev) => ({
					...prev,
					user: response.data,
					isLoading: false,
					isAuthenticated: true,
					error: null,
				}));
				persistUser(response.data);
				return true;
			}

			// Not authenticated
			authStore.setState((prev) => ({
				...prev,
				user: null,
				isLoading: false,
				isAuthenticated: false,
				error: null,
			}));
			persistUser(null);
			return false;
		} catch {
			// Auth check failed (401, 500, network error)
			authStore.setState((prev) => ({
				...prev,
				user: null,
				isLoading: false,
				isAuthenticated: false,
				error: null, // Don't show error for background auth checks
			}));
			persistUser(null);
			return false;
		}
	},

	/**
	 * Login with credentials
	 */
	login: async (username: string, password: string): Promise<boolean> => {
		authStore.setState((prev) => ({ ...prev, isLoading: true, error: null }));

		try {
			const response = await postApiV1AuthLogin({ username, password });

			if (response.status === 200) {
				// After successful login, fetch user info
				const meResponse = await getApiV1AuthMe();
				const user = meResponse.status === 200 ? meResponse.data : null;

				authStore.setState((prev) => ({
					...prev,
					user,
					isLoading: false,
					isAuthenticated: true,
					error: null,
				}));
				persistUser(user);
				return true;
			}

			// Login failed (non-200 status)
			authStore.setState((prev) => ({
				...prev,
				isLoading: false,
				error: "Login failed. Please check your credentials.",
			}));
			toast.error("Login failed. Please check your credentials.");
			return false;
		} catch (error) {
			const message = extractErrorMessage(error);
			authStore.setState((prev) => ({
				...prev,
				isLoading: false,
				error: message,
			}));
			toast.error(message);
			throw error;
		}
	},

	/**
	 * Logout and clear session
	 */
	logout: async (): Promise<void> => {
		authStore.setState((prev) => ({ ...prev, isLoading: true }));

		try {
			await postApiV1AuthLogout();
		} catch {
			// Ignore logout API errors, still clear local state
		}

		authStore.setState((prev) => ({
			...prev,
			user: null,
			isLoading: false,
			isAuthenticated: false,
			error: null,
		}));
		persistUser(null);
	},

	/**
	 * Clear any auth error
	 */
	clearError: (): void => {
		authStore.setState((prev) => ({ ...prev, error: null }));
	},

	/**
	 * Hydrate auth state from localStorage (call on app init)
	 */
	hydrate: (): void => {
		const user = loadPersistedUser();
		if (user) {
			authStore.setState((prev) => ({
				...prev,
				user,
				isAuthenticated: true,
			}));
		}
	},
};
