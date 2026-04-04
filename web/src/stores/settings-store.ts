import { createPersistentStore } from "./persistent-store";

/**
 * Application settings state
 */
export interface AppSettingsState {
	/** Enable notifications */
	notificationsEnabled: boolean;
	/** Auto-refresh interval in seconds (0 = disabled) */
	autoRefreshInterval: number;
	/** Show devtools panel */
	showDevtools: boolean;
	/** Default retry limit for tasks */
	defaultRetryLimit: number;
	/** Default timeout in minutes */
	defaultTimeoutMinutes: number;
}

const defaultSettingsState: AppSettingsState = {
	notificationsEnabled: true,
	autoRefreshInterval: 30,
	showDevtools: true,
	defaultRetryLimit: 3,
	defaultTimeoutMinutes: 30,
};

/**
 * App Settings Store - persists user preferences
 */
export const appSettingsStore = createPersistentStore<AppSettingsState>(
	"litchi-app-settings",
	defaultSettingsState,
);

/**
 * Settings update helpers
 */
export const settingsActions = {
	setNotificationsEnabled: (enabled: boolean) => {
		appSettingsStore.setState((prev) => ({
			...prev,
			notificationsEnabled: enabled,
		}));
	},
	setAutoRefreshInterval: (interval: number) => {
		appSettingsStore.setState((prev) => ({
			...prev,
			autoRefreshInterval: interval,
		}));
	},
	setShowDevtools: (show: boolean) => {
		appSettingsStore.setState((prev) => ({
			...prev,
			showDevtools: show,
		}));
	},
	setDefaultRetryLimit: (limit: number) => {
		appSettingsStore.setState((prev) => ({
			...prev,
			defaultRetryLimit: limit,
		}));
	},
	setDefaultTimeoutMinutes: (minutes: number) => {
		appSettingsStore.setState((prev) => ({
			...prev,
			defaultTimeoutMinutes: minutes,
		}));
	},
	reset: () => {
		appSettingsStore.setState(() => defaultSettingsState);
	},
};
