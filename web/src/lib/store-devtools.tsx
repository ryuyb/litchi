import { EventClient } from "@tanstack/devtools-event-client";
import { appSettingsStore, appUIStore } from "../stores";

type EventMap = {
	"store-devtools:ui-state": {
		sidebarCollapsed: boolean;
		sidebarWidth: number;
		lastRoute: string;
	};
	"store-devtools:settings": {
		notificationsEnabled: boolean;
		autoRefreshInterval: number;
		showDevtools: boolean;
		defaultRetryLimit: number;
		defaultTimeoutMinutes: number;
	};
};

class StoreDevtoolsEventClient extends EventClient<EventMap> {
	constructor() {
		super({
			pluginId: "litchi-store-devtools",
		});
	}
}

const sdec = new StoreDevtoolsEventClient();

// Subscribe to store changes and emit events
appUIStore.subscribe(() => {
	sdec.emit("store-devtools:ui-state", {
		sidebarCollapsed: appUIStore.state.sidebarCollapsed,
		sidebarWidth: appUIStore.state.sidebarWidth,
		lastRoute: appUIStore.state.lastRoute,
	});
});

appSettingsStore.subscribe(() => {
	sdec.emit("store-devtools:settings", {
		notificationsEnabled: appSettingsStore.state.notificationsEnabled,
		autoRefreshInterval: appSettingsStore.state.autoRefreshInterval,
		showDevtools: appSettingsStore.state.showDevtools,
		defaultRetryLimit: appSettingsStore.state.defaultRetryLimit,
		defaultTimeoutMinutes: appSettingsStore.state.defaultTimeoutMinutes,
	});
});

function DevtoolPanel() {
	return (
		<div className="p-4">
			<h3 className="text-sm font-bold text-gray-700 mb-2">Litchi Stores</h3>
			<p className="text-xs text-gray-500">
				UI and Settings stores are persisted to localStorage.
			</p>
		</div>
	);
}

export default {
	name: "Litchi Stores",
	render: <DevtoolPanel />,
};
