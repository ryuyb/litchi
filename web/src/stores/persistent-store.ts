import { Store } from "@tanstack/store";

/**
 * Helper to create a persistent store with localStorage sync
 * @param key - localStorage key for persistence
 * @param initialState - default state if no persisted value exists
 * @returns Store instance with persistence enabled
 */
export function createPersistentStore<T extends object>(
	key: string,
	initialState: T,
): Store<T> {
	// Load initial state from localStorage if available (only in browser)
	let persistedState: T | undefined;
	if (typeof window !== "undefined") {
		try {
			const stored = localStorage.getItem(key);
			if (stored) {
				persistedState = JSON.parse(stored) as T;
			}
		} catch (e) {
			// Ignore parsing errors, use default state
			console.warn(`Failed to parse persisted state for "${key}":`, e);
		}
	}

	// Create store with persisted or initial state
	const store = new Store<T>(persistedState ?? initialState);

	// Subscribe to changes and persist to localStorage
	if (typeof window !== "undefined") {
		store.subscribe(() => {
			try {
				localStorage.setItem(key, JSON.stringify(store.state));
			} catch (e) {
				console.warn(`Failed to persist state for "${key}":`, e);
			}
		});
	}

	return store;
}

/**
 * Helper to load state from localStorage
 * @param key - localStorage key
 * @param defaultValue - default value if key doesn't exist
 * @returns stored value or default
 */
export function loadFromStorage<T>(key: string, defaultValue: T): T {
	if (typeof window === "undefined") {
		return defaultValue;
	}
	try {
		const stored = localStorage.getItem(key);
		if (stored) {
			return JSON.parse(stored) as T;
		}
	} catch (e) {
		console.warn(`Failed to load from storage "${key}":`, e);
	}
	return defaultValue;
}

/**
 * Helper to save state to localStorage
 * @param key - localStorage key
 * @param value - value to store
 */
export function saveToStorage<T>(key: string, value: T): void {
	if (typeof window === "undefined") {
		return;
	}
	try {
		localStorage.setItem(key, JSON.stringify(value));
	} catch (e) {
		console.warn(`Failed to save to storage "${key}":`, e);
	}
}
