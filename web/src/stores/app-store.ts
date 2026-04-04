import { createPersistentStore } from "./persistent-store";

// Storage keys
const STORAGE_KEYS = {
	APP_UI: "litchi-app-ui",
	SIDEBAR_STATE: "litchi-sidebar-state",
};

/**
 * Application UI state
 */
export interface AppUIState {
	/** Sidebar collapsed state */
	sidebarCollapsed: boolean;
	/** Sidebar width preference (in pixels) */
	sidebarWidth: number;
	/** Last visited route */
	lastRoute: string;
}

const defaultAppUIState: AppUIState = {
	sidebarCollapsed: false,
	sidebarWidth: 256,
	lastRoute: "/",
};

/**
 * App UI Store - persists user interface preferences
 */
export const appUIStore = createPersistentStore<AppUIState>(
	STORAGE_KEYS.APP_UI,
	defaultAppUIState,
);

/**
 * Sidebar state helpers
 */
export const sidebarActions = {
	toggle: () => {
		appUIStore.setState((prev) => ({
			...prev,
			sidebarCollapsed: !prev.sidebarCollapsed,
		}));
	},
	setCollapsed: (collapsed: boolean) => {
		appUIStore.setState((prev) => ({
			...prev,
			sidebarCollapsed: collapsed,
		}));
	},
	setWidth: (width: number) => {
		appUIStore.setState((prev) => ({
			...prev,
			sidebarWidth: width,
		}));
	},
};

/**
 * Route tracking helper
 */
export const routeActions = {
	setLastRoute: (route: string) => {
		appUIStore.setState((prev) => ({
			...prev,
			lastRoute: route,
		}));
	},
};

// Export store instance and actions
export { STORAGE_KEYS };
