import {
	createFileRoute,
	isRedirect,
	Outlet,
	redirect,
} from "@tanstack/react-router";
import { authActions, authStore } from "#/stores";

export const Route = createFileRoute("/_authenticated")({
	beforeLoad: async ({ location }) => {
		try {
			// Use auth store to check authentication
			const isAuthenticated = await authActions.checkAuth();

			if (isAuthenticated) {
				// Return user from store for route context
				return { user: authStore.state.user };
			}

			// Not authenticated, redirect to login
			throw redirect({
				to: "/login",
				search: { redirect: location.href },
			});
		} catch (error) {
			// If it's already a redirect, re-throw it
			if (isRedirect(error)) {
				throw error;
			}

			// Auth check failed (401, 500, network error, etc.)
			throw redirect({
				to: "/login",
				search: { redirect: location.href },
			});
		}
	},
	component: () => <Outlet />,
});
