import {
	createFileRoute,
	isRedirect,
	Outlet,
	redirect,
} from "@tanstack/react-router";
import { getApiV1AuthMe } from "#/api/auth/auth";

export const Route = createFileRoute("/_authenticated")({
	beforeLoad: async ({ location }) => {
		try {
			const response = await getApiV1AuthMe();

			// Check if response is successful (status 200)
			if (response.status === 200) {
				return { user: response.data };
			}

			// Any non-200 status means unauthenticated
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
